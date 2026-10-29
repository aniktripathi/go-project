// All communication with the Go backend goes through this file.
//
// Token storage: the JWT is kept in localStorage so a page refresh keeps you logged in.
// Trade-off: any script running on the page (XSS) could read it. React escapes all
// rendered text and the app uses no dangerouslySetInnerHTML, which reduces that risk.
// An HttpOnly cookie would be safer against XSS but needs extra CSRF protection.

const BASE = import.meta.env.VITE_API_URL || '/api';
const TOKEN_KEY = 'ft_token';

export const tokenStore = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (t) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
};

export class ApiError extends Error {
  constructor(message, status, code, details) {
    super(message);
    this.status = status;
    this.code = code;
    this.details = details || {};
  }
}

function buildUrl(path, params) {
  const query = new URLSearchParams();
  Object.entries(params || {}).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') query.append(k, v);
  });
  const qs = query.toString();
  return `${BASE}${path}${qs ? `?${qs}` : ''}`;
}

async function send(path, { method = 'GET', body, params } = {}) {
  const headers = {};
  const token = tokenStore.get();
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  try {
    return await fetch(buildUrl(path, params), {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError('Cannot reach the server. Check your connection and make sure the backend is running.', 0, 'network');
  }
}

function handleUnauthorized(res) {
  // A 401 while holding a token means it expired or is invalid: log the user out.
  if (res.status === 401 && tokenStore.get()) {
    tokenStore.clear();
    window.dispatchEvent(new Event('ft:unauthorized'));
  }
}

async function request(path, options) {
  const res = await send(path, options);
  let json = null;
  try {
    json = await res.json();
  } catch {
    /* response was not JSON */
  }
  if (!res.ok || !json || json.success !== true) {
    handleUnauthorized(res);
    throw new ApiError(json?.error?.message || 'Something went wrong. Please try again.', res.status, json?.error?.code, json?.error?.details);
  }
  return json; // { success, data, meta? }
}

// Downloads a file (used for the CSV export) and returns { blob, filename }.
async function download(path, params) {
  const res = await send(path, { params });
  if (!res.ok) {
    let message = 'The download failed.';
    try {
      message = (await res.json())?.error?.message || message;
    } catch {
      /* ignore */
    }
    handleUnauthorized(res);
    throw new ApiError(message, res.status);
  }
  const disposition = res.headers.get('Content-Disposition') || '';
  const match = /filename="([^"]+)"/.exec(disposition);
  return { blob: await res.blob(), filename: match ? match[1] : 'export.csv' };
}

export const api = {
  get: (path, params) => request(path, { params }),
  post: (path, body) => request(path, { method: 'POST', body }),
  put: (path, body) => request(path, { method: 'PUT', body }),
  del: (path) => request(path, { method: 'DELETE' }),
  download,
};
