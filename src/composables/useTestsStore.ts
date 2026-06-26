import { ref, watch } from 'vue';
import { cloneForm, type TestForm } from '../pages/Tests/testForm';

const STORAGE_KEY = 'tests-store-v1';

// Сессия незавершённого прохождения (для «Продолжить»)
export type TestSession = {
  page: number;
  answers: Record<string, unknown>;
  startTs: number; // момент старта (мс)
};

const drafts = ref<TestForm[]>([]);
const published = ref<TestForm[]>([]);
const sessions = ref<Record<number, TestSession>>({});
let nextId = 1;
let loaded = false;

function persist() {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ drafts: drafts.value, published: published.value, sessions: sessions.value, nextId }),
    );
  } catch {
    /* localStorage недоступен — игнорируем */
  }
}

function load() {
  if (loaded) return;
  loaded = true;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const data = JSON.parse(raw);
      drafts.value = Array.isArray(data.drafts) ? data.drafts : [];
      published.value = Array.isArray(data.published) ? data.published : [];
      sessions.value = data.sessions && typeof data.sessions === 'object' ? data.sessions : {};
      nextId = typeof data.nextId === 'number' ? data.nextId : 1;
    }
  } catch {
    /* битые данные — начинаем с нуля */
  }
  watch([drafts, published, sessions], persist, { deep: true });
}

const nowISO = () => new Date().toISOString();

function hmsToSeconds(v: string): number {
  const [h = 0, m = 0, s = 0] = v.split(':').map(Number);
  return h * 3600 + m * 60 + s;
}

function saveDraft(form: TestForm): TestForm {
  const copy = cloneForm(form);
  copy.id = nextId++;
  copy.createdAt = nowISO();
  copy.updatedAt = copy.createdAt;
  drafts.value.push(copy);
  persist();
  return copy;
}

function updateDraft(id: number, form: TestForm): void {
  const i = drafts.value.findIndex((d) => d.id === id);
  if (i === -1) return;
  const copy = cloneForm(form);
  copy.id = id;
  copy.createdAt = drafts.value[i].createdAt ?? nowISO();
  copy.updatedAt = nowISO();
  drafts.value[i] = copy;
  clearSession(id); // вопросы могли измениться — старое прохождение неактуально
  persist();
}

function removeDraft(id: number): void {
  drafts.value = drafts.value.filter((d) => d.id !== id);
  clearSession(id);
  persist();
}

// Публикация из любого источника (новое создание / редактирование / список черновиков).
// fromDraftId — если публикуем из черновика, он удаляется.
function publish(form: TestForm, fromDraftId: number | null = null): TestForm {
  const copy = cloneForm(form);
  copy.id = nextId++;
  copy.createdAt = nowISO();
  copy.updatedAt = copy.createdAt;
  published.value.push(copy);
  if (fromDraftId != null) removeDraft(fromDraftId);
  persist();
  return copy;
}

// ── Сессии прохождения ────────────────────────────────────────────────────────
function getSession(id: number): TestSession | undefined {
  return sessions.value[id];
}
function saveSession(id: number, s: TestSession): void {
  sessions.value[id] = s;
  persist();
}
function clearSession(id: number): void {
  if (sessions.value[id]) {
    delete sessions.value[id];
    persist();
  }
}
// Есть ли незавершённое прохождение, которое можно продолжить (время ещё не вышло)
function hasActiveSession(form: TestForm): boolean {
  if (form.id == null) return false;
  const s = sessions.value[form.id];
  if (!s) return false;
  if (form.useTimeLimit && form.timeLimit) {
    const elapsed = (Date.now() - s.startTs) / 1000;
    return elapsed < hmsToSeconds(form.timeLimit);
  }
  return true;
}

export function useTestsStore() {
  load();
  return {
    drafts,
    published,
    sessions,
    saveDraft,
    updateDraft,
    removeDraft,
    publish,
    getSession,
    saveSession,
    clearSession,
    hasActiveSession,
  };
}
