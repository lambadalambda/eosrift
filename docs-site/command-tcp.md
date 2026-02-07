# `eosrift tcp`

Create a raw TCP tunnel to a local TCP service.

## Usage

```text
eosrift tcp [flags] <local-port|local-addr>
```

If only a port is provided, Eosrift uses `127.0.0.1:<port>`.

## Flags

- `--server <addr>`
- `--authtoken <token>`
- `--remote-port <port>`: request specific remote TCP port (must be in server range).
- `--help`, `-h`

## Examples

```bash
eosrift tcp 5432
eosrift tcp 5432 --server https://eosrift.com
eosrift tcp 5432 --remote-port 20005
eosrift tcp 127.0.0.1:3306
```

The session output includes:

- public endpoint: `tcp://<server-host>:<remote-port>`
- local target: `localhost:<port>` or specified host/port
