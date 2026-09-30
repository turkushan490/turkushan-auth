function cookie(name) {
  const m = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
  return m ? decodeURIComponent(m[1]) : '';
}

// api('/session') GETs; api('/login', {...}) POSTs JSON; api(path, body, 'PUT' | 'DELETE') for the rest.
// Every non-GET carries the CSRF token. Always resolves to { ok, status, data };
// data.error holds a message for the user.
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
    return { ok: false, status: 0, data: { error: 'Cannot reach the server. Check your connection.' } };
  }
  let data = {};
  try {
    data = await res.json();
  } catch {
    // non-JSON response, e.g. a proxy error page
  }
  if (!res.ok && !data.error) data.error = `Something went wrong (${res.status}).`;
  return { ok: res.ok, status: res.status, data };
}
