# Authentication
Backrest supports multiple authentication modes which can be configured under **Settings → Authentication**.

## Disabled Authentication
Only use this when Backrest is not network-accessible (e.g. `localhost` behind a firewall or reverse proxy that handles auth itself).

## Local Authentication (Default)
Add users in **Settings → Authentication**. Passwords are hashed and stored in the Backrest config file.

## OIDC Authentication
OIDC delegates authentication to an external identity provider (IdP) — Keycloak, Dex, Authentik, GitHub, Google, Okta, etc. — using the standard authorization-code flow with PKCE.

### How it works
1. User visits Backrest and is redirected to `/auth/oidc/login`
2. Backrest redirects the browser to the IdP's authorization endpoint
3. The user authenticates with the IdP
4. The IdP redirects back to `/auth/oidc/callback` with an authorization code
5. Backrest exchanges the code for an ID token, verifies it, checks the allow-lists, and issues a session cookie

### Configuring OIDC in the UI

1. Open **Settings → Authentication**
2. Set **Authentication driver** to **OIDC (OpenID Connect)**
3. Fill in the fields that appear:

| Field | Required | Description |
|-------|----------|-------------|
| **Issuer URL** | Yes | OIDC provider base URL. Backrest fetches `{issuer_url}/.well-known/openid-configuration` on startup. E.g. `https://accounts.google.com` |
| **Client ID** | Yes | OAuth2 client ID registered with your IdP. |
| **Client secret** | No | OAuth2 client secret. Leave empty for public clients (PKCE only). |
| **Scopes** | No | Space-separated OAuth2 scopes. Defaults to `openid email profile`. |
| **Redirect URL** | No | Callback URL registered with the IdP. Leave empty to derive it from the request host — set explicitly if Backrest is behind a reverse proxy. |
| **Allowed emails** | No | Comma-separated list of exact email addresses permitted to log in. When both allow-lists are empty, any authenticated email is allowed. |
| **Allowed domains** | No | Comma-separated bare domains (e.g. `example.com`). Only emails from these domains may log in. Checked after allowed emails. |

4. Click **Save**. Backrest will reload and redirect to the IdP on next login.

<details>
<summary><strong>IdP Setup</strong></summary>

#### Register a client

In your IdP, create an **OAuth2 / OIDC client** (sometimes called an "application") with:

- **Client type**: Confidential (if you set a client secret) or Public
- **Allowed redirect URI**: `https://backrest.example.com/auth/oidc/callback`
- **Grant type**: Authorization Code
- **Token endpoint auth method**: `client_secret_post` or `client_secret_basic` (confidential), or `none` (public)

The ID token **must** include an `email` claim. Enable the `email` scope in your IdP if it is not included by default.

#### Keycloak example

1. Create a realm (or use an existing one)
2. **Clients → Create client**
   - Client ID: `backrest`
   - Client authentication: On (confidential)
3. Under **Settings**, add `https://backrest.example.com/auth/oidc/callback` to **Valid redirect URIs**
4. Under **Credentials**, copy the client secret
5. Set **Issuer URL** to `https://keycloak.example.com/realms/<realm-name>`

#### Dex example

```yaml
# dex config snippet
staticClients:
  - id: backrest
    secret: your-client-secret
    redirectURIs:
      - https://backrest.example.com/auth/oidc/callback
    name: Backrest
```

Set **Issuer URL** to your Dex issuer (e.g. `https://dex.example.com`).

</details>

### Reverse proxy notes

If Backrest runs behind a reverse proxy (nginx, Caddy, Traefik), make sure:

- The proxy forwards the `X-Forwarded-Proto: https` header so Backrest can set `Secure` cookies and derive the correct callback URL
- Set **Redirect URL** explicitly to avoid mismatches when the proxy rewrites the `Host` header

### Troubleshooting

| Symptom | Likely cause |
|---------|-------------|
| Redirect loop on login | Auth driver not set to OIDC, or OIDC settings not saved |
| `oidc discovery` error on startup | Issuer URL is wrong or the IdP is unreachable |
| `email … is not permitted` | Email or domain not in the allow-list |
| `oidc nonce mismatch` | Browser blocked the short-lived OIDC cookies (check `SameSite` / cookie policy) |
| Callback URL mismatch | Redirect URL doesn't match what's registered in the IdP |
| Session not persisting | `Secure` cookie rejected over plain HTTP — serve Backrest over HTTPS or set `X-Forwarded-Proto: https` |
