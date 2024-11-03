{
  module,

  pkgs ? import <nixpkgs> { },
  specialArgs ? { },
}:

with pkgs.lib;
with builtins;

let
  lib = pkgs.lib;

  module' = (if isPath module then import module else module) (
    {
      config = { };
      options = { };
    }
    // (specialArgs)
    // ({ inherit lib; })
  );

  parseOptions = options: parseOptions' (filterAttrs (k: v: k != "_module") options);
  parseOptions' = options: mapAttrs (name: parseOption) options;

  parseOption =
    option:
    if (option ? _type) then
      if (option._type == "option") then
        ({ })
        // (parseOption option.type)
        // (flip filterAttrs option (
          k: _:
          elem k [
            "default"
            "defaultText"
            "example"
            "description"
            "descriptionClass"
            "visible"
            "internal"
            "readOnly"
          ]
        ))
        // (
          let
            rs = tryEval (builtins.unsafeGetAttrPos "type" option);
            ok = rs.success && rs.value ? file && !(hasPrefix (toString pkgs.path) rs.value.file);
          in
          optionalAttrs ok {
            location = {
              inherit (rs.value) line column;
            };
          }
        )
      else if (option._type == "option-type") then
        ({
          _option = true;
          _type = option.name;
        })
        // (
          {
            # Types that don't have more types underneath:
            str = { };
            int = { };
            path = { };
            bool = { };
            float = { };
            attrs = { };
            anything = { };
            # boolByOr = { };
            unspecified = { };
            # Integer types, some of which have extra information that we
            # unfortunately can't get:
            intBetween = { };
            positiveInt = { };
            signedInt16 = { };
            signedInt32 = { };
            signedInt8 = { };
            unsignedInt16 = { };
            unsignedInt32 = { };
            unsignedInt8 = { };
            unsignedInt = { };
            # Types that have extra non-type information:
            enum.enum = option.functor.payload;
            separatedString.separator = option.functor.payload;
            # Types that have more types underneath:
            either.either = flattenEither option;
            unique.unique = parseOption option.nestedTypes.elemType;
            nullOr.nullOr = parseOption option.nestedTypes.elemType;
            listOf.listOf = parseOption option.nestedTypes.elemType;
            attrsOf.attrsOf = parseOption option.nestedTypes.elemType;
            submodule.submodule = parseOptions (option.getSubOptions [ ]);
          }
          .${option.name} or (warn "Option type ${option.name} is not fully implemented" { })
        )
      else
        throw "Unknown option type: ${option._type}"
    else
      parseOptions option;

  flattenEither = flattenEither';

  flattenEither' =
    eitherOption:
    let
      l = eitherOption.nestedTypes.left;
      r = eitherOption.nestedTypes.right;
      f = v: if v.name == "either" then flattenEither' v else [ (parseOption v) ];
    in
    [ ] ++ f l ++ f r;
in

assert assertMsg (!(module' ? imports)) "imports is not supported";
parseOptions module'.options
