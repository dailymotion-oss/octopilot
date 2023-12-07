{
  description = "Automate your Gitops workflow, by automatically creating/merging GitHub Pull Requests.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];

      perSystem = { config, self', inputs', pkgs, system, ... }: {

        packages.default = pkgs.buildGoModule rec {
          pname = "octopilot";

          version = with inputs; "${self.shortRev or self.dirtyShortRev or "dirty"}";

          src = inputs.self;

          subPackages = [ "." ];

          ldflags = with inputs; [
            "-s" "-w"
            "-X main.buildVersion=flake-${version}"
            "-X main.buildCommit=${self.rev or self.dirtyRev or "dirty"}"
            "-X main.buildDate=${inputs.self.lastModifiedDate}"
          ];

          vendorHash = null;

          meta = {
            homepage = "https://github.com/dailymotion-oss/octopilot";
          };
        };
      };
    };
}
