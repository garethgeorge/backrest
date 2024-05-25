# Reverse Proxy Examples

## Introduction

Reverse proxies like [Caddy](https://caddyserver.com/) and [Traefik](https://traefik.io/traefik/) can be configured to front and protect your Backrest endpoint.

## Using Caddy
For this example, we'll be running Caddy alongside Backrest via docker-compose.yaml but you can adapt this config to your environment.

Here is an example docker-compose.yaml:
```
version: "3.2"
services:
  backrest:
    image: garethgeorge/backrest
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
    depends_on:
      - caddy
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
```
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
- You can extend this with [acme_dns](https://github.com/caddy-dns/acmedns) to obtain certificates for your endpoint.
- `tls internal` means that Caddy will generate and utilize a self-signed certificate.
- You can create an [authentication portal](https://caddyserver.com/docs/json/apps/http/servers/routes/handle/auth_portal/) to allow login via Google, etc.
- You can opt to have Caddy listen to requests on port 80 (HTTP) but that's not recommended for security reasons.
