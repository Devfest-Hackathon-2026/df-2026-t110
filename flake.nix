{
  description = "mise with the host tools needed to install this project's toolchain";

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
          mise = pkgs.symlinkJoin {
            name = "mise-with-dependencies-${pkgs.mise.version}";
            paths = [ pkgs.mise ];
            nativeBuildInputs = [ pkgs.makeWrapper ];
            postBuild = ''
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
