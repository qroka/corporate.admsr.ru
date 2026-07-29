/**
 * Серверная сессия портала (sessionToken).
 * Используется модулем курсов; не ломает legacy API, которые ещё принимают userId.
 */
const SESSION_KEY = 'auth-session';
const USER_KEY = 'auth-user';

export function getSessionToken(): string | null {
  try {
    return localStorage.getItem(SESSION_KEY);
  } catch {
    return null;
  }
}

export function setSessionToken(token: string | null) {
  try {
    if (token) localStorage.setItem(SESSION_KEY, token);
    else localStorage.removeItem(SESSION_KEY);
  } catch {
    /* ignore */
  }
}

export function getAuthUser(): { id: number; fio?: string; user_group?: string; role?: string; ofo?: unknown } | null {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null');
  } catch {
    return null;
  }
}

export function clearAuthStorage() {
  try {
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(SESSION_KEY);
    localStorage.removeItem('auth-last-check');
  } catch {
    /* ignore */
  }
}

export type ApiResult<T = unknown> = {
  success: boolean;
  message?: string;
  data?: T;
};

/**
 * fetch к API курсов / attempt с заголовком сессии.
 * Не передаёт userId как источник личности.
 */
export async function apiSessionFetch<T = unknown>(
  url: string,
  options: RequestInit & { json?: unknown } = {},
): Promise<ApiResult<T>> {
  const headers = new Headers(options.headers || {});
  const token = getSessionToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('X-Session-Token', token);
  }

  let body = options.body;
  if (options.json !== undefined) {
    headers.set('Content-Type', 'application/json');
    body = JSON.stringify(options.json);
  }

  const res = await fetch(url, { ...options, headers, body });
  let data: ApiResult<T>;
  try {
    data = await res.json();
  } catch {
    data = { success: false, message: `HTTP ${res.status}` };
  }
  if (!res.ok && !data.message) {
    data.message = `Ошибка ${res.status}`;
  }
  return data;
}

export async function apiSessionUpload<T = unknown>(
  url: string,
  formData: FormData,
): Promise<ApiResult<T>> {
  const headers = new Headers();
  const token = getSessionToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('X-Session-Token', token);
  }
  const res = await fetch(url, { method: 'POST', headers, body: formData });
  try {
    return await res.json();
  } catch {
    return { success: false, message: `HTTP ${res.status}` };
  }
}
