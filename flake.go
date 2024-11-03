package main

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-json-experiment/json"
	"github.com/urfave/cli/v3"
	"libdb.so/nixmod2go/nixmodule"
)

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

func getFlakeInfo(ctx context.Context, path string) (*flakeInfo, error) {
	out, err := exec.CommandContext(ctx,
		"nix", "flake", "metadata", "--json", path).Output()
	if err != nil {
		return nil, err
	}

	var flake flakeInfo
	if err := json.Unmarshal(out, &flake); err != nil {
		return nil, fmt.Errorf("unmarshal flake info: %w", err)
	}

	return &flake, nil
}

func currentFlake(ctx context.Context, cmd *cli.Command) (*flakeInfo, error) {
	flake := cmd.String("flake")
	if flake == "" {
		return nil, nil
	}
	return getFlakeInfo(ctx, cmd.String("flake"))
}

func (f flakeInfo) String() string {
	return f.URL
}

func (f flakeInfo) flakeExpr() nixmodule.NixExpr {
	return nixmodule.NixExpr(fmt.Sprintf(`builtins.getFlake %q`, f.URL))
}
