package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/go-json-experiment/json"
	"github.com/urfave/cli/v3"
	"libdb.so/nixmod2go/nixmodule"
)

const systemPkgsExpr nixmodule.NixExpr = `import <nixpkgs> { }`

func pkgsExpr(ctx context.Context, cmd *cli.Command) (nixmodule.NixExpr, error) {
	if pkgs := cmd.String("pkgs"); pkgs != "" {
		return nixmodule.NixExpr(pkgs), nil
	}

	if cmd.String("flake") != "" {
		flake, err := getFlakeInfo(ctx, cmd.String("flake"))
		if err == nil {
			e, err := pkgsExprFromFlake(ctx, flake, cmd.String("flake-input"))
			if err != nil {
				return "", err
			}

			slog.DebugContext(ctx,
				"using nixpkgs from flake's input",
				"flake", flake.URL,
				"flake-input", cmd.String("flake-input"))

			return e, nil
		}
	}

	slog.DebugContext(ctx,
		"using system nixpkgs",
		"nixpkgs", "<nixpkgs>",
		"flake", false)

	return systemPkgsExpr, nil
}

func pkgsExprFromFlake(ctx context.Context, flake flakeInfo, flakeInput string) (nixmodule.NixExpr, error) {
	if _, hasNixpkgs := flake.Locks.Nodes[flakeInput]; !hasNixpkgs {
		return "", fmt.Errorf("flake doesn't have a nixpkgs input")
	}

	pkgsExpr := nixmodule.NixExpr(fmt.Sprintf(
		`import (%s).inputs.%q { }`,
		flake.flakeExpr(), flakeInput,
	))

	slog.DebugContext(ctx,
		"created pkgs expression from flake input",
		"expr", pkgsExpr,
		"flake", flake.URL,
		"flake-input", flakeInput)

	if err := pkgsExpr.Validate(ctx); err != nil {
		return "", fmt.Errorf("validate Nix expression: %w", err)
	}

	return pkgsExpr, nil
}

type flakeInfo struct {
	Locks flakeLocks `json:"locks"`
	URL   string     `json:"url"`
}

type flakeLocks struct {
	Version int                      `json:"version"`
	Nodes   map[string]flakeLockNode `json:"nodes"`
}

type flakeLockNode struct {
	Locked struct {
		NARHash      string `json:"narHash"`
		LastModified int    `json:"lastModified"`
	} `json:"locked"`
}

type flakeNixpkgsData struct {
	Flake      flakeInfo
	FlakeInput string
}

func getFlakeInfo(ctx context.Context, path string) (flakeInfo, error) {
	var flake flakeInfo

	out, err := exec.CommandContext(ctx,
		"nix", "flake", "metadata", "--json", path).Output()
	if err != nil {
		return flake, err
	}

	if err := json.Unmarshal(out, &flake); err != nil {
		return flake, fmt.Errorf("unmarshal flake info: %w", err)
	}

	return flake, nil
}

func (f flakeInfo) String() string {
	return f.URL
}

func (f flakeInfo) flakeExpr() nixmodule.NixExpr {
	return nixmodule.NixExpr(fmt.Sprintf(`builtins.getFlake %q`, f.URL))
}
