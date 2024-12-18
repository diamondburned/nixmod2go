package nixmodule

import (
	"reflect"

	"github.com/go-json-experiment/json/jsontext"
)

// Option represents a Nix option.
type Option interface {
	// Type returns the type of the option.
	Type() string
	// Doc returns common documentation for this Option.
	Doc() OptionDoc
}

// IsType returns true if [o] is of type [T].
// It is a shorter way of writing Go's type assertion syntax.
func IsType[T Option](o Option) bool {
	_, ok := o.(T)
	return ok
}

type optionPrepValue struct {
	GoType  reflect.Type
	NixType string
}

func prepOption[OptionT Option]() optionPrepValue {
	var z OptionT
	return optionPrepValue{
		GoType:  reflect.TypeFor[OptionT](),
		NixType: z.Type(),
	}
}

// serves as both a type registry and a type assertion.
var optionMap = func(preps ...optionPrepValue) map[string]reflect.Type {
	m := make(map[string]reflect.Type, len(preps))
	for _, p := range preps {
		m[p.NixType] = p.GoType
	}
	return m
}(
	prepOption[StrOption](),
	prepOption[IntOption](),
	prepOption[IntBetweenOption](),
	prepOption[PositiveIntOption](),
	prepOption[SignedInt8Option](),
	prepOption[SignedInt16Option](),
	prepOption[SignedInt32Option](),
	prepOption[UnsignedInt8Option](),
	prepOption[UnsignedInt16Option](),
	prepOption[UnsignedInt32Option](),
	prepOption[UnsignedIntOption](),
	prepOption[PathOption](),
	prepOption[BoolOption](),
	prepOption[FloatOption](),
	prepOption[AttrsOption](),
	prepOption[PackageOption](),
	prepOption[AnythingOption](),
	prepOption[UnspecifiedOption](),
	prepOption[EnumOption](),
	prepOption[SeparatedString](),
	prepOption[UniqueOption](),
	prepOption[EitherOption](),
	prepOption[NullOrOption](),
	prepOption[ListOfOption](),
	prepOption[AttrsOfOption](),
	prepOption[SubmoduleOption](),
)

func (StrOption) Type() string           { return "str" }
func (IntOption) Type() string           { return "int" }
func (IntBetweenOption) Type() string    { return "intBetween" }
func (PositiveIntOption) Type() string   { return "positiveInt" }
func (SignedInt8Option) Type() string    { return "signedInt8" }
func (SignedInt16Option) Type() string   { return "signedInt16" }
func (SignedInt32Option) Type() string   { return "signedInt32" }
func (UnsignedInt8Option) Type() string  { return "unsignedInt8" }
func (UnsignedInt16Option) Type() string { return "unsignedInt16" }
func (UnsignedInt32Option) Type() string { return "unsignedInt32" }
func (UnsignedIntOption) Type() string   { return "unsignedInt" }
func (PathOption) Type() string          { return "path" }
func (BoolOption) Type() string          { return "bool" }
func (FloatOption) Type() string         { return "float" }
func (AttrsOption) Type() string         { return "attrs" }
func (PackageOption) Type() string       { return "package" }
func (AnythingOption) Type() string      { return "anything" }
func (UnspecifiedOption) Type() string   { return "unspecified" }
func (EnumOption) Type() string          { return "enum" }
func (SeparatedString) Type() string     { return "separatedString" }
func (UniqueOption) Type() string        { return "unique" }
func (EitherOption) Type() string        { return "either" }
func (NullOrOption) Type() string        { return "nullOr" }
func (ListOfOption) Type() string        { return "listOf" }
func (AttrsOfOption) Type() string       { return "attrsOf" }
func (SubmoduleOption) Type() string     { return "submodule" }

// OptionDoc represents the documentation for a Nix option.
// It is extracted directly from mkOption.
// All fields are optional and may be empty.
type OptionDoc struct {
	Example          any    `json:"example,omitzero"`
	Default          any    `json:"default,omitzero"`
	DefaultText      string `json:"defaultText,omitzero"`
	Description      string `json:"description,omitzero"`
	DescriptionClass string `json:"descriptionClass,omitzero"`
	Visible          bool   `json:"visible,omitzero"`
	Internal         bool   `json:"internal,omitzero"`
	ReadOnly         bool   `json:"readOnly,omitzero"`
}

// Doc returns itself.
// Types can embed [OptionDoc] to satisfy the [Option] interface.
func (o OptionDoc) Doc() OptionDoc { return o }

// Module describes a Nix module.
// It is represented as a map of string keys to [Option]s.
type Module map[string]Option

// Type returns an empty string.
func (m Module) Type() string { return "" }

// Doc returns an empty [OptionDoc].
func (m Module) Doc() OptionDoc { return OptionDoc{} }

// ByPath returns a nested option by path.
// It traverses all [Module] and [SubmoduleOption] types.
// It returns nil if the path does not exist or any option within the path
// (except for the last path) is not a submodule.
// ByPath(0) returns itself.
func (m Module) ByPath(path ...string) Option {
	if len(path) == 0 {
		return m
	}

	switch o := m[path[0]].(type) {
	case Module:
		return o.ByPath(path[1:]...)
	case SubmoduleOption:
		return o.Submodule.ByPath(path[1:]...)
	case Option:
		return o
	default:
		return nil
	}
}

