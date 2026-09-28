<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import type { DropdownMenuItem } from '@nuxt/ui';
import * as XLSX from 'xlsx';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useOfoTree } from '../../../composables/useOfoTree';
import { fmtDuration } from '../../../composables/useTestStats';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';

const route = useRoute();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs();
const {
  categories,
  ensureLoaded: ensureOfo,
  rootUnitsOf,
  rootLabelOf,
} = useOfoTree();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
/** Раньше ошибка загрузки давала пустой список и надпись «Пока нет участников». */
const loadError = ref<string | null>(null);
const summary = ref<Record<string, number | null>>({});
/** Сколько строк запрашиваем за раз; остальное — через поиск/фильтр или выгрузку. */
const PAGE_LIMIT = 200;
const rows = ref<any[]>([]);
const selectedId = ref<number | null>(
  route.query.enrollmentId ? Number(route.query.enrollmentId) : null,
);
const participant = ref<any | null>(null);
const detailLoading = ref(false);
const resetOpen = ref(false);
const resetTarget = ref<any | null>(null);
const resetting = ref(false);
const exporting = ref(false);

const answersOpen = ref(false);
const answersTitle = ref('');
const answersLoading = ref(false);
const answersError = ref('');
const answersData = ref<any | null>(null);

const searchQuery = ref('');
/** '_all' = все ОФО; иначе id корневого подразделения (строка) */
const ofoFilter = ref<string>('_all');
/** '_all' = все статусы */
const statusFilter = ref<string>('_all');
let searchTimer: ReturnType<typeof setTimeout> | null = null;
let filtersReady = false;

const statusItems = [
  { label: 'Все статусы', value: '_all' },
  { label: 'Не начат', value: 'not_started' },
  { label: 'В процессе', value: 'in_progress' },
  { label: 'Завершён', value: 'completed' },
  { label: 'Не сдан', value: 'failed' },
  { label: 'Просрочен', value: 'overdue' },
  { label: 'Отменён', value: 'cancelled' },
];

/** Корневые ОФО (без родителя) — как верхний уровень вкладки ОФО на дашборде. */
const ofoItems = computed(() => {
  const items: { label: string; value: string }[] = [{ label: 'Все ОФО', value: '_all' }];
  const cats = [...categories.value].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id);
  for (const cat of cats) {
    for (const u of rootUnitsOf(cat.id)) {
      items.push({ label: u.name, value: String(u.id) });
    }
  }
  return items.sort((a, b) => {
    if (a.value === '_all') return -1;
    if (b.value === '_all') return 1;
    return a.label.localeCompare(b.label, 'ru');
  });
});

const hasActiveFilters = computed(
  () => Boolean(searchQuery.value.trim()) || ofoFilter.value !== '_all' || statusFilter.value !== '_all',
);

function displayOfoName(ofoId: number | null | undefined, fallback?: string | null) {
  const root = rootLabelOf(ofoId);
  if (root) return root;
  return fallback || null;
}

const detail = computed(() => {
  const p = participant.value;
  if (!p) return null;
  const enr = p.enrollment || {};
  const user = p.user || {};
  const selectedRow = rows.value.find((r) => r.id === selectedId.value);
  const ofoId = user.ofoId ?? selectedRow?.ofoId ?? null;
  return {
    fio: user.fio || selectedRow?.fio || 'Сотрудник',
    ofoName: displayOfoName(ofoId, user.ofoName || selectedRow?.ofoName || null),
    status: enr.status || selectedRow?.status,
    progressPercent: enr.progress?.percent ?? selectedRow?.progressPercent ?? 0,
    finalScore: enr.finalScore ?? p.completion?.finalScore ?? selectedRow?.finalScore ?? null,
    tests: Array.isArray(p.tests) ? p.tests : [],
    topics: Array.isArray(p.topics) ? p.topics : [],
  };
});

