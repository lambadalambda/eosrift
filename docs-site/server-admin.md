# Server Admin

The admin frontend is optional and token-gated.

## Enable admin UI

Set an admin token on the server:

```bash
EOSRIFT_ADMIN_TOKEN=<strong-random-token>
```

Then open:

```text
https://<your-base-domain>/admin
```

## Login flow

- The admin page starts on a login screen.
- Enter the admin token to unlock the dashboard.
- Session is stored in browser local storage.
- Use `Log Out` to clear the local session.

## API

API endpoints are under `/api/admin/*` and require:

```text
Authorization: Bearer <admin-token>
```

Current resources:

- tokens
- reserved subdomains
- reserved TCP ports
