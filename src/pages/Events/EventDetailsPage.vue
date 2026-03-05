<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BlogPostProps } from '@nuxt/ui';
import rawEvents from '../../data/events.json';
import { currentRole } from '../../stores/role';

type EventPost = BlogPostProps & {
  badge?: string;
};

const route = useRoute();
const router = useRouter();

const index = computed(() => {
  const id = Number(route.params.id);
  return Number.isFinite(id) ? id : -1;
});

const event = computed<EventPost | null>(() => {
  const all = rawEvents as EventPost[];
  if (index.value < 0 || index.value >= all.length) return null;
  return all[index.value];
});

const isAdmin = computed(() => currentRole.value === 'admin');

// Локальное редактируемое состояние мероприятия для Slideover
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

const badgeOptions = Array.from(
  new Set((rawEvents as any[]).map((e) => e.badge).filter(Boolean)),
).map((b) => ({ value: b as string, label: b as string }));

function fillEditStateFromEvent() {
  if (!event.value) return;
  editState.title = event.value.title ?? '';
  editState.description = event.value.description ?? '';
  editState.badge = (event.value as any).badge ?? null;
  editState.date = (event.value as any).date ?? '';
  editState.image = (event.value as any).image ?? '';
  editState.link = typeof event.value.to === 'string' ? (event.value.to as string) : '#';
}

watch(
  event,
  (val) => {
    if (val) {
      fillEditStateFromEvent();
    }
  },
  { immediate: true },
);

function openEdit() {
  fillEditStateFromEvent();
  editOpen.value = true;
}

function editSyncDate(val: SingleDateValue) {
  const d = val?.value ?? val;
  if (d && typeof d.toString === 'function') {
    editState.date = d.toString();
  } else {
    editState.date = '';
  }
}

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
  try {
    // Локальный режим: обновляем только локальное отображение (без сохранения в файл).
    (event.value as any).title = editState.title;
    (event.value as any).description = editState.description;
    (event.value as any).badge = editState.badge ?? undefined;
    (event.value as any).date = editState.date;
    (event.value as any).image = editState.image;
    (event.value as any).to = editState.link || '#';
    editOpen.value = false;
  } finally {
    editSubmitting.value = false;
  }
}

function handleDelete() {
  console.log('Удаление мероприятия (локальный режим):', event.value);
}
</script>

<template>
  <content class="flex flex-col gap-6 w-full h-full min-h-0">
    <Headline class="text-zinc-700 dark:text-zinc-50">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between w-full">
        <div>
          <h1 class="text-4xl leading-12 font-display">
            {{ event?.title ?? 'Мероприятие не найдено' }}
          </h1>
          <p class="text-xl leading-6 text-zinc-500" v-if="event">
            {{ event.description }}
          </p>
          <p class="text-xl leading-6 text-zinc-500" v-else>
            Проверьте корректность ссылки или вернитесь к списку мероприятий.
          </p>
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
          <div
            v-if="isAdmin && event"
            class="flex gap-2 justify-end"
          >
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
              @click="handleDelete"
            >
              Удалить
            </UButton>
          </div>
        </div>
      </div>
    </Headline>

    <UMain v-if="event" class="flex flex-1 min-h-0 flex-col w-full h-full gap-6">
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

            <div v-if="event.to && event.to !== '#'" class="pt-2">
              <UButton
                as="a"
                :href="typeof event.to === 'string' ? event.to : '#'"
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

    <UMain v-else class="flex flex-1 items-center justify-center">
      <UAlert
        color="red"
        variant="subtle"
        icon="i-lucide-alert-circle"
        title="Мероприятие не найдено"
      >
        Возможно, оно было удалено или вы перешли по неверной ссылке.
      </UAlert>
    </UMain>

    <!-- Slideover редактирования мероприятия (только для администратора) -->
    <USlideover v-model:open="editOpen" side="right" title="Редактирование мероприятия">
      <template #body>
        <div class="flex flex-col gap-4 py-2">
          <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
            <UFormGroup label="Название" name="title" required>
              <UInput v-model="editState.title" size="lg" placeholder="Название мероприятия" />
            </UFormGroup>

            <UFormGroup label="Категория (бейдж)" name="badge">
              <USelect
                v-model="editState.badge"
                :items="badgeOptions"
                placeholder="Выберите категорию"
                size="lg"
                class="w-full"
              />
            </UFormGroup>

            <UFormGroup label="Описание" name="description">
              <UTextarea
                v-model="editState.description"
                size="lg"
                :rows="3"
                placeholder="Описание мероприятия"
              />
            </UFormGroup>

            <UFormGroup label="Дата проведения" name="date" required>
              <UInputDate
                v-model="editDateValue"
                size="lg"
                class="w-full"
                @update:model-value="editSyncDate"
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
            </UFormGroup>

            <UFormGroup label="Ссылка на подробности" name="link">
              <UInput
                v-model="editState.link"
                size="lg"
                placeholder="https://..."
              />
            </UFormGroup>

            <UFormGroup label="Изображение (URL)" name="image">
              <UInput v-model="editState.image" size="lg" placeholder="/src/img/event-cover.svg" />
            </UFormGroup>

            <UAlert
              v-if="editError"
              color="red"
              variant="subtle"
              icon="i-lucide-alert-circle"
            >
              {{ editError }}
            </UAlert>
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
  </content>
</template>

