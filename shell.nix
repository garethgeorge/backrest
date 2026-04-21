# Backward-compatible wrapper for users without flakes enabled.
# The canonical definition lives in flake.nix.
(builtins.getFlake (toString ./.)).devShells.${builtins.currentSystem}.default
