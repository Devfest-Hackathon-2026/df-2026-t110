{
  description = "mise with toolchain dependencies and a Linux FHS environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      perSystem = { pkgs, ... }:
        let
          runtimeTools = with pkgs; [
            # Download, verify and unpack releases, including legacy asdf plugins.
            bash
            coreutils
            curl
            git
            gnupg
            gawk
            gnugrep
            gnused
            gnutar
            gzip
            bzip2
            xz
            unzip
            # Node.js builds from source by default on NixOS.
            python3
            gnumake
            stdenv.cc
            pkg-config
          ];
          fhsEnv = pkgs.buildFHSEnv {
            name = "mise-fhs";
            targetPkgs = _: runtimeTools ++ [ pkgs.openssl pkgs.zlib ];
            runScript = pkgs.writeShellScript "mise-fhs-entry" ''
              # Host shells must dispatch tools through this FHS launcher.
              # Inspect only mise's subcommand, not arguments passed to a tool or task.
              extraArgs=()
              case "''${MISE_FHS_ARGV0##*/}" in
                mise|mise-*)
                  args=("$@")
                  for ((i = 0; i < ''${#args[@]}; i++)); do
                    case "''${args[i]}" in
                      -C|--cd|-E|--env|-P|--profile|-j|--jobs|--log-level|--output)
                        i=$((i + 1)) ;;
                      --) break ;;
                      -*) ;;
                      activate)
                        extraArgs=(--shims)
                        for arg in "''${args[@]}"; do
                          if [[ "$arg" == --shims ]]; then extraArgs=(); break; fi
                        done
                        break ;;
                      *) break ;;
                    esac
                  done
                  ;;
              esac
              # Keep activation hooks and generated shims pointing at the FHS launcher.
              exec -a "$MISE_FHS_ARGV0" ${pkgs.lib.getExe pkgs.mise} "$@" "''${extraArgs[@]}"
            '';
          };
          mise = pkgs.symlinkJoin {
            name = "mise-with-dependencies-${pkgs.mise.version}";
            paths = [ pkgs.mise ];
            nativeBuildInputs = [ pkgs.makeWrapper ];
            postBuild =
              if pkgs.stdenv.hostPlatform.isLinux then ''
                rm "$out/bin/mise"
                makeWrapper ${pkgs.lib.getExe fhsEnv} "$out/bin/mise" \
                  --set __MISE_BIN "$out/bin/mise" \
                  --prefix PATH : "$out/bin" \
                  --run 'export MISE_FHS_ARGV0="$0"'
              '' else ''
                wrapProgram "$out/bin/mise" \
                  --prefix PATH : ${pkgs.lib.makeBinPath runtimeTools}
              '';
            meta = pkgs.mise.meta;
          };
        in
        {
          packages = {
            inherit mise;
            default = mise;
          };
        };
    };
}
