# `eosrift start`

Start one or more named tunnels from `tunnels:` in config.

## Usage

```text
eosrift start [flags] [<tunnel> ...]
```

Valid invocation patterns:

- `eosrift start <name> [<name> ...]`
- `eosrift start --all`

Invalid:

- `eosrift start` (no names and no `--all`)
- `eosrift start --all web` (`--all` with names)

## Flags

- `--server <addr>`
- `--authtoken <token>`
- `--all`
- `--inspect=<true|false>`: default inspector setting for HTTP tunnels.
- `--inspect-addr <host:port>`: shared inspector listen address.
- `--upstream-tls-skip-verify`: disable cert verification for HTTPS upstreams (HTTP tunnels).
- `--help`, `-h`

## Tunnel selection and validation

On startup, Eosrift:

1. Loads config file.
2. Resolves selected tunnel names (`--all` or explicit names).
3. Validates each selected tunnel.
4. Starts all requested tunnels and prints a combined session view.

Validation errors include the tunnel name and fail fast.

## Examples

```bash
eosrift start web
eosrift start web db
eosrift start --all
eosrift start --all --server https://eosrift.com
eosrift start --all --inspect=false
eosrift start --all --upstream-tls-skip-verify
```

## Inspector behavior with `start`

- One inspector server is started per process when at least one selected HTTP tunnel has inspector enabled.
- Per-tunnel `inspect: false` disables capture for that tunnel.
