FROM mcr.microsoft.com/devcontainers/base:bookworm

# Install Nix in single-user mode for the vscode user
USER vscode
RUN curl -L https://nixos.org/nix/install | bash -s -- --no-daemon
ENV PATH="/home/vscode/.nix-profile/bin:${PATH}"

# Install direnv via Nix and configure shell hooks
RUN . /home/vscode/.nix-profile/etc/profile.d/nix.sh \
    && nix-env -iA nixpkgs.direnv
RUN echo '. /home/vscode/.nix-profile/etc/profile.d/nix.sh' >> /home/vscode/.bashrc \
    && echo 'eval "$(direnv hook bash)"' >> /home/vscode/.bashrc \
    && echo '. /home/vscode/.nix-profile/etc/profile.d/nix.sh' >> /home/vscode/.zshrc \
    && echo 'eval "$(direnv hook zsh)"' >> /home/vscode/.zshrc

# Pre-populate the nix store with project dependencies so first open is fast
COPY shell.nix /tmp/shell.nix
RUN . /home/vscode/.nix-profile/etc/profile.d/nix.sh \
    && nix-shell /tmp/shell.nix --run "echo 'nix dependencies cached'"
