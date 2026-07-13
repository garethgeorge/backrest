# SFTP & SSH Remotes

Backrest can back up to any machine reachable over SSH, such as a NAS or a VPS, using restic's SFTP backend. A built-in setup flow generates an SSH key pair and records the server's host key, so most of the setup can be done from the web UI.

Under the hood, restic's SFTP backend shells out to the `ssh` binary, so an OpenSSH client must be installed on the machine (or in the container) running Backrest.

## Repository URI Formats

restic accepts two URI shapes for SFTP:

```text
sftp:user@host:/path/to/repo        # scp-style (cannot carry a port)
sftp://user@host:2222/path/to/repo  # URL-style (port allowed)
```

If your server listens on a non-standard port, either use the URL-style form or set the **SFTP Port** field described below. The path should be absolute, or relative to the SSH user's home directory.

## Built-In Setup Flow

When you enter an `sftp:` URI in the Add Repository form, an SFTP configuration section appears with three fields (**SFTP Identity File**, **SFTP Port**, and **SFTP Known Hosts**) plus an optional **Setup SSH Key** helper.

<img src="/screenshots/sftp-repo-setup.png" alt="Add Repository modal with an SFTP URI and the Setup SSH Key section expanded" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

### Generate a Key

Expand **Setup SSH Key** and click **Generate Key**. Backrest will:

1. **Generate an Ed25519 key pair** dedicated to this host, stored as `id_ed25519_<host>` (plus `.pub`) in a Backrest-managed SSH directory: `.backrest-ssh/` next to your config file (in Docker: `/config/.backrest-ssh/`). If a key for this host already exists there, it is reused rather than replaced.
2. **Scan the server's host key** with `ssh-keyscan` and append it to `.backrest-ssh/known_hosts`. If the host is already present in that file or in your `~/.ssh/known_hosts`, the scan is skipped. If the host is unreachable, key generation still succeeds and Backrest shows a warning so you can add the host key manually later.
3. **Fill in the fields** — the identity file and known hosts paths are populated automatically, and the public key is displayed for you to copy.

### Authorize the Key on the Server

Backrest does not install the key on the server for you. Append the displayed public key to the SSH user's `authorized_keys` on the remote machine:

```bash
# on the remote server, as the backup user
mkdir -p ~/.ssh && chmod 700 ~/.ssh
echo 'ssh-ed25519 AAAA... backrest' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

Then submit the form. Backrest translates the three SFTP fields into a restic flag on the repository, so this is equivalent to configuring:

```text
--option=sftp.args='-oBatchMode=yes -i "/path/to/.backrest-ssh/id_ed25519_host" -oUserKnownHostsFile="/path/to/.backrest-ssh/known_hosts"'
```

::: info Batch mode
Backrest always runs SSH with `-oBatchMode=yes` for SFTP repositories (it injects this flag automatically if you have not set `sftp.args` yourself). This makes SSH fail immediately with a clear error instead of hanging on an interactive password or host-key prompt, which a background service cannot answer.
:::

::: warning Windows
The automated key setup flow is not available on Windows. Use an existing key configured through your SSH client instead, as described below.
:::

## Using Your Own Keys

The built-in flow is optional. If you already manage SSH keys, you have two equivalent options:

**Option 1 — point the Identity File field at your key.** Enter the path to your private key (and optionally a known_hosts file and port) in the SFTP fields. The key must not have a passphrase, since restic runs non-interactively.

**Option 2 — set the flag directly.** Add a repository flag yourself:

```text
-o sftp.args="-i /path/to/private_key"
```

Any options you can pass to `ssh` work here (`-p`, `-oUserKnownHostsFile=...`, etc.). When you set `sftp.args` manually, Backrest does not inject `BatchMode=yes`, so include `-oBatchMode=yes` if you want fail-fast behavior.

**Option 3 — use your SSH config.** If the host is defined in `~/.ssh/config` (with `IdentityFile`, `Port`, and so on), a bare `sftp:alias:/path` URI will pick that configuration up, since restic invokes the regular `ssh` binary. Note that a daemonized Backrest runs as its service user — it reads *that* user's SSH config, not your desktop user's.

## SFTP in Docker

The two published image variants differ here:

| Image | OpenSSH client | SFTP support |
| --- | --- | --- |
| `ghcr.io/garethgeorge/backrest:latest` (alpine) | Yes (including `ssh-keyscan`) | Full, including the built-in setup flow |
| `ghcr.io/garethgeorge/backrest:scratch` | No (no shell either) | Not available |

With the default alpine image and the standard `/config` volume mount, the built-in setup flow works out of the box: generated keys and the known_hosts file live in `/config/.backrest-ssh/`, so they survive container recreation along with the rest of your config.

If you prefer to generate keys on the Docker host and mount them into the container yourself, that approach is written up in the [SSH remotes with Docker Compose cookbook](/cookbooks/ssh-remote).

## Troubleshooting

**"Host key verification failed" or "Remote host identification has changed"**

The server's host key is not in the known_hosts file restic is using, or it no longer matches. A changed host key can mean the server was reinstalled or, rarely, that something is intercepting the connection. Re-run **Setup SSH Key** to scan the current host key, or update the file manually:

```bash
ssh-keyscan -H your-server >> /path/to/.backrest-ssh/known_hosts
```

If the key genuinely changed, delete the stale entry first: `ssh-keygen -R your-server -f /path/to/known_hosts`.

**"Permission denied (publickey)"**

- The public key is not in the remote user's `~/.ssh/authorized_keys`, or the wrong user is in the URI.
- Permissions on the remote side are too loose — sshd refuses keys when `~/.ssh` is not `700` or `authorized_keys` is not `600`.
- The identity file path is wrong from Backrest's point of view (remember: in Docker it must be a container-side path).

**Operations hang or fail immediately with a cryptic SSH error**

With `BatchMode=yes` in place, anything that would normally prompt interactively (password auth, unknown host key, encrypted key passphrase) fails immediately instead. Read the error text; restic forwards SSH's stderr. Reproduce the connection outside Backrest to isolate the problem:

```bash
ssh -oBatchMode=yes -i /path/to/key user@host true
```

If that command succeeds silently, the SSH layer is fine and the issue is elsewhere (path permissions on the repository directory, for example).

**Repository directory not writable**

restic needs to create the repository directory structure under the URI path. Ensure the SSH user owns the target path on the server.

## Alternative: rest-server over SSH

SFTP works with any SSH server, but restic's [rest-server](https://github.com/restic/rest-server) is faster and supports an append-only mode that protects existing backups even if the client machine is compromised. If you control the remote machine, consider running rest-server on it instead; see the `rest:` section of the [Storage Backends guide](/guides/storage-backends).

## Next Steps

- Browse all supported backends in the [Storage Backends guide](/guides/storage-backends)
- Mount-your-own-keys Docker setup: [SSH remotes with Docker Compose cookbook](/cookbooks/ssh-remote)
- Configure [scheduling](/guides/scheduling) and [retention & repo health](/guides/repo-health) for your new repository
