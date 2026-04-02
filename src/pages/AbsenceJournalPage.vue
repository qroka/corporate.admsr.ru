<script setup lang="ts">
import { computed, h, onMounted, ref, resolveComponent, watch } from 'vue';
import type { TableColumn, TabsItem } from '@nuxt/ui';
import { setHasActiveAbsence } from '../stores/absenceJournal';
import { currentRole } from '../stores/role';
import { useAppToast } from '../composables/useAppToast';

type JsonRow = Record<string, unknown>;

type CurrentUser = {
  id: string;
  fio: string;
  ofoId: string;
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
  status: AbsenceStatus;
};

function extractTableData(raw: unknown, tableName: string): JsonRow[] {
  if (!Array.isArray(raw)) return [];
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue;
    const rec = item as { type?: string; name?: string; data?: unknown[] };
    if (rec.type === 'table' && rec.name === tableName && Array.isArray(rec.data)) {
      return rec.data as JsonRow[];
    }
  }
  return [];
}

function asText(v: unknown): string {
  return String(v ?? '').replace(/\t/g, '').trim();
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
const records = ref<AbsenceRecord[]>([]);

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

const isAdmin = computed(() => currentRole.value === 'admin');

const mySearchQuery = ref('');
const myStatusFilter = ref<'' | AbsenceStatus>('');

const statusFilterItems = [
  { value: '', label: 'Все статусы' },
  { value: 'active', label: 'Не завершено' },
  { value: 'completed', label: 'Завершено' },
];

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
  const id = currentUser.value?.id;
  if (!id) return [];
  return records.value.filter((r) => r.userId === id);
});

