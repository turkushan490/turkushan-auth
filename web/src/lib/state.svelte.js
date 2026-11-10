import { api } from './api.js';
import { confirmDialog } from './confirm.svelte.js';
import { i18n, pickLanguage, setLanguage, t } from './i18n.svelte.js';

export const app = $state({
  loading: true,
  session: null, // { authenticated, user, app_url, brand }
  path: location.pathname,
  // One-off message for the next page, e.g. "You're logged out.": a text, or a
  // function that returns it (so it follows a language change while shown).
  flash: '',
  toast: null, // { text, kind, id } shown bottom-right for a few seconds
  preview: null, // unsaved appearance being tried out in Admin panel → Appearance
});

// The look to render right now: the admin's unsaved draft, else what's saved.
export function look() {
  return app.preview || app.session?.appearance || {};
}

export function toast(text, kind = 'success') {
  const id = Date.now() + Math.random();
  app.toast = { text, kind, id };
  setTimeout(() => {
    if (app.toast?.id === id) app.toast = null;
  }, kind === 'error' ? 6000 : 3000);
}

let languageChosen = false;

export async function refreshSession() {
  const r = await api('/session');
  app.session = r.ok ? r.data : { authenticated: false };
  // The first time (page load), pick the language before anything is shown.
  // Signing in later also follows the language saved on the account.
  const accountLang = app.session.user?.lang;
  if (!languageChosen || (accountLang && accountLang !== i18n.lang)) {
    languageChosen = true;
    await setLanguage(pickLanguage(app.session));
  }
  app.loading = false;
}

// A page with unsaved changes sets this to a function that returns a question
// ("Leave without saving?") while there's something to lose, or '' when not.
let leaveGuard = null;
export function setLeaveGuard(fn) {
  leaveGuard = fn;
  return () => {
    if (leaveGuard === fn) leaveGuard = null;
  };
}

window.addEventListener('beforeunload', (e) => {
  if (leaveGuard?.()) e.preventDefault();
});

export function navigate(to) {
  history.pushState({}, '', to);
  app.path = location.pathname;
  window.scrollTo(0, 0);
}

// onclick handler for in-app <a href="..."> links.
export async function link(e) {
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
  e.preventDefault();
  const to = e.currentTarget.getAttribute('href');
  const question = leaveGuard?.();
  if (question && !(await confirmDialog({ title: question, ok: t('common.leave'), cancel: t('common.stay'), danger: true }))) return;
  navigate(to);
}

// The ?rd= the user came in with, passed along between login and register.
export function currentRD() {
  return new URLSearchParams(location.search).get('rd') || '';
}

// Which site the visitor is heading to (from ?rd=) and what it needs: { known, name, require_email, ... }.
export async function siteInfo(rd) {
  if (!rd) return { known: false };
  const r = await api('/site?rd=' + encodeURIComponent(rd));
  return r.ok ? r.data : { known: false };
}

// The message for an ?oauth_error= code the server sends back after a Discord/Google attempt.
export function oauthError(code) {
  const key = 'oauth.' + code;
  const text = t(key);
  return text === key ? t('oauth.generic') : text;
}

export function withRD(path, rd) {
  return rd ? `${path}?rd=${encodeURIComponent(rd)}` : path;
}

window.addEventListener('popstate', () => {
  app.path = location.pathname;
});
