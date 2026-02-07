# Client CLI

The current client command surface is:

- `eosrift http`
- `eosrift tcp`
- `eosrift tls`
- `eosrift start`
- `eosrift config`
- `eosrift version`
- `eosrift help`

## Global usage

```text
eosrift [--config <path>] <command> [args]
```

Global flag:

- `--config <path>`: override config file path (default is OS-specific, see [Configuration](/configuration)).

## Environment variables

- `EOSRIFT_CONFIG`: config path override.
- `EOSRIFT_SERVER_ADDR`: preferred server base address.
- `EOSRIFT_CONTROL_URL`: legacy full `ws(s)` control URL.
- `EOSRIFT_AUTHTOKEN`: client authtoken.
- `EOSRIFT_INSPECT_ADDR`: local inspector listen address.
- `EOSRIFT_AUTH_TOKEN`: compatibility fallback for authtoken when `EOSRIFT_AUTHTOKEN` is unset.

## Server address formats

Commands that accept `--server` support:

- `https://host` or `https://host:port`
- `http://host:port`
- `wss://host/control` or `ws://host/control`
- bare host forms (for example `example.com` or `example.com:8080`)

For `http(s)` server values, the client derives a websocket control URL by appending `/control`.

## Local target formats

- `http`: `<port>`, `<host:port>`, or `http(s)://<host[:port]>` (URL form must not include path/query/fragment).
- `tcp` and `tls`: `<port>` or `<host:port>`.

If you pass only a port, Eosrift targets `127.0.0.1:<port>`.

## Command docs

- [HTTP command reference](/command-http)
- [TCP command reference](/command-tcp)
- [TLS command reference](/command-tls)
- [Start command reference](/command-start)
- [Config command reference](/command-config)

## Quick examples

```bash
eosrift config add-authtoken <token>
eosrift config set-server https://eosrift.com
eosrift http 3000 --domain demo.tunnel.eosrift.com
eosrift tcp 5432 --remote-port 20005
eosrift start --all
```
