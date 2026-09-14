<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { useMediaQuery } from '@vueuse/core';
import { useRouter } from 'vue-router';
import type { DropdownMenuItem, TabsItem } from '@nuxt/ui';
import {
  CALENDAR_SOURCE_META,
  parseDateKey,
  toDateKey,
  useCalendarFeed,
  type CalendarItem,
  type CalendarSource,
} from '../composables/useCalendarFeed';
import { useAppToast } from '../composables/useAppToast';
import { useSectionAccess } from '../composables/useSectionAccess';
import { toCalendarDate } from '../utils/date';

type ViewMode = 'month' | 'week' | 'day';

const HOUR_HEIGHT = 56;
const DAY_HOURS = Array.from({ length: 24 }, (_, i) => i);
const dayScrollEl = ref<HTMLElement | null>(null);

const router = useRouter();
const { toast } = useAppToast();
const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const {
  loading,
  error,
  items,
  ensureLoaded,
  addLocalEntry,
  removeLocalEntry,
} = useCalendarFeed();
ensureLoaded();

const isLg = useMediaQuery('(min-width: 1024px)');
const isMd = useMediaQuery('(min-width: 768px)');

const viewMode = ref<ViewMode>('month');
const hideWeekends = ref(false);
const enabledSources = ref<Record<CalendarSource, boolean>>({
  event: true,
  meeting: true,
  birthday: true,
  learning: true,
  personal: true,
});

const cursor = ref(new Date());
const selectedDate = ref(new Date());
const dayDrawerOpen = ref(false);
const settingsOpen = ref(false);
const createOpen = ref(false);
const createKind = ref<'meeting' | 'personal'>('personal');

const createForm = ref({
  title: '',
  date: toCalendarDate(toDateKey(new Date())),
  timeStart: '09:00',
  timeEnd: '10:00',
  location: '',
});

watch(
  isMd,
  (md) => {
    if (!md && viewMode.value !== 'day') {
      viewMode.value = 'day';
    }
  },
  { immediate: true },
);

watch(error, (val) => {
  if (!val) return;
  toast.add({
    title: 'Не удалось загрузить календарь',
    description: String(val),
    color: 'error',
    icon: 'i-lucide-alert-circle',
  });
});

const monthNames = [
  'Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
  'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь',
] as const;

const monthNamesShort = [
  'янв', 'фев', 'мар', 'апр', 'май', 'июн',
  'июл', 'авг', 'сен', 'окт', 'ноя', 'дек',
] as const;

const monthNamesGenitive = [
  'января', 'февраля', 'марта', 'апреля', 'мая', 'июня',
  'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря',
] as const;

type MonthCell = { date: Date; inMonth: boolean };

const weekdayLong = [
  'Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота',
] as const;

const weekDaysFull = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'] as const;
const weekDays = computed(() =>
  hideWeekends.value ? weekDaysFull.slice(0, 5) : [...weekDaysFull],
);

const viewTabs = computed<TabsItem[]>(() => [
  { label: 'Месяц', value: 'month', disabled: !isMd.value },
  { label: 'Неделя', value: 'week', disabled: !isMd.value },
  { label: 'День', value: 'day' },
]);

const createMenuItems = computed<DropdownMenuItem[][]>(() => {
  const local: DropdownMenuItem[] = [
    {
      label: 'Встречу',
      icon: 'i-lucide-users',
      onSelect: () => openCreate('meeting'),
    },
    {
      label: 'Личное событие',
      icon: 'i-lucide-circle',
      onSelect: () => openCreate('personal'),
    },
  ];
  if (!canEditSection('events')) return [local];
  return [
    local,
    [
      {
        label: 'Мероприятие',
        icon: 'i-lucide-calendar',
        to: '/events',
      },
    ],
  ];
});

const sourceFilters = (Object.keys(CALENDAR_SOURCE_META) as CalendarSource[]).map((key) => ({
  key,
  ...CALENDAR_SOURCE_META[key],
}));

const filteredItems = computed(() =>
  items.value.filter((item) => enabledSources.value[item.source]),
);

const itemsByDate = computed(() => {
  const map: Record<string, CalendarItem[]> = {};
  for (const item of filteredItems.value) {
    if (!map[item.dateKey]) map[item.dateKey] = [];
    map[item.dateKey].push(item);
  }
  return map;
});

const cursorMonth = computed(() => cursor.value.getMonth());
const cursorYear = computed(() => cursor.value.getFullYear());
const monthTitle = computed(
  () => `${monthNames[cursorMonth.value]} ${cursorYear.value}`,
);

const selectedKey = computed(() => toDateKey(selectedDate.value));
const selectedDayItems = computed(() => itemsByDate.value[selectedKey.value] ?? []);

