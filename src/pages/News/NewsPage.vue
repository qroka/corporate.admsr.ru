<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { useRoute, useRouter } from 'vue-router';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';
import { useAppToast } from '../../composables/useAppToast';
import { currentRole } from '../../stores/role';

type NewsPost = BlogPostProps & { id: string; rawDate: string };

const route = useRoute();
const router = useRouter();
const newsDetailPrefix = computed(() =>
  route.matched?.some((r) => r.meta?.kiosk) ? '/kiosk/news/' : '/news/',
);

const { loading, error, sortedNews, ensureLoaded, reload } = useNewsData();
ensureLoaded();

const { toast } = useAppToast();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить новости',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const isAdmin = computed(() => currentRole.value === 'admin');

const searchQuery = ref('');
const sortKey = ref<'newest' | 'oldest' | 'title-asc' | 'title-desc'>('newest');
const sortOptions = [
  { value: 'newest',     label: 'Сначала новые' },
  { value: 'oldest',     label: 'Сначала старые' },
  { value: 'title-asc',  label: 'По названию (А‑Я)' },
  { value: 'title-desc', label: 'По названию (Я‑А)' },
];

const postsBase = computed<NewsPost[]>(() =>
  sortedNews.value.map((n) => {
    const imageSrc = resolveNewsImageSrc(n.imagePath);
    return {
      id:          n.id,
      rawDate:     n.date,
      title:       n.title || `Новость #${n.id}`,
      description: n.description.slice(0, 240),
      date:        formatNewsDate(n.date),
      badge:       n.category || undefined,
      to:          `${newsDetailPrefix.value}${n.id}`,
      image:       imageSrc ? { src: imageSrc, alt: n.title } : { src: '/src/img/Logo.svg', alt: n.title },
    };
  }),
);

const filtered = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  const base = postsBase.value;
  const items = q
    ? base.filter((p) => {
        const t = String(p.title ?? '').toLowerCase();
        const d = String(p.description ?? '').toLowerCase();
        return t.includes(q) || d.includes(q) || p.id.includes(q);
      })
    : base.slice();

  items.sort((a, b) => {
    if (sortKey.value === 'newest')     return String(b.rawDate ?? '').localeCompare(String(a.rawDate ?? ''));
    if (sortKey.value === 'oldest')     return String(a.rawDate ?? '').localeCompare(String(b.rawDate ?? ''));
    if (sortKey.value === 'title-asc')  return String(a.title ?? '').localeCompare(String(b.title ?? ''), 'ru-RU');
    return String(b.title ?? '').localeCompare(String(a.title ?? ''), 'ru-RU');
  });

  return items;
});

// ─── Create form ──────────────────────────────────────────────────────────────
const createBadgeOptions = [
  { value: 'Новости',      label: 'Новости' },
  { value: 'Мероприятия',  label: 'Мероприятия' },
  { value: 'Объявления',   label: 'Объявления' },
  { value: 'Архив',        label: 'Архив' },
];

const createOpen = ref(false);

type CreateFormState = {
  title:       string;
  category:    string | null;
  description: string;
  date:        string;
};

const createState = reactive<CreateFormState>({
  title:       '',
  category:    null,
  description: '',
  date:        '',
});

const createImageFile    = ref<File | null>(null);
const createImagePreview = ref('');

function onCreateImageSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  createImageFile.value = file;
  if (createImagePreview.value) URL.revokeObjectURL(createImagePreview.value);
  createImagePreview.value = file ? URL.createObjectURL(file) : '';
}

type SingleDateValue = { value?: any } | any | null;
const createDateValue  = ref<SingleDateValue>(null);
const createSubmitting = ref(false);
const createError      = ref<string | null>(null);

watch(createDateValue, (val) => {
  const d = val?.value ?? val;
  createState.date = d && typeof d.toString === 'function' ? d.toString() : '';
});

function resetCreateForm() {
  createState.title       = '';
  createState.category    = null;
  createState.description = '';
  createState.date        = '';
  createImageFile.value   = null;
  if (createImagePreview.value) URL.revokeObjectURL(createImagePreview.value);
  createImagePreview.value = '';
  createDateValue.value    = null;
  createError.value        = null;
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
  createError.value      = null;
  try {
    let imagePath: string | null = null;

    if (createImageFile.value) {
      const formData = new FormData();
      formData.append('image', createImageFile.value);
      const uploadRes  = await fetch('/api/Upload/upload.php', { method: 'POST', body: formData });
      const uploadJson = await uploadRes.json();
      if (!uploadJson.success) throw new Error(uploadJson.message || 'Ошибка загрузки изображения');
      imagePath = uploadJson.data.image as string;
    }

    const res = await fetch('/api/news.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title:       createState.title.trim(),
        category:    createState.category || '',
        description: createState.description.trim(),
        date:        createState.date,
        image_path:  imagePath,
      }),
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка создания');

    toast.add({ title: 'Новость создана', color: 'success', icon: 'i-lucide-check-circle' });
    createOpen.value = false;
    resetCreateForm();
    await reload();
  } catch (e: any) {
    createError.value = e.message ?? 'Ошибка при создании новости';
  } finally {
    createSubmitting.value = false;
  }
}
</script>

<template>
  <UMain class="relative flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
      <UPageHeader class="border-none p-0 w-full">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Новости</h1>
        </template>
        <template #links>
          <UButton
            v-if="isAdmin"
            label="Добавить новость"
            color="neutral"
            variant="outline"
            size="xl"
            icon="i-lucide-plus"
            @click="openCreate"
          />
        </template>
      </UPageHeader>

      <UContainer class="flex flex-col max-w-full w-full sm:flex-row gap-3 items-stretch sm:items-center sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
        <UInput
          v-model="searchQuery"
          icon="i-lucide-search"
          size="xl"
          color="neutral"
          variant="outline"
          placeholder="Поиск по новостям..."
          class="flex-1"
        />
        <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" />
      </UContainer>

      <p v-if="!loading && filtered.length !== postsBase.length" class="text-sm text-muted -mt-2">
        Найдено: {{ filtered.length }} из {{ postsBase.length }}
      </p>
    </UContainer>

    <div class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UContainer class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:p-0 max-w-full w-full md:p-0 lg:p-0 xl:p-0 mx-0">
        <USkeleton v-if="loading" class="h-28 w-full" />
        <UBlogPost
          v-for="post in filtered"
          v-else
          :key="post.id"
          v-bind="post"
          class="h-full max-w-full w-full"
        >
          <template #title>
            <h3
              class="text-base text-pretty font-semibold text-highlighted overflow-hidden [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]"
            >
              {{ post.title }}
            </h3>
          </template>
          <template #description>
            <p
              class="text-sm text-muted text-pretty overflow-hidden [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]"
            >
              {{ post.description }}
            </p>
          </template>
        </UBlogPost>
      </UContainer>

      <UEmpty
        v-if="!loading && !filtered.length"
        icon="i-lucide-newspaper"
        title="Новостей не найдено"
        description="Попробуйте изменить запрос или сортировку."
        class="mt-3"
      />
    </div>

    <USlideover v-model:open="createOpen" side="right" title="Новое мероприятие">
      <template #body>
        <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
          <UFormField label="Название мероприятия" name="title" required>
            <UInput v-model="createState.title" size="xl" class="w-full" placeholder="Введите название мероприятия" />
          </UFormField>

          <UFormField label="Категория" name="category">
            <USelect v-model="createState.category" :items="createBadgeOptions" placeholder="Выберите категорию" size="xl" class="w-full" />
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