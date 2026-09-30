function cookie(name) {
  const m = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
  return m ? decodeURIComponent(m[1]) : '';
}

// api('/session') does a GET; api('/login', {...}) POSTs JSON with the CSRF token.
// Always resolves to { ok, status, data }; data.error holds a message for the user.
export async function api(path, body) {
  const opts = { method: 'GET', headers: { Accept: 'application/json' }, credentials: 'same-origin' };
  if (body !== undefined) {
    opts.method = 'POST';
    opts.headers['Content-Type'] = 'application/json';
    opts.headers['X-CSRF-Token'] = cookie('ta_csrf');
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
