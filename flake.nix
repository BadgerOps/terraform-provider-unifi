{
  description = "Development shell for terraform-provider-unifi";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        terraformCompat = pkgs.writeShellScriptBin "terraform" ''
          exec ${pkgs.opentofu}/bin/tofu "$@"
        '';
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            actionlint
            curl
            git
            go_1_25
            golangci-lint
            gopls
            gnumake
            jq
            opentofu
            terraformCompat
            terraform-ls
            tfsec
            tflint
            unzip
          ];
        };

        formatter = pkgs.nixpkgs-fmt;
      });
}