function testKindLabel(t: any) {
  if (t?.type === 'final') return 'Итоговый';
  return t?.topicTitle ? `Тема: ${t.topicTitle}` : 'Тест темы';
}

function testResultLabel(t: any) {
  if (t?.passed === true) return 'Сдан';
  if (t?.passed === false) return 'Не сдан';
  // Попытки модуля тестов завершаются статусом 'completed' (V2__tests_module.sql).
  if (t?.status === 'completed' || t?.status === 'finished') return 'Завершён';
  if (t?.status === 'expired') return 'Время вышло';
  if (t?.status && t.status !== 'not_started') return 'В процессе';
  return 'Не пройден';
}

function testResultColor(t: any): 'success' | 'error' | 'warning' | 'neutral' {
  if (t?.passed === true) return 'success';
  if (t?.passed === false) return 'error';
  if (t?.status === 'completed' || t?.status === 'finished') return 'success';
  if (t?.status && t.status !== 'not_started') return 'warning';
  return 'neutral';
}

function rowMenuItems(row: any): DropdownMenuItem[][] {
  return [
    [
      {
        label: 'Открыть карточку',
        icon: 'i-lucide-user',
        onSelect() {
          void openDetail(row.id);
        },
      },
      {
        label: 'Обнулить результат',
        icon: 'i-lucide-rotate-ccw',
        color: 'error' as const,
        onSelect() {
          resetTarget.value = row;
          resetOpen.value = true;
        },
      },
    ],
  ];
}

function statusLabel(status?: string | null) {
  const s = String(status || '');
  return statusItems.find((i) => i.value === s)?.label || s || '—';
}

