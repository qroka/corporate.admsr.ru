<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';
import { useAppToast } from '../../composables/useAppToast';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
  image_full?: string;
  link?: string;
};

const route = useRoute();
const router = useRouter();

// ─── State ───────────────────────────────────────────────────────────────────
const event = ref<EventPost | null>(null);
const loading = ref(false);
const fetchError = ref<string | null>(null);

const isAdmin = computed(() => currentRole.value === 'admin');
const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const isArchivedEvent = computed(() => {
  const badge = String(event.value?.badge ?? '').trim().toLowerCase();
  return badge.includes('архив');
});
const isNewEvent = computed(() => {
  const badge = String(event.value?.badge ?? '').trim().toLowerCase();
  return badge.includes('нов');
});

const { toast } = useAppToast();

const editBadgeOptions = [
  { value: 'Новое', label: 'Новое' },
  { value: 'Архив', label: 'Архив' },
];

// ─── API ─────────────────────────────────────────────────────────────────────
function firstSentence(text: string): string {
  if (!text) return text;
  return text.match(/^[^.!?]*[.!?]/)?.[0].trim() ?? text;
}

function mapEvent(raw: any): EventPost {
  return {
    id:          raw.id,
    title:       raw.title,
    description: raw.description ?? '',
    badge:       raw.badge ?? undefined,
    date:        raw.date,
    to:          `/events/${raw.id}`,
    image:       raw.image      ? { src: raw.image,      alt: raw.title } : undefined,
    image_full:  raw.image_full || raw.image || '',
    link:        raw.link ?? undefined,
  };
}

