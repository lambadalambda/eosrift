# Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial project documentation and planning.
- Docker Compose deployment (Caddy + server).
- Coolify deployment notes (`deploy/COOLIFY.md`).
- Caddy on-demand TLS `ask` endpoint (`/caddy/ask`).
- TCP tunneling (alpha): websocket control plane (`/control`) + `eosrift tcp`.
- HTTP tunneling (alpha): host routing under `EOSRIFT_TUNNEL_DOMAIN` + `eosrift http`.
- Control plane auth (alpha): SQLite-backed authtokens (create/list/revoke via `eosrift-server token ...`) and `--authtoken` / `EOSRIFT_AUTHTOKEN` on the client.
- Local inspector (alpha): capture HTTP exchanges, redact common secrets, serve `/api/requests`, and support best-effort replay.
- Docker Compose-based integration test harness.
- GitHub Actions CI (unit tests + integration tests).

### Changed

- Set project name to EosRift (`eosrift.com`).
