# Deploying on Coolify

EosRift is Docker Compose-first, so deploying on Coolify is mostly “run the compose”.

## Requirements

- A server where **ports 80 and 443 are available** for this app.
  - This repo runs **Caddy** in `docker-compose.yml` and expects Caddy to own 80/443.
  - If Coolify’s built-in proxy is using 80/443, either disable it for this application
    (run “no proxy”) or deploy on a dedicated host/VPS.
- DNS records pointing at your server:
  - `EOSRIFT_BASE_DOMAIN` (example: `eosrift.com`) → A/AAAA record
  - `EOSRIFT_TUNNEL_DOMAIN` (example: `tunnel.eosrift.com`) → A/AAAA record (optional)
  - `*.${EOSRIFT_TUNNEL_DOMAIN}` (example: `*.tunnel.eosrift.com`) → wildcard A/AAAA record
- Firewall/security group allows:
  - `80/tcp` and `443/tcp` (Caddy)
  - your TCP tunnel range (default `20000-21000/tcp`)

## Deploy steps

1. In Coolify, create a **New Resource → Docker Compose** application (from this repo).
2. Use this repo’s `docker-compose.yml` as the compose file.
3. Ensure Coolify does **not** create/proxy domains for this app (Caddy handles HTTPS).
4. Set environment variables (App Env Vars):
   - `EOSRIFT_BASE_DOMAIN` (example: `eosrift.com`)
   - `EOSRIFT_TUNNEL_DOMAIN` (example: `tunnel.eosrift.com`)
   - `EOSRIFT_AUTH_TOKEN` (required for production; protects `/control`)
   - `EOSRIFT_ACME_EMAIL` (recommended; used by Caddy for ACME registration)
   - Optional: `EOSRIFT_TCP_PORT_RANGE_START` / `EOSRIFT_TCP_PORT_RANGE_END`
5. Deploy.

## Verify

On the server, check the health endpoint (it’s bound to loopback only):

- `curl -fsS http://127.0.0.1:8080/healthz`

From your laptop (client):

- Build the client (see `README.md`)
- Create a tunnel:
  - `./bin/eosrift http 8080 --server wss://<EOSRIFT_BASE_DOMAIN>/control --authtoken <EOSRIFT_AUTH_TOKEN>`

## Notes

- **On-demand TLS:** `deploy/Caddyfile` uses on-demand TLS with an `ask` endpoint. This is
  convenient for alpha, but can hit CA rate limits if you generate lots of subdomains.
  For production, prefer a wildcard certificate (DNS challenge).
- **Cloudflare/“proxy” DNS:** if using a DNS proxy, make sure ACME challenges and websockets
  work for your setup (often easiest: “DNS only” / no proxy while testing).
