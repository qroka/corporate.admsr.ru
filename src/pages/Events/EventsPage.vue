<script setup lang="ts">
import { computed, ref, watch, reactive } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';
import { useEventsData } from '../../composables/useEventsData';
import { useAppToast } from '../../composables/useAppToast';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
};

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

const { toast } = useAppToast();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить мероприятия',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const posts = computed<EventPost[]>(() =>
  events.value.map((e, idx) => ({
    id: e.id,
    title: e.title,
    description: e.description,
    date: e.date,
    badge: e.badge,
    to: `/events/${e.id}`,
    image: { src: coverAt(e.coverIndex ?? idx), alt: e.title },
  })),
);

// ─── Фильтры и поиск ─────────────────────────────────────────────────────────
const searchQuery = ref('');

// ─── Сортировка (как в Gallery) ───────────────────────────────────────────────
const sortKey = ref<'newest' | 'oldest' | 'title-asc' | 'title-desc'>('newest');
const sortOptions = [
  { value: 'newest', label: 'Сначала новые' },
  { value: 'oldest', label: 'Сначала старые' },
  { value: 'title-asc', label: 'По названию (А‑Я)' },
  { value: 'title-desc', label: 'По названию (Я‑А)' },
];

// ─── Форма создания мероприятия ───────────────────────────────────────────────
const createOpen = ref(false);

type CreateFormState = {
  title: string;
  description: string;
  badge: string | null;
  date: string;
  image: string;
  link: string;
};

const createState = reactive<CreateFormState>({
  title: '',
  description: '',
  badge: null,
  date: '',
  image: '/src/img/tailwindcss-v4.svg',
  link: '#',
});

type SingleDateValue = { value?: any } | any | null;
const createDateValue = ref<SingleDateValue>(null);
const createSubmitting = ref(false);
const createError = ref<string | null>(null);

const badgeOptions = computed(() => badges.value.map((b) => ({ value: b, label: b })));

const filteredPosts = computed(() => {
  let list = posts.value;

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase();
    list = list.filter((p) => {
      const title = p.title ?? '';
      const description = p.description ?? '';
      return title.toLowerCase().includes(q) || description.toLowerCase().includes(q);
    });
  }

  return list;
});

function resetCreateForm() {
  createState.title = '';
  createState.description = '';
  createState.badge = null;
  createState.date = '';
  createState.image = '/src/img/tailwindcss-v4.svg';
  createState.link = '#';
  createDateValue.value = null;
  createError.value = null;
}

function openCreate() {
  resetCreateForm();
  createOpen.value = true;
}

const headerLinks = computed(() => {
  if (currentRole.value !== 'admin') return [];
  return [
    {
      label: 'Добавить мероприятие',
      color: 'neutral',
      variant: 'outline',
      size: 'xl',
      onClick: openCreate,
    },
  ];
});

// watch надёжнее @update:model-value — срабатывает и при ручном вводе, и при выборе из календаря
watch(createDateValue, (val) => {
  const d = val?.value ?? val;
  createState.date = (d && typeof d.toString === 'function') ? d.toString() : '';
});

function validateCreate(): boolean {
  if (!createState.title.trim()) {
    createError.value = 'Заполните название мероприятия.';
    return false;
  }
  if (!createState.date) {
    createError.value = 'Выберите дату проведения.';
    return false;
  }
  createError.value = null;
  return true;
}

async function handleCreateSubmit() {
  if (!validateCreate()) return;
  createSubmitting.value = true;
  createError.value = null;
  try {
    const nextId = Math.max(0, ...events.value.map((p) => p.id ?? 0)) + 1;
    events.value = [
      {
        id: nextId,
        title: createState.title,
        description: createState.description,
        badge: createState.badge ?? undefined,
        date: createState.date,
        coverIndex: nextId,
      },
      ...events.value,
    ];
    createOpen.value = false;
    resetCreateForm();
  } catch (e: any) {
    createError.value = e?.message ?? 'Ошибка при создании мероприятия';
  } finally {
    createSubmitting.value = false;
  }
}

