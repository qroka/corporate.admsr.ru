<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted, onUnmounted } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';
import { useGalleryData } from '../../composables/useGalleryData';
import { useAppToast } from '../../composables/useAppToast';
import { formatDateRuLong } from '../../utils/date';

type Album = BlogPostProps & { id: string; date: string; rawDate: string };

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

const { loading, error, albums: albumRecords, ensureLoaded, reload } = useGalleryData();
ensureLoaded();

const { toast } = useAppToast();
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

const albums = computed<Album[]>(() =>
  albumRecords.value.map((a, idx) => ({
    id: a.id,
    title: a.title,
    description: a.description,
    rawDate: a.date,
    date: formatDateRuLong(a.date) || a.date,
    to: `/gallery/${a.id}`,
    image: { src: a.image || coverAt(idx), alt: 'Обложка альбома' },
  })),
);

const searchQuery = ref('');
const sortKey = ref<'newest' | 'oldest' | 'title-asc' | 'title-desc'>('newest');

const sortOptions = [
  { value: 'newest', label: 'Сначала новые' },
  { value: 'oldest', label: 'Сначала старые' },
  { value: 'title-asc', label: 'По названию (А‑Я)' },
  { value: 'title-desc', label: 'По названию (Я‑А)' },
];

const filteredAlbums = computed(() => {
  let list = albums.value;

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase();
    list = list.filter((a) => {
      const title = a.title ?? '';
      const description = a.description ?? '';
      return title.toLowerCase().includes(q) || description.toLowerCase().includes(q);
    });
  }

  return [...list].sort((a, b) => {
    if (sortKey.value === 'newest') return (b.rawDate ?? '').localeCompare(a.rawDate ?? '');
    if (sortKey.value === 'oldest') return (a.rawDate ?? '').localeCompare(b.rawDate ?? '');
    if (sortKey.value === 'title-asc') return (a.title ?? '').localeCompare(b.title ?? '', 'ru');
    if (sortKey.value === 'title-desc') return (b.title ?? '').localeCompare(a.title ?? '', 'ru');
    return 0;
  });
});

// ── Создание альбома ──────────────────────────────────────────────────────────
const createOpen       = ref(false);
const createSubmitting = ref(false);
const createFiles      = ref<File[] | undefined>(undefined);
type SingleDateValue   = { value?: any } | any | null;
const createDateValue  = ref<SingleDateValue>(null);
const createErrors     = reactive({ title: '', general: '' });

const createState = reactive({ title: '', description: '', date: '' });

function resetCreateForm() {
  createState.title       = '';
  createState.description = '';
  createState.date        = '';
  createDateValue.value   = null;
  createFiles.value       = undefined;
  createErrors.title      = '';
  createErrors.general    = '';
}

function openCreate() {
  resetCreateForm();
  createOpen.value = true;
}

const headerLinks = computed(() => {
  if (currentRole.value !== 'admin') return [];
  return [
    {
      label:   'Добавить альбом',
      icon:    'i-lucide-folder-plus',
      color:   'neutral',
      variant: 'outline',
      size:    'xl',
      onClick: openCreate,
    },
  ];
});

watch(createDateValue, (val) => {
  const d = val?.value ?? val;
  createState.date = d && typeof d.toString === 'function' ? d.toString() : '';
});

function validateCreate() {
  createErrors.title   = '';
  createErrors.general = '';
  if (!createState.title.trim()) createErrors.title = 'Заполните название альбома.';
  return !createErrors.title;
}

