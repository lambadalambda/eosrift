# Plan (milestones)

This project starts **docs-first** and proceeds **TDD-first** (unit tests + Docker-based
integration tests). The goal is a self-hostable tunnel service with an ngrok-like UX.

Examples may reference `eosrift.com`; self-hosters can substitute their own domain.

## Milestone tracker (keep updated)

Last updated: **2026-02-05**

- [x] Milestone 0 — Repository + delivery scaffolding
- [x] Milestone 1 — Control plane + TCP tunnel (end-to-end MVP)
- [x] Milestone 2 — HTTP tunnel with host routing (alpha)
- [x] Milestone 3 — Local inspector (`localhost:4040` equivalent)
- [x] Milestone 4 — CLI + config compatibility
- [x] Milestone 5 — Auth + reserved names (SQLite-backed)
- [x] Milestone 6 — Packaging + deployment polish
- [x] Milestone 7 — Hardening + observability
- [x] Milestone 8 — RC track (HTTP correctness + compat)
- [x] Milestone 9 — Config parity + Caddy smoke + release dry-run
- [x] Milestone 10 — Named tunnels + `start` (ngrok-like)
- [x] Milestone 11 — `start` polish + TCP remote ports
- [x] Milestone 12 — HTTP upstream HTTPS
- [x] Milestone 13 — Per-tunnel access control
- [x] Milestone 14 — Reserved TCP ports
- [x] Milestone 15 — HTTP header transforms

Current focus: **Milestone 16**.

## Guiding principles

- **TDD:** write a failing test first, then implement, then refactor.
- **Docker-first testing:** prefer running integration tests inside `docker compose` networks.
- **Caddy owns HTTPS:** the tunnel server runs plain HTTP behind Caddy for web traffic.
- **Small, topical commits:** one logical change per commit; keep the changelog current.
- **Linux server / macOS+Linux client:** server is tested/packaged for Linux; client targets macOS + Linux.

## Engineering backlog (rolling)

Items captured during periodic review passes. These are not necessarily milestones, but should be
kept in mind as we move toward `v1.0.0`.

- [ ] Improve unit test coverage (esp. glue in `cmd/*`; keep relying on Docker integration for e2e).
  - [x] `cmd/server`: admin commands (`token`, `reserve`, `tcp-reserve`)
  - [x] `internal/cli`: config + flag validation tests (no-network)
  - [x] `internal/client`: helper logic (no-network)
  - [x] `internal/client`: host header rewrite tests (no-network)
  - [x] `internal/server`: http tunnel helper tests (no-network)
  - [x] `internal/server`: control helper tests (no-network)
  - [x] `internal/logging`: parse + text formatting tests (no-network)
- [x] Reduce duplication between CLI and server validation (header transforms: shared header name/value validation).
- [x] Reduce duplication between CLI and server validation (CIDR parsing).
- [x] Deduplicate defaults/precedence logic across `http`, `tcp`, and `start` commands.
- [x] HTTP edge proxy perf: avoid per-request `Transport`/`ReverseProxy` construction; reuse shared proxy/transport.
- [x] Control-plane hardening: limit initial JSON request bytes; cap list lengths (CIDRs/headers); validate header values.
- [ ] Decide policy for allowing transforms on `Forwarded` / `X-Forwarded-*` (currently allowed).

## Milestone 0 — Repository + delivery scaffolding

**Goal:** reproducible builds/tests and a runnable skeleton.

**Status:** done (2026-02-03)

- Go module layout (`cmd/server`, `cmd/client`, internal packages)
- Dockerfiles for server/client (client optional in Docker)
- `docker-compose.yml` for “naked server” deployment (Caddy + server + SQLite volume)
- Embedded landing page on the base domain (`GET /` + `GET /style.css`)
- `docker-compose.test.yml` for deterministic integration tests
- Basic CI (build + unit tests + integration tests in containers)
- `CHANGELOG.md` kept up to date

**Acceptance tests**

- `docker compose up -d` starts Caddy + server with a health endpoint
- `docker compose -f docker-compose.test.yml up --build --exit-code-from test` is green

## Milestone 1 — Control plane + TCP tunnel (end-to-end MVP)

**Goal:** a working tunnel for raw TCP with automated tests.

**Status:** done (2026-02-03, alpha)

