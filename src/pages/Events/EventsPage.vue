<script setup lang="ts">
import { computed, ref, watch, reactive } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
};

const coverModules = import.meta.glob('../../img/EventsWebp/*.webp', {
  eager: true,
  import: 'default',
});
const coverSrcs = Object.entries(coverModules)
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([, src]) => src as string);
function coverAt(index: number): string {
  return coverSrcs.length ? coverSrcs[index % coverSrcs.length] : '/src/img/Logo.svg';
}

// ─── Данные ──────────────────────────────────────────────────────────────────
const posts = ref<EventPost[]>([
  {
    id: 110,
    title: 'Зимний корпоратив',
    description: 'Встречаемся всей командой: награждения, конкурсы и фото-зона.',
    date: '2026-01-25',
    badge: 'Корпоратив',
    to: `/events/110`,
    image: { src: coverAt(0), alt: 'Зимний корпоратив' },
  },
  {
    id: 111,
    title: 'День донора',
    description: 'Корпоративная донорская акция. Участие добровольное.',
    date: '2026-02-12',
    badge: 'Волонтёрство',
    to: `/events/111`,
    image: { src: coverAt(1), alt: 'День донора' },
  },
  {
    id: 112,
    title: 'Спартакиада сотрудников',
    description: 'Командные соревнования и спортивный праздник.',
    date: '2026-03-10',
    badge: 'Спорт',
    to: `/events/112`,
    image: { src: coverAt(2), alt: 'Спартакиада' },
  },
  {
    id: 113,
    title: 'Лекция: новые инструменты',
    description: 'Внутренняя лекция и демо полезных практик.',
    date: '2026-03-22',
    badge: 'Обучение',
    to: `/events/113`,
    image: { src: coverAt(3), alt: 'Лекция' },
  },
  {
    id: 114,
    title: 'Субботник',
    description: 'Наводим порядок и завершаем чаепитием.',
    date: '2026-04-06',
    badge: 'Команда',
    to: `/events/114`,
    image: { src: coverAt(4), alt: 'Субботник' },
  },
  {
    id: 115,
    title: 'Экскурсия для новичков',
    description: 'Знакомство с офисом, сервисами и командой.',
    date: '2026-04-18',
    badge: 'Новичкам',
    to: `/events/115`,
    image: { src: coverAt(5), alt: 'Экскурсия' },
  },
]);
const loading = ref(false);

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

const uniqueBadges = computed(() => {
  const set = new Set<string>();
  posts.value.forEach((p) => p.badge && set.add(p.badge));
  return Array.from(set).sort();
});
const badgeOptions = computed(() => uniqueBadges.value.map((b) => ({ value: b, label: b })));

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
  if (currentRole !== 'admin') return [];
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
    const nextId = Math.max(0, ...posts.value.map((p) => p.id ?? 0)) + 1;
    posts.value = [
      {
        id: nextId,
        title: createState.title,
        description: createState.description,
        badge: createState.badge ?? undefined,
        date: createState.date,
        to: `/events/${nextId}`,
        image: { src: coverAt(nextId), alt: createState.title },
      },
      ...posts.value,
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
    if (sortKey.value === 'newest') return (b.date ?? '').localeCompare(a.date ?? '');
    if (sortKey.value === 'oldest') return (a.date ?? '').localeCompare(b.date ?? '');
    if (sortKey.value === 'title-asc') return (a.title ?? '').localeCompare(b.title ?? '', 'ru');
    if (sortKey.value === 'title-desc') return (b.title ?? '').localeCompare(a.title ?? '', 'ru');
    return 0;
  });

  return list;
});
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
        <UBlogPost v-for="post in sortedPosts" :key="post.id ?? post.title" v-bind="post" class="h-full" />
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
