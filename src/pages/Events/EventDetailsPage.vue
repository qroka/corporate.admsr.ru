<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';
import { useEventsData } from '../../composables/useEventsData';
import { useAppToast } from '../../composables/useAppToast';
import UContentSurround from '../../components/UContentSurround.vue';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
  link?: string;
};

const route = useRoute();
const router = useRouter();

const coverModules = (import.meta as any).glob('../../img/EventsWebp/*.webp', {
  eager: true,
  import: 'default',
});
const coverSrcs = Object.entries(coverModules)
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([, src]) => src as string);
function coverAt(index: number): string {
  return coverSrcs.length ? coverSrcs[index % coverSrcs.length] : '/src/img/Logo.svg';
}

const { loading, error, events, badges, ensureLoaded } = useEventsData();
ensureLoaded();
const eventId = computed(() => Number(route.params.id));
const event = computed<EventPost | null>(() => {
  const idx = events.value.findIndex((e) => e.id === eventId.value);
  const e = idx >= 0 ? events.value[idx] : null;
  if (!e) return null;
  return {
    id: e.id,
    title: e.title,
    description: e.description,
    date: e.date,
    badge: e.badge,
    to: `/events/${e.id}`,
    image: { src: coverAt(e.coverIndex ?? idx), alt: e.title },
  };
});

const sortedEvents = computed(() => {
  const list = events.value.slice();
  list.sort((a, b) => String(b.date ?? '').localeCompare(String(a.date ?? ''), 'ru-RU') || (b.id ?? 0) - (a.id ?? 0));
  return list;
});

const surround = computed(() => {
  const id = event.value?.id;
  if (!id) return { prev: null, next: null };
  const idx = sortedEvents.value.findIndex((x) => x.id === id);
  if (idx < 0) return { prev: null, next: null };

  const prev = idx > 0 ? sortedEvents.value[idx - 1] : null;
  const next = idx >= 0 && idx < sortedEvents.value.length - 1 ? sortedEvents.value[idx + 1] : null;
  const prefix = isKiosk.value ? '/kiosk/events/' : '/events/';

  return {
    prev: prev
      ? {
          title: prev.title,
          description: prev.description,
          to: `${prefix}${prev.id}`,
        }
      : null,
    next: next
      ? {
          title: next.title,
          description: next.description,
          to: `${prefix}${next.id}`,
        }
      : null,
  };
});

const badgeOptions = computed(() => badges.value.map((b) => ({ value: b, label: b })));

const isAdmin = computed(() => currentRole.value === 'admin');
const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));

const { toast } = useAppToast();

watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить мероприятие',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

// ─── RSVP (local demo) ────────────────────────────────────────────────────────
const RSVP_STORAGE_KEY = 'events-rsvp:v1';
const joinPulse = ref(0);

function getRsvpMap(): Record<string, boolean> {
  try {
    const raw = window.localStorage.getItem(RSVP_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Record<string, boolean>) : {};
  } catch {
    return {};
  }
}

function setRsvpMap(map: Record<string, boolean>) {
  try {
    window.localStorage.setItem(RSVP_STORAGE_KEY, JSON.stringify(map));
  } catch {
    // ignore
  }
}

const isJoined = computed(() => {
  // joinPulse is only here to make re-computation deterministic after toggle
  // eslint-disable-next-line @typescript-eslint/no-unused-expressions
  joinPulse.value;
  const id = event.value?.id;
  if (!id) return false;
  if (typeof window === 'undefined') return false;
  return !!getRsvpMap()[String(id)];
});

