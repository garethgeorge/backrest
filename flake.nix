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
              go_1_24  # go 1.25 not yet in nixpkgs; use latest available
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
            ];

            shellHook = ''
              echo "backrest dev shell"
              echo "  go     : $(go version)"
              echo "  node   : $(node --version)"
              echo "  pnpm   : $(pnpm --version)"
              echo "  protoc : $(protoc --version)"
              echo "  buf    : $(buf --version)"
            '';
          };
        });
    };
}
