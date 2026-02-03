# Project rules for agents (read first)

This repo is building **EosRift**, a self-hosted, ngrok-like tunnel service. Follow these rules for
all changes unless a human explicitly overrides them.

## Core constraints

- **Server support:** Linux only.
- **Client support:** macOS + Linux.
- **HTTPS:** terminate TLS via **Caddy** (server runs plain HTTP behind it for web traffic).
- **Persistence:** **SQLite only** (no Postgres/Redis/etc.).
- **UX:** keep the CLI and config as close to ngrok as practical.
- **Testing:** develop via **TDD** (tests drive design; behavior changes require tests).

## Workflow

- Make **small, topical commits** (one logical change, minimal churn).
- Prefer **Docker/Compose-based integration tests** when host networking is flaky.
- Keep docs current:
  - Update `CHANGELOG.md` for user-visible changes.
  - Update `ARCHITECTURE.md` when the design changes.
  - Keep `PLAN.md` aligned with reality (don’t leave stale milestones).

## Default tech choices (unless changed deliberately)

- Go for `server` + `client`
- Caddy for TLS and HTTP routing
- Optional later: React SPA for admin/inspector UI (server remains API-first)

## Testing expectations

- Unit tests: fast, hermetic, run via `go test ./...`
- Integration tests: run in Docker networks (compose) and cover:
  - client ↔ server session establishment
  - an end-to-end HTTP tunnel
  - an end-to-end TCP tunnel
  - inspector API smoke tests

## Code organization (target layout)

- `cmd/server` — server entrypoint
- `cmd/client` — client entrypoint
- `internal/...` — internal packages (protocol, allocators, tunnel router, sqlite store)
- `web/...` — optional UI assets (only if/when needed)
- `deploy/...` — Caddy + compose examples

## Compatibility rules

- Treat CLI help text and user-facing output as API:
  - add golden tests for output changes
  - avoid breaking flags/config keys without a migration path
- Keep defaults safe for self-hosting (no open admin endpoints by default).

## What to do when unsure

- Prefer the simplest implementation that can be tested end-to-end in Docker.
- Ask for clarification before adding major dependencies or widening scope.
