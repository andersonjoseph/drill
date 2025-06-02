{ pkgs, lib, config, inputs, ... }:

{
  languages.go.enable = true;

  packages = [
    pkgs.watchexec
  ];
}
