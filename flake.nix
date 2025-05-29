{
  description = "drill project";

  inputs = {
    nixpkgs.url = "https://github.com/NixOS/nixpkgs/archive/refs/tags/25.05.tar.gz";
  };

  outputs = { self, nixpkgs }: 
  let 
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in
  {
    devShells.${system}.default = pkgs.mkShellNoCC {
      packages = with pkgs; [
	go_1_24
	watchexec
      ];
    };
  };
}