const sortedPosts = computed(() => {
  const list = [...filteredPosts.value];

  list.sort((a, b) => {
    const aDate = String((a as any).date ?? '');
    const bDate = String((b as any).date ?? '');
    const aTitle = String((a as any).title ?? '');
    const bTitle = String((b as any).title ?? '');
    if (sortKey.value === 'newest') return bDate.localeCompare(aDate);
    if (sortKey.value === 'oldest') return aDate.localeCompare(bDate);
    if (sortKey.value === 'title-asc') return aTitle.localeCompare(bTitle, 'ru');
    if (sortKey.value === 'title-desc') return bTitle.localeCompare(aTitle, 'ru');
    return 0;
  });

  return list;
});

function isNewBadge(value: unknown) {
  return String(value ?? '').trim().toLowerCase().includes('нов');
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <!-- Верхняя панель как в Gallery (фиксированная) -->
    <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
      <UPageHeader :links="headerLinks" class="border-none p-0 w-full">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Мероприятия</h1>
        </template>
      </UPageHeader>

      <!-- Поиск + сортировка -->
      <UContainer class="flex flex-col max-w-full w-full sm:flex-row gap-3 items-stretch sm:items-center sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          size="xl"
          color="neutral"
          variant="outline"
          placeholder="Поиск по названию..."
          class="flex-1"
        />
        <USelect
          v-model="sortKey"
          :items="sortOptions"
          size="xl"
          color="neutral"
        />
      </UContainer>

      <p v-if="!loading && filteredPosts.length !== posts.length" class="text-sm text-muted -mt-2">
        Найдено: {{ filteredPosts.length }} из {{ posts.length }}
      </p>
    </UContainer>

    <!-- Скроллится только список -->
    <UContainer class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <!-- Список мероприятий -->
      <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
        <UBlogPost v-for="post in sortedPosts" :key="post.id ?? post.title" v-bind="post" class="h-full">
          <template #badge>
            <UBadge
              v-if="post.badge"
              :color="isNewBadge(post.badge) ? 'primary' : 'neutral'"
              :variant="isNewBadge(post.badge) ? 'solid' : 'subtle'"
            >
              {{ post.badge }}
            </UBadge>
          </template>
        </UBlogPost>
      </div>
    </UContainer>

      <!-- Slideover создания мероприятия (только для администратора) -->
      <USlideover v-model:open="createOpen" side="right" title="Новое мероприятие" description="">
        <template #body>
          <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
            <UFormField label="Название мероприятия" name="title" required>
              <UInput v-model="createState.title" size="xl" class="w-full"
                placeholder="Введите название мероприятия" />
            </UFormField>

            <UFormField label="Категория" name="badge">
              <USelect v-model="createState.badge" :items="badgeOptions" placeholder="Выберите категорию" size="xl"
                class="w-full" />
            </UFormField>

            <UFormField label="Описание" name="description">
              <UTextarea v-model="createState.description" size="xl" class="w-full" :rows="3"
                placeholder="Кратко опишите цель и формат мероприятия..." />
            </UFormField>

            <UFormField label="Дата проведения" name="date" required>
              <UInputDate v-model="createDateValue" size="xl" class="w-full">
                <template #trailing>
                  <UPopover>
                    <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату"
                      class="px-0" />
                    <template #content>
                      <UCalendar v-model="createDateValue" class="p-2" />
                    </template>
                  </UPopover>
                </template>
              </UInputDate>
            </UFormField>
            <UFormField label="Изображение (URL)" name="image">
              <UFileUpload label="Drop your image here"
                description="SVG, PNG, JPG or GIF (max. 2MB)" class="w-full min-h-48" />
            </UFormField>
            <UAlert v-if="createError" color="red" variant="subtle" icon="i-lucide-alert-circle"
              :description="createError" />
          </UForm>
        </template>
        <template #footer>
          <div class="flex justify-between gap-3 items-center w-full">
            <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center"
              @click="createOpen = false">
              Отмена
            </UButton>
            <UButton size="xl" class="w-full justify-center" :loading="createSubmitting" @click="handleCreateSubmit">
              Создать
            </UButton>
          </div>
        </template>
      </USlideover>
  </UMain>
</template>
