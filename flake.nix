{
	inputs = {
		nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
	};
	outputs = {self, nixpkgs, ...}: 
		let
			systems = [ "aarch64-darwin" "x86_64-linux" ];
		in {
			devShells = nixpkgs.lib.genAttrs systems (system: rec { 
				pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };
				default = pkgs.mkShell {
					buildInputs = with pkgs; [ go gopls ];
				};
			});
		};
}
