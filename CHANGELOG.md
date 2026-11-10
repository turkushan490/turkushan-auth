# Changelog

## 1.2.0 (2026-11-10)

Easier and safer to use, with screen readers and keyboard too.

### New
- **The app's own "Are you sure?" dialog.** It replaces the browser's pop-up everywhere. Blocking a user, removing an admin and removing someone from a group or rule now ask first too.
- **Accent color check.** The Appearance tab warns when your accent color makes button text or links hard to read (WCAG contrast). Button text now picks dark or white by real contrast.
- **Warning before you lose changes.** Leaving the Appearance tab (or closing the page) with unsaved changes asks first.
- **The Users tab remembers your filters.** Filters, sorting and the opened user are kept in the address, so a reload or a shared link shows the same list. Long lists load 50 at a time.
- **Audit log search and paging.** Search by who, what or which, and load older entries. It used to stop at the newest 300.
- **Live password check on the reset page.** The page shows "passwords don't match" while you type.

### Fixed
- **Dialogs work properly from the keyboard.** Focus moves into the dialog, Tab stays inside it, Esc closes it, and focus goes back where it was.
- **Screen readers get more.**
  - They now announce messages.
  - They read form hints and password rules together with the field.
  - Switches show a focus ring.
  - The user search box has a label.
- **Old access requests show up again.** Requests no longer disappear from the panel after the 500th.
- **Less checking in the background.** The "waiting for approval" page only checks again while its tab is visible.

### Internal
- The Users and Groups tabs are split into smaller components.

## 1.1.0 (2026-11-10)

Backups, version tags and arm64.

### New
- **Backups.** An admin can save one in two ways:
  - **Admin panel → Settings → Download backup** downloads a copy of the database.
  - `docker exec turkushan-auth /turkushan-auth backup` writes one to `/data/backups/`.
  
  Both are safe while the portal runs.
- **More image platforms and tags.** Images are now built for **arm64** too (Raspberry Pi, Apple silicon). Besides `latest`, every release is tagged `1.1.0`, `1.1` and `1`, so you can stay on a version.
- **Version shown in the admin panel.** The running version is at the bottom of the admin panel, and in the log at start-up.
- **Log settings.** `LOG_LEVEL` (`debug`, `info`, `warn`, `error`) and `LOG_FORMAT=json`.
- **`reset-password` asks for the password.** Leave the password off the command and it asks for it, so the password doesn't end up in the terminal's history.
- **Private security reports** via GitHub (Security → Report a vulnerability).

### Changed
- **Builds are reproducible.** Go dependencies are pinned (`go.sum`) and verified.
- **CI checks more.** It also checks formatting and the frontend, runs the tests with the race detector, and scans the image with Trivy.
- **Releases and updates are automated.** A version tag now creates the GitHub Release on its own, and Dependabot keeps dependencies up to date.
- **Unraid template.**
  - The `ADMIN_PASSWORD` field is no longer required, since it's only needed on the first start.
  - The WebUI button is gone, because signing in through the IP address can't work.
  - New optional log settings.

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