- Client establishes a **single outbound control connection** to the server (WSS in prod)
- Multiplex streams over that connection (e.g., yamux)
- Server allocates a TCP port from a configured range and listens publicly
- Each inbound TCP connection maps to a new multiplexed stream to the client
- Client connects to a local TCP upstream (`127.0.0.1:<port>`) and proxies bytes both ways

**Acceptance tests (Docker)**

- Start a local echo service next to the client container
- Create a TCP tunnel
- Connect from a third container to the server’s allocated port and assert echo works

## Milestone 2 — HTTP tunnel with host routing (ngrok-style URLs)

**Goal:** `http 8080` yields a stable public URL on a wildcard domain.

**Status:** done (2026-02-03, alpha; websocket coverage TODO)

- Subdomain allocator: `https://<id>.<tunnel-domain>` (example: `https://abcd1234.tunnel.eosrift.com`)
- Host-based routing on the server (map `<id>` → active tunnel)
- Reverse-proxy HTTP over multiplexed streams
- Websocket and streaming support where possible (don’t buffer entire bodies)
- Add `X-Forwarded-*` headers consistent with common proxies

**Acceptance tests (Docker)**

- Run a small HTTP upstream next to the client container
- Request `http://server:8080/...` with `Host: <id>.<tunnel-domain>` and assert response
- (TODO) Request via Caddy+TLS and assert response
- Basic websocket smoke test (optional in this milestone if it complicates MVP)

## Milestone 3 — Local inspector (`localhost:4040` equivalent)

**Goal:** developer UX parity: view recent requests and replay.

**Status:** done (2026-02-03, alpha)

- [x] In-memory store + local JSON API (`/api/requests`)
- [x] Capture request/response previews for HTTP tunnels
- [x] Start local inspector server by default (`127.0.0.1:4040`)
- [x] “Replay” / “resend” support for HTTP tunnels (best-effort)
- [x] Inspector web UI (single-page) talking to the local JSON API

**Acceptance tests**

- Unit tests for request capture and redaction/size limits
- Integration test that generates traffic and verifies inspector API returns entries

## Milestone 4 — CLI + config compatibility

**Goal:** “feels like ngrok” for common flows.

- [x] Commands/flags modeled after ngrok (subset first): `http`, `tcp`, `config`, `version`, `help`
- [x] ngrok-style YAML config parsing (compatible subset): `authtoken`, `server_addr`, `inspect`, `inspect_addr`
- [x] Golden tests for help text and key command outputs (initial)
- [x] CLI session output formatting (initial)
- [x] CLI output formatting and errors closer to ngrok (initial)
- [x] Document config precedence + migration notes

## Milestone 5 — Auth + reserved names (SQLite-backed)

**Goal:** multi-user support with durable configuration.

**Status:** done (2026-02-04, alpha)

- [x] Authtokens stored/validated server-side (SQLite-backed)
- [x] Token management CLI (`eosrift-server token create|list|revoke`)
- [x] Bootstrap token (`EOSRIFT_AUTH_TOKEN`) to create an initial authtoken
- [x] Reserved subdomains and/or custom domains (admin-managed)
- [x] Resource limits: max active tunnels per token (`EOSRIFT_MAX_TUNNELS_PER_TOKEN`)
- [x] Basic rate limiting (tunnel creates per token): `EOSRIFT_MAX_TUNNEL_CREATES_PER_MIN`

## Milestone 6 — Packaging + deployment polish

**Goal:** easy “install and run” across environments.

- [x] Multi-arch server Docker images (linux/amd64 + linux/arm64)
- [x] Signed client releases (macOS + Linux), single-file binaries
- [x] Install script (GitHub Release artifacts)
- [x] Production docs: firewall ports, Caddy wildcard cert setup

## Milestone 7 — Hardening + observability

**Goal:** stability under real-world networks.

- [x] Reconnect/backoff and session resumption (beyond initial dial retries)
- [x] Health checks (`/healthz`)
- [x] Metrics endpoint (`/metrics`, Prometheus text format; token-gated)
- [x] Structured logs (JSON) and log levels
- [x] Load testing harness in Docker
- [x] Security review checklist (auth, input validation, SSRF/host header concerns)

## Ongoing

