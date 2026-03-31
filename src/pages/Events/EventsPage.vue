
<script setup lang="ts">
import { computed, ref, watch, reactive, onMounted } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
};

// ─── State ───────────────────────────────────────────────────────────────────
const posts = ref<EventPost[]>([]);
const loading = ref(false);
const fetchError = ref<string | null>(null);

// ─── Filters ─────────────────────────────────────────────────────────────────
const searchQuery = ref('');
const badgeFilter = ref('_all');

const sortKey = ref<'newest' | 'oldest' | 'title-asc' | 'title-desc'>('newest');
const sortOptions = [
  { value: 'newest',     label: 'Сначала новые' },
  { value: 'oldest',     label: 'Сначала старые' },
  { value: 'title-asc',  label: 'По названию (А‑Я)' },
  { value: 'title-desc', label: 'По названию (Я‑А)' },
];

function isNewBadge(value: unknown) {
  return String(value ?? '').trim().toLowerCase().includes('нов');
}

const createBadgeOptions = [
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
    description: firstSentence(raw.description ?? ''),
    badge:       raw.badge ?? undefined,
    date:        raw.date,
    to:          `/events/${raw.id}`,
    image:       raw.image ? { src: raw.image, alt: raw.title } : undefined,
  };
}

async function fetchEvents() {
  loading.value = true;
  fetchError.value = null;
  try {
    const params = new URLSearchParams();
    if (badgeFilter.value && badgeFilter.value !== '_all') params.set('badge', badgeFilter.value);

    const qs = params.toString();
    const res = await fetch(`/api/events.php${qs ? '?' + qs : ''}`);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);

    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки');

    posts.value = (json.data as any[]).map(mapEvent);
  } catch (e: any) {
    fetchError.value = e.message ?? 'Не удалось загрузить мероприятия';
  } finally {
    loading.value = false;
  }
}

onMounted(fetchEvents);
watch(badgeFilter, fetchEvents);

const filteredPosts = computed(() => {
  if (!searchQuery.value.trim()) return posts.value;
  const q = searchQuery.value.trim().toLowerCase();
  return posts.value.filter(p =>
    (p.title ?? '').toLowerCase().includes(q) ||
    (p.description ?? '').toLowerCase().includes(q)
  );
});

const sortedPosts = computed(() => {
  const list = [...filteredPosts.value];
  list.sort((a, b) => {
    if (sortKey.value === 'newest')     return String(b.date ?? '').localeCompare(String(a.date ?? ''));
    if (sortKey.value === 'oldest')     return String(a.date ?? '').localeCompare(String(b.date ?? ''));
    if (sortKey.value === 'title-asc')  return (a.title ?? '').localeCompare(b.title ?? '', 'ru');
    if (sortKey.value === 'title-desc') return (b.title ?? '').localeCompare(a.title ?? '', 'ru');
    return 0;
  });
  return list;
});

// ─── Create form ──────────────────────────────────────────────────────────────
const createOpen = ref(false);

type CreateFormState = {
  title: string;
  description: string;
  badge: string | null;
  date: string;
};

const createState = reactive<CreateFormState>({
  title: '',
  description: '',
  badge: null,
  date: '',
});

const createImageFile = ref<File | null>(null);
const createImagePreview = ref('');

function onCreateImageSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  createImageFile.value = file;
  if (createImagePreview.value) URL.revokeObjectURL(createImagePreview.value);
  createImagePreview.value = file ? URL.createObjectURL(file) : '';
}

type SingleDateValue = { value?: any } | any | null;
const createDateValue = ref<SingleDateValue>(null);
const createSubmitting = ref(false);
const createError = ref<string | null>(null);

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

watch(createDateValue, (val) => {
  const d = val?.value ?? val;
  createState.date = d && typeof d.toString === 'function' ? d.toString() : '';
});

function resetCreateForm() {
  createState.title = '';
  createState.description = '';
  createState.badge = null;
  createState.date = '';
  createImageFile.value = null;
  if (createImagePreview.value) URL.revokeObjectURL(createImagePreview.value);
  createImagePreview.value = '';
  createDateValue.value = null;
  createError.value = null;
}

