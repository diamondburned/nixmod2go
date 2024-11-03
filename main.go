package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"

	"github.com/diamondburned/gotk4/gir/girgen/strcases"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
	"libdb.so/nixmod2go/nixmod2go"
	"libdb.so/nixmod2go/nixmodule"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(ctx, err.Error())
		os.Exit(1)
	}
}

var cmd = &cli.Command{
	Name:      "nixmod2go",
	Usage:     "parse and generate Go struct definitions from Nix modules",
	ArgsUsage: "<module-path> [output-file]",
	Before:    appBefore,
	Action:    appAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:      "format",
			Aliases:   []string{"f"},
			Usage:     "output format",
			Value:     "go",
			Validator: enumValidator("go", "json"),
		},
		&cli.StringSliceFlag{
			Name:  "initials",
			Usage: "list of words that should be all-caps, such as API or URL",
		},
		&cli.StringMapFlag{
			Name:  "initials-replace",
			Usage: "like initials, but with a replacement instead of all-caps",
		},
		&cli.StringFlag{
			Name:  "flake",
			Usage: "path to flake (default: current)",
			Value: ".",
		},
		&cli.StringFlag{
			Name:  "flake-pkgs",
			Usage: "the input name of the Nixpkgs to use in the flake (must be a root input)",
			Value: "nixpkgs",
		},
		&cli.StringFlag{
			Name:  "pkgs",
			Usage: "Nix expression to specify Nixpkgs (default: current flake's Nixpkgs or <nixpkgs>)",
			Action: func(ctx context.Context, cmd *cli.Command, value string) error {
				return nixmodule.NixExpr(value).Validate(ctx)
			},
		},
		&cli.BoolFlag{
			Name:  "json-pretty",
			Usage: "pretty print JSON output",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "go-package",
			Usage: "the package name of the generated Go file",
			Value: "main",
		},
		&cli.StringFlag{
			Name:  "go-type-name",
			Usage: "the type name of the generated root Go struct",
			Value: "Config",
		},
		&cli.StringFlag{
			Name:  "options-path",
			Usage: "dot-separated path to the options module to generate, default to all",
		},
		&cli.StringMapFlag{
			Name:  "special-args",
			Usage: "special arguments to pass to the module, one key=value pair per flag",
			Action: func(ctx context.Context, cmd *cli.Command, value map[string]string) error {
				for k, v := range value {
					if err := nixmodule.NixExpr(v).Validate(ctx); err != nil {
						return fmt.Errorf("expression error at %q: %w", k, err)
					}
				}
				return nil
			},
		},
		&cli.BoolFlag{
			Name:  "special-args-pkgs",
			Usage: "add pkgs to special-args",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "special-args-self",
			Usage: "add current flake (as self) to special-args, errors if not in a flake",
			Value: true,
		},
		&cli.BoolFlag{
			Name:    "expr",
			Aliases: []string{"E"},
			Usage:   "treat module-path as a Nix expression",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "enable verbose output",
		},
	},
}

func appBefore(ctx context.Context, cmd *cli.Command) error {
	logLevel := slog.LevelInfo
	if cmd.Bool("verbose") {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:   logLevel,
		NoColor: os.Getenv("NO_COLOR") != "" || !isatty.IsTerminal(os.Stderr.Fd()),
	}))
	slog.SetDefault(logger)

	strcases.AddPascalSpecials(cmd.StringSlice("initials"))
	strcases.SetPascalWords(cmd.StringMap("initials-replace"))

	return nil
}

func appAction(ctx context.Context, cmd *cli.Command) error {
	if !cmd.Args().Present() {
		cli.ShowAppHelp(cmd)
		return cli.Exit("invalid usage", 1)
	}

	var input nixmodule.ModuleInput
	if cmd.Bool("expr") {
		input = nixmodule.ModuleExpr(cmd.Args().Get(0))
	} else {
		input = nixmodule.ModulePath(cmd.Args().Get(0))
	}

	flake, err := getFlakeInfo(ctx, cmd.String("flake"))
	if err != nil {
		return fmt.Errorf("flake error: %w", err)
	}

	if flake != nil {
		slog.Debug(
			"using current flake",
			"url", flake.URL,
			"inputs", slices.Collect(maps.Keys(flake.Locks.Nodes)))
	}

	pkgsExpr, err := pkgsExpr(ctx, cmd, pkgsOpts{Flake: flake})
	if err != nil {
		return fmt.Errorf("pkgs expression: %w", err)
	}

	specialArgs := map[string]nixmodule.NixExpr{}
	for k, v := range cmd.StringMap("special-args") {
		specialArgs[k] = nixmodule.NixExpr(v)
	}

	if cmd.Bool("special-args-pkgs") {
		specialArgs["pkgs"] = pkgsExpr
		slog.Debug(
			"added pkgs to special-args",
			"pkgs", specialArgs["pkgs"])
	}

	if cmd.Bool("special-args-self") {
		specialArgs["self"] = flake.flakeExpr()
		slog.Debug(
			"added self to special-args",
			"self", specialArgs["self"])
	}

	var optionsPath []string
	if strPath := cmd.String("options-path"); strPath != "" {
		optionsPath = strings.Split(strPath, ".")
	}

	module, err := nixmodule.DumpModule(ctx, input,
		nixmodule.DumpModuleWithPkgs(pkgsExpr),
		nixmodule.DumpModuleWithSpecialArgs(specialArgs),
		nixmodule.DumpModuleWithOptionsPath(optionsPath))
	if err != nil {
		return err
	}

	var o io.Writer = os.Stdout
	if output := cmd.Args().Get(1); output != "" {
		if filepath.Ext(output) == "" {
			output += "." + cmd.String("format")
		}

		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("unable to create output file: %w", err)
		}
		defer f.Close()
		o = f
	}

	switch format := cmd.String("format"); format {
	case "json":
		jsonOpts := nixmodule.JSONOptions
		if cmd.Bool("json-pretty") {
			jsonOpts = json.JoinOptions(jsonOpts, jsontext.WithIndent("  "))
		}

		if err := json.MarshalWrite(o, module, jsonOpts); err != nil {
			return fmt.Errorf("JSON marshal error: %w", err)
		}
	case "go":
		goPackage := cmd.String("go-package")
		goOpts := nixmod2go.Opts{RootName: cmd.String("go-type-name")}

		code, err := nixmod2go.Generate(module, goPackage, goOpts)
		if err != nil {
			return fmt.Errorf("Go generate error: %w", err)
		}

		if _, err := io.WriteString(o, code); err != nil {
			return fmt.Errorf("cannot write to file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format %q", format)
	}

	return nil
}

func enumValidator[T comparable](vs ...T) func(T) error {
	return func(v T) error {
		if slices.Index(vs, v) == -1 {
			return fmt.Errorf("value %v is not in %v", v, vs)
		}
		return nil
	}
}