function formatDate(v?: string | null) {
  if (!v) return '';
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return String(v);
  return d.toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function mapResultRow(r: any) {
  const enr = r.enrollment || r;
  const user = r.user || {};
  const ofoId = user.ofoId ?? (user.ofo != null && user.ofo !== '' && user.ofo !== '-1' ? Number(user.ofo) : null);
  return {
    ...r,
    id: enr.id ?? r.enrollmentId ?? r.id,
    fio: user.fio || r.fio || r.fullName || 'Сотрудник',
    login: user.login || r.login || '',
    ofoId: Number.isFinite(ofoId) ? ofoId : null,
    ofoName: displayOfoName(
      Number.isFinite(ofoId) ? ofoId : null,
      user.ofoName || r.ofoName || null,
    ),
    status: enr.status ?? r.status,
    progressPercent: enr.progress?.percent ?? r.progressPercent ?? 0,
    finalScore: enr.finalScore ?? r.finalScore,
    assignedAt: enr.assignedAt ?? r.assignedAt ?? null,
    deadlineAt: enr.deadlineAt ?? r.deadlineAt ?? null,
    completedAt: enr.completedAt ?? r.completedAt ?? null,
  };
}

async function exportResults() {
  exporting.value = true;
  try {
    await store.loadCourse(courseId.value);
    const data = (await store.loadResults({
      courseId: courseId.value,
      versionId: store.version.value?.id,
      q: searchQuery.value.trim() || undefined,
      ofoId: ofoFilter.value !== '_all' ? Number(ofoFilter.value) : undefined,
      status: statusFilter.value !== '_all' ? statusFilter.value : undefined,
      limit: 5000,
    })) as any;
    const list = (data?.items || []).map(mapResultRow);
    if (!list.length) {
      toast.add({
        title: 'Нечего выгружать',
        description: 'Нет участников по текущим фильтрам.',
        color: 'warning',
        icon: 'i-lucide-alert-triangle',
      });
      return;
    }

    const headers = [
      'ФИО',
      'ОФО',
      'Статус',
      'Прогресс %',
      'Итоговый балл',
      'Назначен',
      'Срок',
      'Завершён',
    ];
    const sheetData = [
      headers,
      ...list.map((r: any) => [
        r.fio,
        r.ofoName || '',
        statusLabel(r.status),
        r.progressPercent ?? 0,
        r.finalScore ?? '',
        formatDate(r.assignedAt),
        formatDate(r.deadlineAt),
        formatDate(r.completedAt),
      ]),
    ];

    const ws = XLSX.utils.aoa_to_sheet(sheetData);
    ws['!cols'] = [
      { wch: 32 },
      { wch: 28 },
      { wch: 14 },
      { wch: 12 },
      { wch: 14 },
      { wch: 18 },
      { wch: 18 },
      { wch: 18 },
    ];
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, 'Результаты');

    const courseTitle = (store.current.value?.title || 'курс').replace(/[\\/:*?"<>|]+/g, '_').trim();
    const stamp = new Date().toISOString().slice(0, 10);
    XLSX.writeFile(wb, `Результаты_${courseTitle}_${stamp}.xlsx`);
    toast.add({
      title: 'Выгрузка готова',
      description: `Строк: ${list.length}`,
      color: 'success',
      icon: 'i-lucide-download',
    });
  } catch (e: any) {
    toast.add({ title: 'Не удалось выгрузить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    exporting.value = false;
  }
}

async function load() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadCourse(courseId.value);
    const data = (await store.loadResults({
      courseId: courseId.value,
      versionId: store.version.value?.id,
      q: searchQuery.value.trim() || undefined,
      ofoId: ofoFilter.value !== '_all' ? Number(ofoFilter.value) : undefined,
      status: statusFilter.value !== '_all' ? statusFilter.value : undefined,
      limit: PAGE_LIMIT,
    })) as any;
    const agg = data?.aggregates || data?.summary || data?.stats || {};
    summary.value = {
      total: agg.total,
      not_started: agg.notStarted ?? agg.not_started,
      completed: agg.completed,
      in_progress: agg.inProgress ?? agg.in_progress,
      overdue: agg.overdue,
      avg_score: agg.avgScore ?? agg.avg_score,
    };
    rows.value = (data?.items || data?.rows || data?.participants || []).map(mapResultRow);
    if (selectedId.value && !rows.value.some((r) => r.id === selectedId.value)) {
      selectedId.value = null;
      participant.value = null;
    }
  } catch (e: any) {
    rows.value = [];
    loadError.value = e?.message || 'Не удалось загрузить отчёт';
  } finally {
    loading.value = false;
  }
}

/** Всего по фильтру больше, чем пришло строк, — говорим об этом, а не обрезаем молча. */
const truncated = computed(() => Number(summary.value.total ?? 0) > rows.value.length);

/** Карточки сводки: у каждой — число и доля от всех назначенных по текущему фильтру. */
const summaryCards = computed(() => {
  const total = Number(summary.value.total ?? rows.value.length);
  const card = (key: string, label: string, tone = '') => {
    const n = Number(summary.value[key] ?? 0);
    return { key, label, n, of: total, tone: n > 0 ? tone : '' };
  };
  return [
    card('not_started', 'Не начали'),
    card('in_progress', 'В процессе'),
    card('overdue', 'Просрочили', 'text-error'),
    card('completed', 'Завершили', 'text-success'),
  ];
});

onMounted(async () => {
  await ensureOfo();
  await load();
  filtersReady = true;
  if (selectedId.value) await openDetail(selectedId.value);
});

watch([ofoFilter, statusFilter], () => {
  if (!filtersReady) return;
  void load();
});

watch(searchQuery, () => {
  if (!filtersReady) return;
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    void load();
  }, 300);
});

function clearFilters() {
  searchQuery.value = '';
  ofoFilter.value = '_all';
  statusFilter.value = '_all';
}

async function openDetail(enrollmentId: number) {
  selectedId.value = enrollmentId;
  detailLoading.value = true;
  try {
    participant.value = await store.loadParticipant({ enrollmentId, courseId: courseId.value });
  } catch (e: any) {
    toast.add({ title: 'Не удалось открыть карточку', description: e?.message, color: 'error', icon: 'i-lucide-x' });
    participant.value = null;
  } finally {
    detailLoading.value = false;
  }
}

