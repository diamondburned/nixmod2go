package example_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/google/go-cmp/cmp"
	"libdb.so/nixmod2go/example"
)

func TestUnmarshal(t *testing.T) {
	// Testing strategy:
	// 1. Unmarshal the Nix-generated JSON into the Go struct.
	// 2. Marshal the Go struct back into JSON.
	// 3. Compare the original JSON with the marshaled JSON (both
	//    canonicalized).

	nixConfig, err := evalExampleConfig(context.Background())
	assert.NoError(t, err, "evalExampleConfig failed")

	var m example.Config
	err = json.Unmarshal(nixConfig, &m)
	assert.NoError(t, err, "json.Unmarshal failed")

	goConfig, err := json.Marshal(m)
	assert.NoError(t, err, "json.Marshal failed")

	nixConfig = canonicalizeJSON(t, nixConfig)
	goConfig = canonicalizeJSON(t, goConfig)

	if diff := cmp.Diff(nixConfig, goConfig); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func canonicalizeJSON(t *testing.T, in []byte) []byte {
	var v any

	err := json.Unmarshal(in, &v)
	assert.NoError(t, err, "json.Unmarshal failed")

	b, err := json.MarshalIndent(v, "", "  ")
	assert.NoError(t, err, "json.MarshalIndent failed")

	return b
}

func evalExampleConfig(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "nix", "eval", "--json", ".#lib.exampleModule.config")

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, err
		}
		return nil, fmt.Errorf(
			"nix eval failed with exit code %d: %s",
			exitErr.ExitCode(), string(exitErr.Stderr),
		)
	}

	return output, nil
}
