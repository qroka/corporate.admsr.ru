<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
  image_full?: string;
};

const route = useRoute();
const router = useRouter();

// ─── State ───────────────────────────────────────────────────────────────────
const event = ref<EventPost | null>(null);
const loading = ref(false);
const fetchError = ref<string | null>(null);

const isAdmin = computed(() => currentRole.value === 'admin');

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
      const uploadRes = await fetch('/api/upload.php', { method: 'POST', body: formData });
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
  <div class="flex flex-col gap-6 w-full h-full min-h-0">
    <div class="text-zinc-700 dark:text-zinc-50">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between w-full">
        <div>
          <template v-if="loading">
            <USkeleton class="h-12 w-64 rounded-xl" />
            <USkeleton class="h-6 w-48 rounded-xl mt-2" />
          </template>
          <template v-else>
            <h1 class="text-4xl leading-12 font-display">
              {{ event?.title ?? 'Мероприятие не найдено' }}
            </h1>
            <p class="text-xl leading-6 text-zinc-500" v-if="event">
              {{ firstSentence(event.description ?? '') }}
            </p>
            <p class="text-xl leading-6 text-zinc-500" v-else-if="!fetchError">
              Проверьте корректность ссылки или вернитесь к списку мероприятий.
            </p>
          </template>
        </div>
        <div class="flex flex-col gap-2 w-full sm:w-auto">
          <UButton color="neutral" variant="ghost" size="xl" class="w-full justify-center" @click="router.push({ name: 'events' })">
            К списку мероприятий
          </UButton>
          <div v-if="isAdmin && event && !loading" class="flex gap-2 justify-end">
            <UButton color="neutral" variant="outline" size="lg" class="w-full sm:w-auto justify-center" @click="openEdit">
              Изменить
            </UButton>
            <UButton color="red" variant="soft" size="lg" class="w-full sm:w-auto justify-center" @click="deleteConfirmOpen = true">
              Удалить
            </UButton>
          </div>
        </div>
      </div>
    </div>

    <UAlert v-if="fetchError" color="error" variant="subtle" icon="i-lucide-alert-circle" :title="fetchError">
      <template #footer>
        <UButton size="sm" color="error" variant="ghost" @click="fetchEvent(route.params.id)">Повторить</UButton>
      </template>
    </UAlert>

    <UMain v-if="loading" class="flex flex-1 min-h-0 flex-col w-full gap-6">
      <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-6 items-start">
        <USkeleton class="h-64 rounded-xl" />
        <USkeleton class="h-48 rounded-xl" />
      </div>
    </UMain>

    <UMain v-else-if="event" class="flex flex-1 min-h-0 flex-col w-full h-full gap-6">
      <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-6 items-start">
        <UCard class="w-full">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <span v-if="event.badge" class="inline-flex items-center rounded-full bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-200 px-3 py-1 text-xs font-medium">
                {{ event.badge }}
              </span>
              <span class="ml-auto text-sm text-zinc-500 dark:text-zinc-400">{{ event.date }}</span>
            </div>
          </template>
          <div class="space-y-4">
            <p v-if="event.description" class="text-base leading-6 text-zinc-700 dark:text-zinc-100">
              {{ event.description }}
            </p>
            <p v-else class="text-sm text-zinc-500 dark:text-zinc-400">
              Описание мероприятия пока не добавлено.
            </p>
          </div>
        </UCard>

        <div class="space-y-4">
          <UCard v-if="event.image_full || event.image" class="overflow-hidden p-0">
            <img
              :src="event.image_full || (typeof event.image === 'object' ? (event.image as any).src : event.image as string)"
              :alt="event.title"
              class="w-full object-cover rounded-xl"
            />
          </UCard>
        </div>
      </div>
    </UMain>

    <UMain v-else-if="!fetchError" class="flex flex-1 items-center justify-center">
      <UAlert color="error" variant="subtle" icon="i-lucide-alert-circle" title="Мероприятие не найдено"
        description="Возможно, оно было удалено или вы перешли по неверной ссылке." />
    </UMain>

    <USlideover v-model:open="editOpen" side="right" title="Редактирование мероприятия" description="Измените данные мероприятия">
      <template #body>
        <div class="flex flex-col gap-4 py-2">
          <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
            <UFormField label="Название" name="title" required>
              <UInput v-model="editState.title" size="lg" placeholder="Название мероприятия" class="w-full" />
            </UFormField>

            <UFormField label="Категория (бейдж)" name="badge">
              <UInput v-model="editState.badge" size="lg" placeholder="Корпоратив, Спорт..." class="w-full" />
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

    <UModal v-model:open="deleteConfirmOpen" title="Удалить мероприятие?" description="Подтвердите удаление мероприятия">
      <template #body>
        <p class="text-zinc-700 dark:text-zinc-200">
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
  </div>
</template>