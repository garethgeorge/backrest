{
  description = "Backrest development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          devShellPkgs = with pkgs; [
            go goreleaser nodejs_22 pnpm
            protobuf buf protoc-gen-go protoc-gen-go-grpc protoc-gen-connect-go
            gnumake git restic rclone zsh oh-my-posh
            act docker
          ];
        in
        {
          default = pkgs.buildEnv {
            name = "backrest-dev";
            paths = devShellPkgs;
          };
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              # Go backend
              go
              goreleaser

              # Frontend
              nodejs_22
              pnpm

              # Protobuf / code generation
              protobuf
              buf
              protoc-gen-go
              protoc-gen-go-grpc
              protoc-gen-connect-go

              # General build tools
              gnumake
              git

              # Runtime dependencies (for local testing)
              restic
              rclone

              # Local GitHub Actions debugging: `act` runs the workflows in
              # .github/workflows inside Docker containers (runner images are
              # pinned in .actrc). `docker` is the client act talks to; a
              # running Docker daemon is a host prerequisite (on NixOS enable
              # `virtualisation.docker`).
              act
              docker

              # Shell
              zsh
              oh-my-posh
            ];

            SHELL = "${pkgs.zsh}/bin/zsh";
            OMP_THEME = "${pkgs.oh-my-posh}/share/oh-my-posh/themes/star.omp.json";

            # Playwright browsers for webui e2e tests: stock downloaded
            # browsers don't run on NixOS, so point Playwright at the nixpkgs
            # bundle. Keep webui's @playwright/test pinned to
            # ${pkgs.playwright-driver.version} (nixpkgs playwright-driver).
            PLAYWRIGHT_BROWSERS_PATH = "${pkgs.playwright-driver.browsers}";
            PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS = "true";

            shellHook = ''
              if [ -z "$IN_NIX_SHELL_ZSH" ]; then
                export IN_NIX_SHELL_ZSH=1
                export ZDOTDIR=$(mktemp -d)
                cat > "$ZDOTDIR/.zshrc" <<'ZSHRC'
              [[ -f ~/.zshrc ]] && source ~/.zshrc
              eval "$(oh-my-posh init zsh --config "$OMP_THEME")"
              ZSHRC
                exec ${pkgs.zsh}/bin/zsh
              fi
            '';
          };
        });
    };
}
