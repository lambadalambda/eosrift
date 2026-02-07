# EosRift

Project domain: `eosrift.com`

Self-hosted, Docker-first, open-source tunnel service aiming for an ngrok-like UX.

**Status:** alpha. TCP + HTTP tunnels work, but expect rough edges and breaking changes.

This project is close to being “useful by default”, but it is not a 1.0-quality, battle-tested
service yet:

- Backwards compatibility is **not** guaranteed until `v1.0.0`.
- Treat internet exposure as risky until you’ve reviewed `SECURITY.md` and locked things down.
- Default behavior and CLI/config may change as we approach the first stable release.

## Goals

- Functional, self-hostable “ngrok clone”
- Easy deployment via **Docker / Docker Compose**
- **HTTPS termination via Caddy**
- **Server:** officially supported on Linux (Docker images + CI); likely works on other Unix-like OSes but not tested
- **Client:** Linux + macOS
- CLI UX as close to ngrok as practical (commands, flags, config, output)
- Persistence via **SQLite** (no external DB dependencies)
- Development via **TDD** (unit tests + Docker-based integration tests)

## Non-goals (initially)

- Multi-region “edge” network
- Proprietary ngrok features (SAML, paid dashboards, etc.)
- Supporting Windows clients

## Proposed stack

- **Go** (server + client): static binaries, great networking, fast iteration
- **Caddy**: TLS/HTTPS for `*.tunnel.<yourdomain>` (recommended: wildcard cert via DNS challenge)
- **SQLite**: persistence (pure-Go driver via `modernc.org/sqlite`, no CGO)
- Optional later: **React SPA** for admin UI (server stays API-first)

## What “ngrok-like” means here

- `http` and `tcp` tunnels with similar command structure and flags
- Local inspector UI (ngrok’s `localhost:4040` equivalent)
- YAML config file support (ngrok-style, at least a compatible subset)
- Authtokens and reserved names (alpha)

## Documentation

- `PLAN.md` — milestones and TDD approach
- `ARCHITECTURE.md` — proposed architecture and protocols
- `CHANGELOG.md` — notable changes (Keep a Changelog format)
- `RELEASING.md` — release checklist and “v1.0-ready” definition
- `deploy/PRODUCTION.md` — deployment notes (DNS, Caddy, firewall)
- `SECURITY.md` — threat model and security checklist
- `docs-site/` — VitePress source docs, embedded and served at `/docs/`

## Quickstart

### Server (Docker, alpha)

- `cp .env.example .env` (edit for your domain)
- (Optional) Set `EOSRIFT_AUTH_TOKEN` in `.env` to bootstrap the first authtoken
- (Optional) Set `EOSRIFT_ADMIN_TOKEN` in `.env` to enable the server admin frontend/API
- (Optional) Set `EOSRIFT_MAX_TUNNELS_PER_TOKEN` to cap active tunnels per authtoken (0 = unlimited)
- (Optional) Set `EOSRIFT_MAX_TUNNEL_CREATES_PER_MIN` to rate limit tunnel creations per authtoken (0 = unlimited)
- (Optional) Set `EOSRIFT_LOG_FORMAT=json` for structured logs
- `docker compose up -d --build`
- `curl -fsS http://127.0.0.1:8080/healthz`

Notes:

- By default, `docker-compose.yml` builds the server image locally. If you prefer a prebuilt image, use `ghcr.io/lambadalambda/eosrift-server:v0.1.1`.
- TCP tunnels require opening `EOSRIFT_TCP_PORT_RANGE_START..EOSRIFT_TCP_PORT_RANGE_END` in your firewall/security group.
- `/control` requires an authtoken (stored in SQLite). If you didn’t bootstrap one via `EOSRIFT_AUTH_TOKEN`, create one with `docker compose exec server /eosrift-server token create`.
- `docker-compose.yml` defaults `EOSRIFT_TRUST_PROXY_HEADERS=1` (safe with the default localhost-only server bind + Caddy in front). If you expose the server directly to untrusted clients, set it to `0` to prevent `X-Forwarded-*` spoofing.
- Once deployed with DNS + Caddy, `https://<EOSRIFT_BASE_DOMAIN>/` serves a small landing page (the tunnel subdomains still route to tunnels).
- If `EOSRIFT_ADMIN_TOKEN` is set, `https://<EOSRIFT_BASE_DOMAIN>/admin` serves the admin frontend.
- `https://<EOSRIFT_BASE_DOMAIN>/docs/` serves the embedded docs site.

### Client (build, recommended for now)

This repo doesn’t require Go on your host; you can build with Docker:

- Linux (example): `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./scripts/go build -o bin/eosrift ./cmd/client`
- macOS (example): `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 ./scripts/go build -o bin/eosrift ./cmd/client`

### Client (install from GitHub Releases)

Release artifacts are built by GitHub Actions on tags (`v*`). Install the client on macOS/Linux using:

- Latest release: `./scripts/install.sh`
- Specific version: `./scripts/install.sh --version v0.1.1`

