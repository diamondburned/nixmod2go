package nixmodule

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	// "github.com/puzpuzpuz/xsync/v3"
)

// JSONOptions is the list of options that allow for parsing Nix options.
var JSONOptions = json.JoinOptions(
	json.RejectUnknownMembers(false),
	json.Deterministic(true),
	json.WithUnmarshalers(json.JoinUnmarshalers(
		json.UnmarshalFromFunc(unmarshalOption),
		json.UnmarshalFromFunc(unmarshalModule),
	)),
	json.WithMarshalers(json.JoinMarshalers(
		json.MarshalToFunc(marshalModule),
		json.MarshalToFunc(marshalOption),
	)),
)

func marshalModule(enc *jsontext.Encoder, m Module) error {
	return json.MarshalEncode(enc, (map[string]Option)(m), enc.Options())
}

func marshalOption(enc *jsontext.Encoder, o Option) error {
	b, err := json.Marshal(o)
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

	return json.MarshalEncode(enc, final, enc.Options())
}

func unmarshalOption(dec *jsontext.Decoder, o *Option) error {
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

	if err := json.Unmarshal(value, &option, dec.Options()); err != nil {
		return fmt.Errorf("unmarshal to dummy value: %w", err)
	}

	if !option.Option {
		// Parse as a module.
		var m Module

		if err := json.Unmarshal(value, &m, dec.Options()); err != nil {
			return fmt.Errorf("error while unmarshaling as module: %w", err)
		}

		*o = m
		return nil
	}

	rt, ok := optionMap[option.Type]
	if !ok {
		u := UnspecifiedOption{}
		if err := json.Unmarshal(value, &u, dec.Options()); err != nil {
			return fmt.Errorf("unmarshal unspecified option: %w", err)
		}
		*o = u
		return nil
	}

	rv := reflect.New(rt)
	if err := json.Unmarshal(value, rv.Interface(), dec.Options()); err != nil {
		return fmt.Errorf("unmarshal option: %w", err)
	}
	*o = rv.Elem().Interface().(Option)

	return nil
}

func unmarshalModule(dec *jsontext.Decoder, m *Module) error {
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
		if err := json.UnmarshalDecode(dec, &k, dec.Options()); err != nil {
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
		if err := json.Unmarshal(vvalue, &v, dec.Options()); err != nil {
			return fmt.Errorf("error unmarshaling module attr %s: %w", k, err)
		}

		(*m)[k] = v
	}

	if _, err := dec.ReadToken(); err != nil {
		return err
	}

	return nil
}
