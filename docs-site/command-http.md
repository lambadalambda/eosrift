# `eosrift http`

Create an HTTP tunnel to a local upstream.

## Usage

```text
eosrift http [flags] <local-port|local-addr|local-url>
```

Flags can appear before or after the local target.

## Local target forms

- `3000` (uses `127.0.0.1:3000`)
- `127.0.0.1:3000`
- `https://127.0.0.1:8443` (HTTPS upstream)

URL targets support only scheme + host(+port). Path/query/fragment are rejected.

## Flags

- `--server <addr>`: server address (`https://host`, `http://host:port`, `ws(s)://host/control`).
- `--authtoken <token>`: auth token.
- `--domain <fqdn>`: request a specific domain under tunnel domain.
- `--subdomain <name>`: request reserved subdomain.
- `--basic-auth <user:pass>`: require basic auth at public edge.
- `--allow-cidr <cidr-or-ip>` (repeatable): allowlist client IPs.
- `--deny-cidr <cidr-or-ip>` (repeatable): denylist client IPs.
- `--allow-method <method>` (repeatable): allow request methods.
- `--allow-path </path>` (repeatable): allow exact paths.
- `--allow-path-prefix </prefix>` (repeatable): allow path prefixes.
- `--request-header-add "Name: value"` (repeatable): add/override request headers.
- `--request-header-remove "Name"` (repeatable): remove request headers.
- `--response-header-add "Name: value"` (repeatable): add/override response headers.
- `--response-header-remove "Name"` (repeatable): remove response headers.
- `--host-header <preserve|rewrite|value>`: host header mode.
- `--upstream-tls-skip-verify`: skip cert verification for HTTPS upstreams.
- `--inspect=<true|false>`: enable/disable local inspector.
- `--inspect-addr <host:port>`: inspector listen address.
- `--help`, `-h`

## Validation rules

- `--domain` and `--subdomain` cannot be set together.
- `--basic-auth` must contain `:`.
- CIDR/IP values are validated.
- Header transforms are validated (header names/values).

## Examples

```bash
eosrift http 3000
eosrift http 3000 --domain demo.tunnel.eosrift.com
eosrift http 3000 --subdomain demo
eosrift http 3000 --basic-auth user:pass
eosrift http 3000 --allow-cidr 203.0.113.0/24
eosrift http 3000 --allow-method GET --allow-path /healthz
eosrift http 3000 --request-header-add "X-API-Key: secret"
eosrift http 3000 --host-header=rewrite
eosrift http https://127.0.0.1:8443 --upstream-tls-skip-verify
```
