<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useBirthdayColleagues } from '../composables/useBirthdayColleagues';
import { useAppToast } from '../composables/useAppToast';

type BirthdayUserLike = {
  id: number;
  fio: string;
  avatar: string;
  positionTitle: string;
};

const { loading, error, ensureLoaded, byMonthDay } = useBirthdayColleagues();
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

const currentDate = ref(new Date());
const currentMonth = computed(() => currentDate.value.getMonth());
const currentYear = computed(() => currentDate.value.getFullYear());

function prevMonth() {
  currentDate.value = new Date(currentYear.value, currentMonth.value - 1, 1);
}

function nextMonth() {
  currentDate.value = new Date(currentYear.value, currentMonth.value + 1, 1);
}

const monthNames = [
  'Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
  'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'
];
const monthName = computed(() => monthNames[currentMonth.value]);

function monthDayKey(m: number, d: number): string {
  return `${m}-${d}`;
}

function birthdaysForDay(day: number | null): BirthdayUserLike[] {
  if (!day) return [];
  // currentMonth is 0-indexed, month in byMonthDay is 1-indexed
  return (byMonthDay.value[monthDayKey(currentMonth.value + 1, day)] ?? []) as BirthdayUserLike[];
}

function countBirthdaysOnDay(day: number | null): number {
  return birthdaysForDay(day).length;
}

const calendarGrid = computed(() => {
  const year = currentYear.value;
  const month = currentMonth.value;
  const firstDay = new Date(year, month, 1);
  const lastDay = new Date(year, month + 1, 0);
  const daysInMonth = lastDay.getDate();
  
  let startOffset = firstDay.getDay() - 1;
  if (startOffset === -1) startOffset = 6; // Воскресенье
  
  const weeks: Array<Array<number | null>> = [];
  let currentWeek: Array<number | null> = [];
  
  for (let i = 0; i < startOffset; i++) {
    currentWeek.push(null);
  }
  
  for (let d = 1; d <= daysInMonth; d++) {
    currentWeek.push(d);
    if (currentWeek.length === 7) {
      weeks.push(currentWeek);
      currentWeek = [];
    }
  }
  
  if (currentWeek.length > 0) {
    while (currentWeek.length < 7) {
      currentWeek.push(null);
    }
    weeks.push(currentWeek);
  }
  
  return weeks;
});

const weekDays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

const isToday = (day: number | null) => {
  if (!day) return false;
  const today = new Date();
  return today.getDate() === day && today.getMonth() === currentMonth.value && today.getFullYear() === currentYear.value;
};
</script>

<template>
  <UMain class="flex w-full min-h-0 flex-col h-full gap-6 p-px">

    <UPageHeader class="border-none p-0 w-full max-w-none">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Дни рождения коллег</h1>
        </template>
      </UPageHeader>
      <USkeleton v-if="loading" class="h-[520px] w-full rounded-2xl" />
      <template v-else>
        <div class="flex items-center justify-between bg-primary rounded-lg p-3">
          <h2 class="text-2xl font-semibold text-inverted capitalize">{{ monthName }}</h2>
          <div class="flex items-center gap-2">
            <UButton icon="i-lucide-chevron-left" variant="solid" color="neutral" @click="prevMonth" />
            <UButton icon="i-lucide-chevron-right" variant="solid" color="neutral" @click="nextMonth" />
          </div>
        </div>
        
        <table class="w-full h-full min-h-0 border border-default flex-1 table-fixed rounded-lg overflow-hidden">
          <thead>
            <tr>
              <th v-for="(wd, i) in weekDays" :key="wd" class="p-3 text-center text-sm font-medium text-muted border-default bg-default" :class="{'border-r': i !== 6}">
                {{ wd }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(week, wIdx) in calendarGrid" :key="wIdx">
              <td 
                v-for="(day, dIdx) in week" 
                :key="dIdx" 
                class="border border-default p-0 align-top transition-colors h-px"
                :class="[day ? 'bg-default/10 hover:bg-default/30' : 'bg-transparent']"
              >
                <div v-if="day" class="w-full h-full min-h-0 p-2 flex flex-col gap-2">
                  <div class="flex items-center justify-between">
                    <span 
                      class="text-sm font-semibold flex items-center justify-center size-7 rounded-full" 
                      :class="[
                        isToday(day) ? 'bg-primary text-white shadow-sm' : (countBirthdaysOnDay(day) ? 'text-primary' : 'text-toned')
                      ]"
                    >
                      {{ day }}
                    </span>
                  </div>
                  <div v-if="countBirthdaysOnDay(day)" class="mt-auto pb-1 flex justify-start">
                    <UAvatarGroup size="sm" class="justify-start">
                      <UTooltip
                        v-for="person in birthdaysForDay(day).slice(0, 3)"
                        :key="person.id"
                        arrow
                        :content="{ side: 'top' }"
                        :text="person.fio"
                      >
                        <UAvatar
                          :src="person.avatar"
                          :alt="person.fio"
                        />
                      </UTooltip>
                      
                      <UPopover
                        v-if="countBirthdaysOnDay(day) > 3"
                        mode="hover"
                        arrow
                        :content="{ side: 'top' }"
                      >
                        <UAvatar
                          :text="`+${countBirthdaysOnDay(day) - 3}`"
                          color="neutral"
                          class="ring-1 ring-default text-xs font-medium"
                        />
                        <template #content>
                          <div class="flex flex-col gap-1 py-1 px-3">
                            <span v-for="person in birthdaysForDay(day).slice(3)" :key="person.id" class="text-xs text-highlighted">
                              {{ person.fio }}
                            </span>
                          </div>
                        </template>
                      </UPopover>
                    </UAvatarGroup>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </template>
  </UMain>
</template>

<style scoped>
.hidden-border-spacing {
  border-spacing: 0;
}
</style>
