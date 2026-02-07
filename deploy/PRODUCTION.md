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
- `EOSRIFT_TRUST_PROXY_HEADERS=1` (recommended behind Caddy; keep the server bound to localhost-only)

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

## Optional: structured logs

The server supports structured JSON logs:

- `EOSRIFT_LOG_FORMAT=json`
- `EOSRIFT_LOG_LEVEL=info` (or `debug`, `warn`, `error`)

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

## 8) Optional: auto-deploy from GitHub webhook (no deploy secrets in Actions)

This model uses a signed GitHub webhook callback instead of SSH from GitHub Actions.

### What happens

- `Docker Image` workflow succeeds on `main`
- GitHub sends a signed `workflow_run` webhook to `https://<base-domain>/hooks/deploy`
- `deployhook` verifies signature + workflow/branch/repo
- `deploy/webhook/eosrift-deploy.sh` runs:
  - `docker compose pull server`
  - `docker compose up -d --no-deps --force-recreate server`
  - health check (`/healthz`)
  - writes deploy metadata/status to `EOSRIFT_DEPLOY_STATUS_PATH` (default: `/data/deploy-status.json`)

### Server setup

1. In `.env`, set:
   - `EOSRIFT_DEPLOY_WEBHOOK_SECRET=<strong-random-secret>`
   - `EOSRIFT_DEPLOY_WEBHOOK_REPOSITORY=<owner>/<repo>`
2. Start stack with auto-deploy override:
   - `docker compose -f docker-compose.yml -f deploy/docker-compose.autodeploy.yml up -d --build`
3. Confirm the receiver is running:
   - `docker compose -f docker-compose.yml -f deploy/docker-compose.autodeploy.yml logs -f deployhook`

### GitHub webhook setup

In the repository:

- Settings → Webhooks → Add webhook
- Payload URL: `https://<base-domain>/hooks/deploy`
- Content type: `application/json`
- Secret: exactly `EOSRIFT_DEPLOY_WEBHOOK_SECRET`
- Trigger: **Workflow runs**

### Security notes

- The `deployhook` service mounts `/var/run/docker.sock`; treat it as privileged.
- Keep the webhook secret strong and rotate it if leaked.
- Keep `EOSRIFT_DEPLOY_WEBHOOK_REPOSITORY` set to avoid cross-repo trigger abuse.

### Admin visibility

If `EOSRIFT_ADMIN_TOKEN` is set, `/admin` shows the latest deploy state (`running` / `success` / `error`),
SHA, workflow link, and status message from the status file.