- Keep `ARCHITECTURE.md` accurate as implementation evolves
- Keep `CHANGELOG.md` updated per release
- Keep changes reviewable: small PRs/commits, tests required for behavior changes

## Milestone 8 — RC track (HTTP correctness + compat)

**Goal:** close the biggest “real-world” gaps before tagging anything. No stability promises yet,
but this milestone should make the system feel solid in daily use.

- [x] WebSocket support over HTTP tunnels (end-to-end)
- [x] Streaming/chunked responses over HTTP tunnels (end-to-end)
- [x] Control-plane keepalive so idle sessions survive NAT/proxies
- [x] Proxy header hygiene (strip `Forwarded` / `X-Forwarded-*` unless proxy headers are trusted)
- [x] HTTP tunnel compat knobs (host header rewriting)
- [x] Pre-release checklist (what “v1.0-ready” means for EosRift)

**Acceptance tests (Docker)**

- HTTP tunnel WebSocket echo smoke test
- HTTP tunnel streaming response does not buffer entire body before first bytes

## Milestone 9 — Config parity + Caddy smoke + release dry-run

**Goal:** close a few remaining “ops and UX” gaps and make it easier to validate release builds
before tagging.

- [x] Config parity: support `host_header` in `eosrift.yml` and add `eosrift config set-host-header ...`.
- [x] Add a Docker Compose smoke test that runs the existing integration tests through Caddy
  (reverse proxy in front of the server) to catch proxy/websocket/streaming regressions.
- [x] Add a GitHub Actions “release dry-run” path (manual workflow dispatch) that builds the
  same artifacts as tagged releases but uploads them as workflow artifacts (no tag required).

**Acceptance tests**

- Unit tests: `./scripts/go test ./...` is green.
- Integration tests: `docker compose -f docker-compose.test.yml up --build --exit-code-from test` is green.
- Caddy smoke: `docker compose -f docker-compose.caddytest.yml up --build --exit-code-from test` is green.

## Milestone 10 — Named tunnels + `start` (ngrok-like)

**Goal:** run config-defined tunnels without repeating flags/args and get closer to ngrok’s “named
tunnels” workflow.

- [x] Config: support an ngrok-like `tunnels:` map in `eosrift.yml`.
  - Each tunnel defines: `proto` (`http`/`tcp`), `addr`, and optional HTTP settings like
    `domain` / `subdomain` / `host_header`.
- [x] CLI: add `eosrift start <name>` and `eosrift start --all`.
- [x] Output: show a clear per-tunnel session summary (multiple tunnels should be readable).
- [x] Tests: unit tests for config parsing/validation + Docker integration coverage for `start`.

**Acceptance tests**

- `./scripts/go test ./...` is green.
- `docker compose -f docker-compose.test.yml up --build --exit-code-from test --abort-on-container-exit` is green.

## Milestone 11 — `start` polish + TCP remote ports

**Goal:** close a few remaining gaps in the “named tunnels” workflow and tighten config parity.

- [x] TCP `remote_port`:
  - Support `remote_port` in `tunnels:` for TCP tunnels.
  - Add a `--remote-port` flag to `eosrift tcp ...` (ngrok-like convenience; still optional).
  - Client sends requested port to the server; server validates range/availability.
- [x] Stronger tunnel validation:
  - Fail fast with clear errors for invalid `tunnels:` entries (missing/invalid `proto`, invalid `addr`,
    `domain`+`subdomain` conflicts, unsupported keys).
  - Improve errors to identify the tunnel name that failed.
- [x] UX + docs:
  - Add more `eosrift start` examples to CLI help output and `README.md`.
  - Include a full config example showing multiple named tunnels (HTTP + TCP) and typical flags.

**Acceptance tests**

- Unit tests: `./scripts/go test ./...` is green.
- Integration tests (Docker): `docker compose -f docker-compose.test.yml up --build --exit-code-from test --abort-on-container-exit` is green.
- New integration coverage:
  - TCP tunnel can request a specific port within the configured range and it is honored.
  - Error cases: requested port out-of-range/unavailable produce clear messages.

## Milestone 12 — HTTP upstream HTTPS

**Goal:** support forwarding HTTP tunnels to local HTTPS upstreams (ngrok-like), while keeping
websockets, streaming, and the local inspector working.

