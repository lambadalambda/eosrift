# Security

This document is a living security review checklist and threat model for **EosRift**.

If you are deploying EosRift to the public internet, read `deploy/PRODUCTION.md` first.

## Threat model (baseline)

Assume:

- The server is internet-exposed (ports 80/443 and a TCP tunnel range).
- Untrusted internet clients will hit tunnel URLs.
- Some clients/agents may be compromised or malicious (valid authtokens).
- Operators may accidentally expose internal ports (e.g. binding `:8080` publicly).

Protect:

- Authtokens (client auth) and any admin/ops tokens (metrics).
- Reserved names (ownership of stable subdomains).
- Availability: certificate issuance, control plane, and tunnel proxying.
- User traffic confidentiality/integrity (TLS handled by Caddy).

Non-goals (currently):

- Multi-tenant isolation beyond token-based ownership and coarse limits.
- Advanced abuse prevention (WAF/bot protection) out of the box.

## Attack surface inventory

Server endpoints:

- `GET /healthz` — health
- `GET /caddy/ask` — restricts on-demand TLS issuance (Caddy)
- `GET /metrics` — optional Prometheus text endpoint (token-gated)
- `GET /` + `GET /style.css` — landing page (base domain only)
- `GET/POST ...` on `*.tunnel.<domain>` — HTTP tunnel proxy
- `WS /control` — authenticated control plane (client ↔ server)
- TCP listeners in the configured range — TCP tunnel endpoints

Client endpoints:

- Local inspector HTTP server (default `127.0.0.1:4040`)

Persistence:

- SQLite DB: authtokens, reserved subdomains

## Deployment safety checklist (operator)

- [ ] Caddy terminates TLS; server HTTP port (`:8080`) is not publicly exposed (bind to loopback or firewall).
- [ ] DNS is correct for base domain and wildcard tunnel domain.
- [ ] Choose a TLS strategy:
  - [ ] Wildcard cert via DNS challenge (recommended at scale), or
  - [ ] On-demand TLS with a strict `ask` endpoint (default).
- [ ] Open only required ports: 80/443 and your TCP tunnel range.
- [ ] Back up the SQLite volume if you care about token/reservation durability.
- [ ] Set a metrics token only if you intend to scrape metrics: `EOSRIFT_METRICS_TOKEN`.

## Application security checklist (dev)

### Authentication / authorization

- [ ] `/control` requires an authtoken (server-side validation).
- [ ] Token storage uses hashes-at-rest (no plaintext token in SQLite).
- [ ] Token revocation is supported and enforced.
- [ ] Reserved subdomains enforce ownership (token-bound).

### Input validation

- [ ] Validate subdomain labels (DNS label rules) for reservations and requested domains.
- [ ] Validate `Host` routing for HTTP tunnels (must be exactly `<id>.<tunnel-domain>`).
- [ ] Validate requested TCP ports are in range.

### SSRF / proxying concerns

- [ ] Server HTTP tunnel proxy never dials arbitrary upstreams (streams only).
- [ ] Hop-by-hop headers are handled safely by the reverse proxy.
- [ ] Keep-alives and connection reuse are bounded to avoid resource leaks.

### On-demand TLS abuse

- [ ] `/caddy/ask` does not allow arbitrary hostname issuance (only base domain + active/reserved tunnels).
- [ ] Recommend wildcard cert for large fleets or high churn.

### DoS / abuse controls

- [ ] Rate limit tunnel creation (per authtoken).
- [ ] Cap active tunnels (per authtoken).
- [ ] Consider per-IP limits at the edge (Caddy) for public deployments.

### Secret handling

- [ ] Avoid logging authtokens.
- [ ] Avoid embedding secrets in URLs (query params) in docs/examples.
- [ ] Ensure `/metrics` is token-gated and not enabled by default.

### Client inspector

- [ ] Inspector binds to loopback by default.
- [ ] Redact sensitive headers and common secret query params.
- [ ] Bound capture sizes and entry counts.

## Vulnerability reporting

For now, please open a GitHub issue with **minimal detail** and we’ll follow up privately for
reproduction steps. (We can switch to GitHub Security Advisories once the repo is hosted there.)

