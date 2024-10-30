package nixmod2go

import (
	"context"
	"errors"
	"testing"

	"github.com/go-test/deep"
)

type dumpModuleTest struct {
	name string
	in   DumpModuleInput
	want testResult[Module]
}

var dumpModulePassingTests = []dumpModuleTest{
	{
		name: "empty module",
		in:   DumpModuleExpr(`{ ... }: { options = { }; }`),
		want: expectValue(Module{}),
	},
	{
		name: "submodule",
		in: DumpModuleExpr(`{ lib, ... }: with lib; {
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
		in:   DumpModuleExpr(`{ ... }: { }`),
		want: expectStringError[Module]("nix-instantiate: attribute 'options' missing"),
	},
	{
		name: "invalid expression",
		in:   DumpModuleExpr(`{`),
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

				module, err := DumpModule(ctx, test.in)
				test.want.expect(t, module, err)
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

func (r testResult[T]) expect(t *testing.T, value T, err error) {
	t.Helper()
	t.Logf("got value: (%v, %v)", value, err)
	if r.Error == anyError {
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		return
	}
	if diff := deep.Equal(r, testResult[T]{value, err}); diff != nil {
		t.Fatalf("unexpected result:\n%v", diff)
	}
}
