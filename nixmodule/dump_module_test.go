package nixmodule

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type dumpModuleTest struct {
	name string
	in   ModuleInput
	opts DumpModuleOpts
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
	{
		name: "options path",
		in: ModuleExpr(`{ lib, ... }: with lib; {
			options.services.magics.extras = {
				enable = mkEnableOption "magics service extras";
			};
		}`),
		opts: DumpModuleOpts{
			DumpModuleWithOptionsPath([]string{"services", "magics"}),
		},
		want: expectValue(Module{
			"extras": Module{
				"enable": BoolOption{
					OptionDoc: OptionDoc{
						Example:     true,
						Default:     false,
						Description: "Whether to enable magics service extras.",
					},
				},
			},
		}),
	},
	{
		name: "custom type",
		in: ModuleExpr(`{ lib, ... }: with lib; {
			options.always-fail = mkOption {
				description = "An option that always fails.";
				type = mkOptionType {
					name = "always-fail";
					check = v: false;
					description = "This type always fails.";
				};
			};
		}`),
		want: expectValue(Module{
			"always-fail": UnspecifiedOption{
				OptionDoc: OptionDoc{Description: "An option that always fails."},
				JSON:      jsontext.Value(`{"_option":true,"_type":"always-fail"}`),
			},
		}),
	},
}

var dumpModuleFailingTests = []dumpModuleTest{
	{
		name: "invalid module",
		in:   ModuleExpr(`{ ... }: { }`),
		want: expectAnyError[Module](),
	},
	{
		name: "invalid expression",
		in:   ModuleExpr(`{`),
		want: expectAnyError[Module](),
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
					module, err = DumpModule(ctx, test.in, test.opts...)
					test.want.expect(t, module, err)
				})

				if test.want.expectingError() {
					return
				}

				t.Run("marshaling", func(t *testing.T) {
					actual, err := json.Marshal(module, JSONOptions)
					assert.NoError(t, err, "json.Marshal failed")

					expect, err := dumpModuleAs[jsontext.Value](ctx, test.in, test.opts...)
					assert.NoError(t, err, "dumpModuleAs failed")

					if diff := cmp.Diff(
						canonicalizeJSON(t, actual),
						canonicalizeJSON(t, expect),
					); diff != "" {
						t.Fatalf("unexpected marshaled value (-actual +expect):\n%v", diff)
					}
				})
			})
		}
	}
}

func canonicalizeJSON[T ~[]byte](t *testing.T, in T) []byte {
	// Force unmarshaling to `any` to erase all orderings of object keys.
	// This way, Deterministic will sort the keys in a consistent order.
	var v any
	err := json.Unmarshal(in, &v)
	assert.NoError(t, err, "canonicalizeJSON: json.Unmarshal failed")

	b, err := json.Marshal(v,
		json.Deterministic(true),
		jsontext.WithIndent("  "))
	assert.NoError(t, err, "canonicalizeJSON: json.Marshal failed")

	return b
}

type testResult[T any] struct {
	Value T
	Error error
}

func expectValue[T any](v T) testResult[T] {
	return testResult[T]{v, nil}
}

func expectError[T any](err error) testResult[T] {
	var v T
	return testResult[T]{v, err}
}

func expectAnyError[T any]() testResult[T] {
	var v T
	return testResult[T]{v, cmpopts.AnyError}
}

func (r testResult[T]) expect(t *testing.T, value T, err error) {
	t.Helper()
	t.Logf("got value: (%v, %v)", value, err)
	if diff := cmp.Diff(r, testResult[T]{value, err}, cmpopts.EquateErrors()); diff != "" {
		t.Fatalf("unexpected result (-want +got):\n%s", diff)
	}
}

func (r testResult[T]) expectingError() bool {
	return r.Error != nil
}
