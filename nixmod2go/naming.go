package nixmod2go

import (
	"strings"

	"github.com/diamondburned/gotk4/gir/girgen/strcases"
)

// optionName is a type for naming conversions.
type optionName struct {
	Nix string // original name
	Go  string // Go name
}

func parseName(s string) optionName {
	switch {
	case strings.Contains(s, "-"):
		return optionName{s, strcases.KebabToGo(true, s)}
	default:
		return optionName{s, strcases.SnakeToGo(true, s)}
	}
}

func (n optionName) concat(nixName string) optionName {
	return parseName(n.Nix + nixName)
}

func (n optionName) unexport() optionName {
	n.Go = strcases.UnexportPascal(n.Go)
	return n
}
