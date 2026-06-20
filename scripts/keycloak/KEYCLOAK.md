# Keycloak (local OIDC testing)

A Keycloak + Postgres stack is bundled in `docker-compose.yml` for testing the
backrest OIDC auth driver locally.

## Start

```bash
docker compose up -d postgres keycloak
```

Keycloak comes up at <http://localhost:8080>. The Postgres data dir is `tmpfs`
(in-memory), so the realm state is **not** persisted across container recreation.

## Admin console

- URL: <http://localhost:8080/admin>
- Username: `admin`
- Password: `admin`

## OIDC client for backrest

| Setting       | Value                                                  |
| ------------- | ------------------------------------------------------ |
| Realm         | `master`                                               |
| Issuer URL    | `http://localhost:8080/realms/master`                  |
| Client ID     | `backrest`                                             |
| Client secret | `3jpW4nLOccDiEPm1t1XnbCNH2GKNTsuz`                      |

Set the matching values under **Settings → Auth → OIDC** in backrest (or in
`config.json`). The redirect URL is `http://localhost:9898/auth/oidc/callback`
(derived from the request when left blank); it must be listed as a valid
redirect URI on the `backrest` client in Keycloak.

## Persisting changes

State lives only in the in-memory Postgres. To save realm/client/user changes
you make in the admin console, dump them to `keycloak_dump.sql`:

```bash
scripts/export-keycloak.sh
```

Re-enable the `keycloak_dump.sql` volume mount on the `postgres` service in
`docker-compose.yml` to have that dump re-seed the database on every fresh
container.
