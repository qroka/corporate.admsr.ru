<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { useSectionAccess } from '../../composables/useSectionAccess';
import { apiSessionFetch, apiSessionUpload } from '../../composables/useAuthSession';
import { useGalleryData } from '../../composables/useGalleryData';
import { useAppToast } from '../../composables/useAppToast';
import { formatDateRuLong } from '../../utils/date';
import { useCursorFeed } from '../../composables/useCursorFeed';
import { useFeedSentinel } from '../../composables/useFeedSentinel';

type Album = {
  id: string;
  title: string;
  description: string;
  date: string;
  rawDate: string;
  to: string;
  coverSrc: string;
  photoCount: number;
};

const GALLERY_PAGE_LIMIT = 12;

function isVideo(url: string): boolean {
  return /\.mp4(\?|$)/i.test(url ?? '');
}

const albumPlaceholder = (() => {
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="800" height="450" viewBox="0 0 800 450">` +
    `<rect width="800" height="450" fill="#1e293b"/>` +
    `<text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" ` +
    `fill="#94a3b8" font-family="sans-serif" font-size="72" font-weight="700" letter-spacing="10">АЛЬБОМ</text>` +
    `</svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
})();

function albumCover(image?: string): string {
  return image && !isVideo(image) ? image : albumPlaceholder;
}

function photosLabel(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return `${n} фотография`;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${n} фотографии`;
  return `${n} фотографий`;
}

const route = useRoute();
const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));

const { reload: reloadGalleryCache } = useGalleryData();

const searchQuery = ref('');
const searchForApi = ref('');
let searchTimer: ReturnType<typeof setTimeout> | null = null;
watch(searchQuery, (q) => {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    searchForApi.value = q.trim();
  }, 300);
});

const sortKey = ref<'newest' | 'oldest' | 'title-asc' | 'title-desc'>('newest');
const sortOptions = [
  { value: 'newest', label: 'Сначала новые' },
  { value: 'oldest', label: 'Сначала старые' },
  { value: 'title-asc', label: 'По названию (А‑Я)' },
  { value: 'title-desc', label: 'По названию (Я‑А)' },
];

function mapAlbum(raw: any): Album {
  const rawDate = String(raw?.date ?? '').slice(0, 10);
  return {
    id: String(raw.id),
    title: String(raw.name ?? ''),
    description: String(raw.description ?? ''),
    rawDate,
    date: formatDateRuLong(rawDate) || rawDate,
    to: `/gallery/${raw.id}`,
    coverSrc: albumCover(raw.cover ?? undefined),
    photoCount: Number(raw.photo_count ?? 0) || 0,
  };
}

const {
  items: albumItems,
  loading: feedLoading,
  initialLoading,
  error,
  hasMore,
  loadInitial,
  loadMore,
  refresh,
  sentinelEnabled,
} = useCursorFeed<Album>({
  buildUrl: (cursor) => {
    const params = new URLSearchParams();
    params.set('limit', String(GALLERY_PAGE_LIMIT));
    if (cursor) params.set('cursor', cursor);
    if (searchForApi.value) params.set('search', searchForApi.value);
    return `/api/gallery.php?${params.toString()}`;
  },
  mapItem: (raw) => mapAlbum(raw),
  getId: (item) => item.id,
});

const loading = computed(() => initialLoading.value);

watch(searchForApi, () => {
  void loadInitial(true);
});

const { toast } = useAppToast();
const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить альбомы',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const filteredAlbums = computed(() => {
  const list = [...albumItems.value];
  return list.sort((a, b) => {
    if (sortKey.value === 'newest') return (b.rawDate ?? '').localeCompare(a.rawDate ?? '');
    if (sortKey.value === 'oldest') return (a.rawDate ?? '').localeCompare(b.rawDate ?? '');
    if (sortKey.value === 'title-asc') return (a.title ?? '').localeCompare(b.title ?? '', 'ru');
    if (sortKey.value === 'title-desc') return (b.title ?? '').localeCompare(a.title ?? '', 'ru');
    return 0;
  });
});

