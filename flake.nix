{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
  };
  outputs =
    { self, nixpkgs, ... }:
    let
      systems = [
        "aarch64-darwin"
        "x86_64-linux"
      ];
    in
    {
      packages = nixpkgs.lib.genAttrs systems (system: rec {
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        default = pkgs.buildGoModule {
          name = "gogrep";
          src = ./.;
          vendorHash = "sha256-WIzP3PgNYzB13okvLvjKpgEp7dqxUytyquj9HY4vVIg=";

          # this becomes available in the implicitly defined devShell
          nativeBuildInputs = [ pkgs.gopls ];
        };
      });
    };
}
