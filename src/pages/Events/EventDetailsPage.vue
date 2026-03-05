<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';

const API_URL = '/api/events.php';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
  link?: string;
};

const route = useRoute();
const router = useRouter();

// ─── Данные ──────────────────────────────────────────────────────────────────
const event = ref<EventPost | null>(null);
const loading = ref(true);
const fetchError = ref<string | null>(null);

const badgeOptions = ref<{ value: string; label: string }[]>([]);

async function fetchEvent(id: string | string[]) {
  loading.value = true;
  fetchError.value = null;
  try {
    // Загружаем мероприятие и все события (для списка бейджей) параллельно
    const [resEvent, resAll] = await Promise.all([
      fetch(`${API_URL}?id=${id}`),
      fetch(API_URL),
    ]);

    if (!resEvent.ok) throw new Error(`HTTP ${resEvent.status}`);
    const jsonEvent = await resEvent.json();
    if (!jsonEvent.success) throw new Error(jsonEvent.message);
    event.value = jsonEvent.data;

    if (resAll.ok) {
      const jsonAll = await resAll.json();
      if (jsonAll.success) {
        const badges = [
          ...new Set<string>(
            (jsonAll.data as any[]).map((e) => e.badge).filter(Boolean),
          ),
        ].sort();
        badgeOptions.value = badges.map((b) => ({ value: b, label: b }));
      }
    }
  } catch (e: any) {
    fetchError.value = e?.message ?? 'Не удалось загрузить мероприятие';
    event.value = null;
  } finally {
    loading.value = false;
  }
}

// Загружаем при первом открытии и при смене id в маршруте
watch(
  () => route.params.id,
  (id) => { if (id) fetchEvent(id); },
  { immediate: true },
);