async function handleCreateSubmit() {
  if (!validateCreate()) return;
  createSubmitting.value = true;
  try {
    // 1. Создаём альбом в БД
    const res = await fetch('/api/gallery.php', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({
        name:        createState.title.trim(),
        description: createState.description.trim() || null,
        date:        createState.date || null,
      }),
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка создания альбома');

    const newAlbumId = json.data.id;

    // 2. Загружаем фото если выбраны
    for (const file of (createFiles.value ?? [])) {
      const fd = new FormData();
      fd.append('image',    file);
      fd.append('album_id', String(newAlbumId));
      await fetch('/api/gallery_base.php', { method: 'POST', body: fd });
    }

    // 3. Обновляем список
    await reload();
    createOpen.value = false;
    resetCreateForm();
    toast.add({
      title:       'Альбом создан',
      color:       'success',
      icon:        'i-lucide-check-circle',
    });
  } catch (e: any) {
    createErrors.general = e.message ?? 'Ошибка при создании альбома';
  } finally {
    createSubmitting.value = false;
  }
}

// ── Floating header on scroll up ──────────────────────────────────────────────
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
  if (!el) return;
  el.removeEventListener('scroll', onMainScroll);
  window.removeEventListener('resize', updateFloatingRect as any);
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <!-- Floating header (rendered only when needed, does not affect layout) -->
    <transition name="fade">
      <div
        v-if="showFloatingHeader"
        class="fixed z-30"
        :style="{ left: `${floatingRect.left}px`, top: `${floatingRect.top}px`, width: `${floatingRect.width}px` }"
      >
        <div class="bg-default p-0 pb-6 flex flex-col gap-6">
            <UPageHeader title="" :links="headerLinks" class="border-none p-0 w-full">
              <template #title>
                <h1 class="text-4xl font-normal font-unbounded">Фотогалерея</h1>
              </template>
            </UPageHeader>

            <UContainer class="flex flex-col max-w-full w-full sm:flex-row gap-3 items-stretch sm:items-center sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
              <UInput
                v-model="searchQuery"
                icon="i-lucide-search"
                size="xl"
                color="neutral"
                variant="outline"
                placeholder="Поиск по альбомам"
                class="flex-1"
              />
              <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" />
            </UContainer>
        </div>
      </div>
    </transition>

    <!-- Single scroll container for the whole page -->
    <div ref="mainScrollEl" class="flex flex-col w-full h-full min-h-0 gap-6 overflow-y-auto scrollbar-hide">
      <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
        <UPageHeader title="" :links="headerLinks" class="border-none p-0 w-full">
          <template #title>
            <h1 class="text-4xl font-normal font-unbounded">Фотогалерея</h1>
          </template>
        </UPageHeader>

        <UContainer class="flex flex-col max-w-full w-full sm:flex-row gap-3 items-stretch sm:items-center sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
          <UInput v-model="searchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline" placeholder="Поиск по альбомам" class="flex-1" />
          <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" />
        </UContainer>
      </UContainer>

      <UContainer class="sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px mx-0 flex-1 min-h-0">
        <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
          <USkeleton v-for="i in 6" :key="i" class="h-64 rounded-2xl" />
        </div>
        <UContainer
          v-else
          class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:p-0 max-w-full w-full md:p-0 lg:p-0 xl:p-0 mx-0"
        >
          <UBlogPost v-for="album in filteredAlbums" :key="album.id" v-bind="album" class="h-full max-w-full w-full">
            <template #title>
              <span class="overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]">
                {{ album.title }}
              </span>
            </template>
            <template #description>
              <span class="overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]">
                {{ album.description }}
              </span>
            </template>
          </UBlogPost>
        </UContainer>
      </UContainer>
    </div>

    <!-- Slideover создания альбома -->
    <USlideover
      v-model:open="createOpen"
      side="right"
      title="Новый альбом"
      description="Заполните данные альбома"
    >
      <template #body>
        <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
          <UFormField label="Название альбома" name="title" :error="createErrors.title || undefined" required>
            <UInput v-model="createState.title" size="xl" class="w-full" placeholder="Введите название альбома" />
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
                  <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату" class="px-0" />
                  <template #content>
                    <UCalendar v-model="createDateValue" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>

          <UFormField label="Фотографии (необязательно)" name="files">
            <UFileUpload
              v-model="createFiles"
              multiple
              label="Перетащите фото сюда"
              description="JPG, PNG или WEBP. Можно выбрать несколько."
              class="w-full min-h-48 rounded-lg"
            />
          </UFormField>

          <UAlert v-if="createErrors.general" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="createErrors.general" />
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton type="button" color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="createOpen = false">
            Отмена
          </UButton>
          <UButton type="button" size="xl" class="w-full justify-center" :loading="createSubmitting" @click="handleCreateSubmit">
            Создать альбом
          </UButton>
        </div>
      </template>
    </USlideover>
  </UMain>
</template>