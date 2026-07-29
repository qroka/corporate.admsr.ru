/**
 * Серверная сессия портала (sessionToken).
 * Используется модулем курсов; не ломает legacy API, которые ещё принимают userId.
 */
const SESSION_KEY = 'auth-session';
const USER_KEY = 'auth-user';

export function getSessionToken(): string | null {
  try {
    const direct = localStorage.getItem(SESSION_KEY);
    if (direct) return direct;
    // fallback: токен мог остаться только внутри auth-user после логина
    const user = JSON.parse(localStorage.getItem(USER_KEY) || 'null');
    const nested = user?.sessionToken;
    if (typeof nested === 'string' && nested.length > 0) {
      localStorage.setItem(SESSION_KEY, nested);
      return nested;
    }
    return null;
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

export function getAuthUser(): { id: number; fio?: string; user_group?: string; role?: string; ofo?: unknown; sessionToken?: string } | null {
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

let bootstrapPromise: Promise<boolean> | null = null;

/**
 * Если sessionToken нет, но пользователь залогинен в портале —
 * запросить токен через session_bootstrap (после деплоя V4).
 */
export async function ensureSessionToken(): Promise<boolean> {
  if (getSessionToken()) return true;
  const user = getAuthUser();
  if (!user?.id) return false;

  if (!bootstrapPromise) {
    bootstrapPromise = (async () => {
      try {
        const res = await fetch('/api/session_bootstrap.php', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ id: user.id }),
        });
        const data = await res.json();
        const token = data?.data?.sessionToken || data?.sessionToken;
        if (data?.success && token) {
          setSessionToken(token);
          try {
            const u = getAuthUser() || { id: user.id };
            localStorage.setItem(USER_KEY, JSON.stringify({ ...u, sessionToken: token }));
          } catch {
            /* ignore */
          }
          return true;
        }
        return false;
      } catch {
        return false;
      } finally {
        bootstrapPromise = null;
      }
    })();
  }
  return bootstrapPromise;
}

/**
 * fetch к API курсов / attempt с заголовком сессии.
 * Не передаёт userId как источник личности.
 */
export async function apiSessionFetch<T = unknown>(
  url: string,
  options: RequestInit & { json?: unknown } = {},
): Promise<ApiResult<T>> {
  await ensureSessionToken();

  const doFetch = async (): Promise<ApiResult<T>> => {
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
    if (res.status === 401) {
      data.message = data.message || 'Требуется авторизация';
      (data as any)._unauthorized = true;
    }
    return data;
  };

  let result = await doFetch();
  if ((result as any)._unauthorized && getAuthUser()?.id) {
    // токен протух — попробовать выдать новый один раз
    setSessionToken(null);
    const ok = await ensureSessionToken();
    if (ok) result = await doFetch();
  }

  if ((result as any)._unauthorized) {
    result.message = 'Требуется авторизация. Выйдите из портала и войдите снова.';
  }
  return result;
}

export async function apiSessionUpload<T = unknown>(
  url: string,
  formData: FormData,
): Promise<ApiResult<T>> {
  await ensureSessionToken();
  const headers = new Headers();
  const token = getSessionToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
    headers.set('X-Session-Token', token);
  }
  const res = await fetch(url, { method: 'POST', headers, body: formData });
  try {
    const data = await res.json();
    if (res.status === 401) {
      return {
        success: false,
        message: 'Требуется авторизация. Выйдите из портала и войдите снова.',
      };
    }
    return data;
  } catch {
    return { success: false, message: `HTTP ${res.status}` };
  }
}
