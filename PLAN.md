# Plan (milestones)

This project starts **docs-first** and proceeds **TDD-first** (unit tests + Docker-based
integration tests). The goal is a self-hostable tunnel service with an ngrok-like UX.

Examples may reference `eosrift.com`; self-hosters can substitute their own domain.

## Milestone tracker (keep updated)

Last updated: **2026-02-04**

- [x] Milestone 0 — Repository + delivery scaffolding
- [x] Milestone 1 — Control plane + TCP tunnel (end-to-end MVP)
- [x] Milestone 2 — HTTP tunnel with host routing (alpha)
- [x] Milestone 3 — Local inspector (`localhost:4040` equivalent)
- [x] Milestone 4 — CLI + config compatibility
- [x] Milestone 5 — Auth + reserved names (SQLite-backed)
- [x] Milestone 6 — Packaging + deployment polish
- [ ] Milestone 7 — Hardening + observability

Current focus: **Milestone 7**.

## Guiding principles

- **TDD:** write a failing test first, then implement, then refactor.
- **Docker-first testing:** prefer running integration tests inside `docker compose` networks.
- **Caddy owns HTTPS:** the tunnel server runs plain HTTP behind Caddy for web traffic.
- **Small, topical commits:** one logical change per commit; keep the changelog current.
- **Linux server / macOS+Linux client:** server targets Linux only; client targets macOS + Linux.

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

- Reconnect/backoff and session resumption
- Health checks, structured logs, metrics (Prometheus-friendly)
- Load testing harness in Docker
- Security review checklist (auth, input validation, SSRF/host header concerns)

## Ongoing

- Keep `ARCHITECTURE.md` accurate as implementation evolves
- Keep `CHANGELOG.md` updated per release
- Keep changes reviewable: small PRs/commits, tests required for behavior changes