async function fetchEvent(id: string | string[]) {
  loading.value = true;
  fetchError.value = null;
  event.value = null;
  try {
    const res = await fetch(`/api/events.php?id=${id}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);

    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки');

    event.value = mapEvent(json.data);
  } catch (e: any) {
    fetchError.value = e.message ?? 'Не удалось загрузить мероприятие';
  } finally {
    loading.value = false;
  }
}

onMounted(() => fetchEvent(route.params.id));
watch(() => route.params.id, fetchEvent);

watch(
  fetchError,
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
  if (isArchivedEvent.value) return;
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

// ─── Форма редактирования ─────────────────────────────────────────────────────
type EditFormState = {
  title: string;
  description: string;
  badge: string | null;
  date: string;
};

const editOpen = ref(false);
const editState = reactive<EditFormState>({
  title: '',
  description: '',
  badge: null,
  date: '',
});

const editImageFile = ref<File | null>(null);
const editImagePreview = ref('');

function onEditImageSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  editImageFile.value = file;
  if (editImagePreview.value) URL.revokeObjectURL(editImagePreview.value);
  editImagePreview.value = file ? URL.createObjectURL(file) : '';
}

type SingleDateValue = { value?: any } | any | null;
const editDateValue = ref<SingleDateValue>(null);
const editSubmitting = ref(false);
const editError = ref<string | null>(null);

function fillEditState() {
  if (!event.value) return;
  editState.title       = event.value.title ?? '';
  editState.description = event.value.description ?? '';
  editState.badge       = (event.value as any).badge ?? null;
  editState.date        = (event.value as any).date ?? '';
  editImageFile.value = null;
}

function openEdit() {
  fillEditState();
  editOpen.value = true;
}

watch(editDateValue, (val) => {
  const d = val?.value ?? val;
  editState.date = d && typeof d.toString === 'function' ? d.toString() : '';
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
  if (!validateEdit() || !event.value?.id) return;
  editSubmitting.value = true;
  editError.value = null;
  try {
    let imagePatch: { image?: string; image_full?: string } = {};
    if (editImageFile.value) {
      const formData = new FormData();
      formData.append('image', editImageFile.value);
      const uploadRes = await fetch('/api/Upload/upload.php', { method: 'POST', body: formData });
      const uploadJson = await uploadRes.json();
      if (!uploadJson.success) throw new Error(uploadJson.message || 'Ошибка загрузки изображения');
      imagePatch = { image: uploadJson.data.image, image_full: uploadJson.data.image_full };
    }

    const res = await fetch(`/api/events.php?id=${event.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title:       editState.title.trim(),
        description: editState.description.trim() || null,
        badge:       editState.badge || null,
        date:        editState.date,
        ...imagePatch,
      }),
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка сохранения');

    event.value = mapEvent(json.data);
    editOpen.value = false;
  } catch (e: any) {
    editError.value = e.message ?? 'Ошибка при сохранении';
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
  if (!event.value?.id) return;
  deleteSubmitting.value = true;
  deleteError.value = null;
  try {
    const res = await fetch(`/api/events.php?id=${event.value.id}`, {
      method: 'DELETE',
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка удаления');

    deleteConfirmOpen.value = false;
    router.push({ name: 'events' });
  } catch (e: any) {
    deleteError.value = e.message ?? 'Ошибка при удалении';
  } finally {
    deleteSubmitting.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-0">
    <UContainer class="flex-1 min-h-0 overflow-y-auto max-w-full w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 scrollbar-hide mx-0 pb-8">
      <UAlert v-if="fetchError" color="error" variant="subtle" icon="i-lucide-alert-circle" :title="fetchError" class="mb-6">
        <template #footer>
          <UButton size="sm" color="error" variant="ghost" @click="fetchEvent(route.params.id)">Повторить</UButton>
        </template>
      </UAlert>

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
            <img
              v-if="event.image_full || (event.image as any)?.src"
              :src="event.image_full || (event.image as any)?.src"
              :alt="(event.image as any)?.alt ?? event.title"
              class="absolute inset-0 h-full w-full object-cover"
            />
            <div class="absolute inset-0 bg-linear-to-t from-black/65 via-black/25 to-transparent" />

            <div class="absolute inset-0 p-6 flex flex-col justify-end">
              <div class="flex flex-wrap items-center gap-2 mb-3">
                <UBadge v-if="event.badge" :color="isNewEvent ? 'primary' : 'neutral'" :variant="isNewEvent ? 'solid' : 'soft'" size="md">
                  {{ event.badge }}
                </UBadge>
                <UBadge color="neutral" variant="soft" size="md" class="backdrop-blur">
                  {{ event.date }}
                </UBadge>
                <UButton
                  class="ml-auto"
                  color="neutral"
                  variant="soft"
                  size="md"
                  icon="i-lucide-arrow-left"
                  @click="router.push({ name: isKiosk ? 'kioskEvents' : 'events' } as any)"
                >
                  К списку
                </UButton>
              </div>

              <div class="flex items-end justify-between gap-4 flex-wrap">
                <div class="min-w-0">
                  <h1 class="text-2xl sm:text-4xl font-semibold tracking-tight text-white">
                    {{ event.title }}
                  </h1>
                  <p v-if="event.description" class="mt-2 text-sm sm:text-base text-white/85 max-w-3xl">
                    {{ firstSentence(event.description ?? '') }}
                  </p>
                </div>

                <div v-if="isAdmin" class="flex gap-2">
                  <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-pencil" @click="openEdit">
                    Изменить
                  </UButton>
                  <UButton color="red" variant="soft" size="lg" icon="i-lucide-trash-2" @click="deleteConfirmOpen = true">
                    Удалить
                  </UButton>
                </div>
              </div>
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
                <UButton
                  :color="isArchivedEvent ? 'neutral' : (isJoined ? 'neutral' : 'primary')"
                  :variant="isArchivedEvent ? 'outline' : (isJoined ? 'soft' : 'solid')"
                  size="lg"
                  :icon="isArchivedEvent ? 'i-lucide-lock' : (isJoined ? 'i-lucide-check' : 'i-lucide-ticket')"
                  :disabled="isArchivedEvent"
                  @click="toggleJoin"
                >
                  {{ isArchivedEvent ? 'Запись недоступна' : (isJoined ? 'Вы записаны' : 'Записаться') }}
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
                    {{ isArchivedEvent ? 'Архив (запись закрыта)' : (isJoined ? 'Вы записаны' : 'Не записаны') }}
                  </dd>
                </div>
              </dl>

              <div v-if="event.link && event.link !== '#'" class="pt-4">
                <UButton as="a" :href="event.link" target="_blank" rel="noopener noreferrer" size="lg" color="neutral" variant="solid" class="w-full justify-center" icon="i-lucide-external-link">
                  Открыть страницу
                </UButton>
              </div>
            </UCard>
          </div>
        </div>
      </div>

      <!-- Не найдено -->
      <div v-else class="py-10">
        <UEmpty
          icon="i-lucide-calendar-x"
          title="Мероприятие не найдено"
          description="Возможно, оно было удалено или вы перешли по неверной ссылке."
        />
      </div>
    </UContainer>

    <!-- Slideover редактирования -->
    <USlideover v-model:open="editOpen" side="right" title="Редактирование мероприятия" description="Измените данные мероприятия">
      <template #body>
        <div class="flex flex-col gap-4 py-2">
          <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
            <UFormField label="Название" name="title" required>
              <UInput v-model="editState.title" size="lg" placeholder="Название мероприятия" class="w-full" />
            </UFormField>

            <UFormField label="Категория (бейдж)" name="badge">
              <USelect v-model="editState.badge" :items="editBadgeOptions" placeholder="Выберите: Новое или Архив" size="lg" class="w-full" />
            </UFormField>

            <UFormField label="Описание" name="description">
              <UTextarea v-model="editState.description" size="lg" :rows="3" placeholder="Описание мероприятия" class="w-full" />
            </UFormField>

            <UFormField label="Дата проведения" name="date" required>
              <UInputDate v-model="editDateValue" size="lg" class="w-full">
                <template #trailing>
                  <UPopover>
                    <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату" class="px-0" />
                    <template #content>
                      <UCalendar v-model="editDateValue" class="p-2" />
                    </template>
                  </UPopover>
                </template>
              </UInputDate>
            </UFormField>

            <UFormField label="Изображение" name="image">
              <div class="flex flex-col gap-2 w-full">
                <input id="editFileInput" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="onEditImageSelected" />
                <img v-if="editImagePreview" :src="editImagePreview" alt="Предпросмотр" class="w-full h-40 object-cover rounded-lg" />
                <label for="editFileInput" class="flex items-center justify-center gap-2 cursor-pointer rounded-lg border border-default px-4 py-2.5 text-sm font-medium text-default hover:bg-elevated/50 transition-colors">
                  <UIcon name="i-lucide-upload" class="text-base shrink-0" />
                  {{ editImageFile ? 'Изменить фото' : 'Заменить фото' }}
                </label>
                <p v-if="editImageFile" class="text-xs text-muted truncate px-1">{{ editImageFile.name }}</p>
              </div>
            </UFormField>

            <UAlert v-if="editError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="editError" />
          </UForm>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="editOpen = false">Отмена</UButton>
          <UButton size="xl" class="w-full justify-center" :loading="editSubmitting" @click="handleEditSubmit">Сохранить</UButton>
        </div>
      </template>
    </USlideover>

    <!-- Модал подтверждения удаления -->
    <UModal v-model:open="deleteConfirmOpen" title="Удалить мероприятие?" description="Подтвердите удаление мероприятия">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ event?.title }}»</strong>? Это действие нельзя отменить.
        </p>
        <UAlert v-if="deleteError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="deleteError" class="mt-3" />
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" size="lg" @click="deleteConfirmOpen = false">Отмена</UButton>
          <UButton color="red" size="lg" :loading="deleteSubmitting" @click="handleDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>