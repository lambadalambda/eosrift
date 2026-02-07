# Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Named tunnels in `eosrift.yml` (`tunnels:`) and `eosrift start` / `eosrift start --all`.
- TCP tunnel `remote_port` support: `eosrift tcp --remote-port` and `tunnels.*.remote_port`.
- TLS tunnel convenience command: `eosrift tls` (thin wrapper around `tcp`).
- Token-gated server admin frontend and API: `/admin` and `/api/admin/*` (enable with `EOSRIFT_ADMIN_TOKEN`).
- Embedded documentation site at `/docs/` (VitePress source in `docs-site/`, served from Go on the base domain).
- Reserved TCP ports (SQLite-backed): `eosrift-server tcp-reserve ...` and auto-reserve on first `--remote-port` use.
- HTTP tunnels can now forward to local HTTPS upstreams (pass a `https://...` local URL).
- HTTP tunnel basic auth (per tunnel): `eosrift http --basic-auth user:pass` and `tunnels.*.basic_auth`.
- HTTP tunnel method/path allowlists (per tunnel): `--allow-method` / `--allow-path` / `--allow-path-prefix` and config keys under `tunnels.*`.
- HTTP tunnel CIDR access control (per tunnel): `--allow-cidr` / `--deny-cidr` and `tunnels.*.allow_cidr` / `tunnels.*.deny_cidr`.
- HTTP header transforms (per tunnel): request/response header add/remove (`--request-header-add`, `--request-header-remove`, `--response-header-add`, `--response-header-remove`) and config keys under `tunnels.*`.

### Changed

- `eosrift start` now validates `tunnels:` config and fails fast on common mistakes (invalid `addr`, invalid option combinations).
- Control-plane requests now enforce basic size limits and validate header transform values (defense in depth).
- Admin UI now starts on a login screen and only shows the dashboard after a valid admin token is entered (with logout/session-expiry handling).
- Expanded `/docs/` with a full client CLI reference (commands, flags, config paths/schema, precedence, named tunnels, inspector behavior).

### Fixed

- HTTP tunnel reconnect now preserves per-tunnel access control settings (basic auth and CIDR rules).
- Control responses now decode reliably when the server closes the stream immediately after sending JSON (fixes flaky `requested port out of range` errors in integration/CI).

## [0.1.1] - 2026-02-04

### Added

- CLI help output now includes examples for `--domain` and `--subdomain`.

### Fixed

- HTTP tunnels now unregister on control-plane disconnect so `--domain` can be reused after the prior session ends.
- TCP tunnels now close their listener on control-plane disconnect.

## [0.1.0] - 2026-02-04

### Added

- Initial project documentation and planning.
- Docker Compose deployment (Caddy + server).
- Caddy on-demand TLS `ask` endpoint (`/caddy/ask`).
- Optional `/metrics` endpoint (Prometheus text format), gated by `EOSRIFT_METRICS_TOKEN`.
- Security review checklist and threat model (`SECURITY.md`).
- Structured server logging with levels and optional JSON format (`EOSRIFT_LOG_LEVEL`, `EOSRIFT_LOG_FORMAT`).
- `EOSRIFT_TRUST_PROXY_HEADERS` to control whether the server trusts `Forwarded` / `X-Forwarded-*` headers from an upstream proxy.
- Embedded landing page on the base domain (`GET /` + `GET /style.css`).
- GitHub Actions release workflow (builds client/server binaries on tags).
- GitHub Actions release workflow now signs `checksums.txt` with keyless Sigstore/cosign.
- GitHub Actions release workflow now supports manual “dry-run” builds (no tag) via `workflow_dispatch`.
- TCP tunneling (alpha): websocket control plane (`/control`) + `eosrift tcp`.
- HTTP tunneling (alpha): host routing under `EOSRIFT_TUNNEL_DOMAIN` + `eosrift http`.
- Reserved subdomains (alpha): `eosrift-server reserve ...` + `eosrift http --subdomain ...` (stable tunnel URLs).
- ngrok-like custom domain flag: `eosrift http --domain <name>.<EOSRIFT_TUNNEL_DOMAIN>` auto-reserves on first use.
- Host header rewriting (alpha): `eosrift http --host-header=preserve|rewrite|<value>`.
- Resource limits (alpha): `EOSRIFT_MAX_TUNNELS_PER_TOKEN` to cap concurrent tunnels per authtoken.
- Basic rate limiting (alpha): `EOSRIFT_MAX_TUNNEL_CREATES_PER_MIN` to cap tunnel create attempts per authtoken.
- Control plane auth (alpha): SQLite-backed authtokens (create/list/revoke via `eosrift-server token ...`) and `--authtoken` / `EOSRIFT_AUTHTOKEN` on the client.
- Client config (alpha): ngrok-style YAML config (`eosrift.yml`) + `eosrift config add-authtoken|set-server|set-host-header|check` and global `--config`.
- Local inspector (alpha): capture HTTP exchanges, redact common secrets, serve a web UI at `/`, expose `/api/requests`, and support best-effort replay.
- Docker Compose-based integration test harness.
- Caddy-in-the-loop integration smoke harness (`docker-compose.caddytest.yml`).
- Docker-based load testing harness (`docker-compose.loadtest.yml` + `test/loadtest`).
- GitHub Actions CI (unit tests + integration tests).
- GitHub Actions workflow to publish multi-arch server Docker images to GHCR.
- Production deployment docs (`deploy/PRODUCTION.md`) covering DNS, Caddy, and firewall ports.
- Client install script (`scripts/install.sh`) for GitHub Release artifacts.

### Changed

- Set project name to EosRift (`eosrift.com`).
- `eosrift version` now prints `eosrift version <version>` (release builds inject the version via `-ldflags -X`).
- Default client server is now `https://eosrift.com` (override via `--server`, `EOSRIFT_SERVER_ADDR`, or `server_addr` in config).
- `eosrift http` / `eosrift tcp` now print an ngrok-like session summary (including forwarding URL and inspector URL).
- The client now retries initial control connection dials with exponential backoff until canceled.
- The client now automatically reconnects and attempts to resume tunnels after control-plane disconnects.
- The control-plane session now sends periodic keepalives to reduce idle disconnects.

### Fixed

- `./scripts/go` now forwards `GOOS`/`GOARCH`/`CGO_ENABLED` into the Docker container so macOS client builds work.
- `eosrift http` / `eosrift tcp` now accept flags after args (ngrok-like): `eosrift http 8080 --server https://...`.
- `eosrift http|tcp|config --help` (and `-h`) now prints help to stdout and exits 0.
- `deploy/Caddyfile` now uses the correct `on_demand_tls { ask ... }` placement for Caddy.
- `/caddy/ask` now only allows on-demand TLS issuance for active or reserved tunnel subdomains (prevents arbitrary ACME issuance abuse).
- Prevent `X-Forwarded-For` spoofing on HTTP tunnels by stripping `Forwarded` / `X-Forwarded-*` unless proxy headers are trusted.
- Ctrl-C shutdown no longer prints spurious tunnel errors.
- Suppress noisy yamux shutdown logs (e.g. `Failed to read header: ... context canceled`) on normal disconnects.
- Local inspector now auto-increments ports until it can bind (up to `:5000`).
- `/metrics` is now served only on the base domain and requires `Authorization: Bearer <token>`.
