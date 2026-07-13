# Reverse Proxies

## Introduction

Reverse proxies like [Caddy](https://caddyserver.com/), [Traefik](https://traefik.io/traefik/), and [nginx](https://nginx.org/) can be configured to front your Backrest instance, adding TLS termination and an extra layer of access control.

Backrest's WebUI and API are served over [ConnectRPC](https://connectrpc.com/), which works over both HTTP/1.1 and HTTP/2 and relies on long-lived server-streaming requests (e.g. the operation event stream that keeps the WebUI updated in real time). [Multihost Sync](/docs/multihost) additionally uses a long-lived bidirectional stream that requires HTTP/2 end-to-end.

This means your proxy should:

- **Not buffer responses** — streamed events must be flushed to the browser as they happen.
- **Allow long-lived requests** — read/write timeouts should be hours, not seconds. The streams are intentionally persistent.
- **Speak HTTP/2 (or h2c) to Backrest if you use multihost sync** — the sync stream will not work across an HTTP/1.1 hop. The WebUI alone works fine over HTTP/1.1.

::: warning
A reverse proxy is not a substitute for authentication. Enable Backrest's built-in authentication, or place an authenticating proxy in front, before exposing the WebUI beyond your trusted network. See [Authentication & Security](/guides/security).
:::

## Caddy

For this example, we'll be running Caddy alongside Backrest via `docker-compose.yml`, but you can adapt this config to your environment.

Here is an example `docker-compose.yml`:

```yaml
services:
  backrest:
    image: ghcr.io/garethgeorge/backrest:latest
    container_name: backrest
    hostname: <YOUR PROXIED FQDN HERE (example: backrest.example.com)>
    volumes:
      - ./backrest/data:/data
      - ./backrest/config:/config
      - ./backrest/cache:/cache
      - /MY-BACKUP-DATA:/userdata # mount your directories to backup somewhere in the filesystem
      - /MY-REPOS:/repos # (optional) mount your restic repositories somewhere in the filesystem.
    environment:
      - BACKREST_DATA=/data # path for backrest data. restic binary and the database are placed here.
      - BACKREST_CONFIG=/config/config.json # path for the backrest config file.
      - XDG_CACHE_HOME=/cache # path for the restic cache which greatly improves performance.
    restart: unless-stopped
  caddy:
    image: caddy
    container_name: caddy
    ports:
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile
    restart: unless-stopped
```

Your Caddyfile should look like this:

```Caddyfile
{
	https_port 443
}

backrest.example.com {
  tls internal
  reverse_proxy backrest:9898
}
```

Some items to note:

- The `reverse_proxy` line in your Caddyfile **must** match your Backrest container's name!
- Caddy applies no read/write timeouts by default and automatically flushes streaming responses, so this minimal config already handles the WebUI's long-lived operation streams correctly.
- You can extend this with [acme_dns](https://github.com/caddy-dns/acmedns) to obtain certificates for your endpoint.
- `tls internal` means that Caddy will generate and utilize a self-signed certificate.
- You can create an [authentication portal](https://caddyserver.com/docs/json/apps/http/servers/routes/handle/auth_portal/) to allow login via Google, etc.
- You can opt to have Caddy listen to requests on port 80 (HTTP) but that's not recommended for security reasons.

### Caddy for Multihost Sync

If your instance acts as a [Multihost Sync](/docs/multihost) server for remote clients, you only need to expose the sync RPC path — the UI and admin API can stay on your trusted network:

```Caddyfile
backrest.example.com {
    @sync path /v1sync.BackrestSyncService/*
    reverse_proxy @sync h2c://backrest:9898 {
        flush_interval -1
        transport http {
            read_timeout 24h
            write_timeout 24h
        }
    }
}
```

Why each setting matters:

- `@sync path /v1sync.BackrestSyncService/*` exposes only the sync endpoint; every other path returns 404, keeping the WebUI and admin API off the public internet.
- `h2c://` proxies to Backrest over cleartext HTTP/2. The sync protocol is a long-lived bidirectional stream and **requires HTTP/2 on the upstream hop** — a plain `http://` upstream will not work.
- `flush_interval -1` disables response buffering so stream messages are forwarded immediately.
- `read_timeout 24h` / `write_timeout 24h` keep the intentionally persistent sync stream from being cut off by proxy timeouts.

If you also want to serve the WebUI through the same proxy, add a second `reverse_proxy backrest:9898` block without the path matcher — but be aware this exposes the admin API too, so keep authentication enabled.

## Traefik

Traefik integrates with Docker via container labels. Traefik streams responses without buffering by default, so the WebUI works with a minimal router + service definition:

```yaml
services:
  backrest:
    image: ghcr.io/garethgeorge/backrest:latest
    container_name: backrest
    volumes:
      - ./backrest/data:/data
      - ./backrest/config:/config
      - ./backrest/cache:/cache
      - /MY-BACKUP-DATA:/userdata
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.backrest.rule=Host(`backrest.example.com`)"
      - "traefik.http.routers.backrest.entrypoints=websecure"
      - "traefik.http.routers.backrest.tls.certresolver=letsencrypt"
      - "traefik.http.services.backrest.loadbalancer.server.port=9898"
    restart: unless-stopped

  traefik:
    image: traefik:v3
    container_name: traefik
    command:
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.email=you@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.tlschallenge=true"
    ports:
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt
    restart: unless-stopped
```

Notes for multihost sync through Traefik:

- The sync stream needs HTTP/2 on the upstream hop. Tell Traefik to speak cleartext HTTP/2 to Backrest by adding the label `traefik.http.services.backrest.loadbalancer.server.scheme=h2c` (the WebUI works over h2c as well).
- The sync stream keeps its request body open indefinitely, so the entrypoint's read timeout must not apply. If sync connections drop after about a minute, disable it: `--entrypoints.websecure.transport.respondingTimeouts.readTimeout=0`.

## nginx

nginx works well in front of the Backrest WebUI. The key requirements are HTTP/1.1 to the upstream, disabled buffering, and long read timeouts for the operation event streams:

```nginx
server {
    listen 443 ssl;
    http2 on; # nginx >= 1.25.1; on older versions use "listen 443 ssl http2;"
    server_name backrest.example.com;

    ssl_certificate     /etc/nginx/certs/backrest.example.com.crt;
    ssl_certificate_key /etc/nginx/certs/backrest.example.com.key;

    location / {
        proxy_pass http://127.0.0.1:9898;

        # ConnectRPC requires at least HTTP/1.1 on the upstream connection.
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Required for the WebUI's long-lived operation event streams:
        proxy_buffering off;         # flush streamed responses to the client immediately
        proxy_request_buffering off; # pass request bodies through as they arrive
        proxy_read_timeout 1d;       # keep idle streams open instead of timing out at 60s
        proxy_send_timeout 1d;
    }
}
```

::: warning
nginx's `proxy_pass` speaks HTTP/1.1 to the upstream, which is fine for the WebUI but **not sufficient for multihost sync** — the sync stream requires HTTP/2 end-to-end. If this instance serves remote sync clients, front the `/v1sync.BackrestSyncService/` path with Caddy or Traefik (using an h2c upstream) instead.
:::

## See Also

- [Multihost Sync](/docs/multihost) — the sync protocol's proxy requirements and troubleshooting.
- [Authentication & Security](/guides/security) — locking down Backrest before exposing it to a network.
