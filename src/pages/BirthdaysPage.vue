<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { CalendarDate, getLocalTimeZone, today } from '@internationalized/date';
import type { DateValue } from '@internationalized/date';
import { useBirthdayColleagues } from '../composables/useBirthdayColleagues';
import { useAppToast } from '../composables/useAppToast';

type BirthdayUserLike = {
  id: number;
  fio: string;
  avatar: string;
  positionTitle: string;
};

const tz = getLocalTimeZone();
const selectedDate = ref<DateValue | undefined>(today(tz));
const miniCalendarPlaceholder = ref<DateValue>(today(tz));
const monthCalendarPlaceholder = ref<DateValue>(today(tz));
const peopleSearch = ref('');

const { loading, error, ensureLoaded, byMonthDay, birthdayColleaguesCount } = useBirthdayColleagues();
ensureLoaded();

const { toast } = useAppToast();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить данные',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const viewModes = ['Day', 'Week', 'Month', 'Year'];
const activeMode = ref<'Day' | 'Week' | 'Month' | 'Year'>('Month');

function monthDayKey(m: number, d: number): string {
  return `${m}-${d}`;
}

function birthdaysForDay(day: { month: number; day: number }): BirthdayUserLike[] {
  return (byMonthDay.value[monthDayKey(day.month, day.day)] ?? []) as BirthdayUserLike[];
}

function countBirthdaysOnDay(day: { month: number; day: number }): number {
  return birthdaysForDay(day).length;
}

const currentMonthPeople = computed(() => {
  const p = monthCalendarPlaceholder.value as CalendarDate;
  if (!p || typeof p.month !== 'number') return [];

  const items: Array<{ user: BirthdayUserLike; day: number }> = [];
  for (let d = 1; d <= 31; d++) {
    const list = birthdaysForDay({ month: p.month, day: d });
    for (const u of list) items.push({ user: u, day: d });
  }

  const query = peopleSearch.value.trim().toLowerCase();
  const filtered = query
    ? items.filter(({ user }) => {
        const n = String(user.fio ?? '').toLowerCase();
        const r = String(user.positionTitle ?? '').toLowerCase();
        return n.includes(query) || r.includes(query);
      })
    : items;

  return filtered
    .sort((a, b) => a.day - b.day || a.user.fio.localeCompare(b.user.fio, 'ru'))
    .slice(0, 14);
});

function goToToday() {
  const t = today(tz);
  selectedDate.value = t;
  miniCalendarPlaceholder.value = t;
  monthCalendarPlaceholder.value = t;
}
</script>

<template>
  <UMain class="flex w-full min-h-0 flex-col gap-4">
    <UCard
      :ui="{ root: 'overflow-hidden rounded-3xl border border-default/60', body: 'p-0' }"
      class="w-full"
    >
      <div class="grid min-h-[78vh] grid-cols-1 lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside class="border-r border-default/50 bg-elevated/40 p-4 sm:p-5">
          <h1 class="text-2xl font-semibold text-highlighted">Calendar</h1>

          <UButton
            icon="i-lucide-plus"
            color="primary"
            variant="solid"
            size="lg"
            class="mt-4 w-full justify-center rounded-xl"
            @click="goToToday"
          >
            К сегодняшней дате
          </UButton>

          <div class="mt-4 rounded-2xl border border-default/50 bg-default/70 p-2">
            <USkeleton v-if="loading" class="h-56 w-full rounded-xl" />
            <UCalendar
              v-else
              v-model="selectedDate"
              v-model:placeholder="miniCalendarPlaceholder"
              :week-starts-on="1"
              size="sm"
              color="primary"
              variant="ghost"
              class="w-full"
            />
          </div>

          <div class="mt-5">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">People</h2>
            <UInput
              v-model="peopleSearch"
              icon="i-lucide-search"
              placeholder="Поиск сотрудника"
              size="sm"
              class="mt-2"
            />
          </div>

          <div class="mt-3 space-y-2">
            <USkeleton v-if="loading" class="h-40 w-full rounded-xl" />
            <template v-else>
              <div
                v-for="item in currentMonthPeople"
                :key="`${item.user.id}-${item.day}`"
                class="flex items-center gap-2 rounded-xl border border-default/50 bg-default/70 p-2"
              >
                <UAvatar :src="item.user.avatar" size="sm" />
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-highlighted">{{ item.user.fio }}</p>
                  <p class="truncate text-xs text-muted">{{ item.user.positionTitle }}</p>
                </div>
                <UBadge size="sm" color="primary" variant="soft">{{ item.day }}</UBadge>
              </div>
              <p v-if="!currentMonthPeople.length" class="text-sm text-muted">
                В этом месяце нет дней рождения
              </p>
            </template>
          </div>
        </aside>

        <section class="min-w-0 bg-default/30 p-4 sm:p-6">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 class="text-2xl font-semibold text-highlighted sm:text-3xl">Calendar</h2>
              <p class="text-sm text-muted">Всего в каталоге: {{ birthdayColleaguesCount }}</p>
            </div>
            <div class="flex items-center gap-1 rounded-xl bg-elevated p-1">
              <UButton
                v-for="mode in viewModes"
                :key="mode"
                size="sm"
                :variant="activeMode === mode ? 'solid' : 'ghost'"
                :color="activeMode === mode ? 'primary' : 'neutral'"
                class="rounded-lg"
                @click="activeMode = mode as typeof activeMode"
              >
                {{ mode }}
              </UButton>
            </div>
          </div>

          <UCard
            :ui="{ root: 'rounded-2xl border border-default/60 bg-elevated', body: 'p-2 sm:p-4' }"
          >
            <USkeleton v-if="loading" class="h-[520px] w-full rounded-2xl" />
            <UCalendar
              v-else
              v-model="selectedDate"
              v-model:placeholder="monthCalendarPlaceholder"
              :week-starts-on="1"
              size="lg"
              color="primary"
              variant="soft"
              class="w-full min-w-0 **:data-[slot=heading]:text-xl **:data-[slot=heading]:font-semibold **:data-[slot=cellTrigger]:min-h-18 sm:**:data-[slot=cellTrigger]:min-h-20"
            >
              <template #day="{ day }">
                <div class="flex h-full w-full flex-col items-start gap-1 p-1.5 text-left">
                  <span
                    class="text-sm font-semibold"
                    :class="countBirthdaysOnDay(day) ? 'text-primary' : 'text-toned'"
                  >
                    {{ day.day }}
                  </span>
                  <template v-if="countBirthdaysOnDay(day)">
                    <span
                      v-for="person in birthdaysForDay(day).slice(0, 2)"
                      :key="person.id"
                      class="w-full truncate rounded-md bg-primary/15 px-1.5 py-0.5 text-[10px] font-medium text-primary"
                    >
                      {{ person.fio }}
                    </span>
                    <span
                      v-if="countBirthdaysOnDay(day) > 2"
                      class="text-[10px] font-medium text-muted"
                    >
                      +{{ countBirthdaysOnDay(day) - 2 }} ещё
                    </span>
                  </template>
                </div>
              </template>
            </UCalendar>
          </UCard>
        </section>
      </div>
    </UCard>
  </UMain>
</template>