async function openTestAnswers(t: any) {
  if (!selectedId.value || !t?.attemptId) return;
  answersTitle.value = t.title || 'Ответы теста';
  answersData.value = null;
  answersError.value = '';
  answersOpen.value = true;
  answersLoading.value = true;
  try {
    answersData.value = await store.loadAttemptAnswers({
      enrollmentId: selectedId.value,
      attemptId: t.attemptId,
    });
  } catch (e: any) {
    answersError.value = e?.message || 'Не удалось загрузить ответы';
  } finally {
    answersLoading.value = false;
  }
}

async function confirmReset() {
  const row = resetTarget.value;
  if (!row?.id) return;
  resetting.value = true;
  try {
    await store.resetEnrollment(row.id);
    toast.add({
      title: 'Результат обнулён',
      description: `${row.fio || 'Участник'} может пройти курс заново.`,
      color: 'success',
      icon: 'i-lucide-check',
    });
    resetOpen.value = false;
    resetTarget.value = null;
    await load();
    if (selectedId.value === row.id) {
      await openDetail(row.id);
    }
  } catch (e: any) {
    toast.add({
      title: 'Не удалось обнулить',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    resetting.value = false;
  }
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
    <UPageHeader
      headline="Обучение"
      title="Результаты"
      :description="store.current.value?.title || undefined"
    >
      <template #links>
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-download"
          :loading="exporting"
          :disabled="loading || Boolean(loadError)"
          @click="exportResults"
        >
          Выгрузить в Excel
        </UButton>
      </template>
    </UPageHeader>

    <div class="min-w-0 w-full flex flex-col gap-4 flex-1">
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          placeholder="Поиск по ФИО или логину…"
          size="lg"
          :ui="{ base: 'w-full' }"
        />
        <USelectMenu
          v-model="ofoFilter"
          :items="ofoItems"
          value-key="value"
          label-key="label"
          placeholder="Все ОФО"
          size="lg"
          color="neutral"
          :search-input="{ placeholder: 'Найти ОФО…' }"
          class="w-full"
          :content="{ align: 'start', sideOffset: 8 }"
        />
        <USelectMenu
          v-model="statusFilter"
          :items="statusItems"
          value-key="value"
          label-key="label"
          placeholder="Все статусы"
          size="lg"
          color="neutral"
          :search-input="false"
          class="w-full"
          :content="{ align: 'start', sideOffset: 8 }"
        />
      </div>

      <div v-if="loading" class="flex flex-col gap-3" aria-busy="true" aria-label="Загрузка отчёта">
        <div class="grid grid-cols-2 md:grid-cols-5 gap-3">
          <USkeleton v-for="n in 5" :key="n" class="h-20 w-full rounded-panel" />
        </div>
        <USkeleton v-for="n in 4" :key="`r${n}`" class="h-16 w-full rounded-panel" />
      </div>

      <UAlert
        v-else-if="loadError"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Не удалось загрузить отчёт"
        :description="loadError"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="load">Повторить</UButton>
        </template>
      </UAlert>

      <template v-else>
        <!-- Сводка с долей от всех назначенных: «ложных 100%» не бывает -->
        <div class="grid grid-cols-2 md:grid-cols-5 gap-3">
          <div v-for="c in summaryCards" :key="c.key" class="rounded-panel bg-elevated p-3 min-w-0">
            <p class="text-xs text-muted">{{ c.label }}</p>
            <p class="text-xl font-semibold tabular-nums" :class="c.tone || 'text-highlighted'">
              {{ c.n }}<span class="text-sm font-normal text-muted"> из {{ c.of }}</span>
            </p>
          </div>
          <div class="rounded-panel bg-elevated p-3 min-w-0">
            <p class="text-xs text-muted">Средний балл</p>
            <p class="text-xl font-semibold tabular-nums text-highlighted">
              {{ summary.avg_score != null ? Math.round(Number(summary.avg_score)) : '—' }}
            </p>
          </div>
        </div>

        <p v-if="truncated && rows.length" class="text-sm text-muted -mt-1">
          Показаны последние {{ rows.length }} из {{ summary.total }}. Уточните поиск или фильтры —
          либо выгрузите всех в Excel.
        </p>

        <UEmpty
          v-if="!rows.length"
          icon="i-lucide-bar-chart-3"
          :title="hasActiveFilters ? 'Никого не найдено' : 'Пока нет участников'"
          :description="hasActiveFilters
            ? 'Измените поиск, ОФО или статус.'
            : 'Назначьте курс сотрудникам, чтобы видеть прогресс.'"
          class="py-10"
        >
          <template v-if="hasActiveFilters" #actions>
            <UButton color="neutral" variant="outline" icon="i-lucide-x" @click="clearFilters">
              Сбросить фильтры
            </UButton>
          </template>
        </UEmpty>

        <div v-else class="flex flex-col lg:flex-row gap-4 min-h-0 flex-1 min-w-0">
          <ul class="flex-1 min-w-0 overflow-auto flex flex-col gap-2 list-none m-0 p-0.5">
            <li
              v-for="row in rows"
              :key="row.id"
              class="rounded-panel ring-1 ring-inset ring-default flex items-center gap-1 pe-2 min-w-0 transition-colors hover:bg-elevated/50"
              :class="selectedId === row.id ? 'ring-primary bg-elevated/50' : ''"
            >
              <button
                type="button"
                class="flex-1 min-w-0 flex items-center gap-3 p-3 text-left rounded-panel focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                :aria-pressed="selectedId === row.id"
                @click="openDetail(row.id)"
              >
                <span class="flex-1 min-w-0 flex flex-col">
                  <span class="font-medium text-highlighted break-words">{{ row.fio }}</span>
                  <span v-if="row.ofoName" class="text-xs text-muted break-words mt-0.5">{{ row.ofoName }}</span>
                  <span class="flex items-center gap-2 mt-1 flex-wrap">
                    <CourseStatusBadge :status="row.status" />
                    <span class="text-xs text-muted tabular-nums">{{ row.progressPercent ?? 0 }}%</span>
                    <span v-if="row.deadlineAt" class="text-xs text-muted">до {{ formatDate(row.deadlineAt).slice(0, 10) }}</span>
                  </span>
                </span>
                <span class="text-sm text-muted tabular-nums shrink-0" :title="row.finalScore == null ? 'Итогового балла нет' : 'Итоговый балл'">
                  {{ row.finalScore == null ? '—' : row.finalScore }}
                </span>
              </button>
              <UTooltip text="Действия">
                <span class="inline-flex">
                  <UDropdownMenu
                    :items="rowMenuItems(row)"
                    :content="{ align: 'end' }"
                    @click.stop
                  >
                    <UButton
                      icon="i-lucide-ellipsis-vertical"
                      color="neutral"
                      variant="ghost"
                      square
                      size="sm"
                      aria-label="Действия"
                      @click.stop
                    />
                  </UDropdownMenu>
                </span>
              </UTooltip>
            </li>
          </ul>

          <aside v-if="selectedId" class="w-full lg:w-[26rem] shrink-0 rounded-xl ring-1 ring-default p-4 flex flex-col gap-3 min-w-0 overflow-y-auto max-h-[70vh] lg:max-h-none">
            <div class="flex items-center justify-between gap-2">
              <h2 class="text-lg font-medium">Участник</h2>
              <div class="flex items-center gap-1">
                <UTooltip v-if="detail" text="Действия">
                  <span class="inline-flex">
                    <UDropdownMenu
                     
                      :items="rowMenuItems({ id: selectedId, fio: detail.fio })"
                      :content="{ align: 'end' }"
                    >
                      <UButton
                        icon="i-lucide-ellipsis-vertical"
                        color="neutral"
                        variant="ghost"
                        square
                        size="sm"
                        aria-label="Действия"
                      />
                    </UDropdownMenu>
                  </span>
                </UTooltip>
                <UTooltip text="Закрыть карточку">
                  <UButton
                    color="neutral"
                    variant="ghost"
                    size="sm"
                    icon="i-lucide-x"
                    aria-label="Закрыть карточку"
                    @click="selectedId = null; participant = null" />
                </UTooltip>
              </div>
            </div>
            <USkeleton v-if="detailLoading" class="h-40 w-full rounded-lg" />
            <template v-else-if="detail">
              <div class="flex flex-col gap-1 min-w-0">
                <p class="font-medium break-words">{{ detail.fio }}</p>
                <p class="text-sm text-muted break-words">
                  ОФО: {{ detail.ofoName || '—' }}
                </p>
              </div>

              <div class="flex items-center gap-2 flex-wrap">
                <CourseStatusBadge :status="detail.status" />
                <span class="text-sm text-muted">Прогресс: {{ detail.progressPercent }}%</span>
              </div>

              <p v-if="detail.finalScore != null" class="text-sm">
                Итоговый балл: <span class="font-medium tabular-nums">{{ detail.finalScore }}</span>
              </p>

              <USeparator />

              <div class="flex flex-col gap-2 min-w-0">
                <h3 class="text-sm font-medium text-highlighted">Тесты</h3>
                <p class="text-xs text-dimmed">Нажмите на тест с попыткой, чтобы открыть ответы по вопросам</p>
                <UEmpty
                  v-if="!detail.tests.length"
                  icon="i-lucide-clipboard-list"
                  title="Тестов в курсе нет"
                  class="py-4"
                />
                <ul v-else class="flex flex-col gap-2 list-none m-0 p-0">
                  <li v-for="t in detail.tests" :key="t.courseTestLinkId">
                    <!-- С попыткой — кнопка (доступна с клавиатуры), без попытки — просто карточка -->
                    <component
                      :is="t.attemptId ? 'button' : 'div'"
                      :type="t.attemptId ? 'button' : undefined"
                      class="w-full text-left rounded-lg ring-1 ring-inset ring-default p-3 flex flex-col gap-1.5 min-w-0"
                      :class="t.attemptId ? 'hover:bg-elevated/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary' : ''"
                      :aria-label="t.attemptId ? `Ответы: ${t.title}` : undefined"
                      @click="t.attemptId ? openTestAnswers(t) : undefined"
                    >
                    <span class="flex items-start justify-between gap-2">
                      <span class="block min-w-0">
                        <span class="block text-sm font-medium text-highlighted break-words">{{ t.title }}</span>
                        <span class="block text-xs text-muted break-words">{{ testKindLabel(t) }}</span>
                      </span>
                      <span class="flex items-center gap-1 shrink-0">
                        <UBadge :color="testResultColor(t)" variant="subtle">
                          {{ testResultLabel(t) }}
                        </UBadge>
                        <UIcon
                          v-if="t.attemptId"
                          name="i-lucide-eye"
                          class="size-4 text-dimmed"
                          aria-hidden="true"
                        />
                      </span>
                    </span>
                    <span class="flex items-center gap-3 text-xs text-muted flex-wrap">
                      <span v-if="t.score != null" class="tabular-nums">Балл: {{ t.score }}</span>
                      <span v-if="t.attemptsCount" class="tabular-nums">Попыток: {{ t.attemptsCount }}</span>
                      <span v-else>Попыток не было</span>
                    </span>
                    </component>
                  </li>
                </ul>
              </div>

              <template v-if="detail.topics?.length">
                <USeparator />
                <div class="flex flex-col gap-2 min-w-0">
                  <h3 class="text-sm font-medium text-highlighted">Темы</h3>
                  <ul class="flex flex-col gap-1.5 list-none m-0 p-0">
                    <li
                      v-for="tp in detail.topics"
                      :key="tp.topicId"
                      class="flex items-center justify-between gap-2 text-sm min-w-0"
                    >
                      <span class="break-words min-w-0">{{ tp.title }}</span>
                      <CourseStatusBadge :status="tp.status" class="shrink-0" />
                    </li>
                  </ul>
                </div>
              </template>
            </template>
          </aside>
        </div>
      </template>
    </div>

    </div>

    <UModal
      v-model:open="answersOpen"
      :title="`Ответы — ${answersTitle}`"
      description=""
      :ui="{ content: 'max-w-2xl' }"
    >
      <template #body>
        <div class="max-h-[70vh] overflow-y-auto px-1 py-1 flex flex-col gap-3">
          <div v-if="answersLoading" class="flex flex-col gap-3" aria-busy="true" aria-label="Загрузка ответов">
            <USkeleton class="h-6 w-40 rounded-lg" />
            <USkeleton v-for="n in 3" :key="n" class="h-20 w-full rounded-xl" />
          </div>
          <div v-else-if="answersError" class="py-8 text-center text-error text-sm">{{ answersError }}</div>
          <template v-else-if="answersData">
            <div class="flex items-center gap-2 flex-wrap">
              <UBadge
                v-if="answersData.attempt?.score != null"
                :color="answersData.attempt.passed === false ? 'error' : 'success'"
                variant="subtle"
                class="tabular-nums"
              >
                {{ Math.round(Number(answersData.attempt.score)) }}%
              </UBadge>
              <UBadge
                v-if="answersData.attempt?.passed != null"
                :color="answersData.attempt.passed ? 'success' : 'error'"
                variant="subtle"
              >
                {{ answersData.attempt.passed ? 'Тест пройден' : 'Не пройден' }}
              </UBadge>
              <span
                v-if="answersData.attempt?.durationSec != null"
                class="text-xs text-dimmed"
              >
                время: {{ fmtDuration(Number(answersData.attempt.durationSec)) }}
              </span>
            </div>
            <div
              v-for="(a, i) in (answersData.answers || [])"
              :key="i"
              class="rounded-xl ring-1 p-3 flex flex-col gap-1"
              :class="a.isCorrect === false
                ? 'ring-error/40 bg-error/5'
                : (a.isCorrect === true ? 'ring-success/40 bg-success/5' : 'ring-default')"
            >
              <div class="flex items-center gap-2">
                <UIcon
                  v-if="a.isCorrect === true"
                  name="i-lucide-check-circle-2"
                  class="size-4 text-success shrink-0"
                  aria-hidden="true"
                />
                <UIcon
                  v-else-if="a.isCorrect === false"
                  name="i-lucide-x-circle"
                  class="size-4 text-error shrink-0"
                  aria-hidden="true"
                />
                <span class="text-sm text-highlighted">{{ i + 1 }}. {{ a.title || 'Без названия' }}</span>
              </div>
              <p class="text-sm text-muted pl-6">Ответ: {{ a.userAnswer }}</p>
              <p
                v-if="a.isCorrect === false && a.correctAnswer"
                class="text-sm text-success pl-6"
              >
                Правильно: {{ a.correctAnswer }}
              </p>
            </div>
            <p v-if="!(answersData.answers || []).length" class="text-sm text-muted py-4 text-center">
              Ответов в попытке нет.
            </p>
          </template>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="resetOpen"
      title="Обнулить результат?"
      description="Прогресс, материалы, попытки тестов и завершение будут удалены. Участник сможет пройти курс заново."
    >
      <template #body>
        <p class="text-sm text-muted">
          Участник:
          <span class="text-highlighted font-medium">{{ resetTarget?.fio || '—' }}</span>
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="resetOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-rotate-ccw" :loading="resetting" @click="confirmReset">
            Обнулить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
