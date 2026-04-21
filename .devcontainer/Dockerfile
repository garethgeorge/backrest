FROM mcr.microsoft.com/devcontainers/base:bookworm

# Install Nix in single-user mode
RUN curl -L https://nixos.org/nix/install | bash -s -- --no-daemon \
    && echo '. /root/.nix-profile/etc/profile.d/nix.sh' >> /etc/bash.bashrc

ENV PATH="/root/.nix-profile/bin:${PATH}"

# Install direnv via Nix and configure the shell hook
RUN . /root/.nix-profile/etc/profile.d/nix.sh \
    && nix-env -iA nixpkgs.direnv \
    && echo 'eval "$(direnv hook bash)"' >> /etc/bash.bashrc \
    && echo 'eval "$(direnv hook zsh)"' >> /etc/zshrc

# Pre-populate the nix store with project dependencies
COPY shell.nix /tmp/shell.nix
RUN . /root/.nix-profile/etc/profile.d/nix.sh \
    && nix-shell /tmp/shell.nix --run "echo 'nix dependencies cached'" \
    && rm /tmp/shell.nix
