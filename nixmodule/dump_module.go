package nixmodule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// ModuleInput represents an input to [DumpModule].
// It can be either [ModulePath] or [ModuleExpr].
type ModuleInput interface {
	isDumpModuleInput()
	argAdder
}

func (ModulePath) isDumpModuleInput() {}
func (ModuleExpr) isDumpModuleInput() {}

// ModulePath represents a path to a Nix module file.
// The path doesn't have to be absolute.
type ModulePath string

func (p ModulePath) add(ctx context.Context, cmd *exec.Cmd) error {
	abs, err := filepath.Abs(string(p))
	if err != nil {
		return fmt.Errorf("resolve module path: %w", err)
	}

	cmd.Args = append(cmd.Args, "--arg", "module", abs)
	return nil
}

// ModuleExpr represents a Nix expression that evaluates to a Nix module.
// The expression must be a function that accepts a `lib` argument and returns a
// Nix module.
type ModuleExpr NixExpr

func (e ModuleExpr) add(ctx context.Context, cmd *exec.Cmd) error {
	if err := NixExpr(e).Validate(ctx); err != nil {
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
		if err := expr.Validate(ctx); err != nil {
			return fmt.Errorf("parse pkgs expression: %w", err)
		}
		cmd.Args = append(cmd.Args, "--arg", "pkgs", string(expr))
		return nil
	})
}

// DumpModuleWithSpecialArgs adds a `specialArgs` argument to the Nix expression.
// By default, { } is used.
func DumpModuleWithSpecialArgs(specialArgs map[string]NixExpr) DumpModuleOpt {
	return dumpModuleOptFunc(func(ctx context.Context, cmd *exec.Cmd) error {
		var b strings.Builder
		b.WriteString("{ ")
		for k, v := range specialArgs {
			if err := v.Validate(ctx); err != nil {
				return fmt.Errorf("specialArgs: error at %q: %w", k, err)
			}
			fmt.Fprintf(&b, "%q = %s; ", k, v)
		}
		b.WriteString(" }")

		slog.DebugContext(ctx,
			"built specialArgs expression",
			"specialArgs", b.String())

		if err := NixExpr(b.String()).Validate(ctx); err != nil {
			return fmt.Errorf("specialArgs: %w", err)
		}

		cmd.Args = append(cmd.Args, "--arg", "specialArgs", b.String())
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

// DumpModuleOpts represents a list of [DumpModuleOpt].
type DumpModuleOpts []DumpModuleOpt

// Add appends more options to the list.
func (opts *DumpModuleOpts) Add(more ...DumpModuleOpt) {
	*opts = append(*opts, more...)
}

func (opts DumpModuleOpts) add(ctx context.Context, cmd *exec.Cmd) error {
	for _, opt := range opts {
		if err := opt.add(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (opts DumpModuleOpts) isDumpModuleOpt() {}

// DumpModule evaluates a Nix module and returns its representation as a
// [Module].
func DumpModule(ctx context.Context, module ModuleInput, opts ...DumpModuleOpt) (Module, error) {
	return dumpModuleAs[Module](ctx, module, opts...)
}

func dumpModuleAs[T any](ctx context.Context, module ModuleInput, opts ...DumpModuleOpt) (T, error) {
	var v T

	cmd := exec.CommandContext(ctx,
		"nix-instantiate",
		"--eval", "--strict", "--json", "--expr")

	if err := module.add(ctx, cmd); err != nil {
		return v, err
	}

	for _, opt := range opts {
		if err := opt.add(ctx, cmd); err != nil {
			return v, err
		}
	}

	cmd.Args = append(cmd.Args, string(dumpModuleNix))

	var stderr strings.Builder
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return v, fmt.Errorf("get stdout pipe: %w", err)
	}
	defer stdout.Close()

	slog.DebugContext(ctx,
		"running nix-instantiate to dump modules",
		"args", cmd.Args[:len(cmd.Args)-1])

	if err := cmd.Start(); err != nil {
		return v, fmt.Errorf("start nix-instantiate: %w", err)
	}

	jsonErr := json.UnmarshalRead(stdout, &v, JSONOptions)

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			nixErrorMatch := nixErrorRe.FindStringSubmatch(stderr.String())
			if nixErrorMatch == nil {
				return v, fmt.Errorf("nix-instantiate: %s", stderr.String())
			}

			nixError := string(nixErrorMatch[1])
			return v, fmt.Errorf("nix-instantiate: %s", nixError)
		}

		return v, fmt.Errorf("nix-instantiate: %w", err)
	}

	if jsonErr != nil {
		return v, fmt.Errorf("unmarshal module: %w", jsonErr)
	}

	return v, nil
}