function toggleJoin() {
  const id = event.value?.id;
  if (!id) return;
  const wasJoined = isJoined.value;
  const map = getRsvpMap();
  const key = String(id);
  map[key] = !map[key];
  setRsvpMap(map);
  joinPulse.value++;

  toast.add({
    title: wasJoined ? 'Запись отменена' : 'Вы записались на мероприятие',
    description: wasJoined ? 'Вы больше не в списке участников (демо).' : 'Добавили вас в список участников (демо).',
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
}

// ─── Share / Calendar helpers ────────────────────────────────────────────────
async function copyEventLink() {
  if (typeof window === 'undefined') return;
  const url = window.location.href;
  try {
    await navigator.clipboard.writeText(url);
    toast.add({
      title: 'Ссылка скопирована',
      description: 'Можно отправлять или открывать на другом устройстве.',
      color: 'success',
      icon: 'i-lucide-circle-check',
    });
  } catch {
    toast.add({
      title: 'Не удалось скопировать ссылку',
      description: 'Браузер мог запретить доступ к буферу обмена.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  }
}

function parseEventDate(value: string): Date | null {
  if (!value) return null;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? null : d;
}

function toIcsDate(d: Date) {
  const pad = (n: number) => String(n).padStart(2, '0');
  return (
    d.getUTCFullYear() +
    pad(d.getUTCMonth() + 1) +
    pad(d.getUTCDate()) +
    'T' +
    pad(d.getUTCHours()) +
    pad(d.getUTCMinutes()) +
    pad(d.getUTCSeconds()) +
    'Z'
  );
}

function downloadIcs() {
  if (!event.value || typeof window === 'undefined') return;
  const title = (event.value.title ?? 'Мероприятие').replace(/\r?\n/g, ' ').trim();
  const desc = (event.value.description ?? '').replace(/\r?\n/g, ' ').trim();
  const dt = parseEventDate(String((event.value as any).date ?? '')) ?? new Date();
  const dtEnd = new Date(dt.getTime() + 60 * 60 * 1000);
  const uid = `event-${event.value.id ?? 'x'}@corporate.admsr.ru`;

  const ics = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//corporate.admsr.ru//Events//RU',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
    'BEGIN:VEVENT',
    `UID:${uid}`,
    `DTSTAMP:${toIcsDate(new Date())}`,
    `DTSTART:${toIcsDate(dt)}`,
    `DTEND:${toIcsDate(dtEnd)}`,
    `SUMMARY:${title}`,
    desc ? `DESCRIPTION:${desc}` : '',
    'END:VEVENT',
    'END:VCALENDAR',
  ]
    .filter(Boolean)
    .join('\r\n');

  const blob = new Blob([ics], { type: 'text/calendar;charset=utf-8' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = `${title || 'event'}.ics`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(a.href);

  toast.add({
    title: 'Файл календаря готов',
    description: 'Откройте .ics, чтобы добавить событие.',
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
}

// ─── Форма редактирования ─────────────────────────────────────────────────────
type EditFormState = {
  title: string;
  description: string;
  badge: string | null;
  date: string;
  image: string;
  link: string;
};

const editOpen = ref(false);
const editState = reactive<EditFormState>({
  title: '',
  description: '',
  badge: null,
  date: '',
  image: '',
  link: '',
});

type SingleDateValue = { value?: any } | any | null;
const editDateValue = ref<SingleDateValue>(null);
const editSubmitting = ref(false);
const editError = ref<string | null>(null);

function fillEditStateFromEvent() {
  if (!event.value) return;
  editState.title = event.value.title ?? '';
  editState.description = event.value.description ?? '';
  editState.badge = (event.value as any).badge ?? null;
  editState.date = (event.value as any).date ?? '';
  editState.image = (event.value as any).image ?? '';
  editState.link = (event.value as any).link ?? '#';
}

// Заполняем форму как только загрузится мероприятие
watch(event, (val) => {
  if (val) fillEditStateFromEvent();
});

function openEdit() {
  fillEditStateFromEvent();
  editOpen.value = true;
}

watch(editDateValue, (val) => {
  const d = val?.value ?? val;
  editState.date = (d && typeof d.toString === 'function') ? d.toString() : '';
});

function validateEdit(): boolean {
  if (!editState.title.trim()) {
    editError.value = 'Заполните название мероприятия.';
    return false;
  }
  if (!editState.date) {
    editError.value = 'Выберите дату проведения.';
    return false;
  }
  editError.value = null;
  return true;
}

async function handleEditSubmit() {
  if (!validateEdit() || !event.value) return;
  editSubmitting.value = true;
  editError.value = null;
  try {
    const idx = events.value.findIndex((e) => e.id === (event.value?.id ?? -1));
    if (idx === -1) {
      editError.value = 'Мероприятие не найдено.';
      return;
    }

    const updated = {
      ...events.value[idx],
      title: editState.title,
      description: editState.description,
      badge: editState.badge ?? undefined,
      date: editState.date,
    };
    events.value = events.value.map((e) => (e.id === updated.id ? updated : e));
    editOpen.value = false;
  } catch (e: any) {
    editError.value = e?.message ?? 'Ошибка при сохранении';
  } finally {
    editSubmitting.value = false;
  }
}

// ─── Удаление ─────────────────────────────────────────────────────────────────
const deleteConfirmOpen = ref(false);
const deleteSubmitting = ref(false);
const deleteError = ref<string | null>(null);

watch(editError, (val) => {
  if (!val) return;
  toast.add({
    title: 'Не удалось сохранить',
    description: String(val),
    color: 'error',
    icon: 'i-lucide-alert-circle',
  });
});

watch(deleteError, (val) => {
  if (!val) return;
  toast.add({
    title: 'Не удалось удалить',
    description: String(val),
    color: 'error',
    icon: 'i-lucide-alert-circle',
  });
});

async function handleDelete() {
  if (!event.value) return;
  deleteSubmitting.value = true;
  deleteError.value = null;
  try {
    events.value = events.value.filter((e) => e.id !== event.value?.id);
    deleteConfirmOpen.value = false;
    router.push({ name: 'events' });
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Ошибка при удалении';
  } finally {
    deleteSubmitting.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-0">
    <UContainer
      class="flex-1 min-h-0 overflow-y-auto max-w-full w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 scrollbar-hide mx-0 pb-8">

      <!-- Скелетон -->
      <div v-if="loading" class="space-y-6">
        <USkeleton class="h-56 sm:h-72 rounded-3xl" />
        <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-6 items-start">
          <USkeleton class="h-64 rounded-3xl" />
          <USkeleton class="h-64 rounded-3xl" />
        </div>
      </div>

      <!-- Контент -->
      <div v-else-if="event" class="flex flex-col gap-6">
        <!-- Hero -->
        <div class="relative overflow-hidden rounded-3xl ring ring-default bg-default">
          <div class="relative h-96">
            <img :src="(event.image as any)?.src" :alt="(event.image as any)?.alt ?? event.title"
              class="absolute inset-0 h-full w-full object-cover" />
            <div class="absolute inset-0 bg-linear-to-t from-black/65 via-black/25 to-transparent" />
            <div class="absolute inset-0 p-6 flex flex-col justify-end">
              <div class="flex flex-wrap items-center gap-2 mb-3">
                <UBadge v-if="event.badge" color="primary" variant="solid" size="md">
                  {{ event.badge }}
                </UBadge>
                <UBadge color="neutral" variant="soft" size="md" class="backdrop-blur">
                  {{ event.date }}
                </UBadge>
              </div>
              <h1 class="text-2xl sm:text-4xl font-semibold tracking-tight text-white">
                {{ event.title }}
              </h1>
              <p v-if="event.description" class="mt-2 text-sm sm:text-base text-white/85 max-w-3xl">
                {{ event.description }}
              </p>
            </div>
          </div>
        </div>

        <!-- Body -->
        <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-6 items-start p-px">
          <UCard class="rounded-3xl">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <UIcon name="i-lucide-info" class="size-5 text-primary" />
                  <span class="text-sm font-semibold text-highlighted">О мероприятии</span>
                </div>
              </div>
            </template>

            <div class="space-y-4">
              <p v-if="event.description" class="text-base leading-6 text-muted">
                {{ event.description }}
              </p>
              <p v-else class="text-sm text-muted">
                Описание мероприятия пока не добавлено.
              </p>

              <div class="flex flex-wrap gap-2 pt-1">
                <UButton :color="isJoined ? 'neutral' : 'primary'" :variant="isJoined ? 'soft' : 'solid'" size="lg"
                  :icon="isJoined ? 'i-lucide-check' : 'i-lucide-ticket'" @click="toggleJoin">
                  {{ isJoined ? 'Вы записаны' : 'Записаться' }}
                </UButton>
                <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-link" @click="copyEventLink">
                  Скопировать ссылку
                </UButton>
              </div>
            </div>
          </UCard>

          <div class="space-y-4">
            <UCard class="rounded-3xl">
              <template #header>
                <div class="flex items-center gap-2">
                  <UIcon name="i-lucide-calendar-days" class="size-5 text-primary" />
                  <span class="text-sm font-semibold text-highlighted">Детали</span>
                </div>
              </template>

              <dl class="grid grid-cols-1 gap-3">
                <div class="flex items-start justify-between gap-3">
                  <dt class="text-sm text-muted">Дата</dt>
                  <dd class="text-sm font-medium text-highlighted text-right">
                    {{ event.date }}
                  </dd>
                </div>
                <div v-if="event.badge" class="flex items-start justify-between gap-3">
                  <dt class="text-sm text-muted">Категория</dt>
                  <dd class="text-sm font-medium text-highlighted text-right">
                    {{ event.badge }}
                  </dd>
                </div>
                <div class="flex items-start justify-between gap-3">
                  <dt class="text-sm text-muted">Статус</dt>
                  <dd class="text-sm font-medium text-highlighted text-right">
                    {{ isJoined ? 'Вы записаны' : 'Не записаны' }}
                  </dd>
                </div>
              </dl>

              <div v-if="(event as any).link && (event as any).link !== '#'" class="pt-4">
                <UButton as="a" :href="(event as any).link" target="_blank" rel="noopener noreferrer" size="lg"
                  color="neutral" variant="solid" class="w-full justify-center" icon="i-lucide-external-link">
                  Открыть страницу
                </UButton>
              </div>
            </UCard>
          </div>
        </div>

        <UContentSurround v-if="surround.prev || surround.next" :prev="surround.prev" :next="surround.next" />
      </div>

      <!-- Не найдено -->
      <div v-else class="py-10">
        <UEmpty icon="i-lucide-calendar-x" title="Мероприятие не найдено"
          description="Возможно, оно было удалено или вы перешли по неверной ссылке." />
      </div>
    </UContainer>

    <!-- Slideover редактирования -->
    <USlideover v-model:open="editOpen" side="right" title="Редактирование мероприятия"
      description="Измените данные мероприятия">
      <template #body>
        <div class="flex flex-col gap-4 py-2">
          <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
            <UFormField label="Название" name="title" required>
              <UInput v-model="editState.title" size="lg" placeholder="Название мероприятия" />
            </UFormField>

            <UFormField label="Категория (бейдж)" name="badge">
              <USelect v-model="editState.badge" :items="badgeOptions" placeholder="Выберите категорию" size="lg"
                class="w-full" />
            </UFormField>

            <UFormField label="Описание" name="description">
              <UTextarea v-model="editState.description" size="lg" :rows="3" placeholder="Описание мероприятия" />
            </UFormField>

            <UFormField label="Дата проведения" name="date" required>
              <UInputDate v-model="editDateValue" size="lg" class="w-full">
                <template #trailing>
                  <UPopover>
                    <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату"
                      class="px-0" />
                    <template #content>
                      <UCalendar v-model="editDateValue" class="p-2" />
                    </template>
                  </UPopover>
                </template>
              </UInputDate>
            </UFormField>

            <UFormField label="Ссылка на подробности" name="link">
              <UInput v-model="editState.link" size="lg" placeholder="https://..." />
            </UFormField>

            <UFormField label="Изображение (URL)" name="image">
              <UInput v-model="editState.image" size="lg" placeholder="/src/img/event-cover.svg" />
            </UFormField>

          </UForm>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="editOpen = false">
            Отмена
          </UButton>
          <UButton size="xl" class="w-full justify-center" :loading="editSubmitting" @click="handleEditSubmit">
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>

    <!-- Модал подтверждения удаления -->
    <UModal v-model:open="deleteConfirmOpen" title="Удалить мероприятие?"
      description="Подтвердите удаление мероприятия">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить
          <strong>«{{ event?.title }}»</strong>?
          Это действие нельзя отменить.
        </p>
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" size="lg" @click="deleteConfirmOpen = false">
            Отмена
          </UButton>
          <UButton color="red" size="lg" :loading="deleteSubmitting" @click="handleDelete">
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