By default, this installs to `~/.local/bin/eosrift` (override with `--dir`).

By default, the install script downloads from `lambadalambda/eosrift` (override with `--repo`).

Release assets include `checksums.txt` so you can verify downloads.

### Client config (alpha)

Save your authtoken (ngrok-like):

- `./bin/eosrift config add-authtoken <token>`

Set your default server (recommended for self-hosting):

- `./bin/eosrift config set-server https://<yourdomain>`
- or env: `EOSRIFT_SERVER_ADDR=https://<yourdomain>`

If unset, the client defaults to `https://eosrift.com`.

Set your default host header mode (optional; ngrok-like):

- `./bin/eosrift config set-host-header preserve`
- `./bin/eosrift config set-host-header rewrite`
- `./bin/eosrift config set-host-header <value>`

Default config path:

- Linux: `~/.config/eosrift/eosrift.yml` (or `$XDG_CONFIG_HOME/eosrift/eosrift.yml`)
- macOS: `~/Library/Application Support/eosrift/eosrift.yml`

Supported keys (compatible subset): `authtoken`, `server_addr`, `host_header`, `inspect`, `inspect_addr`.

Named tunnel keys (alpha) live under `tunnels:`:

- Per tunnel: `proto` (`http`/`tcp`), `addr`
- HTTP-only: `domain`, `subdomain`, `basic_auth`, `allow_method`, `allow_path`, `allow_path_prefix`, `allow_cidr`, `deny_cidr`, `request_header_add`, `request_header_remove`, `response_header_add`, `response_header_remove`, `host_header`
- TCP-only: `remote_port`
- Optional: `inspect` (HTTP tunnels only)

Config precedence:

- Flags > environment > config file > defaults
- Server address (`--server`):
  - `--server` > `EOSRIFT_SERVER_ADDR` > `EOSRIFT_CONTROL_URL` > `server_addr` > `https://eosrift.com`
- Authtoken (`--authtoken`):
  - `--authtoken` > `EOSRIFT_AUTHTOKEN` > `authtoken` > empty
- Inspector:
  - `--inspect` > `inspect` > `true`
  - `--inspect-addr` > `EOSRIFT_INSPECT_ADDR` > `inspect_addr` > `127.0.0.1:4040` (tries up to `:5000`)
- Host header (`--host-header`):
  - `--host-header` > `host_header` > `preserve`

### Named tunnels + `start` (alpha)

Define tunnels in `eosrift.yml`, then start them by name (ngrok-like):

```yaml
version: 1
server_addr: https://eosrift.com
authtoken: <token>

tunnels:
  web:
    proto: http
    addr: 3000
    domain: demo.tunnel.eosrift.com
    basic_auth: user:pass
    allow_method:
      - GET
    allow_path:
      - /healthz
    allow_cidr:
      - 203.0.113.0/24
    request_header_add:
      X-Edge-Debug: "1"
    response_header_remove:
      - Server
  db:
    proto: tcp
    addr: 5432
    remote_port: 20005
```

- Start one: `./bin/eosrift start web`
- Start many: `./bin/eosrift start web db`
- Start all: `./bin/eosrift start --all`
- Override server/token: `./bin/eosrift start --all --server https://<yourdomain> --authtoken <token>`
- HTTPS upstreams: set `addr: https://127.0.0.1:8443` and (if needed) add `--upstream-tls-skip-verify`.

### TCP tunnel (alpha)

Expose a local TCP port through the server:

- `./bin/eosrift tcp 8080 --server https://<yourdomain>`
- Request a specific remote port: `./bin/eosrift tcp 8080 --remote-port 20005 --server https://<yourdomain>`

The client prints the allocated remote port, e.g. `Forwarding tcp://<yourdomain>:20001 -> 127.0.0.1:8080`.

Notes:

- If `--remote-port` is unused, the server auto-reserves it to your authtoken on first use.
- Manage reservations on the server with `eosrift-server tcp-reserve add|list|remove`.

### TLS tunnel (alpha)

Expose a local TLS service through a raw TCP tunnel (EosRift does **not** terminate TLS):

- `./bin/eosrift tls 443 --server https://<yourdomain>`
- Request a specific remote port: `./bin/eosrift tls 443 --remote-port 20005 --server https://<yourdomain>`

### HTTP tunnel (alpha)

Expose a local HTTP port through the server:

- `./bin/eosrift http 8080 --server https://<yourdomain>`
- Request a stable domain (ngrok-like): `./bin/eosrift http --domain demo.tunnel.<yourdomain> 127.0.0.1:8080`
- Require basic auth on the public URL: `./bin/eosrift http 8080 --basic-auth user:pass`
- Allowlist methods/paths (per tunnel): `./bin/eosrift http 8080 --allow-method GET --allow-path /healthz --allow-path-prefix /api/`
- Allowlist client IPs (CIDR): `./bin/eosrift http 8080 --allow-cidr 203.0.113.0/24`
- Header transforms (per tunnel): `./bin/eosrift http 8080 --request-header-add "X-API-Key: secret" --response-header-remove "Server"`
- Host header rewriting (ngrok-like): `./bin/eosrift http --host-header=rewrite 127.0.0.1:8080`
- Forward to a local HTTPS upstream: `./bin/eosrift http https://127.0.0.1:8443 --upstream-tls-skip-verify`

