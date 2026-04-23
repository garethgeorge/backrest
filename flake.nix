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
              nodejs_20
              pnpm_9

              # Protobuf / code generation
              protobuf
              buf
              protoc-gen-go
              protoc-gen-go-grpc

              # General build tools
              gnumake
              git

              # Runtime dependencies (for local testing)
              restic
              rclone

              # Shell
              zsh
              oh-my-posh
            ];

            SHELL = "${pkgs.zsh}/bin/zsh";
            OMP_THEME = "${pkgs.oh-my-posh}/share/oh-my-posh/themes/star.omp.json";

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
