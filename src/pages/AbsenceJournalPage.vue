<script setup lang="ts">
import { computed, h, onMounted, ref, resolveComponent, shallowRef, watch } from 'vue';
import type { TableColumn, TabsItem } from '@nuxt/ui';
import { Time } from '@internationalized/date';
import { setHasActiveAbsence } from '../stores/absenceJournal';
import { useSectionAccess } from '../composables/useSectionAccess';
import { useAppToast } from '../composables/useAppToast';
import { apiSessionFetch } from '../composables/useAuthSession';
import { toCalendarDate } from '../utils/date';
import { slideoverPopoverContent, slideoverSelectContent } from '../composables/slideoverFieldUi';

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
  if (Number.isNaN(dt.getTime())) return '—';
  const date = dt.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' });
  const time = dt.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${date}, ${time}`;
}

function formatDateTimeWrap(dt: Date): string {
  return formatDateTime(dt);
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
  const base = hours > 0 ? `${hours}ч ${minutes}мин` : `${minutes}мин`;
  return sign < 0 ? `−${base}` : base;
}

function formatDurationRuSpaced(ms: number): string {
  if (!Number.isFinite(ms)) return '—';
  const sign = ms < 0 ? -1 : 1;
  const abs = Math.abs(ms);
  const totalMinutes = Math.round(abs / 60000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  const base = hours > 0 ? `${hours} ч ${minutes} мин` : `${minutes} мин`;
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
const startDateValue = shallowRef<ReturnType<typeof toCalendarDate>>(null);
const startTimeValue = shallowRef<Time | null>(null);
const startHour = ref(0);
const startMinute = ref(0);
const startReason = ref('');
const filterPeriod = ref<'all' | 'today' | 'week' | 'month'>('all');
const mySortDesc = ref(true);
const historySearchOpen = ref(false);
const historyFilterOpen = ref(false);
const myRecordsStore = ref<AbsenceRecord[]>([]);
const adminRecordsStore = ref<AbsenceRecord[]>([]);

const TIME_STEP_MINUTES = 5;
let syncingStartTime = false;

const hourOptions = Array.from({ length: 24 }, (_, h) => ({
  label: two(h),
  value: h,
}));

const minuteOptions = Array.from({ length: 60 / TIME_STEP_MINUTES }, (_, i) => {
  const m = i * TIME_STEP_MINUTES;
  return { label: two(m), value: m };
});

function roundMinutesToStep(minutes: number, step = TIME_STEP_MINUTES): number {
  const rounded = Math.round(minutes / step) * step;
  if (rounded >= 60) return 60 - step;
  return Math.max(0, rounded);
}

function applyStartTimeParts(hour: number, minute: number) {
  const h = Math.min(23, Math.max(0, hour));
  const m = roundMinutesToStep(minute);
  startHour.value = h;
  startMinute.value = m;
  startTimeValue.value = new Time(h, m, 0);
}

function syncStartPartsFromCombined() {
  const d = parseDateTimeInputValue(startAbsenceAt.value) ?? new Date();
  startDateValue.value = toCalendarDate(d);
  syncingStartTime = true;
  applyStartTimeParts(d.getHours(), d.getMinutes());
  syncingStartTime = false;
}

function syncCombinedFromParts() {
  const datePart = startDateValue.value as { year?: number; month?: number; day?: number } | null;
  const timePart = startTimeValue.value;
  if (!datePart?.year || !datePart?.month || !datePart?.day || !timePart) return;
  startAbsenceAt.value =
    `${datePart.year}-${two(datePart.month)}-${two(datePart.day)}` +
    `T${two(timePart.hour)}:${two(timePart.minute)}`;
}

syncStartPartsFromCombined();

watch([startDateValue, startTimeValue], () => {
  syncCombinedFromParts();
});

watch(startTimeValue, (time) => {
  if (syncingStartTime || !time) return;
  syncingStartTime = true;
  startHour.value = time.hour;
  startMinute.value = roundMinutesToStep(time.minute);
  if (time.minute !== startMinute.value) {
    startTimeValue.value = new Time(time.hour, startMinute.value, 0);
  }
  syncingStartTime = false;
});

watch([startHour, startMinute], ([hour, minute]) => {
  if (syncingStartTime) return;
  syncingStartTime = true;
  startTimeValue.value = new Time(hour, roundMinutesToStep(minute), 0);
  syncingStartTime = false;
});

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
  'Энгельса 10',
  'Работа в архиве',
  'Совещание',
] as const;

// У пользователя должно быть заполнено ОФО (числовой id), иначе бэкенд вернёт «ofo обязателен»
const hasOfo = computed(() => {
  const v = (currentUser.value?.ofoId ?? '').trim();
  return /^[0-9]+$/.test(v) && Number(v) > 0;
});

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

const monthStats = computed(() => {
  const now = new Date();
  const y = now.getFullYear();
  const m = now.getMonth();
  const monthLabel = now.toLocaleDateString('ru-RU', { month: 'long' });
  const list = myRecordsStore.value.filter(
    (r) => r.startAt.getFullYear() === y && r.startAt.getMonth() === m,
  );
  let totalMs = 0;
  for (const r of list) {
    totalMs += r.endAt ? diffMs(r.startAt, r.endAt) : diffMs(r.startAt, now);
  }
  return {
    label: `За ${monthLabel}`,
    count: list.length,
    duration: formatDurationRuSpaced(totalMs),
  };
});

const canStartAbsence = computed(() =>
  Boolean(startAbsenceAt.value)
  && Boolean(startReason.value.trim())
  && Boolean(currentUser.value)
  && hasOfo.value
  && !activeRecord.value,
);

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
const UIcon = resolveComponent('UIcon');

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

function onPageScroll(e: Event) {
  if (adminTab.value === 'my') onMyScroll(e);
  else onAdminScroll(e);
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

  if (!hasOfo.value) {
    toast.add({ title: 'Не указано подразделение (ОФО)', description: 'Обратитесь к администратору — в вашем профиле не заполнено ОФО.', color: 'error', icon: 'i-lucide-alert-circle' });
    return;
  }

  const reason = startReason.value.trim();
  if (!reason) {
    toast.add({ title: 'Не указана причина', description: 'Укажите причину отсутствия.', color: 'error', icon: 'i-lucide-alert-circle' });
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
        reason,
      },
    });
    if (!data.success) throw new Error(data.message || (data as any).error || 'Ошибка создания записи');

    const newRecord = mapApiRecord(data.data as JsonRow, ofoTitleById.value);
    myRecordsStore.value = [newRecord, ...myRecordsStore.value];
    if (isAdmin.value) adminRecordsStore.value = [newRecord, ...adminRecordsStore.value];
    startReason.value = '';
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
  createdLabel: string;
};

const tableRows = computed<AbsenceRow[]>(() => {
  const rows = filteredMyRecords.value.map((r) => {
    const end = r.endAt;
    const durationMs = end ? diffMs(r.startAt, end) : diffMs(r.startAt, new Date());
    return {
      ...r,
      startLabel: formatDateTimeWrap(r.startAt),
      endLabel: end ? formatDateTimeWrap(end) : '—',
      durationLabel: formatDurationRu(durationMs),
      createdLabel: formatDateTimeWrap(r.createdAt),
    };
  });
  return mySortDesc.value
    ? rows
    : [...rows].reverse();
});

function headerWithIcon(icon: string, label: string) {
  return () =>
    h('div', { class: 'flex items-center gap-1.5 text-highlighted' }, [
      h(UIcon, { name: icon, class: 'size-4 shrink-0 text-muted' }),
      h('span', label),
    ]);
}

function exportMyHistory() {
  const rows = tableRows.value;
  if (!rows.length) {
    toast.add({
      title: 'Нечего выгружать',
      description: 'История отсутствия пуста.',
      color: 'neutral',
      icon: 'i-lucide-info',
    });
    return;
  }
  const header = ['Статус', 'Создание записи', 'Начало', 'Конец', 'Длительность', 'Причина'];
  const lines = rows.map((r) =>
    [
      recordStatusLabel(r.status),
      r.createdLabel,
      r.startLabel,
      r.endLabel,
      r.durationLabel,
      r.reason || '',
    ]
      .map((cell) => `"${String(cell).replace(/"/g, '""')}"`)
      .join(';'),
  );
  const bom = '\uFEFF';
  const blob = new Blob([bom + [header.join(';'), ...lines].join('\n')], {
    type: 'text/csv;charset=utf-8',
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `absence-history-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

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
        syncStartPartsFromCombined();
      },
    },
  ];
}

const columns: TableColumn<AbsenceRow>[] = [
  {
    accessorKey: 'status',
    header: headerWithIcon('i-lucide-settings-2', 'Статус'),
    meta: { class: { th: 'w-[160px]', td: 'whitespace-nowrap' } },
    cell: ({ row }) => {
      const s = row.getValue('status') as AbsenceStatus;
      return h(
        UBadge,
        {
          variant: 'subtle',
          color: recordStatusColor(s),
          leading: true,
          leadingIcon: s === 'active' ? 'i-lucide-timer' : 'i-lucide-check',
        },
        () => recordStatusLabel(s),
      );
    },
  },
  {
    accessorKey: 'createdLabel',
    header: headerWithIcon('i-lucide-pencil', 'Создание записи'),
    meta: { class: { th: 'min-w-[160px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'startLabel',
    header: headerWithIcon('i-lucide-calendar-clock', 'Начало'),
    meta: { class: { th: 'min-w-[160px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'endLabel',
    header: headerWithIcon('i-lucide-calendar-check-2', 'Конец'),
    meta: { class: { th: 'min-w-[160px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'durationLabel',
    header: headerWithIcon('i-lucide-timer', 'Длительность'),
    meta: { class: { th: 'w-[130px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'reason',
    header: headerWithIcon('i-lucide-message-circle', 'Причина'),
    meta: { class: { th: 'min-w-[200px]', td: 'whitespace-normal' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceRow;
      return h(
        'span',
        { class: [r.reason ? 'text-default' : 'text-muted', 'break-words'].join(' ') },
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

const tabItems = computed<TabsItem[]>(() => [
  { label: 'Мои отсутствия', value: 'my' },
  {
    label: 'Отсутствия подразделения',
    value: 'ofo',
    badge: isAdmin.value && adminRecordsStore.value.length
      ? String(adminRecordsStore.value.length)
      : undefined,
  },
]);

const adminTab = ref<'my' | 'ofo'>('my');

watch(adminTab, (tab) => {
  if (tab !== 'ofo' || !isAdmin.value) return;
  if (ofoFilter.value === '_none' && currentUser.value?.ofoId) {
    ofoFilter.value = currentUser.value.ofoId;
  }
});

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
  <UMain class="relative w-full h-full min-h-0">
    <div
      class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8"
      @scroll.passive="onPageScroll"
    >
      <UPageHeader
        headline="Сервисы"
        title="Журнал отсутствия"
        description="Отмечайте рабочие отсутствия и просматривайте историю"
      />

      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-alert-triangle"
        title="Ошибка загрузки"
        :description="error"
      />

      <UTabs
        v-model="adminTab"
        :items="tabItems"
        variant="link"
        color="primary"
        size="md"
        :content="false"
        class="w-full border-b border-default"
        :ui="{
          list: 'w-full gap-1',
          trigger: 'grow-0',
        }"
      />

      <!-- Мои отсутствия -->
      <div
        v-if="adminTab === 'my'"
        class="flex flex-col gap-6 w-full"
      >
        <UCard
          variant="soft"
          class="w-full rounded-panel"
          :ui="{
            root: 'rounded-panel bg-elevated ring-0 border-0 divide-y-0',
            body: 'flex flex-col gap-4 p-4 sm:p-5',
          }"
        >
          <div
            v-if="currentUser && !hasOfo"
            class="rounded-lg ring-1 ring-warning/40 bg-warning/5 p-3 text-sm text-warning flex items-start gap-2"
          >
            <UIcon name="i-lucide-alert-triangle" class="size-4 shrink-0 mt-0.5" />
            <span>
              У вас не указано подразделение (ОФО). Отметить отсутствие нельзя — обратитесь к администратору, чтобы заполнить ОФО в профиле.
            </span>
          </div>

          <div class="flex flex-col xl:flex-row xl:items-end gap-4 xl:gap-6">
            <div class="flex min-w-0 flex-1 flex-col gap-3">
              <div class="flex items-center gap-1">
                <h2 class="text-lg font-semibold text-highlighted">
                  {{ activeRecord ? 'Завершить отсутствие' : 'Начать отсутствие' }}
                </h2>
                <UTooltip
                  :text="activeRecord
                    ? 'Завершите текущую запись, чтобы начать новую'
                    : 'Укажите дату, время и причину начала отсутствия'"
                >
                  <UButton
                    type="button"
                    color="neutral"
                    variant="ghost"
                    size="md"
                    icon="i-lucide-info"
                    square
                    aria-label="Подсказка"
                  />
                </UTooltip>
              </div>

              <div class="flex flex-col lg:flex-row gap-2 lg:items-center">
                <template v-if="!activeRecord">
                  <UInputDate
                    v-model="startDateValue"
                    size="md"
                    color="neutral"
                    class="w-full lg:w-48"
                    :disabled="loading"
                  >
                    <template #trailing>
                      <UPopover :content="slideoverPopoverContent">
                        <UButton
                          color="neutral"
                          variant="link"
                          size="md"
                          icon="i-lucide-calendar"
                          aria-label="Выбрать дату"
                          class="px-0"
                          :disabled="loading"
                        />
                        <template #content>
                          <UCalendar v-model="startDateValue" class="p-2" />
                        </template>
                      </UPopover>
                    </template>
                  </UInputDate>
                  <UInputTime
                    v-model="startTimeValue"
                    size="md"
                    color="neutral"
                    :hour-cycle="24"
                    :step="{ minute: 5 }"
                    step-snapping
                    granularity="minute"
                    class="w-full lg:w-44"
                    :disabled="loading"
                  >
                    <template #trailing>
                      <UPopover :content="slideoverPopoverContent">
                        <UButton
                          color="neutral"
                          variant="link"
                          size="md"
                          icon="i-lucide-clock"
                          aria-label="Выбрать время"
                          class="px-0"
                          :disabled="loading"
                        />
                        <template #content>
                          <div class="p-3 w-64 flex flex-col gap-2">
                            <div class="flex items-end gap-2">
                              <UFormField label="Часы" class="min-w-0 flex-1">
                                <USelectMenu
                                  v-model="startHour"
                                  :items="hourOptions"
                                  :content="slideoverSelectContent"
                                  :search-input="false"
                                  value-key="value"
                                  label-key="label"
                                  size="md"
                                  color="neutral"
                                  class="w-full"
                                />
                              </UFormField>
                              <span class="pb-2.5 text-lg text-muted shrink-0" aria-hidden="true">:</span>
                              <UFormField label="Минуты" class="min-w-0 flex-1">
                                <USelectMenu
                                  v-model="startMinute"
                                  :items="minuteOptions"
                                  :content="slideoverSelectContent"
                                  :search-input="false"
                                  value-key="value"
                                  label-key="label"
                                  size="md"
                                  color="neutral"
                                  class="w-full"
                                />
                              </UFormField>
                            </div>
                          </div>
                        </template>
                      </UPopover>
                    </template>
                  </UInputTime>
                  <UInput
                    v-model="startReason"
                    color="neutral"
                    size="md"
                    placeholder="Причина отсутствия"
                    class="w-full min-w-0 flex-1"
                    :disabled="loading"
                    @keydown.enter="startAbsence"
                  />
                  <UButton
                    color="primary"
                    variant="solid"
                    size="md"
                    class="shrink-0 justify-center"
                    :disabled="!canStartAbsence || loading"
                    @click="startAbsence"
                  >
                    Начать отсутствие
                  </UButton>
                </template>
                <UButton
                  v-else
                  color="primary"
                  variant="solid"
                  size="md"
                  class="w-full lg:w-auto justify-center"
                  icon="i-lucide-check-circle-2"
                  @click="openFinish(activeRecord)"
                >
                  Завершить отсутствие
                </UButton>
              </div>
            </div>

            <div class="hidden xl:block w-px self-stretch bg-default shrink-0" aria-hidden="true" />

            <div class="flex flex-col gap-2 xl:min-w-[280px] xl:shrink-0">
              <p class="text-sm text-muted capitalize">{{ monthStats.label }}</p>
              <div class="flex flex-wrap gap-6">
                <div>
                  <p class="text-2xl font-semibold text-highlighted tabular-nums leading-none">
                    {{ monthStats.count }}
                  </p>
                  <p class="text-sm text-muted mt-1">отсутствия</p>
                </div>
                <div>
                  <p class="text-2xl font-semibold text-highlighted tabular-nums leading-none">
                    {{ monthStats.duration }}
                  </p>
                  <p class="text-sm text-muted mt-1">общее время отсутствия</p>
                </div>
              </div>
            </div>
          </div>
        </UCard>

        <section class="flex flex-col gap-4" aria-label="История отсутствия">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div class="flex items-baseline gap-2 min-w-0">
              <h2 class="text-lg font-semibold text-highlighted">История отсутствия</h2>
              <span class="text-sm text-muted shrink-0">
                {{ tableRows.length }}
                {{ tableRows.length === 1 ? 'запись' : tableRows.length > 1 && tableRows.length < 5 ? 'записи' : 'записей' }}
              </span>
            </div>

            <div class="flex flex-wrap items-center gap-1.5">
              <UTooltip text="Сортировка по дате">
                <UButton
                  type="button"
                  color="neutral"
                  variant="ghost"
                  size="md"
                  :icon="mySortDesc ? 'i-lucide-arrow-down-narrow-wide' : 'i-lucide-arrow-up-narrow-wide'"
                  square
                  aria-label="Сортировка"
                  @click="mySortDesc = !mySortDesc"
                />
              </UTooltip>

              <UPopover v-model:open="historyFilterOpen">
                <UButton
                  type="button"
                  color="neutral"
                  :variant="filterPeriod !== 'all' ? 'soft' : 'ghost'"
                  size="md"
                  icon="i-lucide-funnel"
                  square
                  aria-label="Фильтр периода"
                />
                <template #content>
                  <div class="p-3 w-56">
                    <USelectMenu
                      v-model="filterPeriod"
                      :items="periodOptions"
                      value-key="value"
                      label-key="label"
                      size="md"
                      color="neutral"
                      class="w-full"
                    />
                  </div>
                </template>
              </UPopover>

              <UPopover v-model:open="historySearchOpen">
                <UButton
                  type="button"
                  color="neutral"
                  :variant="mySearchQuery ? 'soft' : 'ghost'"
                  size="md"
                  icon="i-lucide-search"
                  square
                  aria-label="Поиск"
                />
                <template #content>
                  <div class="p-3 w-72">
                    <UInput
                      v-model="mySearchQuery"
                      icon="i-lucide-search"
                      size="md"
                      color="neutral"
                      placeholder="Поиск по причине, дате…"
                      class="w-full"
                    />
                  </div>
                </template>
              </UPopover>

              <UButton
                color="neutral"
                variant="solid"
                size="md"
                trailing-icon="i-lucide-download"
                @click="exportMyHistory"
              >
                Выгрузить историю
              </UButton>
            </div>
          </div>

          <p v-if="loading" class="text-sm text-muted py-4">Загрузка…</p>

          <UEmpty
            v-else-if="!tableRows.length"
            variant="naked"
            icon="i-lucide-calendar-off"
            title="Записей не найдено"
            description="Начните отсутствие — запись появится в истории."
            class="w-full py-10"
          />

          <div
            v-else
            class="w-full rounded-panel ring-1 ring-default overflow-x-auto"
          >
            <UTable
              :columns="columns"
              :data="tableRows"
              class="w-full"
              :ui="{
                root: 'bg-transparent',
                base: 'min-w-full',
                thead: 'bg-elevated',
                tbody: 'bg-transparent [&>tr]:bg-transparent',
                th: 'px-4 py-3 text-xs font-semibold bg-elevated',
                td: 'px-4 py-3 text-sm bg-transparent',
                tr: 'border-b border-default last:border-0 bg-transparent',
              }"
            />

            <div
              v-if="myLoadingMore || myHasMore"
              class="p-4 text-sm text-muted flex items-center justify-center gap-3"
            >
              <span v-if="myLoadingMore">Загрузка…</span>
              <UButton
                v-else
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-arrow-down"
                @click="loadMoreMy"
              >
                Загрузить ещё
              </UButton>
            </div>
          </div>
        </section>
      </div>

      <!-- Отсутствия подразделения -->
      <div
        v-else
        class="flex flex-col gap-4 w-full"
      >
        <template v-if="!isAdmin">
          <UEmpty
            variant="naked"
            icon="i-lucide-shield"
            title="Недостаточно прав"
            description="Просмотр отсутствий подразделения доступен сотрудникам с доступом к журналу."
            class="w-full py-10"
          />
        </template>

        <template v-else>
          <div class="flex flex-col sm:flex-row gap-3 w-full">
            <UInput
              v-model="adminSearchQuery"
              icon="i-lucide-search"
              size="xl"
              color="neutral"
              variant="outline"
              placeholder="Поиск по ФИО, причине, дате…"
              class="w-full sm:flex-1 sm:min-w-[240px]"
            />
            <USelectMenu
              v-model="ofoFilter"
              :items="ofoItems"
              size="xl"
              color="neutral"
              placeholder="Подразделение"
              class="w-full sm:w-72"
              value-key="value"
              label-key="label"
              :content="{ align: 'start', sideOffset: 8 }"
            />
            <UButton
              color="neutral"
              variant="outline"
              size="xl"
              icon="i-lucide-rotate-ccw"
              class="shrink-0"
              @click="resetAdminFilters"
            >
              Сбросить
            </UButton>
          </div>

          <UEmpty
            v-if="ofoFilter === '_none'"
            variant="naked"
            icon="i-lucide-building-2"
            title="Выберите подразделение"
            description="Укажите ОФО, чтобы посмотреть отсутствия сотрудников."
            class="w-full py-10"
          />

          <UEmpty
            v-else-if="adminInitialLoaded && !adminTableRows.length"
            variant="naked"
            icon="i-lucide-calendar-off"
            title="Записей не найдено"
            description="В выбранном подразделении пока нет завершённых отсутствий."
            class="w-full py-10"
          />

          <div
            v-else
            class="w-full rounded-panel ring-1 ring-default overflow-x-auto"
          >
            <UTable
              :columns="adminColumns"
              :data="adminTableRows"
              class="w-full"
              :ui="{
                root: 'bg-transparent',
                base: 'min-w-full',
                thead: 'bg-elevated',
                tbody: 'bg-transparent [&>tr]:bg-transparent',
                th: 'px-4 py-3 text-xs font-semibold bg-elevated',
                td: 'px-4 py-3 text-sm bg-transparent',
                tr: 'border-b border-default last:border-0 bg-transparent',
              }"
            />

            <div
              v-if="adminLoadingMore || adminHasMore"
              class="p-4 text-sm text-muted flex items-center justify-center gap-3"
            >
              <span v-if="adminLoadingMore">Загрузка…</span>
              <UButton
                v-else
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-arrow-down"
                @click="loadMoreAdmin"
              >
                Загрузить ещё
              </UButton>
            </div>
          </div>
        </template>
      </div>

      <USlideover v-model:open="finishOpen" title="Завершение отсутствия">
        <template #body>
          <UContainer class="space-y-4">
            <UAlert
              v-if="finishingRecord?.status === 'active'"
              color="warning"
              variant="subtle"
              icon="i-lucide-info"
              title="Запись не завершена"
              description="Пока вы не завершите отсутствие, оно будет отображаться как «Не завершено»."
            />

            <UContainer class="grid grid-cols-1 gap-3">
              <UFormField size="xl" label="Начало">
                <UInput
                  class="w-full"
                  :model-value="finishingRecord ? formatDateTime(finishingRecord.startAt) : ''"
                  readonly
                />
              </UFormField>

              <UFormField size="xl" label="Время окончания">
                <UInput class="w-full" v-model="finishForm.endAt" type="datetime-local" />
              </UFormField>

              <UFormField size="xl" label="Причина">
                <UTextarea
                  class="w-full"
                  v-model="finishForm.reason"
                  placeholder="Например: К врачу, работа вне офиса…"
                  :rows="4"
                />

                <UContainer class="mt-2 flex flex-wrap gap-2">
                  <UButton
                    v-for="p in reasonPresets"
                    :key="p"
                    color="neutral"
                    variant="soft"
                    size="md"
                    @click="finishForm.reason = p"
                  >
                    {{ p }}
                  </UButton>
                </UContainer>
              </UFormField>
            </UContainer>

            <UAlert
              v-if="finishError"
              color="error"
              variant="subtle"
              icon="i-lucide-alert-circle"
              :description="finishError"
            />

            <p v-if="finishingRecord" class="text-md text-muted">
              Длительность:
              <span class="tabular-nums">
                {{
                  formatDurationRu(
                    diffMs(
                      finishingRecord.startAt,
                      parseDateTimeInputValue(finishForm.endAt) || finishingRecord.startAt,
                    ),
                  )
                }}
              </span>
            </p>
          </UContainer>
        </template>

        <template #footer>
          <UContainer class="flex justify-between gap-3 items-center w-full">
            <UButton
              color="neutral"
              variant="outline"
              size="xl"
              class="w-full justify-center"
              @click="finishOpen = false"
            >
              Пока не завершать
            </UButton>
            <UButton
              color="primary"
              size="xl"
              class="w-full justify-center"
              :disabled="!canFinish"
              @click="finishAbsence"
            >
              Завершить
            </UButton>
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
                <UTextarea
                  class="w-full"
                  v-model="editForm.reason"
                  placeholder="Причина отсутствия"
                  :rows="4"
                />
              </UFormField>
            </UContainer>

            <UAlert
              v-if="editError"
              color="error"
              variant="subtle"
              icon="i-lucide-alert-circle"
              :description="editError"
            />
          </UContainer>
        </template>

        <template #footer>
          <UContainer class="flex justify-between gap-3 items-center w-full">
            <UButton
              color="neutral"
              variant="outline"
              size="xl"
              class="w-full justify-center"
              @click="editOpen = false"
            >
              Отмена
            </UButton>
            <UButton
              color="primary"
              size="xl"
              class="w-full justify-center"
              :disabled="!canSaveEdit"
              @click="saveEdit"
            >
              Сохранить
            </UButton>
          </UContainer>
        </template>
      </USlideover>
    </div>
  </UMain>
</template>
