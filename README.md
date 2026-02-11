# Starter

## Tech Stack

**Backend**
- Go 1.25+
- Google Genkit (AI)
- PostgreSQL 17
- Valkey (Redis-compatible)

**Frontend**
- SvelteKit (SPA mode)
- Svelte 5
- Tailwind CSS 4
- TypeScript

**Tools**
- goose (migrations)
- sqlc (type-safe SQL)
- Air (hot reload)
- Bun (package manager)

## Requirements

- Go 1.25+
- Bun
- Docker

## Setup

```sh
# Install dependencies
make install

# Copy env and fill in values
cp .env.example .env.development

# Generate secrets
openssl rand -base64 32  # for POSTGRES_PASSWORD
```

## Development

```sh
# Start everything (DB, Valkey, Go, SvelteKit)
make dev
```

- Go API: http://localhost:3400
- SvelteKit: http://localhost:5173
- Genkit UI: http://localhost:4000

## Production

Local development runs over HTTP for convenience. Production must run behind HTTPS only.

### Required settings

- `ENV=production`
- `APP_BASE_URL=https://your-domain` (used in verification links)
- `POSTGRES_SSLMODE=require`
- `GOOGLE_REDIRECT_URI` must be HTTPS and match the Google OAuth console

### Auth + cookies

- Session cookie name is `__Host-session` in production and `session` in development.
- Session cookie flags:
  - `HttpOnly` always
  - `Secure` in production (HTTPS only)
  - `SameSite=Strict` in production
  - `Path=/`, `Max-Age=7 days`
- OAuth state/verifier cookies:
  - `HttpOnly` always
  - `Secure` in production
  - `SameSite=Lax`
  - `Path=/api/auth/google/callback`, `Max-Age=5 minutes`
- `AUTH_COOKIE_SECURE` overrides the secure flag; if set to `false`, the cookie name falls back to `session` (no `__Host-` prefix).

### Reverse proxy / trusted IP

When running behind a reverse proxy (nginx, Cloudflare, AWS ALB), set `TRUSTED_PROXY_HEADER` to the header your proxy uses for the real client IP:

```sh
# nginx / Cloudflare
TRUSTED_PROXY_HEADER="X-Forwarded-For"

# or if your proxy sets X-Real-IP
TRUSTED_PROXY_HEADER="X-Real-IP"
```

This is used for **rate limiting** and **audit logging**. Without it, all requests appear to come from the proxy's IP.

Leave empty (or unset) when not behind a proxy — the app uses `RemoteAddr` directly, which is correct and safe in that case. Do not enable this without a trusted proxy, as clients could spoof the header to bypass rate limits.

### Security headers

- HSTS is enabled in production with `max-age=31536000; includeSubDomains`.

## Other Commands

```sh
make db          # Start Postgres and Valkey
make db-stop     # Stop Postgres and Valkey
make swag        # Generate OpenAPI spec
make types       # Generate TypeScript types
make build       # Production build
make run         # Run production binary
```
