# Architecture (proposed)

This document describes the intended architecture for a self-hosted, ngrok-like tunnel
service: a public **server** (“edge”) and a local **client** (“agent”) that establishes an
outbound connection to the server and carries proxied traffic back to local services.

Examples use `eosrift.com` as the base domain; self-hosters can substitute their own.

## Terminology

- **Edge / server:** publicly reachable service (tested/packaged on Linux; other Unix-like OSes may work but are untested).
- **Agent / client:** CLI tool running on macOS/Linux that exposes local services.
- **Tunnel:** mapping from a public endpoint (host/port) to a local upstream address.
- **Session:** an authenticated, long-lived client connection to the server.
- **Stream:** a single proxied connection carried over the session (multiplexed).

## High-level components

### 1) Caddy (HTTPS termination)

- Listens on **:80/:443**
- Obtains certificates (recommended: wildcard `*.tunnel.<base-domain>` via DNS challenge)
- Reverse-proxies:
  - `https://<id>.tunnel.<base-domain>` → server’s HTTP edge handler
  - `https://<base-domain>/` → server’s embedded landing page
  - `https://<base-domain>/docs/` → server’s embedded docs site
  - `https://<base-domain>/admin` → server’s embedded admin frontend (optional)
  - `https://<base-domain>/control` (websocket upgrade) → server control handler
  - Optional: `https://<base-domain>/hooks/deploy` → deploy webhook receiver
  - Optional: `https://<base-domain>/api/admin/...` → server management API

### 2) Server (Go, Linux target)

Responsibilities:

- Accept and authenticate agent sessions (control plane)
- Allocate/track tunnels:
  - HTTP tunnels: allocate subdomains
  - TCP tunnels: allocate ports from a configured range
- Route inbound traffic to the correct tunnel
- Serve embedded base-domain UX pages (`/`, `/docs/`, optional `/admin`)
- Persist durable config in SQLite (tokens, reservations, users)

### 3) Client (Go, macOS/Linux)

Responsibilities:

- CLI compatible (as close to ngrok as practical)
- Maintain a single outbound session to the server
- Proxy streams to local upstreams
- Local inspector service on `127.0.0.1:4040` (HTTP tunnels)

### 4) Optional deploy webhook receiver (Go)

Responsibilities:

- Accept signed GitHub webhooks (`workflow_run`) on `/hooks/deploy`
- Verify `X-Hub-Signature-256` with a shared webhook secret
- Filter to successful runs for the expected workflow/branch/repository
- Trigger a local deploy script (`docker compose pull/up`) and report status via logs
- Persist deploy status metadata to a shared JSON file for the admin API/UI

## Planes and protocols

### Control plane

- Transport: **WebSocket over TLS** (`wss://…/control`) for NAT friendliness
- Multiplexing: yamux over the WebSocket connection
- Messages: JSON control messages (initially; may move to a stricter schema later)

Control messages cover:

- authenticate (`authtoken`)
- create/update/close tunnel
- heartbeats / keepalive
- server → client “open stream for inbound connection” notifications (if needed)

Current implementation notes:

- Auth uses SQLite-backed authtokens. The client sends `authtoken` in the initial control request and
  the server validates it against the database. `EOSRIFT_AUTH_TOKEN` is a bootstrap convenience to
  ensure an initial token exists.

### Data plane (proxied traffic)

All proxied connections run as **multiplexed streams** over the session:

- Server opens a new stream for each inbound connection/request.
- Client dials the configured local upstream (`127.0.0.1:<port>` or user-provided host).
- Both sides `io.Copy` in both directions until EOF.

This keeps the data plane simple and makes TCP tunnels natural.

## Traffic flows

### HTTP/HTTPS tunnel

1. User runs: `client http 8080`
2. Client connects to `wss://<base-domain>/control`, authenticates (authtoken), requests an HTTP tunnel.
3. Server allocates `<id>.tunnel.<base-domain>` and returns the public URL to the client.
4. Internet request arrives: `https://<id>.tunnel.<base-domain>/path`
5. Caddy terminates TLS and reverse-proxies to server over plain HTTP.
6. Server resolves `<id>` → active tunnel, then proxies the request to the client by opening a stream.
7. Client connects to the local service on `127.0.0.1:8080` and proxies bytes.
8. Response flows back to the internet client via the same path.

Notes:

- For HTTP tunnels, the server can use `net/http` + reverse proxy with a custom dialer that
  opens a tunnel stream (so websockets/streaming work without buffering entire bodies).
- The server adds standard proxy headers (`X-Forwarded-For`, `X-Forwarded-Proto`, etc.).

### TCP tunnel

1. User runs: `client tcp 22`
2. Client requests a TCP tunnel.
3. Server allocates a port from a configured range (e.g., 20000–40000) and listens on it.
4. Internet client connects to `<server-host>:<allocated-port>`.
5. Server opens a stream to the agent and proxies bytes to/from `127.0.0.1:22`.

Notes:

- TCP tunnels do not go through Caddy by default (Caddy’s standard reverse proxy is L7).
- Exposing a port range is expected for “ngrok-style” TCP tunnels.

## Persistence (SQLite)

SQLite is used for durable configuration:

- users/accounts (optional early on; can start single-tenant)
- authtokens
- reserved subdomains (implemented)
- requested domains under `EOSRIFT_TUNNEL_DOMAIN` (implemented; auto-reserved to the first token that uses them)
- arbitrary custom domains (later milestone)
- reserved TCP ports (optional)
- audit log / minimal telemetry (optional; keep privacy-respecting by default)

Ephemeral state (active sessions and in-flight tunnels) is held in memory and rebuilt as
clients reconnect.

## Inspector / UI

- The client hosts a local inspector on `127.0.0.1:4040`.
- For HTTP tunnels, the client captures request/response metadata with strict size limits
  and redaction rules (to avoid leaking secrets).
- The UI can start as server-rendered HTML or a small JSON API, with an optional React SPA later.

## Security model (baseline)

- External TLS terminates at Caddy (Let’s Encrypt).
- Agents authenticate with authtokens; tokens are stored/validated server-side.
- Server enforces per-token limits (max active tunnels, basic create rate limiting); more in later milestones.
- Host header routing is validated to prevent confusion/poisoning attacks.
- Default stance: admin UI/API is disabled unless `EOSRIFT_ADMIN_TOKEN` is set; all admin API requests require bearer auth.

## Scaling (future)

Initial target is a **single-node** deployment (one server + SQLite). A later milestone can
add:

- multiple server nodes behind a load balancer for HTTP tunnels
- shared state (SQLite → replicated DB) or partitioned tunnel allocation

The early design intentionally keeps state and protocols explicit to make this possible later.
