# EosRift

Project domain: `eosrift.com`

Self-hosted, Docker-first, open-source tunnel service aiming for an ngrok-like UX.

**Status:** pre-alpha. TCP + HTTP tunnels work, but expect rough edges and breaking changes.

This project is close to being “useful by default”, but it is not a 1.0-quality, battle-tested
service yet:

- Backwards compatibility is **not** guaranteed until `v1.0.0`.
- Treat internet exposure as risky until you’ve reviewed `SECURITY.md` and locked things down.
- Default behavior and CLI/config may change as we approach the first stable release.

## Goals

- Functional, self-hostable “ngrok clone”
- Easy deployment via **Docker / Docker Compose**
- **HTTPS termination via Caddy**
- **Linux-only server**, **Linux + macOS client**
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
- `deploy/PRODUCTION.md` — deployment notes (DNS, Caddy, firewall)
- `SECURITY.md` — threat model and security checklist

## Quickstart

### Server (Docker, alpha)

- `cp .env.example .env` (edit for your domain)
- (Optional) Set `EOSRIFT_AUTH_TOKEN` in `.env` to bootstrap the first authtoken
- (Optional) Set `EOSRIFT_MAX_TUNNELS_PER_TOKEN` to cap active tunnels per authtoken (0 = unlimited)
- (Optional) Set `EOSRIFT_MAX_TUNNEL_CREATES_PER_MIN` to rate limit tunnel creations per authtoken (0 = unlimited)
- (Optional) Set `EOSRIFT_LOG_FORMAT=json` for structured logs
- `docker compose up -d --build`
- `curl -fsS http://127.0.0.1:8080/healthz`

Notes:

- By default, `docker-compose.yml` builds the server image locally. If you prefer a prebuilt image, use `ghcr.io/<your-gh-org>/eosrift-server:<tag>`.
- TCP tunnels require opening `EOSRIFT_TCP_PORT_RANGE_START..EOSRIFT_TCP_PORT_RANGE_END` in your firewall/security group.
- `/control` requires an authtoken (stored in SQLite). If you didn’t bootstrap one via `EOSRIFT_AUTH_TOKEN`, create one with `docker compose exec server /eosrift-server token create`.
- Once deployed with DNS + Caddy, `https://<EOSRIFT_BASE_DOMAIN>/` serves a small landing page (the tunnel subdomains still route to tunnels).

### Client (build, recommended for now)

This repo doesn’t require Go on your host; you can build with Docker:

- Linux (example): `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ./scripts/go build -o bin/eosrift ./cmd/client`
- macOS (example): `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 ./scripts/go build -o bin/eosrift ./cmd/client`

### Client (install from GitHub Releases, planned)

This repo includes GitHub Actions workflows to build release artifacts on tags (`v*`).

We are not publishing regular releases yet; build locally for now. When releases start, you’ll
be able to install the client on macOS/Linux using:

- Latest release: `./scripts/install.sh`
- Specific version: `./scripts/install.sh --version v0.1.0`

By default, this installs to `~/.local/bin/eosrift` (override with `--dir`).

Release assets include `checksums.txt` so you can verify downloads.

### Client config (alpha)

Save your authtoken (ngrok-like):

- `./bin/eosrift config add-authtoken <token>`

Set your default server (recommended for self-hosting):

- `./bin/eosrift config set-server https://<yourdomain>`
- or env: `EOSRIFT_SERVER_ADDR=https://<yourdomain>`

If unset, the client defaults to `https://eosrift.com`.

Default config path:

- Linux: `~/.config/eosrift/eosrift.yml` (or `$XDG_CONFIG_HOME/eosrift/eosrift.yml`)
- macOS: `~/Library/Application Support/eosrift/eosrift.yml`

Supported keys (compatible subset): `authtoken`, `server_addr`, `inspect`, `inspect_addr`.

Config precedence:

- Flags > environment > config file > defaults
- Server address (`--server`):
  - `--server` > `EOSRIFT_SERVER_ADDR` > `EOSRIFT_CONTROL_URL` > `server_addr` > `https://eosrift.com`
- Authtoken (`--authtoken`):
  - `--authtoken` > `EOSRIFT_AUTHTOKEN` > `authtoken` > empty
- Inspector:
  - `--inspect` > `inspect` > `true`
  - `--inspect-addr` > `EOSRIFT_INSPECT_ADDR` > `inspect_addr` > `127.0.0.1:4040` (tries up to `:5000`)

### TCP tunnel (alpha)

Expose a local TCP port through the server:

- `./bin/eosrift tcp 8080 --server https://<yourdomain>`

The client prints the allocated remote port, e.g. `Forwarding tcp://<yourdomain>:20001 -> 127.0.0.1:8080`.

### HTTP tunnel (alpha)

Expose a local HTTP port through the server:

- `./bin/eosrift http 8080 --server https://<yourdomain>`
- Request a stable domain (ngrok-like): `./bin/eosrift http --domain demo.tunnel.<yourdomain> 127.0.0.1:8080`

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

### Load testing (Docker)

Run a small load test against a throwaway server in a Compose network:

- HTTP: `docker compose -f docker-compose.loadtest.yml up --build --exit-code-from loadtest --abort-on-container-exit`
- TCP: `EOSRIFT_LOAD_MODE=tcp docker compose -f docker-compose.loadtest.yml up --build --exit-code-from loadtest --abort-on-container-exit`

Tune via env:

- `EOSRIFT_LOAD_REQUESTS` (default `2000`)
- `EOSRIFT_LOAD_CONCURRENCY` (default `50`)
- `EOSRIFT_LOAD_TIMEOUT` (default `5s`)
- `EOSRIFT_LOAD_TCP_PAYLOAD_BYTES` (default `1024`, TCP mode only)