const isAdmin = computed(() => currentRole.value === 'admin');

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
    const res = await fetch(`${API_URL}?id=${(event.value as any).id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: editState.title,
        description: editState.description,
        badge: editState.badge,
        date: editState.date,
        image: editState.image,
        link: editState.link,
      }),
    });
    const json = await res.json();
    if (!json.success) {
      editError.value = json.message;
      return;
    }
    // Обновляем локальные данные ответом сервера
    event.value = json.data;
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

async function handleDelete() {
  if (!event.value) return;
  deleteSubmitting.value = true;
  deleteError.value = null;
  try {
    const res = await fetch(`${API_URL}?id=${(event.value as any).id}`, {
      method: 'DELETE',
    });
    const json = await res.json();
    if (!json.success) {
      deleteError.value = json.message;
      return;
    }
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
  <div class="flex flex-col gap-6 w-full h-full min-h-0">
    <Headline class="text-zinc-700 dark:text-zinc-50">
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
              {{ event.description }}
            </p>
            <p class="text-xl leading-6 text-zinc-500" v-else>
              Проверьте корректность ссылки или вернитесь к списку мероприятий.
            </p>
          </template>
        </div>
        <div class="flex flex-col gap-2 w-full sm:w-auto">
          <UButton
            color="neutral"
            variant="ghost"
            size="xl"
            class="w-full justify-center"
            @click="router.push({ name: 'events' })"
          >
            К списку мероприятий
          </UButton>
          <div v-if="isAdmin && event && !loading" class="flex gap-2 justify-end">
            <UButton
              color="neutral"
              variant="outline"
              size="lg"
              class="w-full sm:w-auto justify-center"
              @click="openEdit"
            >
              Изменить
            </UButton>
            <UButton
              color="red"
              variant="soft"
              size="lg"
              class="w-full sm:w-auto justify-center"
              @click="deleteConfirmOpen = true"
            >
              Удалить
            </UButton>
          </div>
        </div>
      </div>
    </Headline>

    <!-- Ошибка загрузки -->
    <UAlert
      v-if="fetchError"
      color="red"
      variant="subtle"
      icon="i-lucide-alert-circle"
      :title="fetchError"
    >
      <template #footer>
        <UButton size="sm" color="red" variant="ghost" @click="fetchEvent(route.params.id)">
          Повторить
        </UButton>
      </template>
    </UAlert>

    <!-- Скелетон -->
    <UMain v-if="loading" class="flex flex-1 min-h-0 flex-col w-full gap-6">
      <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-6 items-start">
        <USkeleton class="h-64 rounded-xl" />
        <USkeleton class="h-48 rounded-xl" />
      </div>
    </UMain>

    <!-- Контент мероприятия -->
    <UMain v-else-if="event" class="flex flex-1 min-h-0 flex-col w-full h-full gap-6">
      <div class="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-6 items-start">
        <UCard class="w-full">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <span
                v-if="event.badge"
                class="inline-flex items-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200 px-3 py-1 text-xs font-medium"
              >
                {{ event.badge }}
              </span>
              <span class="ml-auto text-sm text-zinc-500 dark:text-zinc-400">
                {{ event.date }}
              </span>
            </div>
          </template>

          <div class="space-y-4">
            <p v-if="event.description" class="text-base leading-6 text-zinc-700 dark:text-zinc-100">
              {{ event.description }}
            </p>
            <p v-else class="text-sm text-zinc-500 dark:text-zinc-400">
              Описание мероприятия пока не добавлено.
            </p>

            <div v-if="event.link && event.link !== '#'" class="pt-2">
              <UButton
                as="a"
                :href="event.link"
                target="_blank"
                rel="noopener noreferrer"
                size="lg"
              >
                Перейти к подробностям
              </UButton>
            </div>
          </div>
        </UCard>

        <div class="space-y-4">
          <UCard v-if="event.image" class="overflow-hidden">
            <img
              :src="event.image"
              :alt="event.title"
              class="w-full h-48 object-cover"
            />
          </UCard>
        </div>
      </div>
    </UMain>

    <!-- Мероприятие не найдено -->
    <UMain v-else class="flex flex-1 items-center justify-center">
      <UAlert
        color="red"
        variant="subtle"
        icon="i-lucide-alert-circle"
        title="Мероприятие не найдено"
        description="Возможно, оно было удалено или вы перешли по неверной ссылке."
      />
    </UMain>

    <!-- Slideover редактирования -->
    <USlideover v-model:open="editOpen" side="right" title="Редактирование мероприятия" description="Измените данные мероприятия">
      <template #body>
        <div class="flex flex-col gap-4 py-2">
          <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
            <UFormField label="Название" name="title" required>
              <UInput v-model="editState.title" size="lg" placeholder="Название мероприятия" />
            </UFormField>

            <UFormField label="Категория (бейдж)" name="badge">
              <USelect
                v-model="editState.badge"
                :items="badgeOptions"
                placeholder="Выберите категорию"
                size="lg"
                class="w-full"
              />
            </UFormField>

            <UFormField label="Описание" name="description">
              <UTextarea
                v-model="editState.description"
                size="lg"
                :rows="3"
                placeholder="Описание мероприятия"
              />
            </UFormField>

            <UFormField label="Дата проведения" name="date" required>
              <UInputDate
                v-model="editDateValue"
                size="lg"
                class="w-full"

              >
                <template #trailing>
                  <UPopover>
                    <UButton
                      color="neutral"
                      variant="link"
                      size="sm"
                      icon="i-lucide-calendar"
                      aria-label="Выбрать дату"
                      class="px-0"
                    />
                    <template #content>
                      <UCalendar v-model="editDateValue" class="p-2" />
                    </template>
                  </UPopover>
                </template>
              </UInputDate>
            </UFormField>

            <UFormField label="Ссылка на подробности" name="link">
              <UInput
                v-model="editState.link"
                size="lg"
                placeholder="https://..."
              />
            </UFormField>

            <UFormField label="Изображение (URL)" name="image">
              <UInput v-model="editState.image" size="lg" placeholder="/src/img/event-cover.svg" />
            </UFormField>

            <UAlert
              v-if="editError"
              color="red"
              variant="subtle"
              icon="i-lucide-alert-circle"
              :description="editError"
            />
          </UForm>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
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
            size="xl"
            class="w-full justify-center"
            :loading="editSubmitting"
            @click="handleEditSubmit"
          >
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>

    <!-- Модал подтверждения удаления -->
    <UModal v-model:open="deleteConfirmOpen" title="Удалить мероприятие?" description="Подтвердите удаление мероприятия">
      <template #body>
        <p class="text-zinc-700 dark:text-zinc-200">
          Вы уверены, что хотите удалить
          <strong>«{{ event?.title }}»</strong>?
          Это действие нельзя отменить.
        </p>
        <UAlert
          v-if="deleteError"
          color="red"
          variant="subtle"
          icon="i-lucide-alert-circle"
          :description="deleteError"
          class="mt-3"
        />
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton
            color="neutral"
            variant="outline"
            size="lg"
            @click="deleteConfirmOpen = false"
          >
            Отмена
          </UButton>
          <UButton
            color="red"
            size="lg"
            :loading="deleteSubmitting"
            @click="handleDelete"
          >
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
