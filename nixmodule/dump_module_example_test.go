package nixmodule_test

import (
	"context"
	"fmt"
	"log"

	"libdb.so/nixmod2go/nixmodule"
)

func Example() {
	m, err := nixmodule.DumpModule(context.TODO(), nixmodule.DumpModuleExpr(`
		{ lib, ... }: with lib; {
			options = {
				services.magics = {
					enable = mkEnableOption "magic services";

					port = mkOption {
						type = types.port;
						description = "The port to listen on.";
					};
				};
			};
		}
	`))
	if err != nil {
		log.Fatalln(err)
	}

	printOption := func(n string, o nixmodule.Option) {
		fmt.Printf("%6s (%T): %s\n", n, o, o.Doc().Description)
	}

	magics := m.ByPath("services", "magics").(nixmodule.Module)
	printOption("enable", magics["enable"])
	printOption("port", magics["port"])
	// Output:
	// enable (nixmodule.BoolOption): Whether to enable magic services.
	//   port (nixmodule.UnsignedInt16Option): The port to listen on.
}
