# Named Tunnels

Named tunnels are defined under `tunnels:` in `eosrift.yml` and launched with `eosrift start`.

## Minimal example

```yaml
version: 1
server_addr: https://eosrift.com
authtoken: eos_...

tunnels:
  web:
    proto: http
    addr: 3000
  db:
    proto: tcp
    addr: 5432
```

Run:

```bash
eosrift start web
eosrift start web db
eosrift start --all
```

## Full example

```yaml
version: 1
server_addr: https://eosrift.com
authtoken: eos_...
host_header: preserve
inspect: true
inspect_addr: 127.0.0.1:4040

tunnels:
  web:
    proto: http
    addr: https://127.0.0.1:8443
    domain: app.tunnel.eosrift.com
    basic_auth: user:pass
    allow_method: [GET, POST]
    allow_path: [/healthz]
    allow_path_prefix: [/api/]
    allow_cidr: [203.0.113.0/24]
    deny_cidr: [198.51.100.0/24]
    request_header_add:
      X-Edge-Env: prod
    request_header_remove:
      - X-Remove-Me
    response_header_add:
      X-Served-By: eosrift
    response_header_remove:
      - Server
    host_header: rewrite
    inspect: true

  db:
    proto: tcp
    addr: 5432
    remote_port: 20005
```

Run all with HTTPS-upstream verify disabled:

```bash
eosrift start --all --upstream-tls-skip-verify
```

## Supported tunnel keys

Common:

- `proto`: `http` or `tcp`
- `addr`
- `inspect` (HTTP only; per-tunnel enable/disable)

HTTP-only:

- `domain`, `subdomain` (mutually exclusive)
- `basic_auth`
- `allow_method`, `allow_path`, `allow_path_prefix`
- `allow_cidr`, `deny_cidr`
- `request_header_add`, `request_header_remove`
- `response_header_add`, `response_header_remove`
- `host_header`

TCP-only:

- `remote_port`

## Validation behavior

On `eosrift start`, Eosrift validates:

- `proto` is present and supported.
- `addr` format is valid for selected `proto`.
- `domain` and `subdomain` are not set together.
- `basic_auth` is `user:pass` when set.
- HTTP-only keys are not used on TCP tunnels.
- `remote_port` is only used on TCP and is `>= 0`.

Invalid config fails fast with an error containing the tunnel name.
