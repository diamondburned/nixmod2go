package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"
	"libdb.so/nixmod2go/nixmodule"
)

const systemPkgsExpr nixmodule.NixExpr = `import <nixpkgs> { }`

type pkgsOpts struct {
	Flake *flakeInfo
}

func pkgsExpr(ctx context.Context, cmd *cli.Command, opts pkgsOpts) (nixmodule.NixExpr, error) {
	if pkgs := cmd.String("pkgs"); pkgs != "" {
		return nixmodule.NixExpr(pkgs), nil
	}

	if opts.Flake != nil {
		e, err := pkgsExprFromFlake(ctx, opts.Flake, cmd.String("flake-pkgs"))
		if err != nil {
			return "", err
		}

		slog.DebugContext(ctx,
			"using nixpkgs from flake's input",
			"flake", opts.Flake.URL,
			"flake-pkgs", cmd.String("flake-pkgs"))

		return e, nil
	}

	slog.DebugContext(ctx,
		"using system nixpkgs",
		"nixpkgs", "<nixpkgs>",
		"flake", false)

	return systemPkgsExpr, nil
}

func pkgsExprFromFlake(ctx context.Context, flake *flakeInfo, flakeInput string) (nixmodule.NixExpr, error) {
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
		"flake-pkgs", flakeInput)

	if err := pkgsExpr.Validate(ctx); err != nil {
		return "", fmt.Errorf("validate Nix expression: %w", err)
	}

	return pkgsExpr, nil
}