const filteredMyRecords = computed(() => {
  const now = new Date();
  let list = myRecords.value;

  if (myStatusFilter.value) {
    list = list.filter((r) => r.status === myStatusFilter.value);
  }

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

const USlideover = resolveComponent('USlideover');
const UBadge = resolveComponent('UBadge');
const UButton = resolveComponent('UButton');
const UDropdownMenu = resolveComponent('UDropdownMenu');

const { toast } = useAppToast();

async function load() {
  loading.value = true;
  error.value = null;
  try {
    const [usersRes, ofoRes] = await Promise.all([
      fetch('/data/users.json', { cache: 'force-cache' }),
      fetch('/data/ofo.json', { cache: 'force-cache' }),
    ]);

    if (!usersRes.ok) throw new Error(`Не удалось загрузить users.json (${usersRes.status})`);
    if (!ofoRes.ok) throw new Error(`Не удалось загрузить ofo.json (${ofoRes.status})`);

    const usersRaw = await usersRes.json();
    const ofoRaw = await ofoRes.json();

    const users = extractTableData(usersRaw, 'users');
    const ofoRows = extractTableData(ofoRaw, 'ofo');

    const ofoMap: Record<string, string> = {};
    for (const row of ofoRows) {
      const id = asText(row.id);
      if (!id) continue;
      ofoMap[id] = asText(row.title) || '—';
    }
    ofoTitleById.value = ofoMap;

    const firstActive = users.find((u) => asText(u.active) === '1');
    if (!firstActive) throw new Error('Не найден активный пользователь');

    const user: CurrentUser = {
      id: asText(firstActive.id),
      fio: asText(firstActive.fio) || '—',
      ofoId: asText(firstActive.ofo),
    };
    currentUser.value = user;

    const baseCreated = new Date();
    baseCreated.setHours(10, 6, 0, 0);
    const baseStart = new Date(baseCreated);
    baseStart.setHours(12, 0, 0, 0);
    const baseEnd = new Date(baseCreated);
    baseEnd.setHours(16, 45, 0, 0);

    const activeUsers = users.filter((u) => asText(u.active) === '1');
    const demoRecords: AbsenceRecord[] = activeUsers.slice(0, 24).map((u, idx) => {
      const uid = asText(u.id) || `u-${idx}`;
      const fio = asText(u.fio) || `Сотрудник #${idx + 1}`;
      const ofoId = asText(u.ofo);
      const ofoTitle = ofoMap[ofoId] || (ofoId ? `ОФО #${ofoId}` : '—');

      // spread demo records across last 10 days
      const createdAt = new Date(Date.now() - (idx % 10) * 24 * 60 * 60 * 1000);
      createdAt.setHours(9 + (idx % 6), 10 + (idx % 4) * 10, 0, 0);
      const startAt = new Date(createdAt);
      startAt.setHours(10 + (idx % 6), (idx % 4) * 15, 0, 0);
      const endAt = new Date(startAt.getTime() + (60 + (idx % 6) * 35) * 60 * 1000);

      return {
        id: `demo-${uid}-${createdAt.getTime()}`,
        userId: uid,
        fio,
        ofoId,
        ofoTitle,
        createdAt,
        startAt,
        endAt,
        reason: reasonPresets[idx % reasonPresets.length],
        status: 'completed',
      };
    });

    // Ensure the current user has at least one record (the first row in UI)
    demoRecords.unshift({
      id: 'initial-1',
      userId: user.id,
      fio: user.fio,
      ofoId: user.ofoId,
      ofoTitle: ofoMap[user.ofoId] || (user.ofoId ? `ОФО #${user.ofoId}` : '—'),
      createdAt: baseCreated,
      startAt: baseStart,
      endAt: baseEnd,
      reason: 'Работа вне офиса',
      status: 'completed',
    });

    records.value = demoRecords;
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
  return records.value.find((r) => r.id === id) ?? null;
});

const editingRecord = computed(() => {
  const id = editingId.value;
  if (!id) return null;
  return records.value.find((r) => r.id === id) ?? null;
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

function startAbsence() {
  if (!canStartAbsence.value) return;

  const start = parseDateTimeInputValue(startAbsenceAt.value);
  if (!start) {
    toast.add({
      title: 'Не указано начало',
      description: 'Выберите дату и время начала отсутствия.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    return;
  }

  const now = new Date();
  const u = currentUser.value;
  if (!u) {
    toast.add({
      title: 'Пользователь не определён',
      description: 'Перезагрузите страницу и попробуйте снова.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    return;
  }
  const record: AbsenceRecord = {
    id: `abs-${now.getTime()}`,
    userId: u.id,
    fio: u.fio,
    ofoId: u.ofoId,
    ofoTitle: ofoTitleById.value[u.ofoId] || (u.ofoId ? `ОФО #${u.ofoId}` : '—'),
    createdAt: now,
    startAt: start,
    endAt: null,
    reason: '',
    status: 'active',
  };

  records.value = [record, ...records.value];
  toast.add({
    title: 'Отсутствие начато',
    description: `Начало: ${formatDateTime(start)}.`,
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
}

function openEdit(record: AbsenceRecord) {
  editingId.value = record.id;
  editError.value = null;
  editForm.value.startAt = toLocalDateTimeInputValue(record.startAt);
  editForm.value.endAt = record.endAt ? toLocalDateTimeInputValue(record.endAt) : '';
  editForm.value.reason = record.reason ?? '';
  editOpen.value = true;
}

function saveEdit() {
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
  records.value = records.value.map((r) => {
    if (r.id !== rec.id) return r;
    return {
      ...r,
      startAt: start,
      endAt: end ?? null,
      reason,
      status: end ? 'completed' : 'active',
    };
  });

  editOpen.value = false;
  editingId.value = null;
  toast.add({
    title: 'Изменения сохранены',
    description: end ? `${formatDateTime(start)} → ${formatDateTime(end)}` : `Начало: ${formatDateTime(start)} (незавершено)`,
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
}

function finishAbsence() {
  const rec = finishingRecord.value;
  if (!rec) return;
  const end = parseDateTimeInputValue(finishForm.value.endAt);
  const reason = finishForm.value.reason.trim();
  if (!end) {
    finishError.value = 'Укажите время окончания.';
    toast.add({
      title: 'Не указано окончание',
      description: 'Укажите дату и время окончания отсутствия.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    return;
  }
  if (!reason) {
    finishError.value = 'Укажите причину.';
    toast.add({
      title: 'Не указана причина',
      description: 'Заполните причину отсутствия.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    return;
  }
  if (end.getTime() < rec.startAt.getTime()) {
    finishError.value = 'Окончание не может быть раньше начала.';
    toast.add({
      title: 'Некорректное время',
      description: 'Окончание отсутствия не может быть раньше начала.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    return;
  }

  finishError.value = null;
  records.value = records.value.map((r) => {
    if (r.id !== rec.id) return r;
    return {
      ...r,
      endAt: end,
      reason,
      status: 'completed',
    };
  });

  finishOpen.value = false;
  finishingId.value = null;
  toast.add({
    title: 'Отсутствие завершено',
    description: `${formatDateTime(rec.startAt)} → ${formatDateTime(end)} · ${reason}`,
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
}

function deleteDraft(recordId: string) {
  const row = records.value.find((r) => r.id === recordId);
  records.value = records.value.filter((r) => r.id !== recordId);
  toast.add({
    title: 'Запись удалена',
    description: row ? `Удалено: ${formatDateTime(row.startAt)}` : 'Запись удалена.',
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
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
      startLabel: formatDateTime(r.startAt),
      endLabel: end ? formatDateTime(end) : '—',
      durationLabel: formatDurationRu(durationMs),
    };
  }),
);

type AbsenceAdminRow = AbsenceRow & {
  fioLabel: string;
  ofoLabel: string;
};

const ofoFilter = ref<string>('_all');
const adminSearchQuery = ref('');
const adminStatusFilter = ref<'' | AbsenceStatus>('');

function resetMyFilters() {
  mySearchQuery.value = '';
  myStatusFilter.value = '';
  filterPeriod.value = 'all';
}

function resetAdminFilters() {
  adminSearchQuery.value = '';
  adminStatusFilter.value = '';
  ofoFilter.value = '_all';
}

const ofoItems = computed(() => {
  const base = [{ label: 'Все ОФО', value: '_all' }];
  const entries = Object.entries(ofoTitleById.value)
    .map(([id, title]) => ({ label: title ? `${title}` : `ОФО #${id}`, value: id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru'));
  return base.concat(entries);
});

const adminTableRows = computed<AbsenceAdminRow[]>(() => {
  let list = records.value;

  if (ofoFilter.value !== '_all') {
    list = list.filter((r) => r.ofoId === ofoFilter.value);
  }
  if (adminStatusFilter.value) {
    list = list.filter((r) => r.status === adminStatusFilter.value);
  }
  const q = adminSearchQuery.value.trim().toLowerCase();
  if (q) {
    list = list.filter((r) => recordHaystack(r).includes(q));
  }

  return list.map((r) => {
    const end = r.endAt;
    const durationMs = end ? diffMs(r.startAt, end) : diffMs(r.startAt, new Date());
    return {
      ...r,
      startLabel: formatDateTime(r.startAt),
      endLabel: end ? formatDateTime(end) : '—',
      durationLabel: formatDurationRu(durationMs),
      fioLabel: r.fio || '—',
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
    meta: { class: { th: 'min-w-[180px]', td: 'tabular-nums' } },
  },
  {
    accessorKey: 'endLabel',
    header: 'Конец',
    meta: { class: { th: 'min-w-[180px]', td: 'tabular-nums' } },
  },
  {
    accessorKey: 'durationLabel',
    header: 'Длительность',
    meta: { class: { th: 'w-[120px]', td: 'tabular-nums whitespace-nowrap' } },
  },
  {
    accessorKey: 'reason',
    header: 'Причина',
    meta: { class: { th: 'min-w-[280px]' } },
    cell: ({ row }) => {
      const r = row.original as AbsenceRow;
      return h('span', { class: r.reason ? 'text-default' : 'text-muted' }, r.reason || '—');
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
    accessorKey: 'fioLabel',
    header: 'Сотрудник',
    meta: { class: { th: 'min-w-[220px]', td: 'whitespace-nowrap' } },
  },
  {
    accessorKey: 'ofoLabel',
    header: 'ОФО',
    meta: { class: { th: 'min-w-[240px]' } },
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
        class: 'max-w-[min(340px,100%)] min-w-0 truncate',
        title: title ? `${id} — ${title}` : (id ? `ID: ${id}` : 'Не указано'),
      }, () => label);
    },
  },
  ...(columns as unknown as TableColumn<AbsenceAdminRow>[]),
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
                  {{ activeRecord ? 'Есть незавершённая запись — завершите её.' : 'Выберите дату и время начала, затем нажмите «Начать отсутствие»' }}
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
              Начать отсутствие
            </UButton>
          </UContainer>
        </UCard>


        <UContainer v-if="loading" class="py-6 text-sm text-muted">Загрузка...</UContainer>
        <UContainer v-else class="w-full max-w-none flex flex-col gap-4 h-full min-h-0">
          <UContainer class="flex flex-row gap-3 w-full max-w-none">
            <UInput v-model="mySearchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline"
              placeholder="Поиск по причине, дате, статусу…" class="w-full sm:flex-1 sm:min-w-[240px]" />
            <USelectMenu v-model="myStatusFilter" :items="statusFilterItems" size="xl" color="neutral"
              placeholder="Статус" class="w-full sm:w-52" :content="{ align: 'start', sideOffset: 8 }" />
            <USelectMenu v-model="filterPeriod" :items="periodOptions" value-key="value" label-key="label" size="xl"
              color="neutral" placeholder="Период" class="w-full sm:w-52"
              :content="{ align: 'start', sideOffset: 8 }" />
            <UButton color="neutral" variant="outline" size="xl" icon="i-lucide-rotate-ccw" class="shrink-0"
              @click="resetMyFilters">
              Сбросить
            </UButton>
          </UContainer>

          <UContainer v-if="!tableRows.length" class="py-4 text-sm text-muted">Записей не найдено</UContainer>
          <UScrollArea v-else class="flex-1 min-h-0 w-full rounded-lg border border-default" orientation="both"
            :ui="{ root: 'overflow-auto' }">
            <UTable :columns="columns" :data="tableRows" class="w-full h-full"
              :ui="{ th: 'px-4 sm:px-6', td: 'px-4 sm:px-6' }" />
          </UScrollArea>
        </UContainer>
      </UContainer>


      <!-- Отсутствия по ОФО (только таблица) -->
      <UContainer v-else class="w-full max-w-none flex flex-col gap-4 h-full min-h-0">
        <UContainer class="flex flex-row gap-3 w-full max-w-none">
          <UInput v-model="adminSearchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline"
            placeholder="Поиск по ФИО, ОФО, причине, дате…" class="w-full sm:flex-1 sm:min-w-[240px]" />
          <USelectMenu v-model="adminStatusFilter" :items="statusFilterItems" size="xl" color="neutral"
            placeholder="Статус" class="w-full sm:w-52" :content="{ align: 'start', sideOffset: 8 }" />
          <USelectMenu v-model="ofoFilter" :items="ofoItems" size="xl" color="neutral" placeholder="ОФО"
            class="w-full sm:w-64" value-key="value" label-key="label" :content="{ align: 'start', sideOffset: 8 }" />
          <UButton color="neutral" variant="outline" size="xl" icon="i-lucide-rotate-ccw" class="shrink-0"
            @click="resetAdminFilters">
            Сбросить
          </UButton>
        </UContainer>

        <UContainer v-if="!adminTableRows.length" class="py-4 text-sm text-muted">Записей не найдено</UContainer>
        <UScrollArea v-else class="flex-1 min-h-0 w-full rounded-lg border border-default" orientation="both"
          :ui="{ root: 'overflow-auto' }">
          <UTable :columns="adminColumns" :data="adminTableRows" class="w-full" />
        </UScrollArea>
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
