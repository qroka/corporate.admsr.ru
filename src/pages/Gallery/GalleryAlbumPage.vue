<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useGalleryData } from '../../composables/useGalleryData';
import { useAppToast } from '../../composables/useAppToast';
import { currentRole } from '../../stores/role';

type Photo = { id: string; thumbSrc: string; fullSrc: string };
type DbPhoto = { id: number; album_id: number; image_full_url: string; image_small_url: string };

const route  = useRoute();
const router = useRouter();

const albumId = computed(() => String(route.params.albumId ?? ''));

// Для навигации prev/next используем общий список альбомов
const { albums, ensureLoaded } = useGalleryData();
ensureLoaded();

const { toast } = useAppToast();
const isAdmin   = computed(() => currentRole.value === 'admin');

// ── Данные альбома ────────────────────────────────────────────────────────────
const albumData    = ref<{ id: number; name: string; description: string; date: string } | null>(null);
const albumLoading = ref(false);

const albumTitle       = computed(() => albumData.value?.name        ?? '');
const albumDescription = computed(() => albumData.value?.description ?? '');

async function loadAlbum() {
  albumLoading.value = true;
  try {
    const res  = await fetch(`/api/gallery.php?id=${albumId.value}`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message);
    albumData.value = json.data;
  } catch (e: any) {
    toast.add({ title: 'Не удалось загрузить альбом', description: e.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    albumLoading.value = false;
  }
}

// ── Фотографии ────────────────────────────────────────────────────────────────
const photos        = ref<DbPhoto[]>([]);
const photosLoading = ref(false);

async function loadPhotos() {
  photosLoading.value = true;
  try {
    const res  = await fetch(`/api/gallery_base.php?album_id=${albumId.value}`);
    const json = await res.json();
    if (json.success) photos.value = json.data ?? [];
  } catch {} finally {
    photosLoading.value = false;
  }
}

const items = computed<Photo[]>(() =>
  photos.value.map((p) => ({
    id:       String(p.id),
    thumbSrc: p.image_small_url,
    fullSrc:  p.image_full_url,
  })),
);

// ── Лайтбокс ─────────────────────────────────────────────────────────────────
const selected  = ref<Photo | null>(null);
const modalOpen = computed({
  get:  () => selected.value !== null,
  set:  (open: boolean) => { if (!open) selected.value = null; },
});
function openPhoto(p: Photo) { selected.value = p; }

// ── Добавление фото ───────────────────────────────────────────────────────────
const addPhotosOpen   = ref(false);
const addPhotoFiles   = ref<File[] | undefined>(undefined);
const addSubmitting   = ref(false);
const addError        = ref<string | null>(null);

async function handleAddPhotos() {
  if (!addPhotoFiles.value?.length) { addError.value = 'Выберите хотя бы один файл.'; return; }
  addSubmitting.value = true;
  addError.value      = null;
  try {
    for (const file of addPhotoFiles.value) {
      const fd = new FormData();
      fd.append('image',    file);
      fd.append('album_id', albumId.value);
      const res  = await fetch('/api/gallery_base.php', { method: 'POST', body: fd });
      const json = await res.json();
      if (!json.success) throw new Error(json.message || 'Ошибка загрузки файла');
    }
    await loadPhotos();
    addPhotosOpen.value = false;
    addPhotoFiles.value = undefined;
    toast.add({ title: 'Фото добавлены', color: 'success', icon: 'i-lucide-check-circle' });
  } catch (e: any) {
    addError.value = e.message ?? 'Ошибка загрузки';
  } finally {
    addSubmitting.value = false;
  }
}

// ── Удаление фото ─────────────────────────────────────────────────────────────
const deletingPhotoId = ref<string | null>(null);

async function deletePhoto(photoId: string) {
  deletingPhotoId.value = photoId;
  try {
    await fetch(`/api/gallery_base.php?id=${photoId}`, { method: 'DELETE' });
    photos.value = photos.value.filter((p) => String(p.id) !== photoId);
    if (selected.value?.id === photoId) selected.value = null;
  } finally {
    deletingPhotoId.value = null;
  }
}

// ── Редактирование альбома ────────────────────────────────────────────────────
const editOpen        = ref(false);
const editSubmitting  = ref(false);
const editError       = ref<string | null>(null);
const editState       = ref({ title: '', description: '', date: '' });

async function handleEditSubmit() {
  if (!editState.value.title.trim()) { editError.value = 'Заполните название альбома.'; return; }
  editSubmitting.value = true;
  editError.value      = null;
  try {
    const res  = await fetch(`/api/gallery.php?id=${albumId.value}`, {
      method:  'PUT',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({
        name:        editState.value.title.trim(),
        description: editState.value.description.trim() || null,
        date:        editState.value.date || null,
      }),
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message);
    albumData.value  = json.data;
    editOpen.value   = false;
    toast.add({ title: 'Альбом обновлён', color: 'success', icon: 'i-lucide-check-circle' });
  } catch (e: any) {
    editError.value = e.message ?? 'Ошибка при сохранении';
  } finally {
    editSubmitting.value = false;
  }
}

// ── Удаление альбома ──────────────────────────────────────────────────────────
const deleteConfirmOpen = ref(false);
const deleteSubmitting  = ref(false);
const deleteError       = ref<string | null>(null);

const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));

async function handleDelete() {
  deleteSubmitting.value = true;
  deleteError.value      = null;
  try {
    const res  = await fetch(`/api/gallery.php?id=${albumId.value}`, { method: 'DELETE' });
    const json = await res.json();
    if (!json.success) throw new Error(json.message);
    deleteConfirmOpen.value = false;
    await router.push(isKiosk.value ? '/kiosk/gallery' : '/gallery');
  } catch (e: any) {
    deleteError.value = e.message ?? 'Ошибка при удалении';
  } finally {
    deleteSubmitting.value = false;
  }
}

// ── Навигация prev/next ───────────────────────────────────────────────────────
const sortedAlbums = computed(() =>
  albums.value
    .slice()
    .sort((a, b) =>
      String(b.date ?? '').localeCompare(String(a.date ?? ''), 'ru-RU') ||
      String(b.id).localeCompare(String(a.id), 'ru-RU'),
    ),
);

const surround = computed(() => {
  const id   = albumId.value;
  const list = sortedAlbums.value;
  const idx  = list.findIndex((x) => x.id === id);
  if (idx < 0) return { prev: null, next: null };
  const prev   = idx > 0 ? list[idx - 1] : null;
  const next   = idx < list.length - 1 ? list[idx + 1] : null;
  const prefix = isKiosk.value ? '/kiosk/gallery/' : '/gallery/';
  return {
    prev: prev ? { title: prev.title || `Альбом ${prev.id}`, description: prev.description, to: `${prefix}${prev.id}` } : null,
    next: next ? { title: next.title || `Альбом ${next.id}`, description: next.description, to: `${prefix}${next.id}` } : null,
  };
});

// ── Кнопки заголовка (только для админа) ─────────────────────────────────────
const headerLinks = computed(() => {
  if (!isAdmin.value) return [];
  return [
    {
      label:   'Добавить фото',
      icon:    'i-lucide-image-plus',
      color:   'neutral',
      variant: 'outline',
      size:    'xl',
      onClick: () => {
        addPhotoFiles.value = undefined;
        addError.value      = null;
        addPhotosOpen.value = true;
      },
    },
    {
      label:   'Редактировать',
      color:   'neutral',
      variant: 'outline',
      size:    'xl',
      onClick: () => {
        editState.value = {
          title:       albumData.value?.name        ?? '',
          description: albumData.value?.description ?? '',
          date:        String(albumData.value?.date ?? '').slice(0, 10),
        };
        editError.value  = null;
        editOpen.value   = true;
      },
    },
    {
      label:   'Удалить',
      color:   'error',
      variant: 'outline',
      size:    'xl',
      onClick: () => {
        deleteError.value       = null;
        deleteConfirmOpen.value = true;
      },
    },
  ];
});

// ── Адаптивная сетка ──────────────────────────────────────────────────────────
const viewportWidth = ref(typeof window === 'undefined' ? 1280 : window.innerWidth);
function onResize() { viewportWidth.value = window.innerWidth; }
onMounted(() => {
  window.addEventListener('resize', onResize, { passive: true });
  void loadAlbum();
  void loadPhotos();
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize);
});

