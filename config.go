package main

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

func setValue(cmd *cli.Command, k string, v any) error {
	switch v := v.(type) {
	case map[string]any:
		for k2, v2 := range v {
			if err := cmd.Set(k, fmt.Sprintf("%s=%v", k2, v2)); err != nil {
				return fmt.Errorf("error at key %q: %w", k, err)
			}
		}
	case []any:
		for i, v := range v {
			if err := cmd.Set(k, fmt.Sprint(v)); err != nil {
				return fmt.Errorf("error at index %d: %w", i, err)
			}
		}
	default:
		return cmd.Set(k, fmt.Sprint(v))
	}
	return nil
}