const createOpen = ref(false);
const createSubmitting = ref(false);
const createFiles = ref<File[] | undefined>(undefined);
type SingleDateValue = { value?: any } | any | null;
const createDateValue = ref<SingleDateValue>(null);
const createErrors = reactive({ title: '', general: '' });
const createState = reactive({ title: '', description: '', date: '' });

function resetCreateForm() {
  createState.title = '';
  createState.description = '';
  createState.date = '';
  createDateValue.value = null;
  createFiles.value = undefined;
  createErrors.title = '';
  createErrors.general = '';
}

function openCreate() {
  resetCreateForm();
  createOpen.value = true;
}

const headerLinks = computed(() => {
  if (!canEditSection('gallery') || isKiosk.value) return [];
  return [
    {
      label: 'Добавить альбом',
      color: 'neutral',
      variant: 'outline',
      size: 'md',
      onClick: openCreate,
    },
  ];
});

watch(createDateValue, (val) => {
  const d = val?.value ?? val;
  createState.date = d && typeof d.toString === 'function' ? d.toString() : '';
});

function validateCreate() {
  createErrors.title = '';
  createErrors.general = '';
  if (!createState.title.trim()) createErrors.title = 'Заполните название альбома.';
  return !createErrors.title;
}

async function handleCreateSubmit() {
  if (!validateCreate()) return;
  createSubmitting.value = true;
  try {
    const json = await apiSessionFetch('/api/gallery.php', {
      method: 'POST',
      json: {
        name: createState.title.trim(),
        description: createState.description.trim() || null,
        date: createState.date || null,
      },
    });
    if (!json.success) throw new Error(json.message || 'Ошибка создания альбома');

    const newAlbumId = (json.data as any).id;

    for (const file of createFiles.value ?? []) {
      const fd = new FormData();
      fd.append('image', file);
      fd.append('album_id', String(newAlbumId));
      await apiSessionUpload('/api/gallery_base.php', fd);
    }

    await refresh();
    void reloadGalleryCache();
    createOpen.value = false;
    resetCreateForm();
    toast.add({
      title: 'Альбом создан',
      color: 'success',
      icon: 'i-lucide-check-circle',
    });
  } catch (e: any) {
    createErrors.general = e.message ?? 'Ошибка при создании альбома';
  } finally {
    createSubmitting.value = false;
  }
}

const mainScrollEl = ref<HTMLElement | null>(null);
const gallerySentinelEl = ref<HTMLElement | null>(null);

useFeedSentinel({
  root: mainScrollEl,
  sentinel: gallerySentinelEl,
  enabled: sentinelEnabled,
  onIntersect: () => {
    void loadMore();
  },
  rootMargin: '600px 0px',
});