const lanes        = computed(() => (viewportWidth.value < 640 ? 1 : viewportWidth.value < 1280 ? 2 : 3));
const estimateSize = computed(() => (viewportWidth.value < 640 ? 420 : 480));
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0 max-w-none">
      <UPageHeader :links="headerLinks" class="border-none p-0 w-full">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">
            {{ albumTitle }}
          </h1>
        </template>
      </UPageHeader>
      <p v-if="albumDescription" class="text-sm text-muted">{{ albumDescription }}</p>
    </UContainer>

    <!-- Скелетон загрузки -->
    <div v-if="photosLoading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
      <USkeleton v-for="i in 6" :key="i" class="h-48 rounded-xl" />
    </div>

    <UEmpty
      v-else-if="!items.length"
      icon="i-lucide-image-off"
      title="Нет фотографий"
      description="В этом альбоме пока нет снимков."
      class="flex-1 min-h-0 py-12"
    />

    <UScrollArea
      v-else
      v-slot="{ item, index }"
      :items="items"
      orientation="vertical"
      :virtualize="{ gap: 12, lanes, estimateSize, overscan: 6 }"
      class="flex-1 min-h-0 w-full p-px scrollbar-hide"
    >
      <div class="relative rounded-xl overflow-hidden bg-elevated ring ring-transparent hover:ring-accented transition w-full group">
        <button type="button" class="block w-full" @click="openPhoto(item)">
          <img
            :src="item.thumbSrc"
            alt="Фотография"
            :loading="index > 8 ? 'lazy' : 'eager'"
            decoding="async"
            class="block w-full h-auto"
          />
        </button>
        <!-- Кнопка удаления фото (только для админа) -->
        <button
          v-if="isAdmin"
          type="button"
          class="absolute top-2 right-2 size-7 flex items-center justify-center rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition hover:bg-error"
          :disabled="deletingPhotoId === item.id"
          @click.stop="deletePhoto(item.id)"
          aria-label="Удалить фото"
        >
          <UIcon name="i-lucide-x" class="text-sm" />
        </button>
      </div>
    </UScrollArea>

    <!-- Лайтбокс -->
    <UModal
      v-model:open="modalOpen"
      class="w-[96vw] max-w-7xl h-[88vh] p-0"
      :ui="{ content: 'p-0', header: 'p-0', body: 'p-0', footer: 'p-0' }"
    >
      <template #content>
        <img
          v-if="selected"
          :src="selected.fullSrc"
          alt="Фотография"
          class="block h-[88vh] w-full object-contain"
          decoding="async"
        />
      </template>
    </UModal>

    <!-- Slideover: добавить фото -->
    <USlideover v-model:open="addPhotosOpen" side="right" title="Добавить фотографии" description="">
      <template #body>
        <div class="space-y-4">
          <UFileUpload
            v-model="addPhotoFiles"
            multiple
            label="Перетащите фото сюда"
            description="JPG, PNG или WEBP. Можно выбрать несколько файлов."
            class="w-full min-h-48"
          />
          <UAlert v-if="addError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="addError" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="addPhotosOpen = false">
            Отмена
          </UButton>
          <UButton size="xl" class="w-full justify-center" :loading="addSubmitting" @click="handleAddPhotos">
            Загрузить
          </UButton>
        </div>
      </template>
    </USlideover>

    <!-- Slideover: редактировать альбом -->
    <USlideover v-model:open="editOpen" side="right" title="Редактирование альбома" description="">
      <template #body>
        <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
          <UFormField label="Название" name="title" required>
            <UInput v-model="editState.title" size="xl" class="w-full" placeholder="Название альбома" />
          </UFormField>
          <UFormField label="Описание" name="description">
            <UTextarea v-model="editState.description" size="xl" :rows="4" class="w-full" placeholder="Описание альбома" />
          </UFormField>
          <UFormField label="Дата" name="date">
            <UInput v-model="editState.date" size="xl" class="w-full" placeholder="ГГГГ-ММ-ДД" />
          </UFormField>
          <UAlert v-if="editError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="editError" />
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="editOpen = false">
            Отмена
          </UButton>
          <UButton size="xl" class="w-full justify-center" :loading="editSubmitting" @click="handleEditSubmit">
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>

    <!-- Модал: подтверждение удаления альбома -->
    <UModal v-model:open="deleteConfirmOpen" title="Удалить альбом?" description="">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ albumTitle }}»</strong>? Все фотографии альбома будут удалены. Это действие нельзя отменить.
        </p>
        <UAlert v-if="deleteError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="deleteError" class="mt-3" />
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" size="lg" @click="deleteConfirmOpen = false">Отмена</UButton>
          <UButton color="error" size="lg" :loading="deleteSubmitting" @click="handleDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>