- [x] Accept upstream URLs for HTTP tunnels:
  - `eosrift http https://127.0.0.1:8443` (scheme + host:port), in addition to the existing `<port|host:port>` forms.
  - `tunnels.*.addr` may also be a URL for `proto: http` tunnels.
- [x] Dial upstream with TLS when scheme is `https`.
- [x] Add a TLS verification toggle for upstream HTTPS (default behavior documented).
- [x] Ensure inspector capture + replay work for HTTPS upstreams.

**Acceptance tests**

- Unit tests: `./scripts/go test ./...` is green.
- Integration tests (Docker): `docker compose -f docker-compose.test.yml up --build --exit-code-from test --abort-on-container-exit` is green.
- New integration coverage:
  - HTTP tunnel forwards to an HTTPS upstream (self-signed) and returns expected body.

## Milestone 13 — Per-tunnel access control

**Goal:** add a small but useful subset of ngrok-style edge access control for HTTP tunnels.

- [x] Basic auth:
  - Client flag + config: `--basic-auth user:pass` (and `tunnels.*.basic_auth`).
  - Server enforces auth on inbound HTTP tunnel requests (before proxying).
- [x] IP allowlist/denylist:
  - Client flag: `--allow-cidr ...` / `--deny-cidr ...` (repeatable; also accepts a bare IP).
  - Config keys: `tunnels.*.allow_cidr` / `tunnels.*.deny_cidr`.
  - Must respect `EOSRIFT_TRUST_PROXY_HEADERS` so `X-Forwarded-For` can’t bypass rules.

**Acceptance tests**

- Unit tests for parsing/validation.
- Integration tests for:
  - basic auth (401 without auth; 200 with auth)
  - allow/deny CIDR rules (403 on deny; allowlist permits matching clients)

## Milestone 14 — Reserved TCP ports

**Goal:** ngrok-like “stable TCP address” by reserving ports to authtokens in SQLite.

- [x] Add SQLite persistence for TCP port reservations (token_id ↔ port).
- [x] Server CLI to manage reservations: list/add/remove (mirrors subdomain reservations).
- [x] Control plane enforces reservations:
  - If a port is reserved for another token: reject.
  - Auto-reserve on first use, similar to `--domain`.

**Acceptance tests**

- Unit tests for reservation store behavior.
- Server control-plane tests proving a port can be claimed only by its owning token.

## Milestone 15 — HTTP header transforms (traffic policy lite)

**Goal:** cover a few common ngrok “traffic policy” style needs without implementing a full policy engine.

**Status:** done (2026-02-04)

- [x] Request header add/remove (per tunnel): `--request-header-add`, `--request-header-remove`, config under `tunnels.*`.
- [x] Response header add/remove (per tunnel): `--response-header-add`, `--response-header-remove`, config under `tunnels.*`.

**Acceptance tests**

- Unit tests for parsing/validation.
- Integration test proving headers are modified as configured.

## Milestone 16 — OAuth/OIDC edge auth (optional)

**Goal:** an ngrok-like “login wall” for HTTP tunnels, suitable for self-hosting.

- [ ] OAuth provider config (GitHub first), per-tunnel enable/disable.
- [ ] Cookie/session handling on the server edge; no state stored in the client.

**Acceptance tests**

- Unit tests for config parsing.
- Integration smoke test for redirect + callback flow (best-effort; may require a mocked provider).

## Milestone 17 — TLS tunnels (optional)

**Goal:** ngrok-like `tls` command convenience for exposing local TLS services.

- [ ] `eosrift tls <local-port|local-addr>` as a thin wrapper around `tcp` (byte proxying).
- [ ] Docs explaining common use cases (mTLS, custom certs, etc.).

## Milestone 18 — HTTP request allow/deny (traffic policy lite)

**Goal:** support a small subset of ngrok-style “only expose these endpoints” behavior for HTTP tunnels.

- [ ] Per-tunnel method allowlist: `--allow-method GET` (repeatable) and config under `tunnels.*`.
- [ ] Per-tunnel path allowlist: `--allow-path /healthz` (repeatable) and config under `tunnels.*`.
- [ ] (Optional) Simple prefix matching: `--allow-path-prefix /api/`.

**Acceptance tests**

- Unit tests for parsing/validation and enforcement.
- Integration test proving disallowed requests are rejected on the edge.
