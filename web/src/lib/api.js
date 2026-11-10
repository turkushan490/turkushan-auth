import { errorText, t } from './i18n.svelte.js';

function cookie(name) {
  const m = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
  return m ? decodeURIComponent(m[1]) : '';
}

// The server sends {"error": "English text", "code": "bad_login"}; show our translation
// of the code when there is one. data.code stays available for pages that react to it.
function localizeError(data) {
  if (data.error || data.code) data.error = errorText(data);
  return data;
}

// api('/session') GETs; api('/login', {...}) POSTs JSON; api(path, body, 'PUT' | 'DELETE') for the rest.
// Every non-GET carries the CSRF token. Always resolves to { ok, status, data };
// data.error holds a message for the user, in their language when the server gave a code.
export async function api(path, body, method) {
  method = method || (body === undefined ? 'GET' : 'POST');
  const opts = { method, headers: { Accept: 'application/json' }, credentials: 'same-origin' };
  if (method !== 'GET') opts.headers['X-CSRF-Token'] = cookie('ta_csrf');
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }

  let res;
  try {
    res = await fetch('/api' + path, opts);
  } catch {
    return { ok: false, status: 0, data: { error: t('err.offline') } };
  }
  let data = {};
  try {
    data = await res.json();
  } catch {
    // non-JSON response, e.g. a proxy error page
  }
  if (!res.ok) {
    if (!data.error && !data.code) data.error = t('err.status', { status: res.status });
    else localizeError(data);
  } else if (data.message && data.code) {
    // Success messages with a code: {"message": "Email address removed.", "code": "email_removed"}.
    const key = 'msg.' + data.code;
    const text = t(key, data);
    if (text !== key) data.message = text;
  }
  return { ok: res.ok, status: res.status, data };
}

// apiDownload POSTs and saves the response as a file (database backup).
export async function apiDownload(path, fallbackName) {
  let res;
  try {
    res = await fetch('/api' + path, {
      method: 'POST',
      headers: { 'X-CSRF-Token': cookie('ta_csrf'), 'Content-Type': 'application/json' },
      credentials: 'same-origin',
      body: '{}',
    });
  } catch {
    return { ok: false, error: t('err.offline') };
  }
  if (!res.ok) {
    let error = t('err.download', { status: res.status });
    try {
      const data = await res.json();
      if (data.error || data.code) error = errorText(data);
    } catch {
      // not JSON
    }
    return { ok: false, error };
  }
  const name = /filename="([^"]+)"/.exec(res.headers.get('Content-Disposition') || '')?.[1] || fallbackName;
  const url = URL.createObjectURL(await res.blob());
  const a = Object.assign(document.createElement('a'), { href: url, download: name });
  document.body.append(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 10_000);
  return { ok: true };
}

// apiUpload sends a file as the raw request body (logo / background upload).
export async function apiUpload(path, file) {
  let res;
  try {
    res = await fetch('/api' + path, {
      method: 'POST',
      headers: { Accept: 'application/json', 'X-CSRF-Token': cookie('ta_csrf'), 'Content-Type': 'application/octet-stream' },
      credentials: 'same-origin',
      body: file,
    });
  } catch {
    return { ok: false, status: 0, data: { error: t('err.upload_offline') } };
  }
  let data = {};
  try {
    data = await res.json();
  } catch {
    // e.g. the proxy refused the upload
  }
  if (!res.ok) {
    if (!data.error && !data.code) data.error = res.status === 413 ? t('err.too_big') : t('err.upload', { status: res.status });
    else localizeError(data);
  }
  return { ok: res.ok, status: res.status, data };
}
