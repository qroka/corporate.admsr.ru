<script setup lang="ts">
import { computed, h, onMounted, ref, resolveComponent, watch } from 'vue';
import type { TableColumn, TabsItem } from '@nuxt/ui';
import { setHasActiveAbsence } from '../stores/absenceJournal';
import { useSectionAccess } from '../composables/useSectionAccess';
import { useAppToast } from '../composables/useAppToast';
import { apiSessionFetch } from '../composables/useAuthSession';

type JsonRow = Record<string, unknown>;

type CurrentUser = {
  id: string;
  fio: string;
  ofoId: string;
  role: string;
};

type AbsenceStatus = 'active' | 'completed';
type AbsenceRecord = {
  id: string;
  userId: string;
  fio: string;
  ofoId: string;
  ofoTitle: string;
  createdAt: Date;
  startAt: Date;
  endAt: Date | null;
  reason: string;
  role: string;
  status: AbsenceStatus;
};

function asText(v: unknown): string {
  return String(v ?? '').replace(/\t/g, '').trim();
}

/** Разбирает запись из API в AbsenceRecord */
function mapApiRecord(row: JsonRow, ofoMap: Record<string, string>): AbsenceRecord {
  const ofoId = String(row.ofo ?? '');
  const role = asText((row as any).role ?? (row as any).pos);
  return {
    id:        String(row.id),
    userId:    String(row.user_id),
    fio:       asText(row.fio),
    ofoId,
    ofoTitle:  ofoMap[ofoId] || (ofoId ? `ОФО #${ofoId}` : '—'),
    createdAt: new Date(asText(row.created_at)),
    startAt:   new Date(asText(row.start_datetime)),
    endAt:     row.end_datetime ? new Date(asText(row.end_datetime)) : null,
    reason:    asText(row.reason),
    role,
    status:    row.end_datetime ? 'completed' : 'active',
  };
}

