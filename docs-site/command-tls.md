# `eosrift tls`

Create a TLS-labeled tunnel to a local TCP service.

`tls` is currently a thin wrapper around TCP tunneling; Eosrift does not terminate TLS in this path.

## Usage

```text
eosrift tls [flags] <local-port|local-addr>
```

If only a port is provided, Eosrift uses `127.0.0.1:<port>`.

## Flags

- `--server <addr>`
- `--authtoken <token>`
- `--remote-port <port>`: request specific remote TCP port.
- `--help`, `-h`

## Examples

```bash
eosrift tls 443
eosrift tls 443 --server https://eosrift.com
eosrift tls 443 --remote-port 20005
```

The session output uses `tls://<server-host>:<remote-port>`.
