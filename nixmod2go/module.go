package nixmod2go

import (
	"slices"
	"strings"

	"libdb.so/nixmod2go/nixmodule"
)

type sortedModule []optionItem

type optionItem struct {
	Name   string
	Option *nixmodule.Option // either
	Module *sortedModule     // or
}

// sortModule converts a module into a sorted list of options.
// This function recursively sorts nested modules.
func sortModule(module nixmodule.Module) sortedModule {
	sorted := make(sortedModule, 0, len(module))
	for name, opt := range module {
		item := optionItem{Name: name}
		switch opt := opt.(type) {
		case nixmodule.Module:
			item.Module = ptr(sortModule(opt))
		default:
			item.Option = ptr(opt)
		}
		sorted = append(sorted, item)
	}

	slices.SortFunc(sorted, func(a, b optionItem) int {
		for _, x := range []int{
			// Put in order: [enable, package, ...rest]
			sortFirst(a.Name, b.Name, "enable"),
			sortFirst(a.Name, b.Name, "package"),
			// Rest: sort names alphabetically.
			strings.Compare(a.Name, b.Name),
		} {
			if x != 0 {
				return x
			}
		}
		return 0
	})

	return sorted
}

func ptr[T any](x T) *T {
	return &x
}

func sortFirst[T ~string](a, b T, value T) int {
	if a == value {
		return -1
	}
	if b == value {
		return 1
	}
	return 0
}

type modulePath []optionName

func (m modulePath) Add(v ...optionName) modulePath {
	return slices.Concat(slices.Clone(m), v)
}

func (m modulePath) GoDoc() string {
	var b strings.Builder
	for i, v := range m {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString("[")
		b.WriteString(v.Go)
		b.WriteString("]")
	}
	return b.String()
}

func (m modulePath) GoDocForNixPath() string {
	return "`" + m.NixPath() + "`"
}

func (m modulePath) NixPath() string {
	var b strings.Builder
	for i, v := range m {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(v.Nix)
	}
	return b.String()
}
