# Storage Backends

Backrest stores your backups in standard [restic](https://restic.net) repositories, which means it supports every storage backend that restic supports: local disks, SFTP servers, S3-compatible object stores, Backblaze B2, Azure Blob Storage, Google Cloud Storage, restic's own rest-server, and (via rclone) dozens of additional providers.

This guide covers how to configure each backend in Backrest: the repository URI syntax, the credentials each backend expects, and how Backrest passes environment variables and flags through to restic. For deep details on any individual backend, the [restic documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html) is the authoritative reference.

## How Repository Configuration Works

When you add a repository in Backrest, three fields together determine how restic connects to your storage:

- **Repository URI** — tells restic which backend to use and where the repository lives. The scheme prefix (`s3:`, `b2:`, `sftp:`, ...) selects the backend; a plain filesystem path selects local storage.
- **Environment variables** — per-repository `KEY=VALUE` pairs, passed to every restic command Backrest runs against this repository. This is where backend credentials go.
- **Flags** — extra command-line flags appended to every restic invocation for this repository (for example `-o sftp.args=...` or `--limit-upload`).

::: info Repositories stay portable
Backrest does not wrap or alter the repository format. Any repository you create through Backrest is a plain restic repository, and you can operate on it directly with the restic CLI using the same URI, password, and environment variables.
:::

### Password Precedence

The repository password entered in Backrest takes precedence over any environment-provided password. When a password is set in the repository config, Backrest exports it as `RESTIC_PASSWORD` *after* importing the system environment, and explicitly clears `RESTIC_PASSWORD_FILE` and `RESTIC_PASSWORD_COMMAND` so that values in the host or container environment cannot override it.

If you prefer to manage the password outside of Backrest's config file, leave the password field empty and instead provide one of the following as a per-repository environment variable:

| Variable | Meaning |
| --- | --- |
| `RESTIC_PASSWORD` | The password itself. |
| `RESTIC_PASSWORD_FILE` | Path to a file containing the password. |
| `RESTIC_PASSWORD_COMMAND` | Command that prints the password to stdout. |

The Add Repository form requires a password by one of these mechanisms unless you pass the `--insecure-no-password` flag, which creates an unencrypted repository.

::: warning
Restic cannot decrypt data without the repository password, so losing the password means losing access to your backups. Store it somewhere safe outside the machine being backed up.
:::

### Variable Expansion

The repository URI, environment variables, and flags all support `${VAR}` expansion from the environment of the Backrest process itself. For example, you can keep secrets out of `config.json` by referencing variables set in your systemd unit or docker-compose file:

```json
{
  "id": "s3-backups",
  "uri": "s3:s3.amazonaws.com/${BUCKET_NAME}",
  "env": [
    "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}",
    "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}"
  ]
}
```

Only the `${VAR}` form is expanded (not `$VAR`), and variables that are unset in Backrest's environment expand to an empty string.

## Local Paths

For local storage, use an absolute filesystem path as the URI, with no scheme prefix.

```text
/mnt/backup-drive/backrest-repo
```

No environment variables are required. A few things to keep in mind:

- **Docker**: the path is interpreted inside the container. Mount your backup volume into the container and use the container-side path as the URI.
- **Removable media**: if the drive may be unplugged mid-operation, enable the repository's **auto unlock** option so Backrest automatically removes stale locks left behind by interrupted operations. Only enable this when no other host writes to the same repository, since it will also remove locks legitimately held by another instance.
- Backing up to the same disk you are backing up *from* protects against accidental deletion, but not against disk failure. Pair a local repository with a remote one for important data.

## S3-Compatible Storage

Works with AWS S3 and any S3-compatible service: MinIO, Wasabi, Backblaze B2's S3 endpoint, Cloudflare R2, Garage, and others.

```text
s3:s3.amazonaws.com/bucket-name
```

For non-AWS services, embed the endpoint URL in the URI:

```text
s3:https://minio.example.com/bucket-name
s3:https://s3.us-west-1.wasabisys.com/bucket-name
```

| Environment variable | Purpose |
| --- | --- |
| `AWS_ACCESS_KEY_ID` | Access key ID. |
| `AWS_SECRET_ACCESS_KEY` | Secret access key. |
| `AWS_SHARED_CREDENTIALS_FILE` | Alternative: path to a shared AWS credentials file instead of the two keys above. |

Provide either both key variables or the credentials file; the Add Repository form validates that one of these combinations is present. Region selection and other S3 options are covered in the [restic S3 documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#amazon-s3).

::: tip
Create a dedicated bucket and an access key scoped to just that bucket for Backrest, rather than reusing account-wide credentials.
:::

## Backblaze B2

```text
b2:bucketname:path/to/repo
```

| Environment variable | Purpose |
| --- | --- |
| `B2_ACCOUNT_ID` | Application key ID. |
| `B2_ACCOUNT_KEY` | Application key. |

B2 buckets can also be accessed through their S3-compatible endpoint using the `s3:` scheme above; see the [restic B2 documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#backblaze-b2) for the tradeoffs.

## Azure Blob Storage

```text
azure:container-name:/
```

| Environment variable | Purpose |
| --- | --- |
| `AZURE_ACCOUNT_NAME` | Storage account name (always required). |
| `AZURE_ACCOUNT_KEY` | Account key, **or** |
| `AZURE_ACCOUNT_SAS` | Shared access signature token as an alternative to the account key. |

Set `AZURE_ACCOUNT_NAME` plus either `AZURE_ACCOUNT_KEY` or `AZURE_ACCOUNT_SAS`. Further options are in the [restic Azure documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#microsoft-azure-blob-storage).

## Google Cloud Storage

```text
gs:bucket-name:/
```

| Environment variable | Purpose |
| --- | --- |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to a service account credentials JSON file. |
| `GOOGLE_PROJECT_ID` | The GCP project ID (used together with the credentials file). |
| `GOOGLE_ACCESS_TOKEN` | Alternative: a raw OAuth2 access token instead of the two above. |

::: warning Docker note
`GOOGLE_APPLICATION_CREDENTIALS` is a *path*, and restic runs inside the container. Mount the credentials JSON into the container (for example to `/config/gcs-key.json`) and point the variable at the container-side path.
:::

See the [restic GCS documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#google-cloud-storage) for details on creating a service account.

## SFTP

```text
sftp:user@host:/path/to/repo
```

Backrest has first-class SFTP support, including built-in SSH key generation and host key management from the WebUI. Because there is enough to cover, SFTP has its own page: see the [SFTP & SSH Remotes guide](/guides/sftp).

## rclone

The `rclone:` scheme lets restic reach any of the many providers rclone supports (Google Drive, OneDrive, Dropbox, Proton Drive, and more):

```text
rclone:myremote:path/to/repo
```

where `myremote` is a remote you have configured with `rclone config`. Requirements:

- The `rclone` binary must be installed and on the `PATH` of the machine (or container) running Backrest.
- Your rclone config file must be readable by the Backrest process.

In Docker, the `ghcr.io/garethgeorge/backrest:latest` (alpine-based) image ships with rclone preinstalled. Backrest runs as root in the container, so mount your rclone config to root's default location:

```yaml
services:
  backrest:
    image: ghcr.io/garethgeorge/backrest:latest
    volumes:
      - ./rclone-config:/root/.config/rclone
      # ... other volumes
```

The `:scratch` image variant does not include rclone (or a shell), so the rclone backend is unavailable there. See the [restic rclone documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#other-services-via-rclone) for advanced options such as tuning rclone flags.

## rest-server

restic's [rest-server](https://github.com/restic/rest-server) is a lightweight HTTP server purpose-built for hosting restic repositories. It is a good option for backing up to a machine you control, and is generally faster than SFTP.

```text
rest:https://user:pass@host:8000/repo-name
```

Credentials are carried in the URI rather than environment variables. Run rest-server with `--append-only` on the receiving machine for protection against a compromised client deleting its own backups. See the [restic REST server documentation](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html#rest-server) for setup details.

## Troubleshooting

- **"Missing env vars ... for scheme ..."** — the Add Repository form checks that the expected credential variables for your URI scheme are present (the combinations in the tables above) and lists exactly which are missing.
- **Credentials look right but restic fails to connect** — remember that `${VAR}` references are expanded from *Backrest's* process environment. If the variable is not set for the Backrest service (systemd unit, docker-compose `environment:` block), it silently expands to an empty string.
- **Works with restic CLI but not in Backrest** — compare environments: Backrest passes only the system environment plus the per-repository variables. Shell-only configuration (e.g. variables exported in your `~/.bashrc`) is not visible to a Backrest daemon started by systemd or Docker.
- Test any repository outside Backrest by running the restic CLI with the same URI and environment variables — the repository formats are identical.

## Next Steps

- Set up [SFTP & SSH remotes](/guides/sftp)
- Configure [backup scheduling](/guides/scheduling) and [retention & repo health](/guides/repo-health)
- Review [where Backrest stores its own configuration](/docs/configuration)
