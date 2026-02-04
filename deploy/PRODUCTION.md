# Production deployment

EosRift is designed to be run as a single-node deployment: **Caddy** (TLS) in front of the
**eosrift-server** (plain HTTP + websocket control plane) with a **SQLite** volume.

This doc assumes you are using `docker-compose.yml`.

## 1) Domains + DNS

EosRift uses two domains:

- **Base domain** (`EOSRIFT_BASE_DOMAIN`): landing page + control plane (example: `eosrift.com`)
- **Tunnel domain** (`EOSRIFT_TUNNEL_DOMAIN`): per-tunnel subdomains (example: `tunnel.eosrift.com`)

You need DNS records pointing at your server:

- `A` / `AAAA` for the base domain → your server IP
- `A` / `AAAA` wildcard for the tunnel domain:
  - example: `*.tunnel.eosrift.com` → your server IP

## 2) Firewall / security groups

Open inbound:

- **80/tcp** (HTTP → redirects to HTTPS)
- **443/tcp** (HTTPS)
- **TCP tunnel port range**, e.g. `20000-21000/tcp` (or your configured `EOSRIFT_TCP_PORT_RANGE_*`)

Keep closed to the internet:

- **8080/tcp** (server’s internal HTTP) — `docker-compose.yml` binds this to `127.0.0.1` only.

## 3) Configure environment

Create `.env`:

- `cp .env.example .env`

Set (at minimum):

- `EOSRIFT_BASE_DOMAIN=<your base domain>`
- `EOSRIFT_TUNNEL_DOMAIN=<your tunnel domain>`

Optional (recommended for first-time setup):

- `EOSRIFT_AUTH_TOKEN=<bootstrap token>` (used once to create the first authtoken)

## 4) Caddy TLS strategy

### Option A (default): on-demand TLS (no DNS plugin)

The default `deploy/Caddyfile` uses **on-demand certificates** with an `ask` endpoint on the
server (`/caddy/ask`) to restrict which hostnames can obtain certs.

Pros:

- No DNS plugin required.
- Simple `docker compose up -d`.

Tradeoffs:

- First request to a hostname may be slower (ACME issuance).
- Large numbers of unique hostnames can hit ACME rate limits.

### Option B: wildcard certificates via DNS challenge (recommended at scale)

If you want a wildcard cert for `*.tunnel.<base-domain>`, you’ll need a Caddy build with your
DNS provider module (DNS challenge). This requires a custom Caddy image (built with `xcaddy`)
and provider-specific environment variables.

This repo does not ship a provider-specific config yet; if you want one, tell me which DNS
provider you use and we can add a ready-to-run example.

## 5) Start services

From the repo root:

- `docker compose up -d --build`

Health check:

- `curl -fsS http://127.0.0.1:8080/healthz`

## Optional: metrics

If you set `EOSRIFT_METRICS_TOKEN`, the server exposes Prometheus-style metrics at `/metrics`.

Example (local, via the `127.0.0.1:8080` bind in `docker-compose.yml`):

- `curl -fsS -H 'Authorization: Bearer <token>' http://127.0.0.1:8080/metrics`

## 6) Create an authtoken and remove bootstrap (recommended)

Create a real token for your client:

- `docker compose exec server /eosrift-server token create --label laptop`

Then remove `EOSRIFT_AUTH_TOKEN` from `.env` and restart:

- `docker compose up -d`

## 7) Use the client

Set your authtoken and server:

- `eosrift config add-authtoken <token>`
- `eosrift config set-server https://<your base domain>`

Start a tunnel:

- `eosrift http 3000`
