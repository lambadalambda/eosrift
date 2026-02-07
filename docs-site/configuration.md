# Configuration

Eosrift uses a YAML config file with ngrok-like structure.

## Default config path

Eosrift resolves the default path as follows:

1. If `XDG_CONFIG_HOME` is set: `$XDG_CONFIG_HOME/eosrift/eosrift.yml`
2. Otherwise `os.UserConfigDir()`:
   - Linux: typically `~/.config/eosrift/eosrift.yml`
   - macOS: typically `~/Library/Application Support/eosrift/eosrift.yml`
3. Fallback: `~/.config/eosrift/eosrift.yml`

You can override path with:

- global flag: `--config <path>`
- env: `EOSRIFT_CONFIG=<path>`

## Top-level schema

```yaml
version: 1
authtoken: eos_...
server_addr: https://eosrift.com
host_header: preserve
inspect: true
inspect_addr: 127.0.0.1:4040

tunnels:
  web:
    proto: http
    addr: 3000
  db:
    proto: tcp
    addr: 5432
```

Top-level keys:

- `version` (optional, written as `1` by `eosrift config ...` commands)
- `authtoken`
- `server_addr`
- `host_header` (`preserve`, `rewrite`, or literal host value)
- `inspect` (default inspector behavior)
- `inspect_addr` (starting address for local inspector bind)
- `tunnels` (map of named tunnels)

## Value precedence

General rule: `flag > environment > config > built-in default`.

### Server (`--server`)

1. `--server`
2. `EOSRIFT_SERVER_ADDR`
3. `EOSRIFT_CONTROL_URL` (legacy)
4. `server_addr`
5. built-in default: `https://eosrift.com`

### Authtoken (`--authtoken`)

1. `--authtoken`
2. `EOSRIFT_AUTHTOKEN`
3. `EOSRIFT_AUTH_TOKEN` (compatibility fallback)
4. `authtoken`

### Inspector (`http` and `start`)

- `inspect`:
  - `--inspect` flag
  - then config `inspect`
  - then default `true`
- `inspect_addr`:
  - `--inspect-addr` flag
  - then `EOSRIFT_INSPECT_ADDR`
  - then config `inspect_addr`
  - then default `127.0.0.1:4040`

### Host header (`http`)

1. `--host-header`
2. `host_header`
3. built-in default `preserve`

For `start`, per-tunnel `host_header` overrides top-level `host_header`.

## Config commands

`eosrift config` supports:

- `add-authtoken`
- `set-server`
- `set-host-header`
- `check`

Examples:

```bash
eosrift config add-authtoken eos_...
eosrift config set-server https://eosrift.com
eosrift config set-host-header rewrite
eosrift config check
```

`eosrift config check` verifies file readability and validates top-level `host_header`. It also warns if `authtoken` or `server_addr` is empty.

## Next

- [Named Tunnels](/named-tunnels)
- [Command Reference](/command-http)