const selectedDayTitle = computed(() => {
  const d = selectedDate.value;
  const wd = weekdayLong[d.getDay()];
  return `${wd}, ${d.getDate()} ${monthNamesGenitive[d.getMonth()]}`;
});

const selectedDayCountLabel = computed(() => {
  const n = selectedDayItems.value.length;
  const mod10 = n % 10;
  const mod100 = n % 100;
  let word = 'событий';
  if (mod10 === 1 && mod100 !== 11) word = 'событие';
  else if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) word = 'события';
  return `${n} ${word}`;
});

function isSameDay(a: Date, b: Date) {
  return (
    a.getFullYear() === b.getFullYear()
    && a.getMonth() === b.getMonth()
    && a.getDate() === b.getDate()
  );
}

function isTodayDate(date: Date) {
  return isSameDay(date, new Date());
}

function isSelectedDate(date: Date) {
  return isSameDay(date, selectedDate.value);
}

function goToday() {
  const t = new Date();
  selectedDate.value = new Date(t.getFullYear(), t.getMonth(), t.getDate());
  if (viewMode.value === 'day' || viewMode.value === 'week') {
    cursor.value = new Date(t);
  } else {
    cursor.value = new Date(t.getFullYear(), t.getMonth(), 1);
  }
}

function prevPeriod() {
  if (viewMode.value === 'week') {
    const d = new Date(cursor.value);
    d.setDate(d.getDate() - 7);
    cursor.value = d;
    return;
  }
  if (viewMode.value === 'day') {
    const d = new Date(selectedDate.value);
    d.setDate(d.getDate() - 1);
    selectedDate.value = d;
    cursor.value = new Date(d);
    return;
  }
  cursor.value = new Date(cursorYear.value, cursorMonth.value - 1, 1);
}

function nextPeriod() {
  if (viewMode.value === 'week') {
    const d = new Date(cursor.value);
    d.setDate(d.getDate() + 7);
    cursor.value = d;
    return;
  }
  if (viewMode.value === 'day') {
    const d = new Date(selectedDate.value);
    d.setDate(d.getDate() + 1);
    selectedDate.value = d;
    cursor.value = new Date(d);
    return;
  }
  cursor.value = new Date(cursorYear.value, cursorMonth.value + 1, 1);
}

function selectDay(date: Date) {
  selectedDate.value = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  if (
    viewMode.value === 'month'
    && (date.getMonth() !== cursorMonth.value || date.getFullYear() !== cursorYear.value)
  ) {
    cursor.value = new Date(date.getFullYear(), date.getMonth(), 1);
  }
  if (viewMode.value === 'day') {
    cursor.value = new Date(selectedDate.value);
    dayDrawerOpen.value = false;
    return;
  }
  if (!isLg.value) dayDrawerOpen.value = true;
}

/** Сетка как в Nuxt Calendar Template: 6 недель, дни соседних месяцев. */
const calendarGrid = computed(() => {
  const year = cursorYear.value;
  const month = cursorMonth.value;
  const firstDay = new Date(year, month, 1);
  let startOffset = firstDay.getDay() - 1;
  if (startOffset === -1) startOffset = 6;

  const start = new Date(year, month, 1 - startOffset);
  const weeks: MonthCell[][] = [];
  const walk = new Date(start);

  for (let w = 0; w < 6; w++) {
    const week: MonthCell[] = [];
    for (let d = 0; d < 7; d++) {
      week.push({
        date: new Date(walk),
        inMonth: walk.getMonth() === month,
      });
      walk.setDate(walk.getDate() + 1);
    }
    weeks.push(hideWeekends.value ? week.slice(0, 5) : week);
  }
  return weeks;
});

function dayNumberLabel(cell: MonthCell) {
  const d = cell.date.getDate();
  // Подпись месяца при входе в другой месяц (как Oct 1 в шаблоне Nuxt)
  if (d === 1 && !cell.inMonth) {
    return `${monthNamesShort[cell.date.getMonth()]} ${d}`;
  }
  if (d === 1 && cell.inMonth) {
    return `${d}`;
  }
  return String(d);
}

function mondayOf(date: Date) {
  const d = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const day = d.getDay();
  const diff = day === 0 ? -6 : 1 - day;
  d.setDate(d.getDate() + diff);
  return d;
}

const weekDaysDates = computed(() => {
  const start = mondayOf(cursor.value);
  const days: Date[] = [];
  const count = hideWeekends.value ? 5 : 7;
  for (let i = 0; i < count; i++) {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    days.push(d);
  }
  return days;
});

