# turkushan-auth

Login portal (forward auth) for `*.turkushan.com` behind Nginx Proxy Manager.
One container: a Go backend and a Svelte frontend in a single binary, with SQLite in `/data`.

> Status: **milestone 1**. The skeleton runs: database, admin account bootstrap and `/healthz`.
> Login, forward auth and the admin panel come in the next milestones.

## Install on Unraid

1. Download the template onto the flash drive (Unraid terminal):
   ```bash
   wget -O /boot/config/plugins/dockerMan/templates-user/my-turkushan-auth.xml \
     https://raw.githubusercontent.com/turkushan490/turkushan-auth/main/unraid/turkushan-auth.xml
   ```
2. **Docker → Add Container → Template:** pick `turkushan-auth`.
3. Fill in `ADMIN_USER` and `ADMIN_PASSWORD` (min 10 characters), then click Apply.
4. Check it's up at `http://192.168.0.6:3010/healthz`. It should return `ok`.

The container starts as root only to fix ownership of the appdata folder, then runs as `PUID:PGID` (default `99:100`).

## Settings

| Variable | Default | |
|---|---|---|
| `APP_URL` | `https://auth.turkushan.com` | public portal URL |
| `COOKIE_DOMAIN` | `.turkushan.com` | shared by all subdomains |
| `ADMIN_USER` / `ADMIN_PASSWORD` | | admin created on first start; the password is not overwritten on restart |
| `TRUSTED_PROXIES` | `172.16.0.0/12,192.168.0.6/32` | NPM addresses allowed to pass client IP headers |
| `PORTAL_INTERNAL_URL` | `http://192.168.0.6:3010` | how NPM reaches the portal (used in the generated snippets) |
| `SESSION_SECRET` | auto-generated in `/data/secret` | |
| `PUID` / `PGID` | `99` / `100` | |
| `PORT` | `3010` | port inside the container |

## Build

GitHub Actions (`.github/workflows/build.yml`) builds the frontend, runs `go vet` and `go test`, then pushes `ghcr.io/turkushan490/turkushan-auth:latest` on every push to `main`. A `v1.2.3` tag also pushes `:1.2.3`.

Local development:
```bash
cd web && npm install && npm run build && cd ..
go run ./cmd/turkushan-auth   # with DATA_DIR=./data ADMIN_USER=... ADMIN_PASSWORD=...
```
