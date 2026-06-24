import type { Router } from 'vue-router';

/**
 * Авто-логаут по бездействию.
 * - При реальной активности (клик/ввод/скролл/навигация) шлём heartbeat (не чаще раза в HEARTBEAT_THROTTLE),
 *   сервер обновляет last_activity.
 * - Периодически (CHECK_INTERVAL) и при возврате на вкладку проверяем check-auth (read-only).
 *   Если сервер сказал, что авторизация снята (24ч без активности) — чистим localStorage и уводим на логин.
 */

const HEARTBEAT_THROTTLE = 4 * 60 * 1000; // не чаще раза в 4 минуты
const CHECK_INTERVAL = 5 * 60 * 1000;     // проверять статус каждые 5 минут

let started = false;
let lastBeat = 0;

function getUserId(): number | null {
  try {
    const u = JSON.parse(localStorage.getItem('auth-user') || 'null');
    const id = Number(u?.id);
    return Number.isFinite(id) && id > 0 ? id : null;
  } catch {
    return null;
  }
}

async function heartbeat() {
  const id = getUserId();
  if (!id) return;
  const now = Date.now();
  if (now - lastBeat < HEARTBEAT_THROTTLE) return;
  lastBeat = now;
  try {
    await fetch('/api/heartbeat.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id }),
      keepalive: true,
    });
  } catch { /* офлайн — не критично */ }
}

export function startSessionActivity(router: Router) {
  if (started || typeof window === 'undefined') return;
  started = true;

  function forceLogout() {
    localStorage.removeItem('auth-user');
    localStorage.removeItem('auth-last-check');
    if (router.currentRoute.value.name !== 'login') {
      void router.replace({ name: 'login' });
    }
  }

  async function checkAuth() {
    const id = getUserId();
    if (!id) return;
    try {
      const res = await fetch('/api/check-auth.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
      });
      const data = await res.json();
      if (data?.success && data.auth === false) forceLogout();
    } catch { /* сеть недоступна — пропускаем */ }
  }

  const onActivity = () => { void heartbeat(); };
  for (const ev of ['click', 'keydown', 'scroll', 'mousemove', 'touchstart'] as const) {
    window.addEventListener(ev, onActivity, { passive: true });
  }

  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') {
      void heartbeat();
      void checkAuth();
    }
  });

  // Навигация = активность + проверка статуса
  router.afterEach(() => {
    void heartbeat();
  });

  window.setInterval(() => { void checkAuth(); }, CHECK_INTERVAL);

  // Стартовая отметка/проверка
  void heartbeat();
  void checkAuth();
}
