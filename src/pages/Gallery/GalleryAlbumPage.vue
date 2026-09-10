<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { DropdownMenuItem } from '@nuxt/ui';
import { useGalleryData } from '../../composables/useGalleryData';
import { useAppToast } from '../../composables/useAppToast';
import { useSectionAccess } from '../../composables/useSectionAccess';
import { apiSessionFetch, apiSessionUpload } from '../../composables/useAuthSession';
import { useCursorFeed } from '../../composables/useCursorFeed';
import { useFeedSentinel } from '../../composables/useFeedSentinel';
import { useBreadcrumbCurrentLabel } from '../../composables/usePortalNavigation';
import { formatDateRuLong } from '../../utils/date';

type Photo = { id: string; thumbSrc: string; fullSrc: string };

const ALBUM_PHOTOS_LIMIT = 36;

const route = useRoute();
const router = useRouter();

const albumId = computed(() => String(route.params.albumId ?? ''));

const { ensureLoaded } = useGalleryData();
ensureLoaded();

const { toast } = useAppToast();
const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const isAdmin = computed(() => canEditSection('gallery'));
const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));

const albumData = ref<{
  id: number;
  name: string;
  description: string;
  date: string;
  photo_count?: number;
} | null>(null);
const albumLoading = ref(false);
const relatedEventId = ref<number | null>(null);

const albumTitle = computed(() => albumData.value?.name ?? '');
const albumDescription = computed(() => albumData.value?.description ?? '');

const breadcrumbLabel = useBreadcrumbCurrentLabel();
watch(
  albumTitle,
  (title) => {
    breadcrumbLabel.value = title.trim() || null;
  },
  { immediate: true },
);
const albumDateLabel = computed(() => {
  const raw = String(albumData.value?.date ?? '').slice(0, 10);
  return formatDateRuLong(raw) || raw || '—';
});

