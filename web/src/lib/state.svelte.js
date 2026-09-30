import { api } from './api.js';

export const app = $state({
  loading: true,
  session: null, // { authenticated, user, app_url, brand }
  path: location.pathname,
  flash: '', // one-off message for the next page, e.g. "You're logged out."
});

export async function refreshSession() {
  const r = await api('/session');
  app.session = r.ok ? r.data : { authenticated: false };
  app.loading = false;
}

export function navigate(to) {
  history.pushState({}, '', to);
  app.path = location.pathname;
  window.scrollTo(0, 0);
}

// onclick handler for in-app <a href="..."> links.
export function link(e) {
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
  e.preventDefault();
  navigate(e.currentTarget.getAttribute('href'));
}

// The ?rd= the user came in with, passed along between login and register.
export function currentRD() {
  return new URLSearchParams(location.search).get('rd') || '';
}

export function withRD(path, rd) {
  return rd ? `${path}?rd=${encodeURIComponent(rd)}` : path;
}

window.addEventListener('popstate', () => {
  app.path = location.pathname;
});
