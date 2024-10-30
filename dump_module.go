package nixmod2go

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/go-json-experiment/json"
)

//go:embed dump_module.nix
var dumpModuleNix NixExpr

type argAdder interface {
	add(ctx context.Context, cmd *exec.Cmd) error
}

// DumpModuleInput represents an input to [DumpModule].
// It can be either [DumpModulePath] or [DumpModuleExpr].
type DumpModuleInput interface {
	isDumpModuleInput()
	argAdder
}

func (DumpModulePath) isDumpModuleInput() {}
func (DumpModuleExpr) isDumpModuleInput() {}

// DumpModulePath represents a path to a Nix module file.
// The path doesn't have to be absolute.
type DumpModulePath string

func (p DumpModulePath) add(ctx context.Context, cmd *exec.Cmd) error {
	abs, err := filepath.Abs(string(p))
	if err != nil {
		return fmt.Errorf("resolve module path: %w", err)
	}

	cmd.Args = append(cmd.Args, "--arg", "module", abs)
	return nil
}

// DumpModuleExpr represents a Nix expression that evaluates to a Nix module.
// The expression must be a function that accepts a `lib` argument and returns a
// Nix module.
type DumpModuleExpr NixExpr

func (e DumpModuleExpr) add(ctx context.Context, cmd *exec.Cmd) error {
	if err := NixExpr(e).tryParsing(ctx); err != nil {
		return fmt.Errorf("parse module expression: %w", err)
	}
	cmd.Args = append(cmd.Args, "--arg", "module", string(e))
	return nil
}

// DumpModuleOpt represents an option for [DumpModule].
type DumpModuleOpt interface {
	isDumpModuleOpt()
	argAdder
}

type dumpModuleOptFunc func(ctx context.Context, cmd *exec.Cmd) error

func (f dumpModuleOptFunc) isDumpModuleOpt()                             {}
func (f dumpModuleOptFunc) add(ctx context.Context, cmd *exec.Cmd) error { return f(ctx, cmd) }

// DumpModuleWithPkgs adds a `pkgs` argument to the Nix expression.
// By default, <nixpkgs> is used.
func DumpModuleWithPkgs(expr NixExpr) DumpModuleOpt {
	return dumpModuleOptFunc(func(ctx context.Context, cmd *exec.Cmd) error {
		if err := expr.tryParsing(ctx); err != nil {
			return fmt.Errorf("parse pkgs expression: %w", err)
		}
		cmd.Args = append(cmd.Args, "--arg", "pkgs", string(expr))
		return nil
	})
}

// DumpModuleWithSpecialArgs adds a `specialArgs` argument to the Nix expression.
// By default, { } is used.
func DumpModuleWithSpecialArgs(expr NixExpr) DumpModuleOpt {
	return dumpModuleOptFunc(func(ctx context.Context, cmd *exec.Cmd) error {
		if err := expr.tryParsing(ctx); err != nil {
			return fmt.Errorf("parse specialArgs expression: %w", err)
		}
		cmd.Args = append(cmd.Args, "--arg", "specialArgs", string(expr))
		return nil
	})
}

// DumpModuleWithStderrPassthrough redirects the standard error of the
// `nix-instantiate` command to the standard error of the current process.
func DumpModuleWithStderrPassthrough() DumpModuleOpt {
	return dumpModuleOptFunc(func(ctx context.Context, cmd *exec.Cmd) error {
		cmd.Stderr = os.Stderr
		cmd.Args = append(cmd.Args, "--show-trace")
		return nil
	})
}

// DumpModule evaluates a Nix module and returns its representation as a
// [Module].
func DumpModule(ctx context.Context, module DumpModuleInput, opts ...DumpModuleOpt) (Module, error) {
	cmd := exec.CommandContext(ctx,
		"nix-instantiate",
		"--eval", "--strict", "--json",
		"--expr", string(dumpModuleNix))

	if err := module.add(ctx, cmd); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt.add(ctx, cmd); err != nil {
			return nil, err
		}
	}

	var stderr strings.Builder
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("get stdout pipe: %w", err)
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start nix-instantiate: %w", err)
	}

	var m Module
	jsonErr := json.UnmarshalRead(stdout, &m, JSONOptions)

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			nixErrorMatch := nixErrorRe.FindStringSubmatch(stderr.String())
			if nixErrorMatch != nil {
				return nil, fmt.Errorf("nix-instantiate: %s", nixErrorMatch[1])
			}
			return nil, fmt.Errorf("nix-instantiate: %s", stderr.String())
		}
		return nil, fmt.Errorf("nix-instantiate: %w", err)
	}

	if jsonErr != nil {
		return nil, fmt.Errorf("unmarshal module: %w", jsonErr)
	}

	return m, nil
}