const weekTitle = computed(() => {
  const days = weekDaysDates.value;
  if (!days.length) return monthTitle.value;
  const a = days[0];
  const b = days[days.length - 1];
  if (a.getMonth() === b.getMonth()) {
    return `${a.getDate()}–${b.getDate()} ${monthNamesGenitive[a.getMonth()]} ${a.getFullYear()}`;
  }
  return `${a.getDate()} ${monthNamesGenitive[a.getMonth()]} – ${b.getDate()} ${monthNamesGenitive[b.getMonth()]} ${b.getFullYear()}`;
});

const periodTitle = computed(() => {
  if (viewMode.value === 'week') return weekTitle.value;
  if (viewMode.value === 'day') return selectedDayTitle.value;
  return monthTitle.value;
});

function itemsForDate(date: Date): CalendarItem[] {
  return itemsByDate.value[toDateKey(date)] ?? [];
}

function parseTimeToMinutes(value?: string): number | null {
  if (!value) return null;
  const range = String(value).match(/(\d{1,2}):(\d{2})/);
  if (!range) return null;
  const h = Number(range[1]);
  const m = Number(range[2]);
  if (h > 23 || m > 59) return null;
  return h * 60 + m;
}

function itemStartMinutes(item: CalendarItem): number | null {
  return parseTimeToMinutes(item.timeStart) ?? parseTimeToMinutes(item.timeLabel);
}

function itemEndMinutes(item: CalendarItem, startMin: number): number {
  const end = parseTimeToMinutes(item.timeEnd);
  if (end != null && end > startMin) return end;
  return Math.min(startMin + 60, 24 * 60);
}

const dayAllDayItems = computed(() =>
  selectedDayItems.value.filter((item) => itemStartMinutes(item) == null),
);

