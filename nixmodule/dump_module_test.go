package nixmodule

import (
	"context"
	"errors"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/go-cmp/cmp"
)

type dumpModuleTest struct {
	name string
	in   ModuleInput
	want testResult[Module]
}

var dumpModulePassingTests = []dumpModuleTest{
	{
		name: "empty module",
		in:   ModuleExpr(`{ ... }: { options = { }; }`),
		want: expectValue(Module{}),
	},
	{
		name: "submodule",
		in: ModuleExpr(`{ lib, ... }: with lib; {
			options.submodule = mkOption {
				type = types.submodule {
					options = {
						hello = mkOption {
							type = types.str;
							description = "Hello, world!";
						};
					};
				};
				description = "A submodule.";
			};
		}`),
		want: expectValue(Module{
			"submodule": SubmoduleOption{
				OptionDoc: OptionDoc{Description: "A submodule."},
				Submodule: Module{
					"hello": StrOption{
						OptionDoc: OptionDoc{Description: "Hello, world!"},
					},
				},
			},
		}),
	},
}

var dumpModuleFailingTests = []dumpModuleTest{
	{
		name: "invalid module",
		in:   ModuleExpr(`{ ... }: { }`),
		want: expectStringError[Module]("nix-instantiate: attribute 'options' missing"),
	},
	{
		name: "invalid expression",
		in:   ModuleExpr(`{`),
		want: expectStringError[Module]("parse module expression: nix expression error: syntax error, unexpected end of file"),
	},
}

func TestDumpModule(t *testing.T) {
	suites := []struct {
		name  string
		suite []dumpModuleTest
	}{
		{
			name:  "passing",
			suite: dumpModulePassingTests,
		},
		{
			name:  "failing",
			suite: dumpModuleFailingTests,
		},
	}

	for _, suite := range suites {
		for _, test := range suite.suite {
			t.Run(suite.name+"/"+test.name, func(t *testing.T) {
				ctx := context.Background()

				var module Module
				var err error

				t.Run("expect", func(t *testing.T) {
					module, err = DumpModule(ctx, test.in)
					test.want.expect(t, module, err)
				})

				t.Run("marshaling", func(t *testing.T) {
					expectValue, err := json.Marshal(module, JSONOptions)
					assertNoError(t, err)

					actualValue, err := dumpModuleAs[jsontext.Value](ctx, test.in)
					assertNoError(t, err)

					if diff := cmp.Diff(jsontext.Value(expectValue), actualValue); diff != "" {
						t.Fatalf("unexpected marshaled value:\n%v", diff)
					}
				})
			})
		}
	}
}

type testResult[T any] struct {
	Value T
	Error error
}

var anyError = errors.New("any error")

func expectValue[T any](v T) testResult[T] {
	return testResult[T]{v, nil}
}

func expectError[T any](err error) testResult[T] {
	var v T
	return testResult[T]{v, err}
}

func expectStringError[T any](err string) testResult[T] {
	return expectError[T](errors.New(err))
}

func expectAnyError[T any]() testResult[T] {
	var v T
	return testResult[T]{v, anyError}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	expectValue(struct{}{}).expect(t, struct{}{}, err)
}

func (r testResult[T]) expect(t *testing.T, value T, err error) {
	t.Helper()
	t.Logf("got value: (%v, %v)", value, err)
	if r.Error == anyError {
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		return
	}
	if diff := cmp.Diff(r, testResult[T]{value, err}); diff != "" {
		t.Fatalf("unexpected result:\n%v", diff)
	}
}
