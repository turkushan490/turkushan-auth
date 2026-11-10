# Changelog

## 1.0.0 (2026-11-10)

First tagged release. Mostly security fixes.

> **Action needed after updating:** copy the **NPM config** again for every site (Admin panel → Sites → NPM config → Copy) and paste it over the old one in NPM. The old config could be tricked into checking the wrong site.

### Security
- **The NPM config checks the right site.** It now sends nginx's `$host` instead of the raw `Host` header. With the old config, a visitor with any account could fake the header and get an approval-only site checked as if it were an open one.
- **Helpers can't take over people with more rights.** Someone with the "Manage users" right can no longer reset, block or delete a user who has rights they don't have.
- **Only admins change the mail server and the Discord/Google login apps.** Before this, the "Settings" right was enough to point password-reset mails at your own server. Changing the mail server or port now also clears the saved mail password.
- **Only admins open a site to everyone.** The "Sites" right can no longer turn "Needs approval" off.
- **Discord/Google sign-in is hardened:**
  - It now uses PKCE.
  - The sign-in state cookie is signed and `__Host-` prefixed, so another subdomain can't plant one.
  - Admin and staff accounts are never connected automatically by email. They connect Discord/Google from their account page instead.
  - Blocked users are stopped before anything about them is saved.
- **Account names stay private.**
  - A locked account answers "wrong password" unless the password is right, so the lockout no longer shows which usernames exist.
  - "Forgot password" answers before it sends the mail, so its timing no longer shows which accounts exist.
- Discord notifications can't ping @everyone or roles.

### New
- **Apps behind the portal can see who is visiting.** They receive `Remote-User`, `Remote-Email` (only when the address is verified) and `Remote-Groups` headers. The visitor can't fake them. Apps with "trusted header" login can use these to sign people in automatically.

### Fixed
- **Disconnect works again.** The Discord/Google "Disconnect" button on the account page did nothing.
- People who only sign in with Discord/Google can change a verified email address. They used to get "Wrong password".
- Clicking "Finish" twice on the first Discord/Google sign-in no longer makes two accounts.
- Sending mail over port 465 now gives up when the request times out.
- Expired sessions and links are cleaned up at start-up too, not only once an hour.