function formatClock(minutes: number) {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function formatTimeRange(item: CalendarItem, startMin: number, endMin: number) {
  if (item.timeStart && item.timeEnd) return `${item.timeStart} – ${item.timeEnd}`;
  if (item.timeLabel?.includes('-') || item.timeLabel?.includes('–')) return item.timeLabel;
  return `${formatClock(startMin)} – ${formatClock(endMin)}`;
}

type DayTimedBlock = {
  item: CalendarItem;
  startMin: number;
  endMin: number;
  top: number;
  height: number;
  col: number;
  colCount: number;
  timeRange: string;
};

/** Раскладка пересечений как в Nuxt Calendar Template (колонки). */
const dayTimedLayout = computed((): DayTimedBlock[] => {
  const timed = selectedDayItems.value
    .map((item) => {
      const startMin = itemStartMinutes(item);
      if (startMin == null) return null;
      const endMin = itemEndMinutes(item, startMin);
      return { item, startMin, endMin };
    })
    .filter((x): x is NonNullable<typeof x> => x != null)
    .sort((a, b) => a.startMin - b.startMin || a.endMin - b.endMin);

  const colEnds: number[] = [];
  const withCols = timed.map((ev) => {
    let col = colEnds.findIndex((end) => end <= ev.startMin);
    if (col === -1) {
      col = colEnds.length;
      colEnds.push(ev.endMin);
    } else {
      colEnds[col] = ev.endMin;
    }
    return { ...ev, col };
  });

  return withCols.map((ev) => {
    const overlapping = withCols.filter(
      (o) => o.startMin < ev.endMin && o.endMin > ev.startMin,
    );
    const colCount = Math.max(...overlapping.map((o) => o.col), ev.col) + 1;
    return {
      item: ev.item,
      startMin: ev.startMin,
      endMin: ev.endMin,
      top: (ev.startMin / 60) * HOUR_HEIGHT,
      height: Math.max(((ev.endMin - ev.startMin) / 60) * HOUR_HEIGHT, 24),
      col: ev.col,
      colCount,
      timeRange: formatTimeRange(ev.item, ev.startMin, ev.endMin),
    };
  });
});

const dayHeaderWeekday = computed(
  () => weekDaysFull[(selectedDate.value.getDay() + 6) % 7],
);

const nowLineTop = computed(() => {
  if (!isTodayDate(selectedDate.value)) return null;
  const n = new Date();
  return ((n.getHours() * 60 + n.getMinutes()) / 60) * HOUR_HEIGHT;
});

async function scrollDayIntoView() {
  await nextTick();
  const el = dayScrollEl.value;
  if (!el) return;
  let targetHour = 7;
  if (isTodayDate(selectedDate.value)) {
    targetHour = Math.max(0, new Date().getHours() - 1);
  } else if (dayTimedLayout.value.length) {
    targetHour = Math.max(0, Math.floor(dayTimedLayout.value[0].startMin / 60) - 1);
  }
  el.scrollTop = targetHour * HOUR_HEIGHT;
}

watch(
  [viewMode, selectedDate],
  () => {
    if (viewMode.value === 'day') void scrollDayIntoView();
  },
  { flush: 'post' },
);

function formatHourLabel(hour: number) {
  return `${String(hour).padStart(2, '0')}:00`;
}

function dayBlockStyle(block: DayTimedBlock) {
  const leftPct = (block.col / block.colCount) * 100;
  const widthPct = (1 / block.colCount) * 100;
  return {
    top: `${block.top}px`,
    height: `${block.height}px`,
    left: `calc(${leftPct}% + 2px)`,
    width: `calc(${widthPct}% - 4px)`,
  };
}

const cellVisibleLimit = 3;

function chipTitle(item: CalendarItem) {
  if (item.source === 'birthday') {
    const parts = item.title.trim().split(/\s+/).filter(Boolean);
    if (parts.length >= 3) {
      return `${parts[0]} ${parts[1][0]}.${parts[2][0]}.`;
    }
    if (parts.length === 2) {
      return `${parts[0]} ${parts[1][0]}.`;
    }
    return item.title;
  }
  return item.title;
}

function chipTime(item: CalendarItem) {
  if (item.source === 'birthday') return '';
  return item.timeStart || item.timeLabel || '';
}

function openItem(item: CalendarItem) {
  if (item.source === 'birthday') {
    toast.add({
      title: 'Пока нельзя поздравить',
      description: 'Функция поздравления временно недоступна.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
    return;
  }
  if (item.href) {
    void router.push(item.href);
    return;
  }
  if (item.source === 'meeting' || item.source === 'personal') {
    toast.add({
      title: item.title,
      description: [item.timeLabel, item.location].filter(Boolean).join(' · ') || 'Локальное событие',
      color: 'neutral',
      icon: CALENDAR_SOURCE_META[item.source].icon,
    });
  }
}

function openCreate(kind: 'meeting' | 'personal') {
  createKind.value = kind;
  createForm.value = {
    title: '',
    date: toCalendarDate(selectedKey.value),
    timeStart: '09:00',
    timeEnd: '10:00',
    location: '',
  };
  createOpen.value = true;
}

function submitCreate() {
  const title = createForm.value.title.trim();
  if (!title) {
    toast.add({ title: 'Укажите название', color: 'warning', icon: 'i-lucide-alert-circle' });
    return;
  }
  const dateVal = createForm.value.date as { year?: number; month?: number; day?: number } | null;
  if (!dateVal?.year || !dateVal?.month || !dateVal?.day) {
    toast.add({ title: 'Укажите дату', color: 'warning', icon: 'i-lucide-alert-circle' });
    return;
  }
  const dateKey = `${dateVal.year}-${String(dateVal.month).padStart(2, '0')}-${String(dateVal.day).padStart(2, '0')}`;
  addLocalEntry({
    source: createKind.value,
    dateKey,
    title,
    timeStart: createForm.value.timeStart || undefined,
    timeEnd: createForm.value.timeEnd || undefined,
    location: createForm.value.location.trim() || undefined,
  });
  createOpen.value = false;
  selectDay(parseDateKey(dateKey));
  cursor.value = new Date(dateVal.year, dateVal.month - 1, 1);
  toast.add({
    title: createKind.value === 'meeting' ? 'Встреча добавлена' : 'Событие добавлено',
    color: 'success',
    icon: 'i-lucide-check',
  });
}

function deleteLocal(item: CalendarItem) {
  if (item.source !== 'meeting' && item.source !== 'personal') return;
  removeLocalEntry(item.id);
  toast.add({ title: 'Удалено', color: 'neutral', icon: 'i-lucide-trash-2' });
}

function openDayView() {
  viewMode.value = 'day';
  cursor.value = new Date(selectedDate.value);
  dayDrawerOpen.value = false;
}

const showInlinePanel = computed(
  () => isLg.value && (viewMode.value === 'month' || viewMode.value === 'week'),
);
</script>

<template>
  <UMain class="relative flex min-h-0 w-full flex-1 flex-col">
    <div class="mx-auto flex min-h-0 w-full max-w-[1600px] flex-1 flex-col gap-3 overflow-hidden p-px pb-4">
      <UPageHeader
        class="shrink-0"
        headline="Корпоративная жизнь"
        title="Календарь"
        description="Встречи, мероприятия, обучение и дни рождения коллег"
      >
          <template #links>
            <div class="flex items-center gap-2">
              <UDropdownMenu :items="createMenuItems">
                <UButton
                  label="Создать"
                  icon="i-lucide-plus"
                  trailing-icon="i-lucide-chevron-down"
                  color="primary"
                  size="md"
                />
              </UDropdownMenu>
              <UButton
                label="Настроить календари"
                icon="i-lucide-settings"
                color="neutral"
                variant="outline"
                size="md"
                class="hidden sm:inline-flex"
                @click="settingsOpen = true"
              />
              <UButton
                icon="i-lucide-settings"
                color="neutral"
                variant="outline"
                size="md"
                class="sm:hidden"
                aria-label="Настроить календари"
                @click="settingsOpen = true"
              />
            </div>
          </template>
        </UPageHeader>

        <div class="flex shrink-0 flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              label="Сегодня"
              color="neutral"
              variant="outline"
              size="md"
              @click="goToday"
            />
            <div class="flex items-center gap-1">
              <UButton
                icon="i-lucide-chevron-left"
                color="neutral"
                variant="ghost"
                size="md"
                aria-label="Назад"
                @click="prevPeriod"
              />
              <UButton
                icon="i-lucide-chevron-right"
                color="neutral"
                variant="ghost"
                size="md"
                aria-label="Вперёд"
                @click="nextPeriod"
              />
            </div>
            <h2 class="text-lg font-semibold text-highlighted min-w-40">
              {{ periodTitle }}
            </h2>
          </div>

          <UTabs
            v-model="viewMode"
            :items="viewTabs"
            variant="pill"
            color="primary"
            size="md"
            :content="false"
            class="w-full sm:w-auto"
          />
        </div>

        <USkeleton v-if="loading" class="min-h-0 w-full flex-1 rounded-xl" />

        <div
          v-else
          class="flex min-h-0 flex-1 flex-col gap-4 lg:flex-row lg:items-stretch"
        >
          <!-- Month — по мотивам Nuxt Calendar Template -->
          <div
            v-if="viewMode === 'month'"
            class="flex min-h-0 w-full flex-1 flex-col overflow-hidden rounded-xl border border-default bg-default/10"
          >
            <div
              class="grid shrink-0 border-b border-default"
              :style="{ gridTemplateColumns: `repeat(${weekDays.length}, minmax(0, 1fr))` }"
            >
              <div
                v-for="wd in weekDays"
                :key="wd"
                class="px-2 py-2 text-center text-xs font-medium text-muted"
              >
                {{ wd }}
              </div>
            </div>
            <div class="min-h-0 flex-1 grid grid-rows-6">
              <div
                v-for="(week, wIdx) in calendarGrid"
                :key="wIdx"
                class="grid border-b border-default last:border-b-0 min-h-0"
                :style="{ gridTemplateColumns: `repeat(${weekDays.length}, minmax(0, 1fr))` }"
              >
                <button
                  v-for="(cell, dIdx) in week"
                  :key="toDateKey(cell.date)"
                  type="button"
                  class="min-h-0 border-r border-default last:border-r-0 p-1 flex flex-col justify-start items-stretch gap-0.5 text-left transition-colors hover:bg-elevated/40"
                  :class="[
                    !cell.inMonth ? 'bg-muted/5' : '',
                    isSelectedDate(cell.date) ? 'ring-2 ring-inset ring-primary bg-primary/5' : '',
                  ]"
                  @click="selectDay(cell.date)"
                >
                  <div class="flex items-center justify-start px-0.5">
                    <span
                      class="inline-flex h-6 min-w-6 items-center justify-center rounded-full px-1 text-xs font-medium tabular-nums"
                      :class="[
                        isTodayDate(cell.date)
                          ? 'bg-primary text-inverted'
                          : cell.inMonth
                            ? 'text-highlighted'
                            : 'text-muted',
                      ]"
                    >
                      {{ dayNumberLabel(cell) }}
                    </span>
                  </div>
                  <div class="flex flex-col gap-0.5 min-h-0 overflow-hidden">
                    <div
                      v-for="item in itemsForDate(cell.date).slice(0, cellVisibleLimit)"
                      :key="item.id"
                      class="flex items-center gap-1 rounded px-1 py-0.5 text-[11px] leading-tight min-w-0"
                      :class="cell.inMonth ? 'bg-elevated/70' : 'bg-elevated/40 opacity-70'"
                    >
                      <span
                        class="size-1.5 rounded-full shrink-0"
                        :class="CALENDAR_SOURCE_META[item.source].barClass"
                      />
                      <span class="truncate min-w-0 text-highlighted">{{ chipTitle(item) }}</span>
                      <span
                        v-if="chipTime(item)"
                        class="shrink-0 tabular-nums text-muted ms-auto"
                      >
                        {{ chipTime(item) }}
                      </span>
                    </div>
                    <span
                      v-if="itemsForDate(cell.date).length > cellVisibleLimit"
                      class="text-[11px] text-muted px-1"
                    >
                      +{{ itemsForDate(cell.date).length - cellVisibleLimit }} ещё
                    </span>
                  </div>
                </button>
              </div>
            </div>
          </div>

          <!-- Week -->
          <div
            v-else-if="viewMode === 'week'"
            class="flex min-h-0 w-full flex-1 flex-col overflow-hidden rounded-xl border border-default bg-default/10"
          >
            <div
              class="grid border-b border-default bg-elevated/20"
              :style="{ gridTemplateColumns: `repeat(${weekDaysDates.length}, minmax(0, 1fr))` }"
            >
              <div
                v-for="date in weekDaysDates"
                :key="`h-${toDateKey(date)}`"
                class="px-2 py-2 text-center border-r border-default last:border-r-0"
              >
                <p class="text-xs text-muted">
                  {{ weekDaysFull[(date.getDay() + 6) % 7] }}
                </p>
                <p
                  class="mx-auto mt-1 inline-flex size-7 items-center justify-center rounded-full text-sm font-semibold tabular-nums"
                  :class="isTodayDate(date) ? 'bg-primary text-inverted' : 'text-highlighted'"
                >
                  {{ date.getDate() }}
                </p>
              </div>
            </div>
            <div
              class="grid min-h-0 flex-1 items-stretch"
              :style="{ gridTemplateColumns: `repeat(${weekDaysDates.length}, minmax(0, 1fr))` }"
            >
              <button
                v-for="date in weekDaysDates"
                :key="toDateKey(date)"
                type="button"
                class="flex h-full min-h-0 flex-col items-stretch justify-start gap-1 border-r border-default p-1.5 text-left align-top transition-colors last:border-r-0 hover:bg-elevated/40"
                :class="isSelectedDate(date) ? 'ring-2 ring-inset ring-primary bg-primary/5' : ''"
                @click="selectDay(date)"
              >
                <div
                  v-for="item in itemsForDate(date)"
                  :key="item.id"
                  class="flex items-center gap-1 rounded px-1.5 py-1 text-xs bg-elevated/70 min-w-0 w-full"
                >
                  <span
                    class="size-1.5 rounded-full shrink-0"
                    :class="CALENDAR_SOURCE_META[item.source].barClass"
                  />
                  <span class="truncate min-w-0">{{ chipTitle(item) }}</span>
                  <span v-if="chipTime(item)" class="shrink-0 tabular-nums text-muted ms-auto">
                    {{ chipTime(item) }}
                  </span>
                </div>
              </button>
            </div>
          </div>

          <!-- Day — как в Nuxt Calendar Template -->
          <div
            v-else
            class="flex min-h-0 w-full flex-1 flex-col overflow-hidden rounded-xl border border-default bg-default/10"
          >
            <div class="flex shrink-0 items-end gap-2 border-b border-default px-4 pb-2 pt-3">
              <span class="pb-2 text-sm text-muted">{{ dayHeaderWeekday }}</span>
              <span
                class="inline-flex size-10 items-center justify-center rounded-full text-lg font-semibold tabular-nums"
                :class="isTodayDate(selectedDate) ? 'bg-primary text-inverted' : 'text-highlighted'"
              >
                {{ selectedDate.getDate() }}
              </span>
            </div>

            <div class="grid min-h-10 shrink-0 grid-cols-[4.75rem_1fr] border-b border-default">
              <div class="self-center whitespace-nowrap px-2 py-2 text-[11px] text-muted">
                весь день
              </div>
              <div class="flex min-h-10 flex-nowrap items-center gap-1.5 overflow-x-auto border-l border-default px-2 py-1.5">
                <button
                  v-for="item in dayAllDayItems"
                  :key="item.id"
                  type="button"
                  class="relative inline-flex max-w-64 shrink-0 items-center gap-1.5 overflow-hidden rounded-md border px-2 py-1 pl-3 text-left text-xs hover:brightness-110"
                  :class="CALENDAR_SOURCE_META[item.source].softClass"
                  @click="openItem(item)"
                >
                  <span
                    class="absolute inset-y-0 left-0 w-1"
                    :class="CALENDAR_SOURCE_META[item.source].barClass"
                  />
                  <span class="truncate">{{ item.title }}</span>
                </button>
              </div>
            </div>

            <div
              ref="dayScrollEl"
              class="min-h-0 flex-1 overflow-y-auto"
            >
              <div
                class="relative pt-3"
                :style="{ height: `${DAY_HOURS.length * HOUR_HEIGHT + 12}px` }"
              >
                <div
                  v-for="hour in DAY_HOURS"
                  :key="hour"
                  class="absolute inset-x-0 grid grid-cols-[4.75rem_1fr] pointer-events-none"
                  :style="{ top: `${hour * HOUR_HEIGHT + 12}px`, height: `${HOUR_HEIGHT}px` }"
                >
                  <span class="pr-2 text-right text-[11px] tabular-nums text-muted -translate-y-2">
                    {{ formatHourLabel(hour) }}
                  </span>
                  <div class="border-t border-default/70" />
                </div>

                <div class="absolute bottom-0 left-[4.75rem] right-0 top-3">
                  <div
                    v-if="nowLineTop != null"
                    class="absolute inset-x-0 z-20 pointer-events-none flex items-center"
                    :style="{ top: `${nowLineTop}px` }"
                  >
                    <span class="size-2.5 -ml-1 rounded-full bg-primary shrink-0" />
                    <span class="h-px flex-1 bg-primary" />
                  </div>

                  <button
                    v-for="block in dayTimedLayout"
                    :key="block.item.id"
                    type="button"
                    class="absolute z-10 overflow-hidden rounded-md border text-left px-2 py-1 pl-3 hover:brightness-110 transition"
                    :class="CALENDAR_SOURCE_META[block.item.source].softClass"
                    :style="dayBlockStyle(block)"
                    :aria-label="`${block.item.title}, ${block.timeRange}`"
                    @click="openItem(block.item)"
                  >
                    <span
                      class="absolute inset-y-0 left-0 w-1"
                      :class="CALENDAR_SOURCE_META[block.item.source].barClass"
                    />
                    <p class="text-xs font-medium text-highlighted truncate leading-tight">
                      {{ block.item.title }}
                    </p>
                    <p class="text-[11px] text-muted truncate leading-tight">
                      {{ block.timeRange }}
                    </p>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Desktop day panel (месяц / неделя) -->
          <aside
            v-if="showInlinePanel"
            class="flex max-h-full w-full shrink-0 flex-col overflow-hidden rounded-xl border border-default bg-default/20 lg:w-90 xl:w-100"
          >
            <div class="p-4 border-b border-default">
              <h3 class="text-base font-semibold text-highlighted">{{ selectedDayTitle }}</h3>
              <p class="text-sm text-muted">{{ selectedDayCountLabel }}</p>
            </div>
            <div class="flex-1 overflow-y-auto p-3 flex flex-col gap-3">
              <UEmpty
                v-if="!selectedDayItems.length"
                variant="naked"
                icon="i-lucide-calendar"
                title="Нет событий"
                description="На этот день ничего не запланировано"
                class="py-8"
              />
              <article
                v-for="item in selectedDayItems"
                :key="item.id"
                class="rounded-xl border p-3 flex flex-col gap-3"
                :class="CALENDAR_SOURCE_META[item.source].softClass"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <p class="text-xs text-muted">
                      <template v-if="item.source === 'birthday'">День рождения</template>
                      <template v-else-if="item.timeLabel">{{ item.timeLabel }}</template>
                      <template v-else>{{ CALENDAR_SOURCE_META[item.source].label }}</template>
                    </p>
                    <h4 class="text-sm font-semibold text-highlighted mt-0.5">{{ item.title }}</h4>
                    <p v-if="item.location" class="text-xs text-muted mt-1 flex items-center gap-1">
                      <UIcon name="i-lucide-map-pin" class="size-3.5" />
                      {{ item.location }}
                    </p>
                    <p v-if="item.role" class="text-xs text-muted mt-1">{{ item.role }}</p>
                  </div>
                  <div class="flex items-center gap-1 shrink-0">
                    <UAvatar
                      v-if="item.avatar"
                      :src="item.avatar"
                      :alt="item.title"
                      size="md"
                    />
                    <UDropdownMenu
                      v-if="item.source === 'meeting' || item.source === 'personal'"
                      :items="[[{ label: 'Удалить', icon: 'i-lucide-trash-2', color: 'error', onSelect: () => deleteLocal(item) }]]"
                    >
                      <UButton
                        icon="i-lucide-ellipsis-vertical"
                        color="neutral"
                        variant="ghost"
                        size="xs"
                      />
                    </UDropdownMenu>
                  </div>
                </div>

                <UBadge
                  v-if="item.isJoined"
                  color="primary"
                  variant="subtle"
                  size="sm"
                  label="Вы участвуете"
                  class="self-start"
                />

                <div v-if="item.source === 'learning' && item.progress != null" class="flex flex-col gap-1">
                  <div class="flex justify-between text-xs text-muted">
                    <span>Прогресс</span>
                    <span>{{ item.progress }}%</span>
                  </div>
                  <UProgress :model-value="item.progress" size="sm" color="warning" />
                </div>

                <UButton
                  :label="item.actionLabel"
                  color="neutral"
                  variant="outline"
                  size="sm"
                  trailing-icon="i-lucide-chevron-right"
                  class="self-start"
                  @click="openItem(item)"
                />
              </article>
            </div>
            <div class="p-3 border-t border-default">
              <UButton
                label="Открыть день"
                color="neutral"
                variant="ghost"
                trailing-icon="i-lucide-arrow-right"
                block
                @click="openDayView"
              />
            </div>
          </aside>
        </div>

        <p class="flex shrink-0 items-center gap-2 text-xs text-muted">
          <UIcon name="i-lucide-info" class="size-3.5 shrink-0" />
          События обновляются автоматически из подключённых систем
        </p>
    </div>

    <!-- Medium screens: day drawer -->
    <USlideover
      v-model:open="dayDrawerOpen"
      side="right"
      :title="selectedDayTitle"
      :description="selectedDayCountLabel"
    >
      <template #body>
        <div class="flex flex-col gap-3 p-1">
          <UEmpty
            v-if="!selectedDayItems.length"
            variant="naked"
            icon="i-lucide-calendar"
            title="Нет событий"
            class="py-6"
          />
          <article
            v-for="item in selectedDayItems"
            :key="item.id"
            class="rounded-xl border p-3 flex flex-col gap-3"
            :class="CALENDAR_SOURCE_META[item.source].softClass"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="text-xs text-muted">
                  <template v-if="item.source === 'birthday'">День рождения</template>
                  <template v-else-if="item.timeLabel">{{ item.timeLabel }}</template>
                  <template v-else>{{ CALENDAR_SOURCE_META[item.source].label }}</template>
                </p>
                <h4 class="text-sm font-semibold text-highlighted mt-0.5">{{ item.title }}</h4>
                <p v-if="item.location" class="text-xs text-muted mt-1">{{ item.location }}</p>
              </div>
              <UAvatar v-if="item.avatar" :src="item.avatar" :alt="item.title" size="md" />
            </div>
            <UBadge
              v-if="item.isJoined"
              color="primary"
              variant="subtle"
              size="sm"
              label="Вы участвуете"
              class="self-start"
            />
            <div v-if="item.source === 'learning' && item.progress != null" class="flex flex-col gap-1">
              <UProgress :model-value="item.progress" size="sm" color="warning" />
            </div>
            <UButton
              :label="item.actionLabel"
              color="neutral"
              variant="outline"
              size="sm"
              trailing-icon="i-lucide-chevron-right"
              class="self-start"
              @click="openItem(item); dayDrawerOpen = false"
            />
          </article>
          <UButton
            label="Открыть день"
            color="neutral"
            variant="soft"
            trailing-icon="i-lucide-arrow-right"
            @click="openDayView"
          />
        </div>
      </template>
    </USlideover>

    <USlideover
      v-model:open="settingsOpen"
      side="right"
      title="Настроить календари"
      description="Выберите источники и отображение сетки"
    >
      <template #body>
        <div class="flex flex-col gap-6">
          <div class="flex flex-col gap-2">
            <p class="text-sm font-medium text-highlighted">Источники</p>
            <label
              v-for="src in sourceFilters"
              :key="src.key"
              class="flex items-center gap-3 rounded-lg border border-default p-3 cursor-pointer hover:bg-elevated/40"
            >
              <UCheckbox v-model="enabledSources[src.key]" />
              <span class="size-2.5 rounded-full shrink-0" :class="src.barClass" />
              <UIcon :name="src.icon" class="size-4 text-muted shrink-0" />
              <span class="text-sm text-highlighted">{{ src.label }}</span>
            </label>
          </div>
          <div class="flex flex-col gap-2">
            <p class="text-sm font-medium text-highlighted">Сетка</p>
            <label
              class="flex items-center justify-between gap-3 rounded-lg border border-default p-3 cursor-pointer hover:bg-elevated/40"
            >
              <span class="text-sm text-highlighted">Скрыть выходные</span>
              <USwitch v-model="hideWeekends" />
            </label>
          </div>
        </div>
      </template>
    </USlideover>

    <USlideover
      v-model:open="createOpen"
      side="right"
      :title="createKind === 'meeting' ? 'Новая встреча' : 'Личное событие'"
      description="Сохранится в вашем локальном календаре"
    >
      <template #body>
        <div class="flex flex-col gap-4">
          <UFormField label="Название" required>
            <UInput v-model="createForm.title" size="md" color="neutral" class="w-full" placeholder="Например, планёрка" />
          </UFormField>
          <UFormField label="Дата" required>
            <UInputDate v-model="createForm.date" size="md" color="neutral" class="w-full" />
          </UFormField>
          <div class="grid grid-cols-2 gap-3">
            <UFormField label="Начало">
              <UInput v-model="createForm.timeStart" size="md" color="neutral" placeholder="09:00" />
            </UFormField>
            <UFormField label="Конец">
              <UInput v-model="createForm.timeEnd" size="md" color="neutral" placeholder="10:00" />
            </UFormField>
          </div>
          <UFormField label="Место">
            <UInput v-model="createForm.location" size="md" color="neutral" class="w-full" placeholder="Переговорная" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" label="Отмена" @click="createOpen = false" />
          <UButton color="primary" label="Сохранить" @click="submitCreate" />
        </div>
      </template>
    </USlideover>
  </UMain>
</template>
