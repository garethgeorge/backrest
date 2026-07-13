# Authentication & Security

This page describes Backrest's security model: how authentication works, which addresses Backrest listens on, and which files on disk are sensitive.

## Authentication Model

Backrest ships with **no default credentials**. On first launch, the web UI prompts you to create a username and password before anything else can be configured.

Under the hood:

- **Password storage** — passwords are hashed with bcrypt and stored in the `users` section of the [config file](/docs/configuration). Plaintext passwords are never written to disk.
- **Sessions** — a successful login issues a JWT (signed with HS256) that expires after **7 days**, after which you log in again. The signing secret is 64 random bytes generated on first startup.
- **HTTP Basic auth** — every request also accepts HTTP Basic authentication with the same username and password. This is convenient for scripting against the [API](/docs/api) or for tools that can't handle a login flow.

Multiple users can be defined in the config file. All users have full access; there are no roles or per-resource permissions.

## Disabling Authentication

Authentication can be turned off entirely (via the checkbox in the UI's Settings screen, or by setting `"auth": {"disabled": true}` in the config file). When disabled, all requests are served without any credential checks.

::: warning
Only disable authentication if the interface is genuinely unreachable by anyone but you: bound to `127.0.0.1` on a single-user machine, or sitting behind a reverse proxy that performs its own authentication. Anyone who can reach an unauthenticated Backrest UI can read your repository credentials and delete your snapshots.
:::

## Network Exposure

By default, Backrest binds to `127.0.0.1:9898`, which is reachable only from the local machine. The listen address is controlled by the `BACKREST_PORT` environment variable or the `--bind-address` flag (the flag takes precedence if both are set). A value like `:9898` or `0.0.0.0:9898` listens on all interfaces.

Defaults vary by install method:

| Install method | Default bind |
| --- | --- |
| Linux/macOS `install.sh` | `127.0.0.1:9898` (localhost only); pass `--allow-remote-access` to bind `0.0.0.0:9898` |
| Docker image | `0.0.0.0:9898` inside the container — exposure is governed by your port mapping |
| Running the binary directly | `127.0.0.1:9898` |

::: tip Docker port mapping
With Docker, publish the port as `127.0.0.1:9898:9898` (rather than `9898:9898`) if you only want local access. A bare mapping exposes the UI on every host interface and, with default Docker firewall rules, potentially to your whole network.
:::

For remote access, prefer either a VPN/overlay network (WireGuard, Tailscale) or a TLS-terminating reverse proxy over exposing the port directly. Backrest serves plain HTTP, so credentials sent to a remotely exposed instance without TLS travel unencrypted.

## Secrets on Disk

Two locations contain sensitive data (see [Configuration & Paths](/docs/configuration) for where these live on each OS):

- **`config.json`** — contains your repository passwords and any credentials you entered as repo environment variables (S3 keys, B2 keys, etc.) **in plaintext**. Backrest sets its permissions to `0600` (owner read/write only) whenever it writes the file; keep it that way, and make sure the Backrest user account itself is protected.
- **`<data dir>/jwt-secret`** — the session-signing key, written with `0600` permissions. Deleting this file and restarting Backrest invalidates all existing login sessions.

To keep cloud credentials out of the config file, note that repo environment variables, flags, and URIs support `${VAR}` expansion from Backrest's process environment. You can set `AWS_SECRET_ACCESS_KEY` in the service environment (e.g. a systemd override or Docker secrets) and reference `${AWS_SECRET_ACCESS_KEY}` in the repo config.

::: info Back up your config
`config.json` is also what you need to rebuild your setup after a disaster. Most importantly, it holds your repository passwords, without which your backups cannot be decrypted; restic provides no recovery mechanism. Keep a copy somewhere safe and treat it with the same sensitivity as a password manager export.
:::

## Resetting a Lost Password

If you're locked out of the UI:

1. Stop Backrest.
2. Open `config.json` (Linux/macOS: `~/.config/backrest/config.json`, Windows: `%appdata%\backrest\config.json`, Docker: `/config/config.json` in the container).
3. Delete the `"users"` key from the `"auth"` section.
4. Start Backrest again.

On the next visit, the UI runs first-time setup again and asks you to create a new username and password. Your repos, plans, and operation history are untouched.

## Reverse Proxies & TLS

Backrest has no built-in TLS support; it serves HTTP (with h2c, i.e. cleartext HTTP/2, for its streaming APIs). To get HTTPS, put a reverse proxy such as Caddy, nginx, or Traefik in front of it and terminate TLS there. Two proxy-specific considerations:

- The Web UI uses long-lived streaming requests, so the proxy needs generous read timeouts.
- [Multihost sync](/docs/multihost) between Backrest instances requires the proxy to support h2c or gRPC-style HTTP/2 forwarding.

Working configurations for common proxies are collected in the [reverse proxy cookbook](/cookbooks/reverse-proxy-examples). If your proxy layer already authenticates users (e.g. Authelia, Authentik, or proxy-level basic auth), you can [disable Backrest's built-in authentication](#disabling-authentication) behind it, provided the proxy is the only route to the Backrest port.
