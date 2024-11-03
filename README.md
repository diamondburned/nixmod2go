# nix-mod-to-go

Tool to parse and generate Go struct definitions from Nix modules.

## Usage

```sh
# Print help message
nixmod2go --help

# Generate Go code to stdout for module.nix
nixmod2go -f go module.nix
```

For more information, see the help message and the below example.

## Example

[example/module.nix](./example/module.nix) contains an example Nix module that
contains a lot of different option types. Using this file, 2 more files are
generated:

- [module.gen.json](./example/module.gen.json) contains the
  generated JSON representation of the module.
- [module.gen.go](./example/module.gen.go) contains the
  generated Go structs code that the module config can be unmarshalled into.

[module_test.go](./example/module_test.go) ensures that the generated Go
types can be properly unmarshaled onto and marshaled from.

The command to generate these files is listed below in the
[`update-example`](#update-example) section.

## Tasks

This README file can be executed using `xc`.

### update-example

Update the generated example files.

```sh
go run . -f json ./example/module.nix ./example/module.gen.json
go run . -f go --go-package example ./example/module.nix ./example/module.gen.go
```