The client prints the public URL, e.g. `Forwarding https://abcd1234.tunnel.<yourdomain> -> 127.0.0.1:8080`.

### Auth (alpha)

Authtokens are stored and validated server-side (SQLite). Create one on the server, then pass it from the client:

- flag: `--authtoken <token>`
- env: `EOSRIFT_AUTHTOKEN=<token>`
- config: `eosrift config add-authtoken <token>`

Server token management (Docker):

- Create: `docker compose exec server /eosrift-server token create --label laptop`
- List: `docker compose exec server /eosrift-server token list`
- Revoke: `docker compose exec server /eosrift-server token revoke <id>`

### Server admin frontend (alpha)

A token-gated admin frontend is available on the base domain:

- Enable it by setting `EOSRIFT_ADMIN_TOKEN=<your-strong-random-token>` on the server.
- Open `https://<your-base-domain>/admin`.
- Log in with the token in the UI; it is stored in local browser storage for the current browser profile.
- Use the built-in `Log Out` action to clear the local session token.

Admin API endpoints (same token required via `Authorization: Bearer <token>`):

- `GET|POST /api/admin/tokens`, `DELETE /api/admin/tokens/<id>`
- `GET|POST /api/admin/subdomains`, `DELETE /api/admin/subdomains/<subdomain>`
- `GET|POST /api/admin/tcp-ports`, `DELETE /api/admin/tcp-ports/<port>`

### Reserved subdomains (alpha)

To get a stable URL (instead of a random one each time), reserve a subdomain on the server DB and then request it from the client.

Server (Docker):

- Create a token (note the `id`): `docker compose exec server /eosrift-server token create --label laptop`
- Reserve a subdomain under `EOSRIFT_TUNNEL_DOMAIN`: `docker compose exec server /eosrift-server reserve add --token-id <id> demo`

Client:

- `./bin/eosrift http 8080 --subdomain demo`
- Or (ngrok-like): `./bin/eosrift http --domain demo.tunnel.<yourdomain> 127.0.0.1:8080`

Notes:

- The server will reject `--subdomain` unless it is reserved for your authtoken.
- If `--domain` is unused, the server auto-reserves it to your authtoken on first use.
- Use `docker compose exec server /eosrift-server reserve list` to view reservations.

### Inspector (alpha)

When running `eosrift http ...`, the client starts a local inspector by default:

- `http://127.0.0.1:4040` (web UI)
- `http://127.0.0.1:4040/api/requests` (JSON list)
- Replay: `POST http://127.0.0.1:4040/api/requests/<id>/replay`

Flags:

- Disable: `--inspect=false`
- Change address: `--inspect-addr 127.0.0.1:4041`

Notes:

- The inspector redacts sensitive headers (e.g. `Authorization`, `Cookie`) and common secret query params.
- Replay is best-effort: it forwards to your local upstream and does not include request bodies.
- If the inspector port is in use, the client will try the next port up to `:5000`.
- You must point `*.tunnel.<yourdomain>` at your server (example: `*.tunnel.eosrift.com`).
- In early versions, the host header is preserved (your local service will see `abcd1234.tunnel.<yourdomain>`).

## Trademarks

ngrok is a trademark of its respective owners. This project is not affiliated with ngrok.

## Development

### Tests

- Unit tests: `./scripts/go test ./...`
- Integration tests (Docker): `docker compose -f docker-compose.test.yml up --build --exit-code-from test --abort-on-container-exit`
- Caddy-in-the-loop smoke (Docker): `docker compose -f docker-compose.caddytest.yml up --build --exit-code-from test --abort-on-container-exit`

### Docs site (`/docs`)

- Build embedded docs (Dockerized Node): `./scripts/docs-build`
- Local docs dev server (host Node): `cd docs-site && npm install && npm run docs:dev`
- Local docs dev server (Docker via mise): `mise docs:dev`

### Load testing (Docker)

Run a small load test against a throwaway server in a Compose network:

- HTTP: `docker compose -f docker-compose.loadtest.yml up --build --exit-code-from loadtest --abort-on-container-exit`
- TCP: `EOSRIFT_LOAD_MODE=tcp docker compose -f docker-compose.loadtest.yml up --build --exit-code-from loadtest --abort-on-container-exit`

Tune via env:

- `EOSRIFT_LOAD_REQUESTS` (default `2000`)
- `EOSRIFT_LOAD_CONCURRENCY` (default `50`)
- `EOSRIFT_LOAD_TIMEOUT` (default `5s`)
- `EOSRIFT_LOAD_TCP_PAYLOAD_BYTES` (default `1024`, TCP mode only)
