# Getting Started

## 1) Configure environment

Copy `.env.example` to `.env` and set:

- `EOSRIFT_BASE_DOMAIN` (for example `eosrift.com`)
- `EOSRIFT_TUNNEL_DOMAIN` (for example `tunnel.eosrift.com`)
- Optional: `EOSRIFT_AUTH_TOKEN` (bootstrap token)
- Optional: `EOSRIFT_ADMIN_TOKEN` (enables `/admin`)

## 2) Start server stack

```bash
docker compose up -d --build
curl -fsS http://127.0.0.1:8080/healthz
```

## 3) Create or bootstrap a token

If you did not set `EOSRIFT_AUTH_TOKEN`, create one:

```bash
docker compose exec server /eosrift-server token create --label laptop
```

## 4) Configure client

```bash
./bin/eosrift config add-authtoken <token>
./bin/eosrift config set-server https://<your-base-domain>
```

## 5) Start your first tunnel

```bash
./bin/eosrift http 8080
```

The CLI prints a public forwarding URL and local inspector URL.
