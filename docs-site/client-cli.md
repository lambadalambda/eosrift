# Client CLI

## Core commands

- `eosrift http`
- `eosrift tcp`
- `eosrift tls`
- `eosrift start`
- `eosrift config`

## Typical examples

```bash
eosrift http 3000
eosrift http --domain app.tunnel.eosrift.com 127.0.0.1:3000
eosrift tcp 5432 --remote-port 20005
eosrift start --all
```

## Config precedence

Flags override environment, environment overrides config file, config file overrides defaults.

For server selection:

1. `--server`
2. `EOSRIFT_SERVER_ADDR`
3. `EOSRIFT_CONTROL_URL` (legacy)
4. `server_addr` in config
5. built-in default

## Inspector

HTTP tunnels start a local inspector by default:

- Web UI: `http://127.0.0.1:4040`
- API: `http://127.0.0.1:4040/api/requests`

If a port is busy, Eosrift auto-increments up to `:5000`.
