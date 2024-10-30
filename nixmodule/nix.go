package nixmodule

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// NixExpr represents a Nix expression.
// It exists for documentation purposes only.
type NixExpr string

func (e NixExpr) tryParsing(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "nix-instantiate", "--parse", "--expr", string(e))

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			nixErrorMatch := nixErrorRe.FindStringSubmatch(stderr.String())
			if nixErrorMatch != nil {
				return fmt.Errorf("nix expression error: %s", nixErrorMatch[1])
			}
			return fmt.Errorf("nix-instantiate: %s", stderr.String())
		}
		return fmt.Errorf("nix-instantiate: %w", err)
	}

	return nil
}

var nixErrorRe = regexp.MustCompile(`(?m)^\s*error: (.*)$`)
