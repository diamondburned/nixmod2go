package nixmod2go_test

import (
	"context"
	"fmt"
	"log"

	"libdb.so/nixmod2go"
)

func Example() {
	m, err := nixmod2go.DumpModule(context.TODO(), nixmod2go.DumpModuleExpr(`
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

	printOption := func(n string, o nixmod2go.Option) {
		fmt.Printf("%6s (%T): %s\n", n, o, o.Doc().Description)
	}

	magics := m.ByPath("services", "magics").(nixmod2go.Module)
	printOption("enable", magics["enable"])
	printOption("port", magics["port"])
	// Output:
	// enable (nixmod2go.BoolOption): Whether to enable magic services.
	//   port (nixmod2go.UnsignedInt16Option): The port to listen on.
}
