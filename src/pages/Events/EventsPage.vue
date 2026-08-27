
<script setup lang="ts">
import { computed, ref, watch, reactive, onMounted, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import type { BlogPostProps } from '@nuxt/ui';
import { useSectionAccess } from '../../composables/useSectionAccess';
import { apiSessionFetch } from '../../composables/useAuthSession';
import { formatDateRuLong } from '../../utils/date';
import { useGalleryData } from '../../composables/useGalleryData';
import { slideoverPopoverContent, slideoverSelectContent } from '../../composables/slideoverFieldUi';

type EventPost = BlogPostProps & {
  id?: number;
  badge?: string;
  rawDate?: string;
};

// ─── State ───────────────────────────────────────────────────────────────────
const route = useRoute();
const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const { albums, ensureLoaded: ensureAlbumsLoaded } = useGalleryData();
ensureAlbumsLoaded();

const albumSelectItems = computed(() => [
  { label: 'Без альбома', value: '' },
  ...albums.value.map((a) => ({ label: a.title, value: String(a.id) })),
]);

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
  const rawDate = String(raw?.date ?? '').trim();
  return {
    id:          raw.id,
    title:       raw.title,
    description: firstSentence(raw.description ?? ''),
    badge:       raw.badge ?? undefined,
    rawDate,
    date:        formatDateRuLong(rawDate) || rawDate,
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
    if (sortKey.value === 'newest')     return String(b.rawDate ?? '').localeCompare(String(a.rawDate ?? ''));
    if (sortKey.value === 'oldest')     return String(a.rawDate ?? '').localeCompare(String(b.rawDate ?? ''));
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
  albumId: string;
};

const createState = reactive<CreateFormState>({
  title: '',
  description: '',
  badge: null,
  date: '',
  albumId: '',
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
  if (!canEditSection('events')) return [];
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
  createState.albumId = '';
  createImageFile.value = null;
  if (createImagePreview.value) URL.revokeObjectURL(createImagePreview.value);
  createImagePreview.value = '';
  createDateValue.value = null;
  createError.value = null;
}

function openCreate() {
  ensureAlbumsLoaded();
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
      const uploadRes = await fetch('/api/Upload/upload.php', { method: 'POST', body: formData });
      const uploadJson = await uploadRes.json();
      if (!uploadJson.success) throw new Error(uploadJson.message || 'Ошибка загрузки изображения');
      imagePath     = uploadJson.data.image;
      imageFullPath = uploadJson.data.image_full;
    }

    const json = await apiSessionFetch('/api/events.php', {
      method: 'POST',
      json: {
        title:       createState.title.trim(),
        description: createState.description.trim() || null,
        badge:       createState.badge || null,
        date:        createState.date,
        image:       imagePath,
        image_full:  imageFullPath,
        album_id:    createState.albumId ? Number(createState.albumId) : null,
      },
    });
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

// ── Floating header on scroll up (same as GalleryPage) ────────────────────────
const mainScrollEl = ref<HTMLElement | null>(null);
const showFloatingHeader = ref(false);
let lastScrollTop = 0;
let rafPending = false;

const FLOATING_SHOW_AT = 180;
const FLOATING_HIDE_AT = 0;

const floatingRect = reactive({ left: 0, top: 0, width: 0 });
function updateFloatingRect() {
  const el = mainScrollEl.value;
  if (!el) return;
  const r = el.getBoundingClientRect();
  floatingRect.left = Math.round(r.left);
  floatingRect.top = Math.round(r.top);
  floatingRect.width = Math.round(r.width);
}

function onMainScroll() {
  const el = mainScrollEl.value;
  if (!el) return;
  if (rafPending) return;
  rafPending = true;
  requestAnimationFrame(() => {
    rafPending = false;
    const top = el.scrollTop;
    const goingUp = top < lastScrollTop;
    const goingDown = top > lastScrollTop;

    if (top < FLOATING_HIDE_AT) {
      showFloatingHeader.value = false;
    } else if (goingUp && top > FLOATING_SHOW_AT) {
      showFloatingHeader.value = true;
    } else if (goingDown) {
      showFloatingHeader.value = false;
    }

    lastScrollTop = top;
    if (showFloatingHeader.value) updateFloatingRect();
  });
}

onMounted(() => {
  const el = mainScrollEl.value;
  if (!el) return;
  lastScrollTop = el.scrollTop;
  el.addEventListener('scroll', onMainScroll, { passive: true });
  updateFloatingRect();
  window.addEventListener('resize', updateFloatingRect, { passive: true });
});

onUnmounted(() => {
  const el = mainScrollEl.value;
  if (el) el.removeEventListener('scroll', onMainScroll);
  window.removeEventListener('resize', updateFloatingRect as any);
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <!-- Floating header aligned to page container -->
    <transition name="fade">
      <div
        v-if="showFloatingHeader"
        class="fixed z-30"
        :style="{ left: `${floatingRect.left}px`, top: `${floatingRect.top}px`, width: `${floatingRect.width}px` }"
      >
        <div class="bg-default p-0 pb-6 flex flex-col gap-6">
          <div v-if="isKiosk" class="flex items-center justify-between gap-4 w-full">
            <h1 class="text-4xl font-normal font-unbounded">Мероприятия</h1>
            <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" class="shrink-0" />
          </div>
          <template v-else>
            <UPageHeader :links="headerLinks" class="border-none p-0 w-full">
              <template #title>
                <h1 class="text-4xl font-normal font-unbounded">Мероприятия</h1>
              </template>
            </UPageHeader>

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
              <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" />
            </UContainer>
          </template>

          <p v-if="!isKiosk && !loading && filteredPosts.length !== posts.length" class="text-sm text-muted -mt-2">
            Найдено: {{ filteredPosts.length }} из {{ posts.length }}
          </p>
        </div>
      </div>
    </transition>

    <!-- Single scroll container for the whole page -->
    <div ref="mainScrollEl" class="flex flex-col w-full h-full min-h-0 gap-6 overflow-y-auto scrollbar-hide">
      <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
        <div v-if="isKiosk" class="flex items-center justify-between gap-4 w-full">
          <h1 class="text-4xl font-normal font-unbounded">Мероприятия</h1>
          <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" class="shrink-0" />
        </div>
        <template v-else>
          <UPageHeader :links="headerLinks" class="border-none p-0 w-full">
            <template #title>
              <h1 class="text-4xl font-normal font-unbounded">Мероприятия</h1>
            </template>
          </UPageHeader>

          <UContainer class="flex flex-col max-w-full w-full sm:flex-row gap-3 items-stretch sm:items-center sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
            <UInput v-model="searchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline" placeholder="Поиск по названию..." class="flex-1" />
            <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" />
          </UContainer>
        </template>

        <p v-if="!isKiosk && !loading && filteredPosts.length !== posts.length" class="text-sm text-muted -mt-2">
          Найдено: {{ filteredPosts.length }} из {{ posts.length }}
        </p>
      </UContainer>

      <UContainer class="sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px mx-0 flex-1 min-h-0">
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
            <template #title>
              <span class="overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]">
                {{ post.title }}
              </span>
            </template>
            <template #description>
              <span class="overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]">
                {{ post.description }}
              </span>
            </template>
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
    </div>

    <USlideover v-model:open="createOpen" side="right" title="Новое мероприятие" description="">
      <template #body>
        <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
          <UFormField label="Название мероприятия" name="title" required>
            <UInput v-model="createState.title" size="xl" class="w-full" placeholder="Введите название мероприятия" />
          </UFormField>

          <UFormField label="Категория" name="badge">
            <USelect
              v-model="createState.badge"
              :content="slideoverSelectContent"
              :items="createBadgeOptions"
              placeholder="Выберите: Новое или Архив"
              size="xl"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Описание" name="description">
            <UTextarea v-model="createState.description" size="xl" class="w-full" :rows="3"
              placeholder="Кратко опишите цель и формат мероприятия..." />
          </UFormField>

          <UFormField label="Дата проведения" name="date" required>
            <UInputDate v-model="createDateValue" size="xl" class="w-full">
              <template #trailing>
                <UPopover :content="slideoverPopoverContent">
                  <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату" class="px-0" />
                  <template #content>
                    <UCalendar v-model="createDateValue" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>

          <UFormField label="Альбом фотогалереи" name="albumId">
            <USelectMenu
              v-model="createState.albumId"
              :content="slideoverSelectContent"
              :items="albumSelectItems"
              value-key="value"
              label-key="label"
              placeholder="Выберите альбом"
              size="xl"
              :search-input="false"
              class="w-full"
            />
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

<style>
:global([data-reka-select-content]),
:global([data-reka-popover-content]),
:global([data-reka-combobox-content]) {
  z-index: 100 !important;
}
</style>