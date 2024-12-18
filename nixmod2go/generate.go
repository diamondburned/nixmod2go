package nixmod2go

import (
	_ "embed"
	"fmt"
	"go/doc/comment"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/diamondburned/gotk4/gir/girgen/cmt"
	"github.com/diamondburned/gotk4/gir/girgen/strcases"
	"libdb.so/nixmod2go/nixmodule"

	gen "github.com/moznion/gowrtr/generator"
)

// Opts are the options for generating Go struct definitions from Nix modules.
type Opts struct {
	// RootName is the name of the root struct type.
	// By default, it's "Config".
	RootName string
}

// Generate generates Go struct definitions from Nix modules.
func Generate(module nixmodule.Module, packageName string, opts Opts) (string, error) {
	if opts.RootName == "" {
		opts.RootName = "Config"
	}

	slog := slog.With(
		"package", packageName,
		"root", opts.RootName,
	)

	f := generatingFile{
		imports: make(map[string]struct{}),
		slog:    slog,
	}

	slog.Debug("generating Go struct definitions from Nix modules")
	f.generate(sortModule(module), opts.RootName)

	return gen.NewRoot().
		AddStatements(
			gen.NewComment(" Code generated by nixmod2go. DO NOT EDIT."),
			gen.NewPackage(packageName),
			gen.NewNewline(),
		).
		AddStatements(
			f.importStatement(),
			gen.NewNewline(),
		).
		AddStatements(f.statements...).
		Gofmt("-s").
		Generate(0)
}

type generatingFile struct {
	statements []gen.Statement
	imports    map[string]struct{}
	slog       *slog.Logger
}

func (g *generatingFile) generate(root sortedModule, rootName string) {
	name := rootOptionName(rootName)
	g.generateModuleType(name, modulePath{}, root)
}

type optionOpts struct {
	forceInline bool
}

type optionOpt func(*optionOpts)

