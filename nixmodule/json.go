package nixmodule

import (
	"fmt"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
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
		json.MarshalFuncV2(marshalOption),
	)),
)

func marshalOption(enc *jsontext.Encoder, o Option, opts json.Options) error {
	if unspec, ok := o.(UnspecifiedOption); ok {
		return enc.WriteValue(unspec.JSON)
	}

	b, err := json.Marshal(o, opts, json.WithMarshalers(
		json.MarshalFuncV2(marshalOptionNoop),
	))
	if err != nil {
		return err
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

func marshalOptionNoop(*jsontext.Encoder, Option, json.Options) error {
	return json.SkipFunc
}

func unmarshalOption(dec *jsontext.Decoder, o *Option, opts json.Options) error {
	raw, err := dec.ReadValue()
	if err != nil {
		return fmt.Errorf("read value: %w", err)
	}

	var option struct {
		Option bool   `json:"_option"`
		Type   string `json:"_type"`
	}

	if err := json.Unmarshal(raw, &option, opts); err != nil {
		return fmt.Errorf("unmarshal to dummy value: %w", err)
	}

	if !option.Option {
		// Parse as a module.
		var m Module

		if err := json.Unmarshal(raw, &m, opts); err != nil {
			return fmt.Errorf("error while unmarshaling as module: %w", err)
		}

		*o = m
		return nil
	}

	rt, ok := optionMap[option.Type]
	if !ok {
		u := UnspecifiedOption{}
		if err := json.Unmarshal(raw, &u, opts); err != nil {
			return fmt.Errorf("unmarshal unspecified option: %w", err)
		}
		*o = u
		return nil
	}

	rv := reflect.New(rt)
	if err := json.Unmarshal(raw, rv.Interface(), opts); err != nil {
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

		var v Option
		if err := json.UnmarshalDecode(dec, &v, opts); err != nil {
			return fmt.Errorf("error unmarshaling module attr %s: %w", k, err)
		}

		(*m)[k] = v
	}

	if _, err := dec.ReadToken(); err != nil {
		return err
	}

	return nil
}
