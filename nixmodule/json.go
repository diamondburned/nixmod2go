package nixmodule

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"reflect"
)

// JSONOptions is the list of options that allow for parsing Nix options.
var JSONOptions = json.JoinOptions(
	json.Deterministic(true),
	json.WithMarshalers(json.MarshalToFunc(marshalOption)),
	json.WithUnmarshalers(json.UnmarshalFromFunc(unmarshalOption)),
)

func marshalOption(enc *jsontext.Encoder, o Option) error {
	slog.Debug(
		"nixmodule: marshaling",
		"type", "option",
		"ptr", enc.StackPointer(),
		"value", o,
		"value_type", fmt.Sprintf("%T", o))

	b, err := json.Marshal(o,
		enc.Options(),
		json.WithMarshalers(json.MarshalToFunc(func(enc *jsontext.Encoder, o Option) error {
			// Prevent infinite recursion when marshaling an Option into its raw
			// JSON object. Nested objects are allowed to call marshalOption
			// still.
			if enc.StackDepth() == 0 {
				return json.SkipFunc
			}
			return marshalOption(enc, o)
		})))
	if err != nil {
		return err
	}

	return json.MarshalEncode(enc, struct {
		Option bool           `json:"_option,omitzero"`
		Type   string         `json:"_type,omitzero"`
		Value  jsontext.Value `json:",unknown"`
	}{
		Option: o.Type() != "",
		Type:   o.Type(),
		Value:  b,
	})
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