func buildOptionOpts(opts []optionOpt) optionOpts {
	var o optionOpts
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func addOptions(pre []optionOpt, opts ...optionOpt) []optionOpt {
	return append(slices.Clone(pre), opts...)
}

// optionForceInline forces the generated type to be inline.
// This implies that the generated type does not have a side effect of adding
// code into the *generatingFile instance.
func optionForceInline(o *optionOpts) {
	if o.forceInline {
		panic("option is already forced inline")
	}
	o.forceInline = true
}

func (g *generatingFile) generateItemType(path modulePath, item optionItem, opts ...optionOpt) string {
	g.slog.Debug(
		"generating item type",
		"path", path,
		"name", item.Name,
		"item.Module", item.Module != nil,
		"item.Option", item.Option != nil)

	name := parseName(item.Name)
	switch {
	case item.Module != nil:
		return g.generateModuleType(name, path, *item.Module, opts...)
	case item.Option != nil:
		return g.generateOptionType(name, path, *item.Option, opts...)
	default:
		panic("unreachable")
	}
}

func (g *generatingFile) generateModuleType(name optionName, path modulePath, module sortedModule, opts ...optionOpt) string {
	o := buildOptionOpts(opts)

	g.slog.Debug(
		"generating module type",
		"path", path,
		"name", name.Nix,
		"opts", o,
		"module", module.Keys())

	// Manually generate a struct here. `gen` doesn't support adding comments
	// before each struct field for some reason?
	var s strings.Builder

	fmt.Fprintf(&s, "struct {\n")
	for _, item := range module {
		valueName := parseName(item.Name)
		valueType := g.generateItemType(path.Add(valueName), item)

		if item.Option != nil && (*item.Option).Doc().Description != "" {
			if cmt := docComment((*item.Option).Doc(), valueName.Go, 1); cmt != "" {
				fmt.Fprint(&s, cmt)
			}
		}

		fmt.Fprintf(&s, "\t%s %s `json:%q`\n", valueName.Go, valueType, item.Name)
	}
	fmt.Fprint(&s, "}")

	if o.forceInline {
		return s.String()
	} else {
		// Choose to prepend the struct. This is because [generateItemType] will
		// recursively generate its own type before we can append our struct in, so
		// it'll naturally appear at the end by the time we're here.
		g.prepend(
			gen.NewCommentf(" %s is the struct type for %s.", name.Go, path.GoDocForNixPath()),
			gen.NewRawStatementf("type %s %s\n", name.Go, s.String()),
		)

		return name.Go
	}
}

func (g *generatingFile) generateOptionType(name optionName, path modulePath, option nixmodule.Option, opts ...optionOpt) string {
	g.slog.Debug(
		"generating option type",
		"path", path,
		"name", name.Nix,
		"option", option.Type())

	switch option := option.(type) {
	case nixmodule.StrOption:
		return "string"
	case nixmodule.IntOption:
		return "int"
	case nixmodule.IntBetweenOption:
		return "int"
	case nixmodule.PositiveIntOption:
		return "uint"
	case nixmodule.SignedInt8Option:
		return "int8"
	case nixmodule.SignedInt16Option:
		return "int16"
	case nixmodule.SignedInt32Option:
		return "int32"
	case nixmodule.UnsignedInt8Option:
		return "uint8"
	case nixmodule.UnsignedInt16Option:
		return "uint16"
	case nixmodule.UnsignedInt32Option:
		return "uint32"
	case nixmodule.UnsignedIntOption:
		return "uint"
	case nixmodule.PathOption:
		return "string"
	case nixmodule.PackageOption:
		return "string"
	case nixmodule.BoolOption:
		return "bool"
	case nixmodule.FloatOption:
		return "float64"
	case nixmodule.AttrsOption:
		return "map[string]any"
	case nixmodule.AnythingOption:
		return "any"
	case nixmodule.UnspecifiedOption:
		g.addImport("encoding/json")
		return "json.RawMessage"
	case nixmodule.EnumOption:
		return g.generateEnumType(name, path, option, opts...)
	case nixmodule.SeparatedString:
		return "string" // TODO: generate type that has .Split()
	case nixmodule.UniqueOption:
		return g.generateOptionType(name, path, option.Unique, opts...)
	case nixmodule.EitherOption:
		if eitherIsNumber(option) {
			g.addImport("encoding/json")
			return "json.Number"
		}
		return g.generateEitherType(name, path, option, opts...)
	case nixmodule.NullOrOption:
		return "*" + g.generateOptionType(name, path, option.NullOr, opts...)
	case nixmodule.ListOfOption:
		return "[]" + g.generateOptionType(name, path, option.ListOf, opts...)
	case nixmodule.AttrsOfOption:
		return "map[string]" + g.generateOptionType(name, path, option.AtrrsOf, opts...)
	case nixmodule.SubmoduleOption:
		return g.generateModuleType(name, path, sortModule(option.Submodule), opts...)
	default:
		panic("unreachable")
	}
}

func (g *generatingFile) generateEnumType(name optionName, path modulePath, option nixmodule.EnumOption, _ ...optionOpt) string {
	// Manually construct this code since `gen` doesn't have either `type` or
	// `const` statements supported.
	var s strings.Builder

	fmt.Fprintf(&s, "type %s string\n", name.Go)
	fmt.Fprintln(&s)

	fmt.Fprintln(&s, "const (")
	for _, value := range option.Enum {
		valueName := parseName(value)
		fmt.Fprintf(&s, "%s %s = %q\n", name.Go+valueName.Go, name.Go, value)
	}
	fmt.Fprintln(&s, ")")

	g.append(
		gen.NewCommentf(" %s is the enum type for %s.", name.Go, path.GoDocForNixPath()),
		gen.NewRawStatement(s.String()),
	)

	return name.Go
}

func (g *generatingFile) generateEitherType(name optionName, path modulePath, option nixmodule.EitherOption, opts ...optionOpt) string {
	iface := []gen.Statement{
		gen.NewCommentf(" %s describes the `either` type for %s.", name.Go, path.GoDocForNixPath()),
		gen.NewInterface(name.Go, gen.NewFuncSignature("is"+name.Go)),
	}

	type optionData struct {
		nixmodule.Option
		Name optionName
		Type string
	}

	options := make([]optionData, len(option.Either))
	for i, option := range option.Either {
		optionName := parseName(option.Type())
		optionName.Go = name.Go + optionName.Go

		optionType := g.generateOptionType(optionName, path, option,
			// Force the generated type to be inline.
			// This resolves duplicate names with (either (submodule)).
			addOptions(opts, optionForceInline)...)

		options[i] = optionData{
			Option: option,
			Name:   optionName,
			Type:   optionType,
		}
	}

	var types []gen.Statement
	var typeMethods []gen.Statement
	var typeFuncs []gen.Statement

	for _, option := range options {
		types = append(types,
			gen.NewCommentf(" %s is one of the types that satisfy [%s].", option.Name.Go, name.Go),
			gen.NewRawStatementf("type %s %s", option.Name.Go, option.Type))

		typeMethods = append(typeMethods, gen.NewFunc(
			gen.NewFuncReceiver(strcases.FirstLetter(option.Name.Go), option.Name.Go),
			gen.NewFuncSignature("is"+name.Go),
		))

		typeFuncs = append(typeFuncs,
			gen.NewCommentf(" New%s constructs a value of type `%s` that satisfies [%s].", option.Name.Go, option.Option.Type(), name.Go),
			gen.NewFunc(nil,
				gen.NewFuncSignature("New"+option.Name.Go).
					AddParameters(gen.NewFuncParameter(strcases.FirstLetter(option.Name.Go), option.Type)).
					AddReturnTypes(name.Go),
				gen.NewReturnStatement(fmt.Sprintf("%s(%s)", option.Name.Go, strcases.FirstLetter(option.Name.Go))),
			))
	}

	g.append(iface...)
	g.append(types...)
	g.append(typeMethods...)
	g.append(typeFuncs...)

	g.addImport("encoding/json")
	g.addImport("errors")

	unmarshalFunc := gen.NewFunc(nil,
		gen.NewFuncSignature("unmarshal"+name.Go).
			Parameters(gen.NewFuncParameter("data", "json.RawMessage")).
			ReturnTypes(name.Go, "error"))
	for i, option := range options {
		unmarshalFunc = unmarshalFunc.AddStatements(gen.NewRawStatementf(
			`
			var v%[1]d %[2]s
			if err := json.Unmarshal(data, &v%[1]d); err == nil {
				return %[3]s(v%[1]d), nil
			}
			`,
			i, option.Type, option.Name.Go,
		))
	}
	unmarshalFunc = unmarshalFunc.AddStatements(
		gen.NewReturnStatement(fmt.Sprintf(
			`nil, errors.New("failed to unmarshal %s: unknown type received")`,
			name.Go)))

	g.append(
		gen.NewCommentf(" %sJSON wraps [%[1]s] and implements the json.Unmarshaler interface.", name.Go),
		gen.NewRawStatementf("type %sJSON struct { Value %[1]s }", name.Go),
		gen.NewNewline(),
		gen.NewCommentf(" UnmarshalJSON implements the [json.Unmarshaler] interface for [%s].", name.Go),
		gen.NewFunc(
			gen.NewFuncReceiver(strcases.FirstLetter(name.Go), "*"+name.Go+"JSON"),
			gen.NewFuncSignature("UnmarshalJSON").
				Parameters(gen.NewFuncParameter("data", "[]byte")).
				ReturnTypes("error"),
			gen.NewRawStatementf("_v, err := unmarshal%s(data)", name.Go),
			gen.NewIf(fmt.Sprintf("err != nil"),
				gen.NewReturnStatement("err"),
			),
			gen.NewRawStatementf("%s.Value = _v", strcases.FirstLetter(name.Go)),
			gen.NewReturnStatement("nil"),
		),
		gen.NewNewline(),
		gen.NewCommentf(" MarshalJSON implements the [json.Marshaler] interface for [%s].", name.Go),
		gen.NewFunc(
			gen.NewFuncReceiver(strcases.FirstLetter(name.Go), name.Go+"JSON"),
			gen.NewFuncSignature("MarshalJSON").
				Parameters().
				ReturnTypes("[]byte", "error"),
			gen.NewReturnStatement(fmt.Sprintf(
				"json.Marshal(%s.Value)",
				strcases.FirstLetter(name.Go))),
		),
	)

	g.append(unmarshalFunc)

	return name.Go + "JSON"
}

func (g *generatingFile) append(stmts ...gen.Statement) {
	g.statements = append(g.statements, stmts...)
	g.statements = append(g.statements, gen.NewNewline())
}

func (g *generatingFile) prepend(stmts ...gen.Statement) {
	g.statements = slices.Concat(stmts, []gen.Statement{gen.NewNewline()}, g.statements)
}

func (g *generatingFile) addImport(url string) {
	g.imports[url] = struct{}{}
}

func (g *generatingFile) importStatement() gen.Statement {
	return gen.NewImport(slices.Collect(maps.Keys(g.imports))...)
}

func docComment(doc nixmodule.OptionDoc, what string, indentLvl int) string {
	if doc.Description == "" {
		return ""
	}
	s := cmt.FixGrammar(what, doc.Description)
	return fmtComment(s, indentLvl)
}

func fmtComment(s string, indentLvl int) string {
	printer := &comment.Printer{
		TextPrefix:     strings.Repeat(" ", cmt.CommentsTabWidth*indentLvl) + "// ",
		TextCodePrefix: strings.Repeat(" ", cmt.CommentsTabWidth*indentLvl) + "//\t",
	}
	s = string(printer.Text(new(comment.Parser).Parse(s)))
	return s
}
