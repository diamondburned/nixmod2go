package nixmodule

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/puzpuzpuz/xsync/v3"
)

// JSONOptions is the list of options that allow for parsing Nix options.
var JSONOptions = json.JoinOptions(
	json.DiscardUnknownMembers(false),
	json.Deterministic(true),
	json.WithUnmarshalers(json.NewUnmarshalers(
		json.UnmarshalFuncV2(unmarshalOption),
		json.UnmarshalFuncV2(unmarshalModule),
	)),
	json.WithMarshalers(json.NewMarshalers(
		json.MarshalFuncV2(marshalModule),
		json.MarshalFuncV2(marshalOption),
	)),
)

func marshalModule(enc *jsontext.Encoder, m Module, opts json.Options) error {
	return json.MarshalEncode(enc, (map[string]Option)(m), opts)
}

// marshalingOptions keeps track of the [Option] values that are being
// marshaled. This is used to allow that option to be marshaled recursively
// without causing infinite recursions. We only have to do this because
// go-json-experiment's Marshaler API doesn't have a way to pass state to the
// underlying Marshalers.
var marshalingOptions = xsync.NewMapOf[Option, struct{}]()

func marshalOption(enc *jsontext.Encoder, o Option, opts json.Options) error {
	if _, exists := marshalingOptions.LoadOrStore(o, struct{}{}); exists {
		// This option is already being marshaled, so we skip it to avoid infinite
		// recursion.
		return json.SkipFunc
	}
	defer marshalingOptions.Delete(o)

	b, err := json.Marshal(o, opts)
	if err != nil {
		return err
	}

	// If Option is an UnspecifiedOption, then the JSON library will already
	// include the internal fields for us. Pass it through directly.
	if _, ok := o.(interface{ isUnspecifiedOption() }); ok {
		return enc.WriteValue(b)
	}

	final := struct {
		Option bool           `json:"_option"`
		Type   string         `json:"_type"`
		Value  jsontext.Value `json:",inline"`
	}{
		Option: true,
		Type:   o.Type(),
		Value:  b,
	}

	return json.MarshalEncode(enc, final, opts)
}

func unmarshalOption(dec *jsontext.Decoder, o *Option, opts json.Options) error {
	value, err := dec.ReadValue()
	if err != nil {
		return fmt.Errorf("read value: %w", err)
	}

	slog.Debug(
		"nixmodule: unmarshaling",
		"type", "option",
		"ptr", dec.StackPointer(),
		"value", value)

	var option struct {
		Option bool   `json:"_option"`
		Type   string `json:"_type"`
	}

	if err := json.Unmarshal(value, &option, opts); err != nil {
		return fmt.Errorf("unmarshal to dummy value: %w", err)
	}

	if !option.Option {
		// Parse as a module.
		var m Module

		if err := json.Unmarshal(value, &m, opts); err != nil {
			return fmt.Errorf("error while unmarshaling as module: %w", err)
		}

		*o = m
		return nil
	}

	rt, ok := optionMap[option.Type]
	if !ok {
		u := UnspecifiedOption{}
		if err := json.Unmarshal(value, &u, opts); err != nil {
			return fmt.Errorf("unmarshal unspecified option: %w", err)
		}
		*o = u
		return nil
	}

	rv := reflect.New(rt)
	if err := json.Unmarshal(value, rv.Interface(), opts); err != nil {
		return fmt.Errorf("unmarshal option: %w", err)
	}
	*o = rv.Elem().Interface().(Option)

	return nil
}

func unmarshalModule(dec *jsontext.Decoder, m *Module, opts json.Options) error {
	if k := dec.PeekKind(); k != '{' {
		return fmt.Errorf("expected object start, but encountered %v", k)
	}

	if _, err := dec.ReadToken(); err != nil {
		return err
	}

	if *m == nil {
		*m = make(Module)
	}

	for dec.PeekKind() != '}' {
		var k string
		if err := json.UnmarshalDecode(dec, &k, opts); err != nil {
			return fmt.Errorf("error unmarshaling module name: %w", err)
		}

		vvalue, err := dec.ReadValue()
		if err != nil {
			return fmt.Errorf("error reading module attr %s: %w", k, err)
		}

		slog.Debug(
			"nixmodule: unmarshaling",
			"type", "module",
			"ptr", dec.StackPointer(),
			"value", vvalue)

		var v Option
		if err := json.Unmarshal(vvalue, &v, opts); err != nil {
			return fmt.Errorf("error unmarshaling module attr %s: %w", k, err)
		}

		(*m)[k] = v
	}

	if _, err := dec.ReadToken(); err != nil {
		return err
	}

	return nil
}