function formatDateTime(dt: Date): string {
  return dt.toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatDateTimeWrap(dt: Date): string {
  if (Number.isNaN(dt.getTime())) return '—';
  const date = dt.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' });
  const time = dt.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${date}\n${time}`;
}

function two(n: number) {
  return n < 10 ? `0${n}` : String(n);
}

function toLocalDateTimeInputValue(d: Date): string {
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getFullYear()}-${two(d.getMonth() + 1)}-${two(d.getDate())}T${two(d.getHours())}:${two(d.getMinutes())}`;
}

function parseDateTimeInputValue(s: string): Date | null {
  const trimmed = String(s ?? '').trim();
  if (!trimmed) return null;
  const d = new Date(trimmed);
  return Number.isNaN(d.getTime()) ? null : d;
}

function diffMs(a: Date, b: Date) {
  return b.getTime() - a.getTime();
}

function formatDurationRu(ms: number): string {
  if (!Number.isFinite(ms)) return '—';
  const sign = ms < 0 ? -1 : 1;
  const abs = Math.abs(ms);
  const totalMinutes = Math.round(abs / 60000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  const base = hours > 0 ? `${hours}ч ${minutes}м` : `${minutes}м`;
  return sign < 0 ? `−${base}` : base;
}

type BadgeColor = 'primary' | 'success' | 'warning' | 'info' | 'neutral';
function ofoBadgeColor(ofoId: string): BadgeColor {
  const id = (ofoId ?? '').trim();
  if (!id || id === '-1' || id === '0') return 'neutral';
  const n = Number.parseInt(id, 10);
  const k = Number.isFinite(n) ? Math.abs(n) : Array.from(id).reduce((acc, ch) => acc + ch.charCodeAt(0), 0);
  const palette: BadgeColor[] = ['primary', 'success', 'warning', 'info', 'neutral'];
  return palette[k % palette.length] ?? 'neutral';
}

const loading = ref(true);
const error = ref<string | null>(null);

const currentUser = ref<CurrentUser | null>(null);
const ofoTitleById = ref<Record<string, string>>({});

const startAbsenceAt = ref(toLocalDateTimeInputValue(new Date()));
const filterPeriod = ref<'all' | 'today' | 'week' | 'month'>('all');
const myRecordsStore = ref<AbsenceRecord[]>([]);
const adminRecordsStore = ref<AbsenceRecord[]>([]);

const MY_PAGE_SIZE = 200;
const myOffset = ref(0);
const myHasMore = ref(true);
const myLoadingMore = ref(false);
const myInitialLoaded = ref(false);

const ADMIN_PAGE_SIZE = 80;
const adminOffset = ref(0);
const adminHasMore = ref(true);
const adminLoadingMore = ref(false);
const adminInitialLoaded = ref(false);

const periodOptions = [
  { label: 'За все время', value: 'all' },
  { label: 'За сегодня', value: 'today' },
  { label: 'За неделю', value: 'week' },
  { label: 'За месяц', value: 'month' },
];

const reasonPresets = [
  'Выезд',
  'Энгильса 10',
  'Работа в архиве',
  'Совещание',
] as const;

const canStartAbsence = computed(() =>
  Boolean(startAbsenceAt.value) && Boolean(currentUser.value) && !activeRecord.value,
);

const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const isAdmin = computed(() => canEditSection('absence_journal'));

const mySearchQuery = ref('');
// Фильтр по статусу для "Мои отсутствия" убран по запросу

function recordHaystack(r: AbsenceRecord): string {
  return [
    r.fio,
    r.ofoTitle,
    r.reason,
    recordStatusLabel(r.status),
    formatDateTime(r.createdAt),
    formatDateTime(r.startAt),
    r.endAt ? formatDateTime(r.endAt) : '',
  ]
    .join(' ')
    .toLowerCase();
}

const myRecords = computed(() => {
  return myRecordsStore.value;
});

const filteredMyRecords = computed(() => {
  const now = new Date();
  let list = myRecords.value;

  const q = mySearchQuery.value.trim().toLowerCase();
  if (q) {
    list = list.filter((r) => recordHaystack(r).includes(q));
  }

  return list.filter((r) => {
    if (filterPeriod.value === 'all') return true;
    if (filterPeriod.value === 'today') {
      return r.createdAt.toDateString() === now.toDateString();
    }
    if (filterPeriod.value === 'week') {
      const delta = now.getTime() - r.createdAt.getTime();
      return delta <= 7 * 24 * 60 * 60 * 1000;
    }
    const delta = now.getTime() - r.createdAt.getTime();
    return delta <= 30 * 24 * 60 * 60 * 1000;
  });
});

const activeRecord = computed(() => myRecords.value.find((r) => r.status === 'active') ?? null);

watch([filterPeriod], () => {
  if (!currentUser.value) return;
  void resetAndLoadMy();
});

let mySearchTimer: number | null = null;
watch(mySearchQuery, () => {
  if (!currentUser.value) return;
  if (mySearchTimer !== null) window.clearTimeout(mySearchTimer);
  mySearchTimer = window.setTimeout(() => {
    void resetAndLoadMy();
  }, 300);
});

const USlideover = resolveComponent('USlideover');
const UBadge = resolveComponent('UBadge');
const UButton = resolveComponent('UButton');
const UDropdownMenu = resolveComponent('UDropdownMenu');

const { toast } = useAppToast();

async function load() {
  loading.value = true;
  error.value = null;
  try {
    // Текущий пользователь из localStorage (устанавливается при авторизации)
    const storedUser = JSON.parse(localStorage.getItem('auth-user') ?? 'null');
    if (!storedUser?.id) throw new Error('Пользователь не авторизован');

    const user: CurrentUser = {
      id:    String(storedUser.id),
      fio:   asText(storedUser.fio),
      ofoId: String(storedUser.ofo ?? storedUser.ofo_id ?? ''),
      role:  String(storedUser.role ?? ''),
    };
    currentUser.value = user;

    // Параллельно грузим ОФО-справочник и записи журнала
    const [ofoRes] = await Promise.all([
      fetch('/api/ofo.php', { cache: 'force-cache' }),
    ]);

    if (!ofoRes.ok)     throw new Error(`Не удалось загрузить ОФО (${ofoRes.status})`);

    const ofoRaw     = await ofoRes.json();

    // Строим карту ОФО id → название
    const ofoMap: Record<string, string> = {};
    for (const row of (ofoRaw.data || [])) {
      const id = asText(row.id);
      if (id) ofoMap[id] = asText(row.title) || '—';
    }
    ofoTitleById.value = ofoMap;

    await resetAndLoadMy();

    if (isAdmin.value) {
      await resetAndLoadAdmin();
    } else {
      adminRecordsStore.value = [];
      adminInitialLoaded.value = false;
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки данных';
    toast.add({
      title: 'Не удалось загрузить данные',
      description: String(error.value),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    loading.value = false;
  }
}

function makeMyQueryParams(): URLSearchParams {
  const params = new URLSearchParams();
  params.set('limit', String(MY_PAGE_SIZE));
  params.set('offset', String(myOffset.value));
  params.set('user_id', String(currentUser.value?.id ?? ''));
  if (filterPeriod.value) params.set('period', filterPeriod.value);
  const q = mySearchQuery.value.trim();
  if (q) params.set('q', q);
  return params;
}

async function loadMoreMy(): Promise<void> {
  if (myLoadingMore.value) return;
  if (!myHasMore.value) return;
  const uid = currentUser.value?.id;
  if (!uid) return;

  myLoadingMore.value = true;
  try {
    const params = makeMyQueryParams();
    const res = await fetch(`/api/absence_journal.php?${params.toString()}`);
    if (!res.ok) throw new Error(`Не удалось загрузить журнал (${res.status})`);
    const raw = await res.json();
    if (!raw.success) throw new Error(raw.error || 'Ошибка загрузки журнала');

    const batch = (raw.data as JsonRow[]).map((row) => mapApiRecord(row, ofoTitleById.value));
    myRecordsStore.value = [...myRecordsStore.value, ...batch];
    myOffset.value += batch.length;
    myHasMore.value = batch.length >= MY_PAGE_SIZE;
    myInitialLoaded.value = true;
  } catch (e) {
    toast.add({
      title: 'Ошибка загрузки',
      description: e instanceof Error ? e.message : 'Не удалось загрузить данные',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    myHasMore.value = false;
  } finally {
    myLoadingMore.value = false;
  }
}

async function resetAndLoadMy(): Promise<void> {
  myOffset.value = 0;
  myHasMore.value = true;
  myInitialLoaded.value = false;
  myRecordsStore.value = [];
  await loadMoreMy();
}

function onMyScroll(e: Event) {
  const el = e.target as HTMLElement | null;
  if (!el) return;
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 250) {
    void loadMoreMy();
  }
}

function makeAdminQueryParams(): URLSearchParams {
  const params = new URLSearchParams();
  params.set('limit', String(ADMIN_PAGE_SIZE));
  params.set('offset', String(adminOffset.value));
  if (ofoFilter.value !== '_all' && ofoFilter.value !== '_none') params.set('ofo', ofoFilter.value);
  params.set('status', 'completed');
  const q = adminSearchQuery.value.trim();
  if (q) params.set('q', q);
  return params;
}

async function loadMoreAdmin(): Promise<void> {
  if (!isAdmin.value) return;
  if (adminLoadingMore.value) return;
  if (!adminHasMore.value) return;
  if (ofoFilter.value === '_none') return;
  if (!Object.keys(ofoTitleById.value).length) return;

  adminLoadingMore.value = true;
  try {
    const params = makeAdminQueryParams();
    const raw = await apiSessionFetch(`/api/absence_journal.php?${params.toString()}`, { method: 'GET' });
    if (!raw.success) throw new Error(raw.message || (raw as any).error || 'Ошибка загрузки журнала');

    const batch = ((raw.data as JsonRow[]) || []).map((row) => mapApiRecord(row, ofoTitleById.value));
    adminRecordsStore.value = [...adminRecordsStore.value, ...batch];
    adminOffset.value += batch.length;
    adminHasMore.value = batch.length >= ADMIN_PAGE_SIZE;
    adminInitialLoaded.value = true;
  } catch (e) {
    toast.add({
      title: 'Ошибка загрузки',
      description: e instanceof Error ? e.message : 'Не удалось загрузить данные',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    adminHasMore.value = false;
  } finally {
    adminLoadingMore.value = false;
  }
}

async function resetAndLoadAdmin(): Promise<void> {
  adminOffset.value = 0;
  adminHasMore.value = true;
  adminInitialLoaded.value = false;
  adminRecordsStore.value = [];
  await loadMoreAdmin();
}

function onAdminScroll(e: Event) {
  const el = e.target as HTMLElement | null;
  if (!el) return;
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 250) {
    void loadMoreAdmin();
  }
}

const finishOpen = ref(false);
const finishingId = ref<string | null>(null);
const finishForm = ref({
  endAt: '',
  reason: '',
});
const finishError = ref<string | null>(null);

const editOpen = ref(false);
const editingId = ref<string | null>(null);
const editForm = ref({
  startAt: '',
  endAt: '',
  reason: '',
});
const editError = ref<string | null>(null);

const finishingRecord = computed(() => {
  const id = finishingId.value;
  if (!id) return null;
  return (
    myRecordsStore.value.find((r) => r.id === id)
    ?? adminRecordsStore.value.find((r) => r.id === id)
    ?? null
  );
});

const editingRecord = computed(() => {
  const id = editingId.value;
  if (!id) return null;
  return (
    myRecordsStore.value.find((r) => r.id === id)
    ?? adminRecordsStore.value.find((r) => r.id === id)
    ?? null
  );
});

const canSaveEdit = computed(() => {
  const rec = editingRecord.value;
  if (!rec) return false;
  const start = parseDateTimeInputValue(editForm.value.startAt);
  if (!start) return false;
  const end = parseDateTimeInputValue(editForm.value.endAt);
  if (end && end.getTime() < start.getTime()) return false;
  const reason = editForm.value.reason.trim();
  if (end && !reason) return false;
  return true;
});

const canFinish = computed(() => {
  const rec = finishingRecord.value;
  if (!rec) return false;
  const end = parseDateTimeInputValue(finishForm.value.endAt);
  const reason = finishForm.value.reason.trim();
  if (!end) return false;
  if (!reason) return false;
  return end.getTime() >= rec.startAt.getTime();
});

function openFinish(record: AbsenceRecord) {
  finishingId.value = record.id;
  finishError.value = null;
  const endDraft = record.endAt ?? new Date();
  const safeEnd = endDraft.getTime() < record.startAt.getTime()
    ? new Date(record.startAt.getTime() + 15 * 60 * 1000)
    : endDraft;
  finishForm.value.endAt = toLocalDateTimeInputValue(safeEnd);
  finishForm.value.reason = record.reason?.trim?.() ? record.reason : '';
  finishOpen.value = true;
}

async function startAbsence() {
  if (!canStartAbsence.value) return;

  const start = parseDateTimeInputValue(startAbsenceAt.value);
  if (!start) {
    toast.add({ title: 'Не указано начало', description: 'Выберите дату и время начала отсутствия.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  const u = currentUser.value;
  if (!u) {
    toast.add({ title: 'Пользователь не определён', description: 'Перезагрузите страницу.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  loading.value = true;
  try {
    const data = await apiSessionFetch('/api/absence_journal.php', {
      method: 'POST',
      json: {
        user_id:        Number(u.id),
        fio:            u.fio,
        ofo:            Number(u.ofoId),
        role:           u.role,
        start_datetime: toLocalDateTimeInputValue(start).replace('T', ' ') + ':00',
      },
    });
    if (!data.success) throw new Error(data.message || (data as any).error || 'Ошибка создания записи');

    const newRecord = mapApiRecord(data.data as JsonRow, ofoTitleById.value);
    myRecordsStore.value = [newRecord, ...myRecordsStore.value];
    if (isAdmin.value) adminRecordsStore.value = [newRecord, ...adminRecordsStore.value];
    toast.add({ title: 'Отсутствие начато', description: `Начало: ${formatDateTime(start)}.`, color: 'success', icon: 'i-lucide-circle-check' });
  } catch (e) {
    toast.add({ title: 'Ошибка', description: e instanceof Error ? e.message : 'Не удалось создать запись', color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
}

function openEdit(record: AbsenceRecord) {
  editingId.value = record.id;
  editError.value = null;
  editForm.value.startAt = toLocalDateTimeInputValue(record.startAt);
  editForm.value.endAt = record.endAt ? toLocalDateTimeInputValue(record.endAt) : '';
  editForm.value.reason = record.reason ?? '';
  editOpen.value = true;
}

async function saveEdit() {
  const rec = editingRecord.value;
  if (!rec) return;

  const start = parseDateTimeInputValue(editForm.value.startAt);
  if (!start) {
    editError.value = 'Укажите время начала.';
    toast.add({ title: 'Не указано начало', description: 'Укажите дату и время начала.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  const end = parseDateTimeInputValue(editForm.value.endAt);
  if (end && end.getTime() < start.getTime()) {
    editError.value = 'Окончание не может быть раньше начала.';
    toast.add({ title: 'Некорректное время', description: 'Окончание не может быть раньше начала.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  const reason = editForm.value.reason.trim();
  if (end && !reason) {
    editError.value = 'Укажите причину (для завершённой записи).';
    toast.add({ title: 'Не указана причина', description: 'Для завершённой записи причина обязательна.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  editError.value = null;
  loading.value = true;
  try {
    const body: Record<string, unknown> = {
      start_datetime: toLocalDateTimeInputValue(start).replace('T', ' ') + ':00',
      end_datetime:   end ? toLocalDateTimeInputValue(end).replace('T', ' ') + ':00' : '',
      reason,
    };
    const data = await apiSessionFetch(`/api/absence_journal.php?id=${rec.id}`, {
      method: 'PUT',
      json: body,
    });
    if (!data.success) throw new Error(data.message || (data as any).error || 'Ошибка обновления записи');

    const updated = mapApiRecord(data.data as JsonRow, ofoTitleById.value);
    myRecordsStore.value = myRecordsStore.value.map((r) => r.id === rec.id ? updated : r);
    adminRecordsStore.value = adminRecordsStore.value.map((r) => r.id === rec.id ? updated : r);

    editOpen.value  = false;
    editingId.value = null;
    toast.add({
      title: 'Изменения сохранены',
      description: end ? `${formatDateTime(start)} → ${formatDateTime(end)}` : `Начало: ${formatDateTime(start)} (незавершено)`,
      color: 'success',
      icon: 'i-lucide-circle-check',
    });
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Ошибка';
    toast.add({ title: 'Ошибка сохранения', description: editError.value ?? '', color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
}

async function finishAbsence() {
  const rec = finishingRecord.value;
  if (!rec) return;
  const end    = parseDateTimeInputValue(finishForm.value.endAt);
  const reason = finishForm.value.reason.trim();

  if (!end) {
    finishError.value = 'Укажите время окончания.';
    toast.add({ title: 'Не указано окончание', description: 'Укажите дату и время окончания отсутствия.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }
  if (!reason) {
    finishError.value = 'Укажите причину.';
    toast.add({ title: 'Не указана причина', description: 'Заполните причину отсутствия.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }
  if (end.getTime() < rec.startAt.getTime()) {
    finishError.value = 'Окончание не может быть раньше начала.';
    toast.add({ title: 'Некорректное время', description: 'Окончание отсутствия не может быть раньше начала.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  finishError.value = null;
  loading.value = true;
  try {
    const data = await apiSessionFetch(`/api/absence_journal.php?id=${rec.id}`, {
      method: 'PUT',
      json: {
        end_datetime: toLocalDateTimeInputValue(end).replace('T', ' ') + ':00',
        reason,
      },
    });
    if (!data.success) throw new Error(data.message || (data as any).error || 'Ошибка завершения записи');

    const updated = mapApiRecord(data.data as JsonRow, ofoTitleById.value);
    myRecordsStore.value = myRecordsStore.value.map((r) => r.id === rec.id ? updated : r);
    adminRecordsStore.value = adminRecordsStore.value.map((r) => r.id === rec.id ? updated : r);

    finishOpen.value  = false;
    finishingId.value = null;
    toast.add({
      title: 'Отсутствие завершено',
      description: `${formatDateTime(rec.startAt)} → ${formatDateTime(end)} · ${reason}`,
      color: 'success',
      icon: 'i-lucide-circle-check',
    });
  } catch (e) {
    finishError.value = e instanceof Error ? e.message : 'Ошибка';
    toast.add({ title: 'Ошибка', description: finishError.value ?? '', color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
}

async function deleteDraft(recordId: string) {
  const row = myRecordsStore.value.find((r) => r.id === recordId) ?? adminRecordsStore.value.find((r) => r.id === recordId);
  // Оптимистичное удаление из UI
  myRecordsStore.value = myRecordsStore.value.filter((r) => r.id !== recordId);
  adminRecordsStore.value = adminRecordsStore.value.filter((r) => r.id !== recordId);
  try {
    const data = await apiSessionFetch(`/api/absence_journal.php?id=${recordId}`, { method: 'DELETE' });
    if (!data.success) throw new Error(data.message || (data as any).error || 'Ошибка удаления');
    toast.add({
      title: 'Запись удалена',
      description: row ? `Удалено: ${formatDateTime(row.startAt)}` : 'Запись удалена.',
      color: 'success',
      icon: 'i-lucide-circle-check',
    });
  } catch (e) {
    // Откатываем если ошибка
    if (row) {
      if (row.userId === currentUser.value?.id) myRecordsStore.value = [row, ...myRecordsStore.value];
      if (isAdmin.value) adminRecordsStore.value = [row, ...adminRecordsStore.value];
    }
    toast.add({ title: 'Ошибка удаления', description: e instanceof Error ? e.message : 'Не удалось удалить запись', color: 'error', icon: 'i-lucide-alert-circle' });
  }
}

function applyFilter() {
  // фильтр реактивный, кнопка нужна для UX в стиле старого интерфейса
}

function recordStatusLabel(s: AbsenceStatus) {
  return s === 'active' ? 'Не завершено' : 'Завершено';
}

function recordStatusColor(s: AbsenceStatus) {
  return s === 'active' ? 'warning' : 'success';
}

type AbsenceRow = AbsenceRecord & {
  startLabel: string;
  endLabel: string;
  durationLabel: string;
};

const tableRows = computed<AbsenceRow[]>(() =>
  filteredMyRecords.value.map((r) => {
    const end = r.endAt;
    const durationMs = end ? diffMs(r.startAt, end) : diffMs(r.startAt, new Date());
    return {
      ...r,
      startLabel: formatDateTimeWrap(r.startAt),
      endLabel: end ? formatDateTimeWrap(end) : '—',
      durationLabel: formatDurationRu(durationMs),
    };
  }),
);

type AbsenceAdminRow = AbsenceRow & {
  fioLabel: string;
  userIdLabel: string;
  createdLabel: string;
  ofoLabel: string;
};

const ofoFilter = ref<string>('_none');
const adminSearchQuery = ref('');
const adminStatusFilter = ref<'' | AbsenceStatus>('completed');

watch([ofoFilter, adminStatusFilter], () => {
  if (!isAdmin.value) return;
  if (ofoFilter.value === '_none') {
    adminRecordsStore.value = [];
    adminInitialLoaded.value = true;
    adminHasMore.value = false;
    return;
  }
  void resetAndLoadAdmin();
});

let adminSearchTimer: number | null = null;
watch(adminSearchQuery, () => {
  if (!isAdmin.value) return;
  if (ofoFilter.value === '_none') return;
  if (adminSearchTimer !== null) window.clearTimeout(adminSearchTimer);
  adminSearchTimer = window.setTimeout(() => {
    void resetAndLoadAdmin();
  }, 300);
});

function resetMyFilters() {
  mySearchQuery.value = '';
  filterPeriod.value = 'all';
  void resetAndLoadMy();
}

function resetAdminFilters() {
  adminSearchQuery.value = '';
  adminStatusFilter.value = 'completed';
  ofoFilter.value = '_none';
}

const ofoItems = computed(() => {
  const base = [
    { label: 'Выберите ОФО', value: '_none' },
    { label: 'Все ОФО', value: '_all' },
  ];
  const entries = Object.entries(ofoTitleById.value)
    .map(([id, title]) => ({ label: title ? `${title}` : `ОФО #${id}`, value: id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru'));
  return base.concat(entries);
});

const adminTableRows = computed<AbsenceAdminRow[]>(() => {
  let list = adminRecordsStore.value;
  const q = adminSearchQuery.value.trim().toLowerCase();
  if (q) {
    list = list.filter((r) => recordHaystack(r).includes(q));
  }

  return list.map((r) => {
    const end = r.endAt;
    const durationMs = end ? diffMs(r.startAt, end) : diffMs(r.startAt, new Date());
    return {
      ...r,
      startLabel: formatDateTimeWrap(r.startAt),
      endLabel: end ? formatDateTimeWrap(end) : '—',
      durationLabel: formatDurationRu(durationMs),
      fioLabel: r.fio || '—',
      userIdLabel: String(r.userId ?? ''),
      createdLabel: formatDateTimeWrap(r.createdAt),
      ofoLabel: r.ofoTitle || '—',
    };
  });
});

function rowMenuItems(row: AbsenceRecord) {
  const adminEditItems = isAdmin.value
    ? [
        {
          label: 'Редактировать',
          icon: 'i-lucide-pencil',
          onSelect() {
            openEdit(row);
          },
        },
      ]
    : [];

  if (row.status === 'active') {
    return [
      ...adminEditItems,
      {
        label: 'Завершить',
        icon: 'i-lucide-check-circle-2',
        onSelect() {
          openFinish(row);
        },
      },
      {
        label: 'Удалить',
        icon: 'i-lucide-trash-2',
        onSelect() {
          deleteDraft(row.id);
        },
      },
    ];
  }
  if (isAdmin.value) {
    return [
      ...adminEditItems,
      {
        label: 'Удалить',
        icon: 'i-lucide-trash-2',
        onSelect() {
          deleteDraft(row.id);
        },
      },
    ];
  }
  return [
    ...adminEditItems,
    {
      label: 'Подать заявку на изменение',
      icon: 'i-lucide-edit-2',
      onSelect() {
        startAbsenceAt.value = toLocalDateTimeInputValue(row.startAt);
      },
    },
  ];
}

const columns: TableColumn<AbsenceRow>[] = [
  {
    accessorKey: 'status',
    header: 'Статус',
    meta: { class: { th: 'w-[140px]', td: 'whitespace-nowrap' } },
    cell: ({ row }) => {
      const s = row.getValue('status') as AbsenceStatus;
      return h(
        UBadge,
        { variant: 'subtle', color: recordStatusColor(s), leading: true, leadingIcon: s === 'active' ? 'i-lucide-timer' : 'i-lucide-check' },
        () => recordStatusLabel(s),
      );
    },
  },
  {
    accessorKey: 'startLabel',
    header: 'Начало',
    meta: { class: { th: 'min-w-[150px]', td: 'tabular-nums whitespace-pre-line' } },
  },
  {
    accessorKey: 'endLabel',
    header: 'Конец',
    meta: { class: { th: 'min-w-[150px]', td: 'tabular-nums whitespace-pre-line' } },
  },
  {
    accessorKey: 'durationLabel',
    header: 'Длительность',
    meta: { class: { th: 'w-[120px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'reason',
    header: 'Причина',
    meta: { class: { th: 'min-w-[280px]', td: 'whitespace-normal' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceRow;
      return h(
        'span',
        { class: [r.reason ? 'text-default' : 'text-muted', 'break-words whitespace-pre-line'].join(' ') },
        r.reason || '—',
      );
    },
  },
  {
    id: 'actions',
    header: () => h('span', { class: 'sr-only' }, 'Действия'),
    enableHiding: false,
    meta: { class: { th: 'w-14 text-right', td: 'text-right' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceRow;
      return h(
        UDropdownMenu,
        { content: { align: 'end' }, items: rowMenuItems(r), 'aria-label': 'Действия с записью отсутствия' },
        () =>
          h(UButton, {
            icon: 'i-lucide-ellipsis-vertical',
            color: 'neutral',
            variant: 'ghost',
            square: true,
            size: 'sm',
            'aria-label': 'Действия',
          }),
      );
    },
  },
];

const adminColumns: TableColumn<AbsenceAdminRow>[] = [
  {
    accessorKey: 'userIdLabel',
    header: 'ID',
    meta: { class: { th: '', td: 'tabular-nums whitespace-nowrap text-muted' } },
  },
  {
    accessorKey: 'fioLabel',
    header: 'Сотрудник',
    meta: { class: { th: '', td: 'whitespace-normal break-words' } },
  },
  {
    accessorKey: 'ofoLabel',
    header: 'ОФО',
    meta: { class: { th: 'min-w-[240px]', td: 'whitespace-normal' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceAdminRow;
      const id = (r.ofoId ?? '').trim();
      const title = r.ofoTitle || ofoTitleById.value[id];
      const label = title
        ? title
        : (!id || id === '-1' || id === '0')
          ? 'Не указано'
          : `ОФО #${id}`;

      return h(UBadge, {
        variant: 'subtle',
        color: ofoBadgeColor(id),
        leading: true,
        leadingIcon: 'i-lucide-building-2',
        class: 'max-w-none min-w-0 whitespace-normal break-words',
        title: title ? `${id} — ${title}` : (id ? `ID: ${id}` : 'Не указано'),
      }, () => label);
    },
  },
  {
    accessorKey: 'createdLabel',
    header: 'Создано',
    meta: { class: { th: 'min-w-[150px]', td: 'tabular-nums whitespace-pre-line' } },
  },
  {
    accessorKey: 'startLabel',
    header: 'Начало',
    meta: { class: { th: 'min-w-[150px]', td: 'tabular-nums whitespace-pre-line' } },
  },
  {
    accessorKey: 'endLabel',
    header: 'Конец',
    meta: { class: { th: 'min-w-[150px]', td: 'tabular-nums whitespace-pre-line' } },
  },
  {
    accessorKey: 'durationLabel',
    header: 'Длительность',
    meta: { class: { th: 'w-[120px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'reason',
    header: 'Причина',
    meta: { class: { th: 'min-w-[280px]', td: 'whitespace-normal' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceAdminRow;
      return h(
        'span',
        { class: [r.reason ? 'text-default' : 'text-muted', 'break-words whitespace-pre-line'].join(' ') },
        r.reason || '—',
      );
    },
  },
  {
    id: 'actions',
    header: () => h('span', { class: 'sr-only' }, 'Действия'),
    enableHiding: false,
    meta: { class: { th: 'w-14 text-right', td: 'text-right' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceAdminRow;
      return h(
        UDropdownMenu,
        { content: { align: 'end' }, items: rowMenuItems(r), 'aria-label': 'Действия с записью отсутствия' },
        () =>
          h(UButton, {
            icon: 'i-lucide-ellipsis-vertical',
            color: 'neutral',
            variant: 'ghost',
            square: true,
            size: 'sm',
            'aria-label': 'Действия',
          }),
      );
    },
  },
];

const tabItems = [
  { label: 'Мои отсутствия', value: 'my', slot: 'my' },
  { label: 'Отсутствия по ОФО', value: 'ofo', slot: 'ofo' },
] satisfies TabsItem[];

const adminTab = ref<(typeof tabItems)[number]['value']>('my');

onMounted(() => {
  void load();
});

watch(
  activeRecord,
  (val) => {
    setHasActiveAbsence(Boolean(val));
  },
  { immediate: true },
);
</script>

<template>
  <UMain class="w-full h-full min-h-0 p-px">
    <UContainer class="flex flex-col gap-6 py-1 mx-0 max-w-full h-full min-h-0">
      <UPageHeader class="border-none p-0 w-full max-w-none">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Журнал отсутствия</h1>
        </template>
      </UPageHeader>

      <UAlert v-if="error" color="error" variant="soft" icon="i-lucide-alert-triangle" title="Ошибка загрузки"
        :description="error" />

      <UTabs v-if="isAdmin" v-model="adminTab" :items="tabItems" size="xl" class="w-full" />

      <!-- Мои отсутствия (как у обычного пользователя) -->
      <UContainer v-if="!isAdmin || adminTab === 'my'" class="flex flex-row items-start gap-3 max-w-full w-full h-full min-h-0">
        <UCard class="overflow-hidden w-96 shrink-0">
          <template #header>
            <UContainer class="flex flex-wrap items-start justify-between gap-3">
              <UContainer class="space-y-1">
                <h2 class="text-xl font-semibold">Старт отсутствия</h2>
                <p class="text-sm text-muted">
                  {{ activeRecord ? 'Есть незавершённая запись — завершите её.' : 'Выберите дату и время начала, затем нажмите «Отметить отсутствие»' }}
                </p>
              </UContainer>
            </UContainer>
          </template>

          <UContainer class="flex flex-col gap-3">
            <UFormField size="xl" label="Дата и время начала">
              <UInput v-model="startAbsenceAt" class="w-full" type="datetime-local"
                :disabled="loading || Boolean(activeRecord)" />
            </UFormField>

            <UButton v-if="activeRecord" color="primary" variant="solid" size="xl" class="w-full justify-center"
              icon="i-lucide-check-circle-2" @click="openFinish(activeRecord)">
              Завершить отсутствие
            </UButton>

            <UButton v-else color="primary" variant="solid" size="xl" class="w-full justify-center"
              :disabled="!canStartAbsence || loading" icon="i-lucide-play-circle" @click="startAbsence">
              Отметить отсутствие
            </UButton>
          </UContainer>
        </UCard>


        <UContainer v-if="loading" class="py-6 text-sm text-muted">Загрузка...</UContainer>
        <UContainer v-else class="w-full max-w-none flex flex-col gap-4 h-full min-h-0">
          <UContainer class="flex flex-row gap-3 w-full max-w-none">
            <UInput v-model="mySearchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline"
              placeholder="Поиск по причине, дате, статусу…" class="w-full sm:flex-1 sm:min-w-[240px]" />
            <USelectMenu v-model="filterPeriod" :items="periodOptions" value-key="value" label-key="label" size="xl"
              color="neutral" placeholder="Период" class="w-full sm:w-52"
              :content="{ align: 'start', sideOffset: 8 }" />
            <UButton color="neutral" variant="outline" size="xl" icon="i-lucide-rotate-ccw" class="shrink-0"
              @click="resetMyFilters">
              Сбросить
            </UButton>
          </UContainer>

          <UContainer v-if="!tableRows.length" class="py-4 text-sm text-muted">Записей не найдено</UContainer>
          <div
            v-else
            class="flex-1 min-h-0 w-full rounded-lg border border-default overflow-auto"
            @scroll.passive="onMyScroll"
          >
            <UTable
              :columns="columns"
              :data="tableRows"
              class="w-full h-full"
              :ui="{
                table: 'w-full',
                th: 'px-4 sm:px-6 py-3 text-xs font-semibold text-muted whitespace-normal',
                td: 'px-4 sm:px-6 py-3 text-sm align-top',
                tr: 'hover:bg-muted/40',
              }"
            />

            <div class="p-4 text-sm text-muted flex items-center justify-center gap-3">
              <span v-if="myLoadingMore">Загрузка…</span>
              <template v-else>
                <UButton
                  v-if="myHasMore"
                  color="neutral"
                  variant="soft"
                  size="sm"
                  icon="i-lucide-arrow-down"
                  @click="loadMoreMy"
                >
                  Загрузить ещё
                </UButton>
              </template>
            </div>
          </div>
        </UContainer>
      </UContainer>


      <!-- Отсутствия по ОФО (только таблица) -->
      <UContainer v-else class="w-full max-w-none flex flex-col gap-4 h-full min-h-0">
        <UContainer class="flex flex-row gap-3 w-full max-w-none">
          <UInput v-model="adminSearchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline"
            placeholder="Поиск по ФИО, причине, дате…" class="w-full sm:flex-1 sm:min-w-[240px]" />
          <USelectMenu v-model="ofoFilter" :items="ofoItems" size="xl" color="neutral" placeholder="ОФО"
            class="w-full sm:w-64" value-key="value" label-key="label" :content="{ align: 'start', sideOffset: 8 }" />
          <UButton color="neutral" variant="outline" size="xl" icon="i-lucide-rotate-ccw" class="shrink-0"
            @click="resetAdminFilters">
            Сбросить
          </UButton>
        </UContainer>

        <UContainer v-if="ofoFilter === '_none'" class="py-10 text-sm text-muted">
          Выберите ОФО, чтобы посмотреть отсутствия сотрудников.
        </UContainer>

        <UContainer v-else-if="adminInitialLoaded && !adminTableRows.length" class="py-4 text-sm text-muted">
          Записей не найдено
        </UContainer>

        <div
          v-else
          class="flex-1 min-h-0 w-full rounded-lg border border-default overflow-auto"
          @scroll.passive="onAdminScroll"
        >
          <UTable
            :columns="adminColumns"
            :data="adminTableRows"
            class="w-full"
            :ui="{
              table: 'w-full',
              th: 'px-4 sm:px-6 py-3 text-xs font-semibold text-muted whitespace-normal',
              td: 'px-4 sm:px-6 py-3 text-sm align-top',
              tr: 'hover:bg-muted/40',
            }"
          />

          <div class="p-4 text-sm text-muted flex items-center justify-center gap-3">
            <span v-if="adminLoadingMore">Загрузка…</span>
            <template v-else>
              <UButton
                v-if="adminHasMore"
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-arrow-down"
                @click="loadMoreAdmin"
              >
                Загрузить ещё
              </UButton>
            </template>
          </div>
        </div>
      </UContainer>

      <USlideover v-model:open="finishOpen" title="Завершение отсутствия">
        <template #body>
          <UContainer class="space-y-4">
            <UAlert v-if="finishingRecord?.status === 'active'" color="warning" variant="subtle" icon="i-lucide-info"
              title="Запись не завершена"
              description="Пока вы не завершите отсутствие, оно будет отображаться как «Не завершено»." />

            <UContainer class="grid grid-cols-1 gap-3">
              <UFormField size="xl" label="Начало">
                <UInput class="w-full" :model-value="finishingRecord ? formatDateTime(finishingRecord.startAt) : ''"
                  readonly />
              </UFormField>

              <UFormField size="xl" label="Время окончания">
                <UInput class="w-full" v-model="finishForm.endAt" type="datetime-local" />
              </UFormField>

              <UFormField size="xl" label="Причина">
                <UTextarea class="w-full" v-model="finishForm.reason" placeholder="Например: К врачу, работа вне офиса…"
                  :rows="4" />

                <UContainer class="mt-2 flex flex-wrap gap-2">
                  <UButton v-for="p in reasonPresets" :key="p" color="neutral" variant="soft" size="md"
                    @click="finishForm.reason = p">
                    {{ p }}
                  </UButton>
                </UContainer>
              </UFormField>
            </UContainer>

            <UAlert v-if="finishError" color="error" variant="subtle" icon="i-lucide-alert-circle"
              :description="finishError" />

            <p v-if="finishingRecord" class="text-md text-muted">
              Длительность: <span class="tabular-nums">{{ formatDurationRu(diffMs(finishingRecord.startAt,
                parseDateTimeInputValue(finishForm.endAt) || finishingRecord.startAt)) }}</span>
            </p>
          </UContainer>
        </template>

        <template #footer>
          <UContainer class="flex justify-between gap-3 items-center w-full">
            <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center"
              @click="finishOpen = false">Пока не завершать</UButton>
            <UButton color="primary" size="xl" class="w-full justify-center" :disabled="!canFinish"
              @click="finishAbsence">Завершить</UButton>
          </UContainer>
        </template>
      </USlideover>

      <USlideover v-model:open="editOpen" title="Редактировать отсутствие">
        <template #body>
          <UContainer class="space-y-4">
            <UAlert
              v-if="editingRecord && editingRecord.status === 'active'"
              color="warning"
              variant="subtle"
              icon="i-lucide-info"
              title="Незавершённая запись"
              description="Если вы укажете время окончания, запись станет «Завершено»."
            />

            <UContainer class="grid grid-cols-1 gap-3">
              <UFormField size="xl" label="Начало">
                <UInput class="w-full" v-model="editForm.startAt" type="datetime-local" />
              </UFormField>

              <UFormField size="xl" label="Окончание (опционально)">
                <UInput class="w-full" v-model="editForm.endAt" type="datetime-local" />
              </UFormField>

              <UFormField size="xl" label="Причина">
                <UTextarea class="w-full" v-model="editForm.reason" placeholder="Причина отсутствия" :rows="4" />
              </UFormField>
            </UContainer>

            <UAlert v-if="editError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="editError" />
          </UContainer>
        </template>

        <template #footer>
          <UContainer class="flex justify-between gap-3 items-center w-full">
            <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="editOpen = false">
              Отмена
            </UButton>
            <UButton color="primary" size="xl" class="w-full justify-center" :disabled="!canSaveEdit" @click="saveEdit">
              Сохранить
            </UButton>
          </UContainer>
        </template>
      </USlideover>
    </UContainer>
  </UMain>
</template>