onMounted(() => {
  void loadInitial();
});

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer);
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div
      ref="mainScrollEl"
      class="flex flex-col w-full h-full min-h-0 gap-6 overflow-y-auto scrollbar-hide p-px"
    >
      <div class="flex flex-col gap-6 w-full max-w-[1600px] mx-auto">
        <UPageHeader
          headline="Корпоративная жизнь"
          title="Фотогалерея"
          description="Альбомы корпоративных событий и встреч"
          :links="headerLinks"
        />

        <div
          v-if="!isKiosk"
          class="flex flex-col sm:flex-row gap-3 items-stretch sm:items-center"
        >
          <UInput
            v-model="searchQuery"
            icon="i-lucide-search"
            size="md"
            color="neutral"
            variant="outline"
            placeholder="Найти альбом..."
            class="w-full min-w-0 flex-1"
          />
          <USelect
            v-model="sortKey"
            :items="sortOptions"
            size="md"
            color="neutral"
            class="w-full sm:w-56 shrink-0"
          />
        </div>

        <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <USkeleton v-for="i in 6" :key="i" class="h-64 rounded-panel" />
        </div>

        <UEmpty
          v-else-if="!filteredAlbums.length"
          variant="naked"
          icon="i-lucide-images"
          title="Альбомы не найдены"
          description="Измените поиск или создайте новый альбом."
          class="w-full py-10"
        />

        <div
          v-else
          class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3"
        >
          <RouterLink
            v-for="album in filteredAlbums"
            :key="album.id"
            :to="album.to"
            class="group flex flex-col gap-3 rounded-panel p-px transition ring-1 ring-transparent hover:ring-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <div class="relative aspect-[16/10] overflow-hidden rounded-panel bg-elevated">
              <img
                :src="album.coverSrc"
                :alt="album.title"
                loading="lazy"
                decoding="async"
                class="size-full object-cover transition duration-300 group-hover:scale-[1.02]"
              />
              <span
                class="absolute bottom-2 right-2 inline-flex items-center gap-1.5 rounded-md bg-black/65 px-2 py-1 text-xs text-white tabular-nums backdrop-blur-sm"
              >
                <UIcon name="i-lucide-images" class="size-3.5 shrink-0" />
                {{ photosLabel(album.photoCount) }}
              </span>
            </div>
            <div class="flex min-w-0 flex-col gap-1 px-0.5">
              <p v-if="album.date" class="text-sm text-muted">
                {{ album.date }}
              </p>
              <h2 class="text-base font-semibold text-highlighted line-clamp-2 text-pretty">
                {{ album.title }}
              </h2>
            </div>
          </RouterLink>
        </div>

        <div
          v-if="!loading && feedLoading"
          class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3"
        >
          <USkeleton v-for="i in 3" :key="`more-${i}`" class="h-64 rounded-panel" />
        </div>

        <div
          v-if="filteredAlbums.length"
          ref="gallerySentinelEl"
          class="h-1 w-full shrink-0"
          aria-hidden="true"
        />
      </div>
    </div>

    <USlideover
      v-model:open="createOpen"
      side="right"
      title="Новый альбом"
      description="Заполните данные альбома"
    >
      <template #body>
        <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
          <UFormField
            label="Название альбома"
            name="title"
            :error="createErrors.title || undefined"
            required
          >
            <UInput
              v-model="createState.title"
              size="xl"
              class="w-full"
              placeholder="Введите название альбома"
            />
          </UFormField>

          <UFormField label="Описание" name="description">
            <UTextarea
              v-model="createState.description"
              size="xl"
              class="w-full"
              :rows="3"
              placeholder="Кратко опишите альбом..."
            />
          </UFormField>

          <UFormField label="Дата альбома" name="date">
            <UInputDate v-model="createDateValue" size="xl" class="w-full">
              <template #trailing>
                <UPopover>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="md"
                    icon="i-lucide-calendar"
                    aria-label="Выбрать дату"
                    class="px-0"
                  />
                  <template #content>
                    <UCalendar v-model="createDateValue" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>

          <UFormField label="Фото и видео (необязательно)" name="files">
            <UFileUpload
              v-model="createFiles"
              multiple
              accept="image/jpeg,image/png,image/webp,video/mp4"
              label="Перетащите файлы сюда"
              description="JPG, PNG, WEBP или MP4 (до 200 МБ). Можно выбрать несколько."
              class="w-full min-h-48 rounded-lg"
            />
          </UFormField>

          <UAlert
            v-if="createErrors.general"
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :description="createErrors.general"
          />
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton
            type="button"
            color="neutral"
            variant="outline"
            size="md"
            class="w-full justify-center"
            @click="createOpen = false"
          >
            Отмена
          </UButton>
          <UButton
            type="button"
            size="md"
            class="w-full justify-center"
            :loading="createSubmitting"
            @click="handleCreateSubmit"
          >
            Создать альбом
          </UButton>
        </div>
      </template>
    </USlideover>
  </UMain>
</template>
