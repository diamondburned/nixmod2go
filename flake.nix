{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:

    (flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools

            xc
            nixfmt-rfc-style
          ];
        };
      }
    ))
    // {
      lib = {
        dumpModule = import ./nixmodule/dump_module.nix;
        exampleModule = nixpkgs.lib.evalModules {
          modules = [ ./example/module.nix ];
          specialArgs = {
            pkgs = nixpkgs.legacyPackages.x86_64-linux;
          };
        };
      };
    };
}