function openCreate() {
  resetCreateForm();
  createOpen.value = true;
}

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
    const defaultImg = '/favicon.svg';
    let imagePath     = defaultImg;
    let imageFullPath = defaultImg;

    if (createImageFile.value) {
      const formData = new FormData();
      formData.append('image', createImageFile.value);
      const uploadRes = await fetch('/api/upload.php', { method: 'POST', body: formData });
      const uploadJson = await uploadRes.json();
      if (!uploadJson.success) throw new Error(uploadJson.message || 'Ошибка загрузки изображения');
      imagePath     = uploadJson.data.image;
      imageFullPath = uploadJson.data.image_full;
    }

    const res = await fetch('/api/events.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title:       createState.title.trim(),
        description: createState.description.trim() || null,
        badge:       createState.badge || null,
        date:        createState.date,
        image:       imagePath,
        image_full:  imageFullPath,
      }),
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка создания');

    createOpen.value = false;
    resetCreateForm();
    await fetchEvents();
  } catch (e: any) {
    createError.value = e.message ?? 'Ошибка при создании мероприятия';
  } finally {
    createSubmitting.value = false;
  }
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
      <UAlert
        v-if="fetchError"
        color="error"
        variant="subtle"
        icon="i-lucide-alert-circle"
        :description="fetchError"
        class="mb-4"
      >
        <template #footer>
          <UButton size="sm" color="error" variant="ghost" @click="fetchEvents">
            Повторить
          </UButton>
        </template>
      </UAlert>

      <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
        <USkeleton v-for="i in 6" :key="i" class="h-64 rounded-2xl" />
      </div>

      <div v-else-if="!fetchError && sortedPosts.length === 0" class="flex flex-col items-center justify-center h-48 gap-3 text-muted">
        <UIcon name="i-lucide-calendar-x" class="text-4xl" />
        <p class="text-sm">Мероприятия не найдены</p>
      </div>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
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

    <USlideover v-model:open="createOpen" side="right" title="Новое мероприятие" description="">
      <template #body>
        <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
          <UFormField label="Название мероприятия" name="title" required>
            <UInput v-model="createState.title" size="xl" class="w-full" placeholder="Введите название мероприятия" />
          </UFormField>

          <UFormField label="Категория" name="badge">
            <USelect v-model="createState.badge" :items="createBadgeOptions" placeholder="Выберите: Новое или Архив" size="xl" class="w-full" />
          </UFormField>

          <UFormField label="Описание" name="description">
            <UTextarea v-model="createState.description" size="xl" class="w-full" :rows="3"
              placeholder="Кратко опишите цель и формат мероприятия..." />
          </UFormField>

          <UFormField label="Дата проведения" name="date" required>
            <UInputDate v-model="createDateValue" size="xl" class="w-full">
              <template #trailing>
                <UPopover>
                  <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату" class="px-0" />
                  <template #content>
                    <UCalendar v-model="createDateValue" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>

          <UFormField label="Изображение" name="image">
            <div class="flex flex-col gap-2 w-full">
              <input id="createFileInput" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="onCreateImageSelected" />
              <img v-if="createImagePreview" :src="createImagePreview" alt="Предпросмотр" class="w-full h-40 object-cover rounded-lg" />
              <label for="createFileInput" class="flex items-center justify-center gap-2 cursor-pointer rounded-lg border border-default px-4 py-2.5 text-sm font-medium text-default hover:bg-elevated/50 transition-colors">
                <UIcon name="i-lucide-upload" class="text-base shrink-0" />
                {{ createImageFile ? 'Изменить фото' : 'Загрузить фото' }}
              </label>
              <p v-if="createImageFile" class="text-xs text-muted truncate px-1">{{ createImageFile.name }}</p>
            </div>
          </UFormField>

          <UAlert v-if="createError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="createError" />
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="createOpen = false">Отмена</UButton>
          <UButton size="xl" class="w-full justify-center" :loading="createSubmitting" @click="handleCreateSubmit">Создать</UButton>
        </div>
      </template>
    </USlideover>
  </UMain>
</template>