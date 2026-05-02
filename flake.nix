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

        packages.default = pkgs.buildGoModule {
          vendorHash = "sha256-69AUHmufjQ8b2D0QNBGSDFsq1xJSV/9/X4gxcWoBbbQ=";

          pname = "nixmod2go";
          version = self.rev or "unknown";
          src = self;
          doCheck = false; # requires Nix

          env.GOEXPERIMENT = "jsonv2";
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
