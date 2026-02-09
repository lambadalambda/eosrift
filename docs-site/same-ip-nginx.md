# Run EosRift on the same IP as nginx

Use this when nginx already owns public `:80` / `:443` and you want EosRift on the same host.

## Topology

- nginx remains the public edge.
- Caddy listens on loopback high ports (`127.0.0.1:9080` and `127.0.0.1:9443`).
- nginx routes `eosrift.com` and `*.tunnel.eosrift.com` to Caddy.
- nginx routes all other hostnames to your existing sites.

## 1) Move Caddy to loopback ports

In `docker-compose.yml`, set:

```yaml
services:
  caddy:
    ports:
      - "127.0.0.1:9080:80"
      - "127.0.0.1:9443:443"
```

Then restart:

```bash
docker compose up -d --build
```

## 2) Route TLS by SNI (nginx stream)

`/etc/nginx/stream.d/eosrift.conf`:

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

Ensure nginx has:

```nginx
stream {
    include /etc/nginx/stream.d/*.conf;
}
```

## 3) Route HTTP hostnames on :80

`/etc/nginx/conf.d/eosrift-http.conf`:

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

## 4) Move your existing HTTPS vhosts

Any non-EosRift nginx site should listen on loopback TLS, for example:

```nginx
listen 127.0.0.1:8443 ssl;
```

## 5) Validate

```bash
nginx -t
systemctl reload nginx
curl -I https://eosrift.com
```

For a full operational checklist, see `deploy/NGINX_SAME_IP.md` in the repo.
