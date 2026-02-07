# Inspector

The local inspector captures HTTP tunnel requests/responses and provides a local UI/API.

## Default behavior

- Enabled by default for `eosrift http` and `eosrift start` HTTP tunnels.
- Default listen address: `127.0.0.1:4040`.
- If the port is in use, Eosrift increments to the next port until `:5000`.

## UI and API

- UI: `http://127.0.0.1:4040`
- API list: `http://127.0.0.1:4040/api/requests`
- Replay: `POST /api/requests/<id>/replay`

## Flags

Single HTTP tunnel:

```bash
eosrift http 3000 --inspect=false
eosrift http 3000 --inspect-addr 127.0.0.1:4042
```

Named tunnels (`start`):

```bash
eosrift start --all --inspect=false
eosrift start --all --inspect-addr 127.0.0.1:4042
```

## Notes

- Inspector only applies to HTTP tunnels.
- For `start`, one inspector service is shared by all HTTP tunnels in that process.
- Replay forwards to the configured local upstream and returns response status.
