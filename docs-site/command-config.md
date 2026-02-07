# `eosrift config`

Manage client configuration.

## Usage

```text
eosrift config <command> [args]
```

Subcommands:

- `add-authtoken`
- `set-server`
- `set-host-header`
- `check`

## `add-authtoken`

```text
eosrift config add-authtoken <token>
```

Saves `authtoken` to config file.

## `set-server`

```text
eosrift config set-server <server-addr>
```

Saves `server_addr` to config file.

## `set-host-header`

```text
eosrift config set-host-header <preserve|rewrite|value>
```

Saves top-level `host_header` default for HTTP tunnels.

## `check`

```text
eosrift config check
```

Checks config loadability and basic validity:

- prints `Config OK: <path>` when readable/parseable
- warns if `authtoken` is empty
- warns if `server_addr` is empty
- validates top-level `host_header` value

## Examples

```bash
eosrift config add-authtoken eos_...
eosrift config set-server https://eosrift.com
eosrift config set-host-header rewrite
eosrift config check
```
