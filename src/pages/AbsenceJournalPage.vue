<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';

type JsonRow = Record<string, unknown>;

type CurrentUser = {
  id: string;
  fio: string;
  ofoId: string;
  seatId: string;
};

type AbsenceRecord = {
  id: string;
  fio: string;
  ofoTitle: string;
  seatTitle: string;
  createdAt: Date;
  startAt: Date;
  endAt: Date;
  reason: string;
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

const loading = ref(true);
const error = ref<string | null>(null);

const currentUser = ref<CurrentUser | null>(null);
const ofoTitleById = ref<Record<string, string>>({});
const seatTitleById = ref<Record<string, string>>({});

const startAbsenceAt = ref('');
const filterPeriod = ref<'all' | 'today' | 'week' | 'month'>('all');
const records = ref<AbsenceRecord[]>([]);

const periodOptions = [
  { label: 'За все время', value: 'all' },
  { label: 'За сегодня', value: 'today' },
  { label: 'За неделю', value: 'week' },
  { label: 'За месяц', value: 'month' },
];

const firstReason = 'Вышел в городок, настроить ИИКСР';

const canStartAbsence = computed(() => Boolean(startAbsenceAt.value) && Boolean(currentUser.value));

const currentOfoTitle = computed(() => {
  const user = currentUser.value;
  if (!user) return '—';
  return ofoTitleById.value[user.ofoId] || '—';
});

const currentSeatTitle = computed(() => {
  const user = currentUser.value;
  if (!user) return '—';
  return seatTitleById.value[user.seatId] || 'Инженер';
});

const filteredRecords = computed(() => {
  const now = new Date();
  return records.value.filter((r) => {
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

async function load() {
  loading.value = true;
  error.value = null;
  try {
    const [usersRes, ofoRes, seatsRes] = await Promise.all([
      fetch('/data/users.json', { cache: 'force-cache' }),
      fetch('/data/ofo.json', { cache: 'force-cache' }),
      fetch('/data/office_seats.json', { cache: 'force-cache' }),
    ]);

    if (!usersRes.ok) throw new Error(`Не удалось загрузить users.json (${usersRes.status})`);
    if (!ofoRes.ok) throw new Error(`Не удалось загрузить ofo.json (${ofoRes.status})`);
    if (!seatsRes.ok) throw new Error(`Не удалось загрузить office_seats.json (${seatsRes.status})`);

    const usersRaw = await usersRes.json();
    const ofoRaw = await ofoRes.json();
    const seatsRaw = await seatsRes.json();

    const users = extractTableData(usersRaw, 'users');
    const ofoRows = extractTableData(ofoRaw, 'ofo');
    const seatRows = extractTableData(seatsRaw, 'office_seats');

    const ofoMap: Record<string, string> = {};
    for (const row of ofoRows) {
      const id = asText(row.id);
      if (!id) continue;
      ofoMap[id] = asText(row.title) || '—';
    }
    ofoTitleById.value = ofoMap;

    const seatMap: Record<string, string> = {};
    for (const row of seatRows) {
      const id = asText(row.id);
      if (!id) continue;
      seatMap[id] = asText(row.title) || '—';
    }
    seatTitleById.value = seatMap;

    const firstActive = users.find((u) => asText(u.active) === '1');
    if (!firstActive) throw new Error('Не найден активный пользователь');

    const user: CurrentUser = {
      id: asText(firstActive.id),
      fio: asText(firstActive.fio) || '—',
      ofoId: asText(firstActive.ofo),
      seatId: asText(firstActive.pos),
    };
    currentUser.value = user;

    const baseCreated = new Date();
    baseCreated.setHours(10, 6, 0, 0);
    const baseStart = new Date(baseCreated);
    baseStart.setHours(12, 0, 0, 0);
    const baseEnd = new Date(baseCreated);
    baseEnd.setHours(16, 45, 0, 0);

    records.value = [
      {
        id: 'initial-1',
        fio: user.fio,
        ofoTitle: ofoMap[user.ofoId] || '—',
        seatTitle: seatMap[user.seatId] || 'Инженер',
        createdAt: baseCreated,
        startAt: baseStart,
        endAt: baseEnd,
        reason: firstReason,
      },
    ];
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки данных';
  } finally {
    loading.value = false;
  }
}

function startAbsence() {
  if (!canStartAbsence.value || !currentUser.value) return;
  const start = new Date(startAbsenceAt.value);
  if (Number.isNaN(start.getTime())) return;

  const end = new Date(start.getTime() + 4 * 60 * 60 * 1000);
  const now = new Date();

  const user = currentUser.value;
  const record: AbsenceRecord = {
    id: `abs-${now.getTime()}`,
    fio: user.fio,
    ofoTitle: ofoTitleById.value[user.ofoId] || '—',
    seatTitle: seatTitleById.value[user.seatId] || 'Инженер',
    createdAt: now,
    startAt: start,
    endAt: end,
    reason: firstReason,
  };

  records.value = [record, ...records.value];
}

function applyFilter() {
  // фильтр реактивный, кнопка нужна для UX в стиле старого интерфейса
}

onMounted(() => {
  void load();
});
</script>

<template>
  <UMain class="w-full min-h-0 overflow-y-auto">
    <div class="space-y-5 py-1">
      <h1 class="text-3xl font-semibold text-highlighted">Журнал отсутствия</h1>

      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-alert-triangle"
        title="Ошибка загрузки"
        :description="error"
      />

      <UCard>
        <template #header>
          <h2 class="text-xl font-semibold">Начать отсутствие</h2>
        </template>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <UFormField label="ФИО">
            <UInput :model-value="currentUser?.fio || ''" readonly :loading="loading" />
          </UFormField>
          <UFormField label="ОФО">
            <UInput :model-value="currentOfoTitle" readonly :loading="loading" />
          </UFormField>
          <UFormField label="Дата и время начала отсутствия">
            <UInput v-model="startAbsenceAt" type="datetime-local" :disabled="loading" />
          </UFormField>
          <UFormField label="Должность">
            <UInput :model-value="currentSeatTitle" readonly :loading="loading" />
          </UFormField>
        </div>

        <div class="mt-4">
          <UButton
            color="primary"
            variant="solid"
            :disabled="!canStartAbsence || loading"
            @click="startAbsence"
          >
            Начать отсутствие
          </UButton>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <h2 class="text-xl font-semibold">Фильтр по периоду</h2>
        </template>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-[240px_240px] md:items-end">
          <UFormField label="Период">
            <USelect v-model="filterPeriod" :items="periodOptions" value-key="value" />
          </UFormField>
          <UButton color="primary" variant="solid" @click="applyFilter">Применить</UButton>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <h2 class="text-xl font-semibold">Мои отсутствия</h2>
        </template>

        <div class="overflow-x-auto">
          <table class="w-full min-w-[860px] border-collapse text-sm">
            <thead>
              <tr class="border-b border-default text-left text-muted">
                <th class="py-2 pe-3 font-medium">ФИО</th>
                <th class="py-2 pe-3 font-medium">ОФО</th>
                <th class="py-2 pe-3 font-medium">Должность</th>
                <th class="py-2 pe-3 font-medium">Дата создания записи</th>
                <th class="py-2 pe-3 font-medium">Начало отсутствия</th>
                <th class="py-2 pe-3 font-medium">Конец отсутствия</th>
                <th class="py-2 pe-0 font-medium">Причина</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="7" class="py-4 text-muted">Загрузка...</td>
              </tr>
              <tr v-else-if="!filteredRecords.length">
                <td colspan="7" class="py-4 text-muted">Записей не найдено</td>
              </tr>
              <tr
                v-for="row in filteredRecords"
                :key="row.id"
                class="border-b border-default/60 align-top"
              >
                <td class="py-2 pe-3">{{ row.fio }}</td>
                <td class="py-2 pe-3">{{ row.ofoTitle }}</td>
                <td class="py-2 pe-3">{{ row.seatTitle }}</td>
                <td class="py-2 pe-3">{{ formatDateTime(row.createdAt) }}</td>
                <td class="py-2 pe-3">{{ formatDateTime(row.startAt) }}</td>
                <td class="py-2 pe-3">{{ formatDateTime(row.endAt) }}</td>
                <td class="py-2 pe-0">{{ row.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </UCard>
    </div>
  </UMain>
</template>
