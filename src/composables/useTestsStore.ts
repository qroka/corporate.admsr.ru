import { ref } from 'vue';
import { type TestForm } from '../pages/Tests/testForm';

// Личность пользователя сервер берёт только из сессии (SEC-008): userId в теле
// запросов больше не отправляем — раньше он создавал ложное впечатление, что
// клиент сам сообщает, кто он (IMP-13).

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
const published = ref<TestForm[]>([]); // «Все формы» — только публичные
const mine = ref<TestForm[]>([]);      // «Мои формы» — опубликованные мной (вкл. приватные)
const forMe = ref<TestForm[]>([]);
const loading = ref(false);
let loaded = false;

async function refresh(): Promise<void> {
  loading.value = true;
  try {
    const data = await api('tests_list.php', {});
    drafts.value = Array.isArray(data?.drafts) ? data.drafts : [];
    published.value = Array.isArray(data?.published) ? data.published : [];
    mine.value = Array.isArray(data?.mine) ? data.mine : [];
    forMe.value = Array.isArray(data?.forMe) ? data.forMe : [];
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
  const saved = await api('tests_save.php', { form });
  await refresh();
  return saved;
}
async function updateDraft(id: number, form: TestForm): Promise<void> {
  await api('tests_save.php', { form: { ...form, id } });
  await refresh();
}
async function publish(form: TestForm, fromDraftId: number | null = null): Promise<TestForm> {
  const payload = fromDraftId != null ? { ...form, id: fromDraftId } : form;
  const r = await api('tests_publish.php', { form: payload });
  await refresh();
  return r;
}
async function removeDraft(id: number): Promise<void> {
  await api('tests_delete.php', { formId: id });
  clearSession(id);
  await refresh();
}
async function unpublish(id: number): Promise<void> {
  await api('tests_unpublish.php', { formId: id });
  clearSession(id);
  await refresh();
}
// mode: 'ofo' | 'users'. При needConfirm возвращает { needConfirm:true, already:number[] } без изменений.
async function addDirections(id: number, mode: 'ofo' | 'users', ids: number[], force = false): Promise<any> {
  const r = await api('tests_direct.php', { formId: id, mode, ids, force });
  if (!r?.needConfirm) await refresh();
  return r;
}
async function submitAttempt(formId: number, answers: Record<string, unknown>, durationSec: number): Promise<any> {
  return api('tests_submit.php', { formId, answers, durationSec });
}
async function loadStats(formId: number): Promise<any> {
  return api('tests_stats.php', { formId });
}
async function loadParticipant(formId: number, participantId: number): Promise<any> {
  return api('tests_participant.php', { formId, participantId });
}
// ── Доступ по ссылке ──────────────────────────────────────────────────────────
async function loadByToken(token: string, respondentToken?: string): Promise<any> {
  return api('tests_by_token.php', { token, respondentToken });
}
async function submitByToken(
  token: string,
  payload: { answers: Record<string, unknown>; durationSec: number; guestName?: string; guestOfoId?: number; respondentToken?: string },
): Promise<any> {
  return api('tests_submit.php', { token, ...payload });
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
    mine,
    forMe,
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
    loadParticipant,
    loadByToken,
    submitByToken,
    getSession,
    saveSession,
    clearSession,
    hasActiveSession,
  };
}
