# nix-mod-to-go

Tool to parse and generate Go struct definitions from Nix modules.

## Example

[example-module.nix](./example/example-module.nix) contains an example Nix
module that contains a lot of different option types. Using this file, 2 more
files are generated:

- [example-module.gen.json](./example/example-module.gen.json) contains the
  generated JSON representation of the module.
- [example-module.gen.go](./example/example-module.gen.go) contains the
  generated Go structs code that the module config can be unmarshalled into.

## Tasks

This README file can be executed using `xc`.

### update-example

Update the generated example files.

```sh
go run . -f json ./example/module.nix ./example/module.gen.json
go run . -f go --go-package example ./example/module.nix ./example/module.gen.go
```
