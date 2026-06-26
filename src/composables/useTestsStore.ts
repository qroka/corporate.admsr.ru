import { ref } from 'vue';
import { type TestForm } from '../pages/Tests/testForm';

// ── Текущий пользователь (из auth-user в localStorage) ───────────────────────
function currentUserId(): number {
  try {
    const u = JSON.parse(localStorage.getItem('auth-user') || 'null');
    const id = Number(u?.id ?? 0);
    return Number.isFinite(id) && id > 0 ? id : 0;
  } catch {
    return 0;
  }
}

async function api(path: string, body: Record<string, unknown>): Promise<any> {
  const res = await fetch(`/api/${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const json = await res.json().catch(() => ({ success: false, message: 'Некорректный ответ сервера' }));
  if (!json.success) throw new Error(json.message || 'Ошибка запроса');
  return json.data;
}

// ── Формы (с сервера) ─────────────────────────────────────────────────────────
const drafts = ref<TestForm[]>([]);
const published = ref<TestForm[]>([]);
const loading = ref(false);
let loaded = false;

async function refresh(): Promise<void> {
  loading.value = true;
  try {
    const data = await api('tests_list.php', { userId: currentUserId() });
    drafts.value = Array.isArray(data?.drafts) ? data.drafts : [];
    published.value = Array.isArray(data?.published) ? data.published : [];
  } finally {
    loading.value = false;
  }
}
async function ensureLoaded(): Promise<void> {
  if (loaded) return;
  loaded = true;
  try { await refresh(); } catch { /* покажем пусто */ }
}

async function saveDraft(form: TestForm): Promise<TestForm> {
  const saved = await api('tests_save.php', { userId: currentUserId(), form });
  await refresh();
  return saved;
}
async function updateDraft(id: number, form: TestForm): Promise<void> {
  await api('tests_save.php', { userId: currentUserId(), form: { ...form, id } });
  await refresh();
}
async function publish(form: TestForm, fromDraftId: number | null = null): Promise<TestForm> {
  const payload = fromDraftId != null ? { ...form, id: fromDraftId } : form;
  const r = await api('tests_publish.php', { userId: currentUserId(), form: payload });
  await refresh();
  return r;
}
async function removeDraft(id: number): Promise<void> {
  await api('tests_delete.php', { userId: currentUserId(), formId: id });
  clearSession(id);
  await refresh();
}
async function unpublish(id: number): Promise<void> {
  await api('tests_unpublish.php', { userId: currentUserId(), formId: id });
  clearSession(id);
  await refresh();
}
// mode: 'ofo' | 'users'. При needConfirm возвращает { needConfirm:true, already:number[] } без изменений.
async function addDirections(id: number, mode: 'ofo' | 'users', ids: number[], force = false): Promise<any> {
  const r = await api('tests_direct.php', { userId: currentUserId(), formId: id, mode, ids, force });
  if (!r?.needConfirm) await refresh();
  return r;
}
async function submitAttempt(formId: number, answers: Record<string, unknown>, durationSec: number): Promise<any> {
  return api('tests_submit.php', { userId: currentUserId(), formId, answers, durationSec });
}
async function loadStats(formId: number): Promise<any> {
  return api('tests_stats.php', { formId });
}

// ── Сессии прохождения (для «Продолжить») — клиентские, localStorage ─────────
export type TestSession = { page: number; answers: Record<string, unknown>; startTs: number };
const SESSIONS_KEY = 'tests-sessions-v1';
const sessions = ref<Record<number, TestSession>>(loadSessions());

function loadSessions(): Record<number, TestSession> {
  try { return JSON.parse(localStorage.getItem(SESSIONS_KEY) || '{}') || {}; } catch { return {}; }
}
function persistSessions() {
  try { localStorage.setItem(SESSIONS_KEY, JSON.stringify(sessions.value)); } catch { /* ignore */ }
}
function hmsToSeconds(v: string): number {
  const [h = 0, m = 0, s = 0] = v.split(':').map(Number);
  return h * 3600 + m * 60 + s;
}
function getSession(id: number): TestSession | undefined { return sessions.value[id]; }
function saveSession(id: number, s: TestSession) { sessions.value[id] = s; persistSessions(); }
function clearSession(id: number) { if (sessions.value[id]) { delete sessions.value[id]; persistSessions(); } }
function hasActiveSession(form: TestForm): boolean {
  if (form.id == null) return false;
  const s = sessions.value[form.id];
  if (!s) return false;
  if (form.useTimeLimit && form.timeLimit) return (Date.now() - s.startTs) / 1000 < hmsToSeconds(form.timeLimit);
  return true;
}

export function useTestsStore() {
  return {
    drafts,
    published,
    loading,
    sessions,
    ensureLoaded,
    refresh,
    saveDraft,
    updateDraft,
    publish,
    removeDraft,
    unpublish,
    addDirections,
    submitAttempt,
    loadStats,
    getSession,
    saveSession,
    clearSession,
    hasActiveSession,
  };
}