// StrOption is a Nix string option.
//
// Equivalent Nix type: types.str
type StrOption struct {
	OptionDoc
}

// IntOption is a Nix integer option.
//
// Equivalent Nix type: types.int or types.ints
type IntOption struct {
	OptionDoc
}

// IntBetweenOption is a Nix integer between option.
//
// Equivalent Nix type: types.ints.between
type IntBetweenOption struct {
	OptionDoc
}

// PositiveIntOption is a Nix positive integer option.
//
// Equivalent Nix type: types.ints.positive
type PositiveIntOption struct {
	OptionDoc
}

// SignedInt8Option is a Nix signed int8 option.
//
// Equivalent Nix type: types.ints.s8
type SignedInt8Option struct {
	OptionDoc
}

// SignedInt16Option is a Nix signed int16 option.
//
// Equivalent Nix type: types.ints.s16
type SignedInt16Option struct {
	OptionDoc
}

// SignedInt32Option is a Nix signed int32 option.
//
// Equivalent Nix type: types.ints.s32
type SignedInt32Option struct {
	OptionDoc
}

// UnsignedInt8Option is a Nix unsigned int8 option.
//
// Equivalent Nix type: types.ints.u8
type UnsignedInt8Option struct {
	OptionDoc
}

// UnsignedInt16Option is a Nix unsigned int16 option.
//
// Equivalent Nix type: types.ints.u16
type UnsignedInt16Option struct {
	OptionDoc
}

// UnsignedInt32Option is a Nix unsigned int32 option.
//
// Equivalent Nix type: types.ints.u32
type UnsignedInt32Option struct {
	OptionDoc
}

// UnsignedIntOption is a Nix unsigned int option.
//
// Equivalent Nix type: types.ints.unsigned
type UnsignedIntOption struct {
	OptionDoc
}

// PathOption is a Nix path option.
//
// Equivalent Nix type: types.path
type PathOption struct {
	OptionDoc
}

// BoolOption is a Nix boolean option.
//
// Equivalent Nix type: types.bool
type BoolOption struct {
	OptionDoc
}

// FloatOption is a Nix float option.
//
// Equivalent Nix type: types.float
type FloatOption struct {
	OptionDoc
}

// AttrsOption is a Nix attrs option.
// It is not to be confused with [AttrsOfOption]: this does not define the value
// type, whereas [AttrsOfOption] does.
//
// Equivalent Nix type: types.attrs
type AttrsOption struct {
	OptionDoc
}

// PackageOption is a Nix package option.
// It is functionally equivalent to [StrOption] but the string value will be a
// Nix derivation path.
//
// Equivalent Nix type: types.package
type PackageOption struct {
	OptionDoc
}

// AnythingOption is a Nix anything option.
//
// Equivalent Nix type: types.anything
type AnythingOption struct {
	OptionDoc
}

// EnumOption is a Nix enum option.
//
// Equivalent Nix type: types.enum
type EnumOption struct {
	OptionDoc
	// Enum is the list of possible values.
	Enum []string `json:"enum,string"`
}

// SeparatedString is a Nix separated string option.
//
// Equivalent Nix type: types.lines or co.
type SeparatedString struct {
	OptionDoc
	// Separator is the string that separates elements in the list.
	Separator string `json:"separator"`
}

// UniqueOption is a Nix unique option.
//
// Equivalent Nix type: types.uniq and types.unique
type UniqueOption struct {
	OptionDoc
	// Unique is the underlying type.
	Unique Option `json:"unique"`
}

// EitherOption is a Nix either option.
//
// Equivalent Nix type: types.either or types.number
type EitherOption struct {
	OptionDoc
	// Either is the list of possible types.
	Either []Option `json:"either"`
}

// NullOrOption is a Nix null or option.
//
// Equivalent Nix type: types.nullOr
type NullOrOption struct {
	OptionDoc
	// NullOr is the underlying type.
	NullOr Option `json:"nullOr"`
}

// ListOfOption is a Nix list of option.
//
// Equivalent Nix type: types.listOf
type ListOfOption struct {
	OptionDoc
	// ListOf is the type of the list elements.
	ListOf Option `json:"listOf"`
}

// AttrsOfOption is a Nix attrs of option.
//
// Equivalent Nix type: types.attrsOf
type AttrsOfOption struct {
	OptionDoc
	// AttrsOf is the type of the attributes.
	AtrrsOf Option `json:"attrsOf"`
}

// SubmoduleOption describes a nested Nix module.
// It is functionally equivalent to a [Module] but is used within.
//
// Equivalent Nix type: types.submodule
type SubmoduleOption struct {
	OptionDoc
	Submodule Module `json:"submodule"`
}

// UnspecifiedOption is a Nix unspecified option.
// Types that could not be determined are represented as unspecified.
//
// Equivalent Nix type: types.unspecified
type UnspecifiedOption struct {
	OptionDoc
	// JSON is the raw JSON value of the option.
	// It is used when the type is unknown or unsupported.
	JSON jsontext.Value `json:",unknown"`
}

func (o UnspecifiedOption) isUnspecifiedOption() {}
