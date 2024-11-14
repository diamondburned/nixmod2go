{
  self, # current flake
  pkgs, # current flake's pkgs
  lib,
  ...
}:

with lib;

{
  options.examples.modules.complexModule = {
    enable = mkEnableOption "example-module";

    string = mkOption {
      type = types.str;
      default = "Hello, World!";
      description = "An example string option";
    };

    number = mkOption {
      type = types.int;
      default = 42;
      example = 42;
      description = "An example number option";
    };

    numbers = mkOption {
      type = types.submodule {
        options = {
          int = mkOption { type = types.int; };
          float = mkOption { type = types.float; };
          number = mkOption { type = types.number; };

          u8 = mkOption { type = types.ints.u8; };
          s8 = mkOption { type = types.ints.s8; };
          u16 = mkOption { type = types.ints.u16; };
          s16 = mkOption { type = types.ints.s16; };
          u32 = mkOption { type = types.ints.u32; };
          s32 = mkOption { type = types.ints.s32; };
          between = mkOption { type = types.ints.between 1 10; };
          unsigned = mkOption { type = types.ints.unsigned; };
          positive = mkOption { type = types.ints.positive; };
        };
      };
      description = "An example for various ints.* options";
    };

    lines = mkOption {
      type = types.lines;
      example = ''
        Hello, world!
        Hello, 世界!
      '';
      description = "An example lines option (treated as string)";
    };

    port = mkOption {
      type = types.port;
      description = "An example port number option";
    };

    path = mkOption {
      type = types.path;
      example = "/etc/nixos/configuration.nix";
      description = "An example path option (treated as string)";
    };

    bool = mkOption {
      type = types.bool;
      default = false;
      description = "An example boolean option";
    };

    uniq = mkOption {
      type = types.uniq types.str;
      description = "An example unique string option";
    };

    anything = mkOption {
      type = types.anything;
      default = null;
      description = "An example anything option";
    };

    attrs = mkOption {
      type = types.attrs;
      default = { };
      description = "An example attrs option (treated as map[string]any)";
    };

    enum = mkOption {
      type = types.enum [
        "a"
        "b"
        "c"
      ];
      default = "a";
      description = "An example enum option";
    };

    either = mkOption {
      type = types.either types.int types.str;
      default = 42;
      description = "An example either option (int or string)";
    };

    oneOf = mkOption {
      type = types.oneOf [
        types.int
        types.str
        types.bool
        types.attrs
      ];
      default = false;
      description = "An example oneOf option (int or string or bool)";
    };

    nullable = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "An example nullable string option";
    };

    stringAttrs = mkOption {
      type = types.attrsOf types.str;
      default = {
        hello = "world";
      };
      description = "A map[string]string option";
    };

    stringList = mkOption {
      type = types.listOf types.str;
      default = [
        "Hello"
        "World"
      ];
      description = "A list of strings";
    };

    submodule = mkOption {
      type = types.submodule {
        # description = "An example submodule";
        options = {
          innerString = mkOption {
            type = types.str;
            default = "Hello, World!";
            description = "An example string option";
          };

          innerNullable = mkOption {
            type = types.nullOr types.str;
            default = null;
            description = "An example nullable string option";
          };
        };
      };
      description = "An example submodule option";
    };

    submoduleSelfRef = mkOption {
      type = types.submodule (
        { name, ... }:
        {
          options = {
            currentName = mkOption {
              type = types.str;
              default = name;
              description = "The name of the submodule";
            };
          };
        }
      );
      description = "An example submodule option that references its own name";
    };

    nullableSubmodule = mkOption {
      type = types.nullOr (
        types.submodule {
          options = {
            enable = mkEnableOption "nullable-submodule";
          };
        }
      );
      default = null;
      description = "An example nullable submodule option";
    };

    eitherSubmodule = mkOption {
      type = types.either types.path (
        types.submodule {
          options = {
            hello = mkOption {
              type = types.str;
              default = "world";
            };
          };
        }
      );
      default = "/run/secrets/submodule.json";
      description = "A submodule or path to the submodule";
    };

    submoduleList = mkOption {
      type = types.listOf (
        types.submodule {
          options = {
            enable = mkEnableOption "submodule-list";
          };
        }
      );
      default = [
        {
          enable = true;
        }
        {
          enable = false;
        }
      ];
      description = "An example list of submodules";
    };

    internal = mkOption {
      type = types.bool;
      default = false;
      internal = true;
      description = "An example internal option";
    };

    package = mkOption {
      type = types.package;
      default = pkgs.hello;
      description = "An example package option";
    };
  };

  config = {
    examples.modules.complexModule = {
      enable = true;
      string = "Hello, World!";
      number = 42;
      numbers = {
        int = 42;
        float = 42.0;
        number = 42;
        u8 = 42;
        s8 = 42;
        u16 = 42;
        s16 = 42;
        u32 = 42;
        s32 = 42;
        between = 5;
        unsigned = 42;
        positive = 42;
      };
      lines = "Hello, world!\nHello, 世界!\n";
      port = 80;
      path = "/etc/nixos/configuration.nix";
      bool = false;
      uniq = "Hello, World!";
      anything = null;
      attrs = { };
      enum = "a";
      either = 42;
      oneOf = false;
      nullable = null;
      stringAttrs = {
        hello = "world";
      };
      stringList = [
        "Hello"
        "World"
      ];
      submodule = {
        innerString = "Hello, World!";
        innerNullable = null;
      };
      submoduleSelfRef = {
        currentName = "submoduleSelfRef";
      };
      nullableSubmodule = null;
      submoduleList = [
        {
          enable = true;
        }
        {
          enable = false;
        }
      ];
      package = pkgs.hello;
    };
  };
}
