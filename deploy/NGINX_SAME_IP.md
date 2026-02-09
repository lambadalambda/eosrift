# Running EosRift on the same IP as existing nginx

This guide is for hosts where nginx already owns public ports `80` and `443`, but you still want
to run EosRift (Caddy + server) on the same machine and same public IP.

The pattern is:

- nginx stays the internet edge on `:80` and `:443`
- Caddy listens only on loopback high ports
- nginx routes EosRift hostnames to Caddy
- nginx routes all other traffic to your existing sites

## 1) Change Caddy binds to loopback high ports

In `docker-compose.yml`, change the `caddy` service `ports:` from public binds to loopback:

```yaml
services:
  caddy:
    ports:
      - "127.0.0.1:9080:80"
      - "127.0.0.1:9443:443"
```

Keep the server bound to loopback only:

```yaml
services:
  server:
    ports:
      - "127.0.0.1:8080:8080"
```

Restart EosRift:

```bash
docker compose up -d --build
```

## 2) Configure nginx stream SNI routing on 443

You need nginx `stream` with `ssl_preread` enabled (standard in most distro nginx packages).

Create `/etc/nginx/stream.d/eosrift.conf`:

```nginx
map $ssl_preread_server_name $tls_upstream {
    eosrift.com                 127.0.0.1:9443;
    ~^.+\.tunnel\.eosrift\.com$ 127.0.0.1:9443;
    default                     127.0.0.1:8443;
}

server {
    listen 443;
    proxy_pass $tls_upstream;
    ssl_preread on;
}
```

Then ensure your main `/etc/nginx/nginx.conf` includes stream configs:

```nginx
stream {
    include /etc/nginx/stream.d/*.conf;
}
```

`127.0.0.1:8443` is where your non-EosRift HTTPS vhosts should listen after this change.

## 3) Configure nginx HTTP routing on 80

Create `/etc/nginx/conf.d/eosrift-http.conf`:

```nginx
server {
    listen 80;
    server_name eosrift.com *.tunnel.eosrift.com;

    location / {
        proxy_pass http://127.0.0.1:9080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Your existing HTTP server blocks can keep handling other hostnames.

## 4) Move existing nginx HTTPS vhosts off public 443

Any existing nginx HTTPS vhost that currently does `listen 443 ssl;` should be changed to:

```nginx
listen 127.0.0.1:8443 ssl;
```

This prevents conflict with the stream listener on public `:443`.

## 5) Validate and reload nginx

```bash
nginx -t
systemctl reload nginx
```

## 6) Quick checks

```bash
curl -I http://eosrift.com
curl -I https://eosrift.com
curl -I https://<some-active-subdomain>.tunnel.eosrift.com
```

Also verify a non-EosRift hostname still reaches your existing app.

## Notes

- If you have a second public IP, using a dedicated IP for EosRift is simpler.
- Keep wildcard DNS for `*.tunnel.eosrift.com` pointing to this server.
- Keep `EOSRIFT_TRUST_PROXY_HEADERS=1` when running behind nginx/Caddy.
