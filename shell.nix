let
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/archive/refs/tags/25.05.tar.gz";
  pkgs = import nixpkgs { config = {}; overlays = []; };
in

pkgs.mkShellNoCC {
    packages = with pkgs; [
      go_1_24
      watchexec
    ];
}
