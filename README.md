# EosRift

Project domain: `eosrift.com`

Self-hosted, Docker-first, open-source tunnel service aiming for an ngrok-like UX.

**Status:** pre-alpha (docs-first; implementation starts next).

## Goals

- Functional, self-hostable “ngrok clone”
- Easy deployment via **Docker / Docker Compose** (bare server or platforms like **Coolify**)
- **HTTPS termination via Caddy**
- **Linux-only server**, **Linux + macOS client**
- CLI UX as close to ngrok as practical (commands, flags, config, output)
- Persistence via **SQLite** (no external DB dependencies)
- Development via **TDD** (unit tests + Docker-based integration tests)

## Non-goals (initially)

- Multi-region “edge” network
- Proprietary ngrok features (SAML, paid dashboards, etc.)
- Supporting Windows clients

## Name ideas (alternatives)

1. Riftgate
2. OpenTunnel
3. WireBore
4. PortRelay
5. Holloway
6. Tunnelrod
7. EdgeHook
8. Subduct
9. Driftpipe
10. Cavern

## Proposed stack

- **Go** (server + client): static binaries, great networking, fast iteration
- **Caddy**: TLS/HTTPS for `*.tunnel.<yourdomain>` (recommended: wildcard cert via DNS challenge)
- **SQLite**: persistence (pure-Go driver planned to avoid CGO)
- Optional later: **React SPA** for admin UI (server stays API-first)

## What “ngrok-like” means here

- `http` and `tcp` tunnels with similar command structure and flags
- Local inspector UI (ngrok’s `localhost:4040` equivalent)
- YAML config file support (ngrok-style, at least a compatible subset)
- Authtokens and reserved names (later milestone)

## Documentation

- `PLAN.md` — milestones and TDD approach
- `ARCHITECTURE.md` — proposed architecture and protocols
- `CHANGELOG.md` — notable changes (Keep a Changelog format)

## Quickstart

Tunnels are not implemented yet, but you can run the current server skeleton:

- `cp .env.example .env` (edit for your domain)
- `docker compose up -d --build`
- `curl -fsS http://127.0.0.1:8080/healthz`

When tunnels land, the intended UX is:

- point `*.tunnel.<yourdomain>` (and optionally `<yourdomain>`) at the server (example: `*.tunnel.eosrift.com`)
- run a client locally: `eosrift http 8080` → get a public URL like `https://abcd1234.tunnel.<yourdomain>` (example: `https://abcd1234.tunnel.eosrift.com`)

## Trademarks

ngrok is a trademark of its respective owners. This project is not affiliated with ngrok.
