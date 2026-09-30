# Security

How turkushan-auth protects your sites, where each measure lives in the code, and which test checks it.
Found a problem? Open a [GitHub issue](https://github.com/turkushan490/turkushan-auth/issues). For anything sensitive, ask for a private contact there first.

## Checklist

| Requirement | How | Code | Test |
|---|---|---|---|
| Passwords hashed with argon2id | argon2id, 64 MiB, t=3, p=2, 16-byte random salt, PHC format; constant-time compare | `internal/auth/password.go` | `TestHashAndVerify`, `TestVerifyMalformed` |
| Hashing can't exhaust memory | max 4 hashes at once; absurd parameters in a stored hash are rejected | `internal/auth/password.go` | `TestVerifyMalformed` |
| Passwords never stored or logged in plain text | only the hash is stored; logs contain username + IP, never passwords | `internal/server/handlers_auth.go` | `TestHashAndVerify` |
| Password rules | at least 6 characters, 1 capital letter, 1 symbol (owner's choice); same rules in the browser and on the server | `internal/auth/validate.go` | `TestPassword`, `TestRegisterValidation` |
| Usernames unique and normalized | trimmed + lowercased, `[a-z0-9._-]`, 3–32 characters, UNIQUE in the database | `internal/auth/validate.go`, `internal/db/db.go` | `TestUsername`, `TestRegisterValidation` |
| Session cookie flags | `HttpOnly`, `Secure` (when APP_URL is https), `SameSite=Lax`, `Domain=COOKIE_DOMAIN`, `Path=/` | `internal/server/security.go` | `TestRegisterLoginLogout` |
| Server-side sessions | 256-bit random token; the database only stores its SHA-256, so a leaked database holds no usable sessions | `internal/store/sessions.go` | `TestSessions` |
| Session expiry | 30 days, sliding (extended at most once a minute); expired sessions are deleted hourly | `internal/store/sessions.go`, `cmd/turkushan-auth/main.go` | `TestSessions` |
| Logout really ends the session | the session row is deleted, so the cookie stops working on every subdomain | `internal/server/handlers_auth.go` | `TestLogoutKillsSessionEverywhere`, `TestForwardAuth` |
| No session fixation | a new session is created on every login; the old one is deleted | `internal/server/security.go` | `TestRegisterLoginLogout` |
| Open-redirect protection | `rd` is only followed to https URLs on the cookie domain or its subdomains, without user info, backslashes or control characters; otherwise the portal's own page | `internal/auth/redirect.go` | `TestSafeRedirect`, `TestRegisterLoginLogout` |
| CSRF protection | every non-GET API call needs the double-submit token (`ta_csrf` cookie echoed in `X-CSRF-Token`) and, when sent, a matching `Origin`; JSON-only bodies | `internal/server/security.go` | `TestCSRF` |
| Rate limiting | login: 20/min per IP and 10/min per username; registration: 5/hour per IP | `internal/ratelimit`, `internal/server/handlers_auth.go` | `TestLimiter`, `TestLoginRateLimit`, `TestRegisterRateLimit` |
| Lockout | 5 wrong passwords in a row lock the account for 15 minutes; the admin can unlock | `internal/store/users.go` | `TestLockout`, `TestRegisterUniqueAndLockout` |
| No username probing on login | unknown users get the same message and take the same time (dummy argon2 check) | `internal/server/handlers_auth.go` | `TestLockout` |
| Admin account from env | `ADMIN_USER` / `ADMIN_PASSWORD` create the admin on first start; the password is never overwritten afterwards | `internal/store/users.go` | `TestBootstrapAdmin` |
| Session secret | `SESSION_SECRET`, or generated once into `/data/secret` (0600). Sessions don't depend on it (they are random tokens), so it's kept for future signed data | `internal/config/config.go` | `TestLoadSecretGeneratesAndReuses` |
| Trusted proxies only | `X-Real-IP` / `X-Forwarded-For` are only believed from `TRUSTED_PROXIES`; otherwise the direct peer IP is used | `internal/server/security.go` | `TestClientIP` |
| Admin panel only for admins | every `/api/admin/*` route checks the session and the admin flag; admins can't block/delete admin accounts | `internal/server/admin.go` | `TestAdminAPI` |
| Audit log | every admin action is logged with admin, target and IP | `internal/server/admin.go` | `TestAdminAPI`, `TestAdminUserActions` |
| Deny by default | hosts that aren't registered as a site get 403, even for signed-in users | `internal/server/forward.go` | `TestForwardAuth` |
| Blocked users are out immediately | blocking deletes all their sessions; blocked accounts can't log in | `internal/store/admin.go` | `TestBlockedUserCannotLogin`, `TestAdminAPI` |
| No nginx config injection | site upstreams must be a plain `http(s)://host[:port]` without `; { } $ ' "` or whitespace; a site can't point at itself or at the portal | `internal/server/admin.go` | `TestAdminAPI` |
| No SSRF via settings | the Discord webhook must be a real `discord.com/api/webhooks/…` URL | `internal/notify/discord.go` | `TestValidDiscordWebhook` |
| Browser hardening | CSP (`default-src 'self'`, no inline scripts), `frame-ancestors 'none'`, `X-Frame-Options`, `nosniff`, `Referrer-Policy`, COOP, `Permissions-Policy`, HSTS on https; the frontend never renders raw HTML | `internal/server/server.go` | `TestSPA` |
| Small attack surface | pure-Go static binary on distroless (no shell), runs as `PUID:PGID` (99:100), request size and time limits | `Dockerfile`, `cmd/turkushan-auth` | – |
| Dependencies | `govulncheck` runs on every build | `.github/workflows/build.yml` | CI |

## Known limitations

- **Every protected app sees the session cookie.** SSO works by sharing one cookie across `*.your-domain`, so the apps behind the portal receive it too (same as Tinyauth, Authelia and similar tools). Only put apps you trust under that domain.
- **Anyone can lock an account for 15 minutes** by entering its password wrong 5 times. That's the cost of the lockout; the admin can unlock early in the panel.
- **Short passwords are allowed** (6 characters with a capital and a symbol). Rate limits and the lockout make guessing slow, but longer is still better.
- **Registration is open.** New accounts can't open anything until the admin approves them (unless a site is set to "open for everyone signed in"), and sign-ups are limited to 5 per hour per IP.
- **The portal port (3010) is reachable on the LAN.** The forward-auth endpoint only reveals the caller's own access status. Firewall the port if your LAN is not trusted.
- **Logout can be triggered by a link** from another site. It only signs you out, it can't do anything else.

## Tips for the owner

- After the first start, remove `ADMIN_PASSWORD` from the container settings. It is only used to create the admin account.
- Keep `COOKIE_DOMAIN` as narrow as possible.
- Turn on HSTS and "Force SSL" in NPM for every protected site.