function photosLabel(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return `${n} фотография`;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${n} фотографии`;
  return `${n} фотографий`;
}

const albumTag = computed(() => {
  const text = `${albumTitle.value} ${albumDescription.value}`.toLowerCase();
  if (/встреч/.test(text)) return 'Встреча';
  if (/мероприят/.test(text)) return 'Мероприятие';
  if (/праздник|новый год|корпоратив/.test(text)) return 'Праздник';
  if (/обучен|семинар|лекц/.test(text)) return 'Обучение';
  return null;
});

async function loadAlbum() {
  albumLoading.value = true;
  relatedEventId.value = null;
  try {
    const res = await fetch(`/api/gallery.php?id=${albumId.value}`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message);
    albumData.value = json.data;

    try {
      const evRes = await fetch('/api/events.php');
      const evJson = await evRes.json();
      if (evJson.success && Array.isArray(evJson.data)) {
        const match = evJson.data.find(
          (e: any) => e?.album_id != null && String(e.album_id) === albumId.value,
        );
        relatedEventId.value = match?.id != null ? Number(match.id) : null;
      }
    } catch {
      // optional link
    }
  } catch (e: any) {
    toast.add({
      title: 'Не удалось загрузить альбом',
      description: e.message,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    albumLoading.value = false;
  }
}

const {
  items,
  loading: feedLoading,
  initialLoading: photosInitialLoading,
  loadInitial: loadPhotosInitial,
  loadMore: loadMorePhotos,
  refresh: refreshPhotos,
  sentinelEnabled: photosSentinelEnabled,
  hasMore: photosHasMore,
} = useCursorFeed<Photo>({
  buildUrl: (cursor) => {
    const params = new URLSearchParams();
    params.set('album_id', albumId.value);
    params.set('limit', String(ALBUM_PHOTOS_LIMIT));
    if (cursor) params.set('cursor', cursor);
    return `/api/gallery_base.php?${params.toString()}`;
  },
  mapItem: (raw: any) => {
    if (!raw?.id) return null;
    return {
      id: String(raw.id),
      thumbSrc: String(raw.image_small_url ?? ''),
      fullSrc: String(raw.image_full_url ?? ''),
    };
  },
  getId: (item) => item.id,
});

const photosLoading = computed(() => photosInitialLoading.value);

const thumbLoaded = reactive<Record<string, boolean>>({});
function onThumbLoad(id: string) {
  thumbLoaded[id] = true;
}

watch(albumId, () => {
  for (const k of Object.keys(thumbLoaded)) delete thumbLoaded[k];
  void loadAlbum();
  void loadPhotosInitial(true);
});

async function loadPhotos() {
  await refreshPhotos();
}

const photoSort = ref<'order' | 'reverse'>('order');
const photoSortOptions = [
  { value: 'order', label: 'По порядку' },
  { value: 'reverse', label: 'В обратном порядке' },
];

const gridDensity = ref<'sm' | 'md' | 'lg' | 'xl'>('md');
const gridClass = computed(() => {
  switch (gridDensity.value) {
    case 'sm':
      return 'grid-cols-2 sm:grid-cols-3 xl:grid-cols-5';
    case 'lg':
      return 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3';
    case 'xl':
      return 'grid-cols-1 sm:grid-cols-2';
    default:
      return 'grid-cols-2 sm:grid-cols-3 xl:grid-cols-4';
  }
});

const displayPhotos = computed(() => {
  const list = [...items.value];
  return photoSort.value === 'reverse' ? list.reverse() : list;
});

const photoCount = computed(() => {
  const fromApi = Number(albumData.value?.photo_count ?? 0);
  return Math.max(fromApi, items.value.length);
});

const photoCountLabel = computed(() => photosLabel(photoCount.value));

const selected = ref<Photo | null>(null);
const modalOpen = computed({
  get: () => selected.value !== null,
  set: (open: boolean) => {
    if (!open) {
      stopSlideshow();
      selected.value = null;
    }
  },
});

function openPhoto(p: Photo) {
  selected.value = p;
}

function isVideo(url: string): boolean {
  return /\.mp4(\?|$)/i.test(url ?? '');
}

const selectedIndex = computed(() => {
  if (!selected.value) return -1;
  return displayPhotos.value.findIndex((p) => p.id === selected.value!.id);
});

function downloadPhoto(photo: Photo, e?: Event) {
  e?.preventDefault();
  e?.stopPropagation();
  const a = document.createElement('a');
  a.href = photo.fullSrc || photo.thumbSrc;
  a.download = `photo-${photo.id}`;
  a.target = '_blank';
  a.rel = 'noopener';
  a.click();
}

async function downloadAlbum() {
  const list = displayPhotos.value.filter((p) => !isVideo(p.fullSrc));
  if (!list.length) {
    toast.add({
      title: 'Нечего скачивать',
      description: 'В альбоме пока нет фотографий.',
      color: 'neutral',
      icon: 'i-lucide-info',
    });
    return;
  }
  toast.add({
    title: 'Скачивание',
    description: `Начинаем загрузку ${list.length} файлов…`,
    color: 'primary',
    icon: 'i-lucide-download',
  });
  for (const photo of list.slice(0, 30)) {
    downloadPhoto(photo);
    await new Promise((r) => setTimeout(r, 120));
  }
}

let slideshowTimer: ReturnType<typeof setInterval> | null = null;
const slideshowActive = ref(false);

function stopSlideshow() {
  if (slideshowTimer) {
    clearInterval(slideshowTimer);
    slideshowTimer = null;
  }
  slideshowActive.value = false;
}

function startSlideshow() {
  if (!displayPhotos.value.length) return;
  stopSlideshow();
  slideshowActive.value = true;
  if (!selected.value) selected.value = displayPhotos.value[0]!;
  slideshowTimer = setInterval(() => {
    const list = displayPhotos.value;
    if (!list.length) {
      stopSlideshow();
      return;
    }
    const idx = selected.value
      ? list.findIndex((p) => p.id === selected.value!.id)
      : -1;
    const next = list[(idx + 1) % list.length]!;
    selected.value = next;
  }, 3500);
}

function toggleSlideshow() {
  if (slideshowActive.value) stopSlideshow();
  else startSlideshow();
}

const addPhotosOpen = ref(false);
const addPhotoFiles = ref<File[] | undefined>(undefined);
const addSubmitting = ref(false);
const addError = ref<string | null>(null);

function openAddPhotos() {
  addPhotoFiles.value = undefined;
  addError.value = null;
  addPhotosOpen.value = true;
}

async function handleAddPhotos() {
  if (!addPhotoFiles.value?.length) {
    addError.value = 'Выберите хотя бы один файл.';
    return;
  }
  addSubmitting.value = true;
  addError.value = null;
  try {
    for (const file of addPhotoFiles.value) {
      const fd = new FormData();
      fd.append('image', file);
      fd.append('album_id', albumId.value);
      const json = await apiSessionUpload('/api/gallery_base.php', fd);
      if (!json.success) throw new Error(json.message || 'Ошибка загрузки файла');
    }
    await loadPhotos();
    await loadAlbum();
    addPhotosOpen.value = false;
    addPhotoFiles.value = undefined;
    toast.add({ title: 'Фото добавлены', color: 'success', icon: 'i-lucide-check-circle' });
  } catch (e: any) {
    addError.value = e.message ?? 'Ошибка загрузки';
  } finally {
    addSubmitting.value = false;
  }
}

const deletingPhotoId = ref<string | null>(null);

async function deletePhoto(photoId: string) {
  deletingPhotoId.value = photoId;
  try {
    await apiSessionFetch(`/api/gallery_base.php?id=${photoId}`, { method: 'DELETE' });
    items.value = items.value.filter((p) => p.id !== photoId);
    delete thumbLoaded[photoId];
    if (selected.value?.id === photoId) selected.value = null;
    await loadAlbum();
  } finally {
    deletingPhotoId.value = null;
  }
}

const editOpen = ref(false);
const editSubmitting = ref(false);
const editError = ref<string | null>(null);
const editState = ref({ title: '', description: '', date: '' });

function openEdit() {
  editState.value = {
    title: albumData.value?.name ?? '',
    description: albumData.value?.description ?? '',
    date: String(albumData.value?.date ?? '').slice(0, 10),
  };
  editError.value = null;
  editOpen.value = true;
}

async function handleEditSubmit() {
  if (!editState.value.title.trim()) {
    editError.value = 'Заполните название альбома.';
    return;
  }
  editSubmitting.value = true;
  editError.value = null;
  try {
    const json = await apiSessionFetch(`/api/gallery.php?id=${albumId.value}`, {
      method: 'PUT',
      json: {
        name: editState.value.title.trim(),
        description: editState.value.description.trim() || null,
        date: editState.value.date || null,
      },
    });
    if (!json.success) throw new Error(json.message);
    albumData.value = json.data as any;
    editOpen.value = false;
    toast.add({ title: 'Альбом обновлён', color: 'success', icon: 'i-lucide-check-circle' });
  } catch (e: any) {
    editError.value = e.message ?? 'Ошибка при сохранении';
  } finally {
    editSubmitting.value = false;
  }
}

const deleteConfirmOpen = ref(false);
const deleteSubmitting = ref(false);
const deleteError = ref<string | null>(null);

async function handleDelete() {
  deleteSubmitting.value = true;
  deleteError.value = null;
  try {
    const json = await apiSessionFetch(`/api/gallery.php?id=${albumId.value}`, { method: 'DELETE' });
    if (!json.success) throw new Error(json.message);
    deleteConfirmOpen.value = false;
    await router.push(isKiosk.value ? '/kiosk/gallery' : '/gallery');
  } catch (e: any) {
    deleteError.value = e.message ?? 'Ошибка при удалении';
  } finally {
    deleteSubmitting.value = false;
  }
}

const moreMenuItems = computed<DropdownMenuItem[][]>(() => {
  if (!isAdmin.value) return [];
  return [
    [
      {
        label: 'Редактировать',
        icon: 'i-lucide-pencil',
        onSelect: openEdit,
      },
      {
        label: 'Удалить альбом',
        icon: 'i-lucide-trash-2',
        color: 'error',
        onSelect: () => {
          deleteError.value = null;
          deleteConfirmOpen.value = true;
        },
      },
    ],
  ];
});

const headerLinks = computed(() => {
  if (isKiosk.value) return [];
  const links: Array<Record<string, unknown>> = [];
  if (isAdmin.value) {
    links.push({
      label: 'Добавить фото',
      icon: 'i-lucide-plus',
      color: 'neutral',
      variant: 'outline',
      size: 'md',
      onClick: openAddPhotos,
    });
  }
  links.push({
    label: 'Скачать альбом',
    icon: 'i-lucide-download',
    color: 'neutral',
    variant: 'outline',
    size: 'md',
    onClick: () => {
      void downloadAlbum();
    },
  });
  return links;
});

const mainScrollEl = ref<HTMLElement | null>(null);
const photosSentinelEl = ref<HTMLElement | null>(null);

useFeedSentinel({
  root: mainScrollEl,
  sentinel: photosSentinelEl,
  enabled: photosSentinelEnabled,
  onIntersect: () => {
    void loadMorePhotos();
  },
  rootMargin: '600px 0px',
});

onMounted(() => {
  void loadAlbum();
  void loadPhotos();
});

onUnmounted(() => {
  breadcrumbLabel.value = null;
  stopSlideshow();
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div
      ref="mainScrollEl"
      class="flex flex-col w-full h-full min-h-0 gap-6 overflow-y-auto scrollbar-hide p-px"
    >
      <div class="flex flex-col gap-6 w-full max-w-[1600px] mx-auto pb-8">
        <div class="flex flex-col gap-3">
          <UPageHeader
            headline="Фотоальбом"
            :title="albumLoading ? 'Загрузка…' : albumTitle"
            :links="headerLinks"
          >
            <template v-if="!isKiosk && moreMenuItems.length" #links>
              <UButton
                v-for="(link, index) in headerLinks"
                :key="index"
                v-bind="link"
              />
              <UDropdownMenu :items="moreMenuItems">
                <UButton
                  color="neutral"
                  variant="outline"
                  size="md"
                  icon="i-lucide-ellipsis"
                  square
                  aria-label="Ещё действия"
                />
              </UDropdownMenu>
            </template>
          </UPageHeader>

          <div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted">
            <span class="inline-flex items-center gap-1.5">
              <UIcon name="i-lucide-calendar" class="size-4 shrink-0" />
              {{ albumDateLabel }}
            </span>
            <span class="inline-flex items-center gap-1.5">
              <UIcon name="i-lucide-image" class="size-4 shrink-0" />
              {{ photoCountLabel }}
            </span>
            <UBadge v-if="albumTag" color="info" variant="subtle">
              {{ albumTag }}
            </UBadge>
            <RouterLink
              v-if="relatedEventId"
              :to="`/events/${relatedEventId}`"
              class="inline-flex items-center gap-1 text-primary hover:underline"
            >
              О мероприятии
              <UIcon name="i-lucide-arrow-up-right" class="size-3.5" />
            </RouterLink>
          </div>

          <p v-if="albumDescription" class="text-sm text-muted max-w-3xl text-pretty">
            {{ albumDescription }}
          </p>
        </div>

        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <p class="text-sm text-muted">{{ photoCountLabel }}</p>

          <div class="flex flex-wrap items-center gap-2">
            <USelect
              v-model="photoSort"
              :items="photoSortOptions"
              size="md"
              color="neutral"
              class="w-44"
            />

            <div class="flex items-center gap-1 rounded-lg ring-1 ring-default p-0.5">
              <UButton
                type="button"
                color="neutral"
                :variant="gridDensity === 'sm' ? 'soft' : 'ghost'"
                size="sm"
                icon="i-lucide-grid-3x3"
                square
                :class="gridDensity === 'sm' ? 'ring-1 ring-primary' : ''"
                aria-label="Плотная сетка"
                @click="gridDensity = 'sm'"
              />
              <UButton
                type="button"
                color="neutral"
                :variant="gridDensity === 'md' ? 'soft' : 'ghost'"
                size="sm"
                icon="i-lucide-layout-grid"
                square
                :class="gridDensity === 'md' ? 'ring-1 ring-primary' : ''"
                aria-label="Средняя сетка"
                @click="gridDensity = 'md'"
              />
              <UButton
                type="button"
                color="neutral"
                :variant="gridDensity === 'lg' ? 'soft' : 'ghost'"
                size="sm"
                icon="i-lucide-panels-top-left"
                square
                :class="gridDensity === 'lg' ? 'ring-1 ring-primary' : ''"
                aria-label="Крупная сетка"
                @click="gridDensity = 'lg'"
              />
              <UButton
                type="button"
                color="neutral"
                :variant="gridDensity === 'xl' ? 'soft' : 'ghost'"
                size="sm"
                icon="i-lucide-square"
                square
                :class="gridDensity === 'xl' ? 'ring-1 ring-primary' : ''"
                aria-label="Очень крупная сетка"
                @click="gridDensity = 'xl'"
              />
            </div>

            <UButton
              color="neutral"
              variant="outline"
              size="md"
              :icon="slideshowActive ? 'i-lucide-pause' : 'i-lucide-play'"
              :label="slideshowActive ? 'Стоп' : 'Слайд-шоу'"
              :disabled="!displayPhotos.length"
              @click="toggleSlideshow"
            />
          </div>
        </div>

        <div v-if="photosLoading" :class="['grid gap-3', gridClass]">
          <USkeleton v-for="i in 8" :key="i" class="aspect-[4/3] rounded-panel" />
        </div>

        <UEmpty
          v-else-if="!displayPhotos.length"
          variant="naked"
          icon="i-lucide-image-off"
          title="Нет фотографий"
          description="В этом альбоме пока нет снимков."
          class="w-full py-12"
        />

        <div v-else :class="['grid gap-3', gridClass]">
          <div
            v-for="(item, index) in displayPhotos"
            :key="item.id"
            class="group relative aspect-[4/3] rounded-panel bg-elevated ring-1 ring-transparent hover:ring-primary transition"
          >
            <div class="absolute inset-0 overflow-hidden rounded-[inherit]">
            <button
              type="button"
              class="absolute inset-0 block w-full h-full text-left"
              @click="openPhoto(item)"
            >
              <div
                v-if="!thumbLoaded[item.id]"
                class="absolute inset-0 animate-pulse bg-accented/50"
                aria-hidden="true"
              />
              <template v-if="isVideo(item.fullSrc)">
                <video
                  :src="`${item.thumbSrc}#t=0.1`"
                  muted
                  playsinline
                  preload="metadata"
                  class="absolute inset-0 size-full object-cover transition-opacity duration-300"
                  :class="thumbLoaded[item.id] ? 'opacity-100' : 'opacity-0'"
                  @loadeddata="onThumbLoad(item.id)"
                />
                <span class="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <span class="size-12 rounded-full bg-black/55 flex items-center justify-center">
                    <UIcon name="i-lucide-play" class="text-white text-2xl" />
                  </span>
                </span>
              </template>
              <img
                v-else
                :src="item.thumbSrc"
                alt="Фотография"
                loading="lazy"
                decoding="async"
                class="absolute inset-0 size-full object-cover transition-opacity duration-300"
                :class="thumbLoaded[item.id] ? 'opacity-100' : 'opacity-0'"
                @load="onThumbLoad(item.id)"
              />
            </button>
            </div>

            <div
              class="pointer-events-none absolute inset-0 opacity-0 group-hover:opacity-100 transition"
            >
              <div class="absolute top-2 right-2 flex gap-1 pointer-events-auto">
                <UButton
                  type="button"
                  color="neutral"
                  variant="solid"
                  size="xs"
                  icon="i-lucide-download"
                  square
                  class="bg-black/60 hover:bg-black/80 text-white ring-0"
                  aria-label="Скачать"
                  @click="downloadPhoto(item, $event)"
                />
                <UButton
                  type="button"
                  color="neutral"
                  variant="solid"
                  size="xs"
                  icon="i-lucide-expand"
                  square
                  class="bg-black/60 hover:bg-black/80 text-white ring-0"
                  aria-label="Открыть"
                  @click="openPhoto(item)"
                />
                <UButton
                  v-if="isAdmin"
                  type="button"
                  color="error"
                  variant="solid"
                  size="xs"
                  icon="i-lucide-trash-2"
                  square
                  class="bg-black/60 hover:bg-error text-white ring-0"
                  :disabled="deletingPhotoId === item.id"
                  aria-label="Удалить"
                  @click.stop="deletePhoto(item.id)"
                />
              </div>
              <span
                class="absolute bottom-2 left-2 rounded-md bg-black/60 px-2 py-0.5 text-xs text-white tabular-nums"
              >
                {{ index + 1 }} из {{ photoCount }}
              </span>
            </div>
          </div>
        </div>

        <div
          v-if="!photosLoading && feedLoading"
          :class="['grid gap-3', gridClass]"
        >
          <USkeleton v-for="i in 4" :key="`more-${i}`" class="aspect-[4/3] rounded-panel" />
        </div>

        <div
          v-if="displayPhotos.length"
          ref="photosSentinelEl"
          class="h-1 w-full shrink-0"
          aria-hidden="true"
        />
      </div>
    </div>

    <UModal
      v-model:open="modalOpen"
      class="p-0"
      :ui="{ content: 'bg-transparent shadow-none ring-0 w-auto max-w-[95vw]', header: 'hidden', body: 'p-0' }"
    >
      <template #content>
        <div v-if="selected" class="flex flex-col items-center justify-center gap-3 p-0">
          <video
            v-if="isVideo(selected.fullSrc)"
            :src="selected.fullSrc"
            controls
            autoplay
            playsinline
            class="block w-auto h-auto max-w-[95vw] max-h-[85vh] rounded-lg"
          />
          <img
            v-else
            :src="selected.fullSrc"
            alt="Фотография"
            decoding="async"
            class="block w-auto h-auto max-w-[95vw] max-h-[85vh] rounded-lg"
          />
          <p v-if="selectedIndex >= 0" class="text-sm text-white/80 tabular-nums">
            {{ selectedIndex + 1 }} из {{ photoCount }}
          </p>
        </div>
      </template>
    </UModal>

    <USlideover v-model:open="addPhotosOpen" side="right" title="Добавить фотографии" description="">
      <template #body>
        <div class="space-y-4">
          <UFileUpload
            v-model="addPhotoFiles"
            multiple
            accept="image/jpeg,image/png,image/webp,video/mp4"
            label="Перетащите файлы сюда"
            description="JPG, PNG, WEBP или MP4 (до 200 МБ). Можно выбрать несколько файлов."
            class="w-full min-h-48"
          />
          <UAlert
            v-if="addError"
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :description="addError"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton
            color="neutral"
            variant="outline"
            size="xl"
            class="w-full justify-center"
            @click="addPhotosOpen = false"
          >
            Отмена
          </UButton>
          <UButton
            size="xl"
            class="w-full justify-center"
            :loading="addSubmitting"
            @click="handleAddPhotos"
          >
            Загрузить
          </UButton>
        </div>
      </template>
    </USlideover>

    <USlideover v-model:open="editOpen" side="right" title="Редактирование альбома" description="">
      <template #body>
        <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
          <UFormField label="Название" name="title" required>
            <UInput
              v-model="editState.title"
              size="xl"
              class="w-full"
              placeholder="Название альбома"
            />
          </UFormField>
          <UFormField label="Описание" name="description">
            <UTextarea
              v-model="editState.description"
              size="xl"
              :rows="4"
              class="w-full"
              placeholder="Описание альбома"
            />
          </UFormField>
          <UFormField label="Дата" name="date">
            <UInput
              v-model="editState.date"
              size="xl"
              class="w-full"
              placeholder="ГГГГ-ММ-ДД"
            />
          </UFormField>
          <UAlert
            v-if="editError"
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :description="editError"
          />
        </UForm>
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

    <UModal v-model:open="deleteConfirmOpen" title="Удалить альбом?" description="">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ albumTitle }}»</strong>? Все фотографии
          альбома будут удалены. Это действие нельзя отменить.
        </p>
        <UAlert
          v-if="deleteError"
          color="error"
          variant="subtle"
          icon="i-lucide-alert-circle"
          :description="deleteError"
          class="mt-3"
        />
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" size="lg" @click="deleteConfirmOpen = false">
            Отмена
          </UButton>
          <UButton color="error" size="lg" :loading="deleteSubmitting" @click="handleDelete">
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
