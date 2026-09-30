# turkushan-auth

Self-hosted login portal (forward auth) for your subdomains behind Nginx Proxy Manager.
People register their own account, you approve per site who may enter, and one login works on all subdomains.
One container: a Go backend and a Svelte frontend in a single binary, with SQLite in `/data`.

> **Beta / work in progress.** The skeleton runs: database, admin account bootstrap and `/healthz`.
> Login, forward auth and the admin panel are being built.

## Install on Unraid

**Via Community Apps:** search for `turkushan-auth` in the **Apps** tab (once it's listed).

**Manually:** open the Unraid terminal and run:
```bash
wget -O /boot/config/plugins/dockerMan/templates-user/my-turkushan-auth.xml \
  https://raw.githubusercontent.com/turkushan490/turkushan-auth/main/templates/turkushan-auth.xml
```
Then go to **Docker → Add Container → Template** and pick `turkushan-auth`.

Fill in `APP_URL` (e.g. `https://auth.example.com`), `ADMIN_USER` and `ADMIN_PASSWORD` (at least 6 characters, 1 capital letter, 1 symbol), then click Apply.
Check `http://<server-ip>:3010/healthz`: it should return `ok`.

The container starts as root only to fix ownership of the appdata folder, then runs as `PUID:PGID` (default `99:100`).

## Settings

| Variable | Default | |
|---|---|---|
| `APP_URL` | **required** | public portal URL, e.g. `https://auth.example.com` |
| `ADMIN_USER` / `ADMIN_PASSWORD` | | admin created on first start; the password is not overwritten on restart |
| `COOKIE_DOMAIN` | derived from `APP_URL` | `auth.example.com` gives `.example.com`, shared by all subdomains |
| `PORTAL_INTERNAL_URL` | | how NPM reaches the portal, e.g. `http://192.168.1.10:3010` (used in the generated nginx snippets) |
| `TRUSTED_PROXIES` | `172.16.0.0/12` | proxy addresses allowed to pass client IP headers |
| `SESSION_SECRET` | auto-generated in `/data/secret` | |
| `PUID` / `PGID` | `99` / `100` | |
| `PORT` | `3010` | port inside the container |

## Build

GitHub Actions (`.github/workflows/build.yml`) builds the frontend, runs `go vet` and `go test`, then pushes `ghcr.io/turkushan490/turkushan-auth:latest` on every push to `main`. A `v1.2.3` tag also pushes `:1.2.3`.

Local development:
```bash
cd web && npm install && npm run build && cd ..
APP_URL=http://localhost:3010 DATA_DIR=./data ADMIN_USER=admin ADMIN_PASSWORD='Change!me' go run ./cmd/turkushan-auth
```

## License

MIT
