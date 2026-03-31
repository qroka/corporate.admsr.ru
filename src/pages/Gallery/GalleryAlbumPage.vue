<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useGalleryData } from '../../composables/useGalleryData';
import { useAppToast } from '../../composables/useAppToast';
import { currentRole } from '../../stores/role';

type Photo = {
  id: string;
  thumbSrc: string;
  fullSrc: string;
};

const route = useRoute();
const router = useRouter();

const albumId = computed(() => String(route.params.albumId ?? ''));

const {
  loading: galleryLoading,
  error: galleryError,
  getAlbum,
  ensureLoaded,
  albums,
  updateAlbum,
  removeAlbum,
} = useGalleryData();
ensureLoaded();

const { toast } = useAppToast();
watch(
  galleryError,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить альбом',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const album = computed(() => getAlbum(albumId.value));
const albumTitle = computed(() => album.value.title);
const albumDescription = computed(() => album.value.description);
const isAdmin = computed(() => currentRole.value === 'admin');

const editOpen = ref(false);
const editSubmitting = ref(false);
const editError = ref<string | null>(null);
const editState = ref({
  title: '',
  description: '',
  date: '',
});
const editFiles = ref<File[] | undefined>(undefined);

function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ''));
    reader.onerror = () => reject(new Error('Не удалось прочитать файл'));
    reader.readAsDataURL(file);
  });
}

const deleteConfirmOpen = ref(false);
const deleteSubmitting = ref(false);
const deleteError = ref<string | null>(null);

const headerLinks = computed(() => {
  if (!isAdmin.value) return [];
  return [
    {
      label: 'Редактировать',
      color: 'neutral',
      variant: 'outline',
      size: 'xl',
      onClick: () => {
        editState.value = {
          title: album.value.title ?? '',
          description: album.value.description ?? '',
          date: album.value.date ?? '',
        };
        editFiles.value = undefined;
        editError.value = null;
        editOpen.value = true;
      },
    },
    {
      label: 'Удалить',
      color: 'error',
      variant: 'outline',
      size: 'xl',
      onClick: () => {
        deleteError.value = null;
        deleteConfirmOpen.value = true;
      },
    },
  ];
});

const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const sortedAlbums = computed(() => {
  const list = albums.value.slice();
  list.sort(
    (a, b) =>
      String(b.date ?? '').localeCompare(String(a.date ?? ''), 'ru-RU') ||
      String(b.id).localeCompare(String(a.id), 'ru-RU'),
  );
  return list;
});

const surround = computed(() => {
  const id = albumId.value;
  if (!id) return { prev: null, next: null };
  const list = sortedAlbums.value;
  const idx = list.findIndex((x) => x.id === id);
  if (idx < 0) return { prev: null, next: null };

  const prev = idx > 0 ? list[idx - 1] : null;
  const next = idx >= 0 && idx < list.length - 1 ? list[idx + 1] : null;
  const prefix = isKiosk.value ? '/kiosk/gallery/' : '/gallery/';

  return {
    prev: prev ? { title: prev.title || `Альбом ${prev.id}`, description: prev.description, to: `${prefix}${prev.id}` } : null,
    next: next ? { title: next.title || `Альбом ${next.id}`, description: next.description, to: `${prefix}${next.id}` } : null,
  };
});

function resolveGalleryPhotoSrcs(src: string): { thumbSrc: string; fullSrc: string } {
  const s = String(src ?? '').trim();
  if (!s) return { thumbSrc: '', fullSrc: '' };
  if (s.startsWith('data:') || s.startsWith('blob:')) {
    return { thumbSrc: s, fullSrc: s };
  }
  if (s.includes('/img/FullPic/')) {
    return {
      fullSrc: s,
      thumbSrc: s.replace('/img/FullPic/', '/img/SmallPic/'),
    };
  }
  if (s.includes('/img/SmallPic/')) {
    return {
      thumbSrc: s,
      fullSrc: s.replace('/img/SmallPic/', '/img/FullPic/'),
    };
  }
  return { thumbSrc: s, fullSrc: s };
}

const items = computed<Photo[]>(() => {
  const linkedPhotos = Array.isArray(album.value.photoLinks)
    ? album.value.photoLinks.filter((x) => String(x ?? '').trim().length > 0)
    : [];

  return linkedPhotos.map((src, index) => {
    const { thumbSrc, fullSrc } = resolveGalleryPhotoSrcs(src);
    return {
      id: `photo-${index + 1}`,
      thumbSrc,
      fullSrc,
    };
  });
});

const viewportWidth = ref<number>(typeof window === 'undefined' ? 1280 : window.innerWidth);
function onResize() {
  viewportWidth.value = window.innerWidth;
}

onMounted(() => {
  window.addEventListener('resize', onResize, { passive: true });
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize);
});

const lanes = computed(() => (viewportWidth.value < 640 ? 1 : viewportWidth.value < 1280 ? 2 : 3));
const estimateSize = computed(() => (viewportWidth.value < 640 ? 420 : 480));

const selected = ref<Photo | null>(null);
const modalOpen = computed({
  get: () => selected.value !== null,
  set: (open: boolean) => {
    if (!open) selected.value = null;
  },
});

function openPhoto(p: Photo) {
  selected.value = p;
}

async function handleEditSubmit() {
  if (!albumId.value) return;
  if (!editState.value.title.trim()) {
    editError.value = 'Заполните название альбома.';
    return;
  }
  editSubmitting.value = true;
  editError.value = null;
  try {
    const nextPhotoLinks =
      editFiles.value?.length
        ? await Promise.all(editFiles.value.map((f) => fileToDataUrl(f)))
        : undefined;
    const nextCover = nextPhotoLinks?.[0];

    updateAlbum(albumId.value, {
      title: editState.value.title.trim(),
      description: editState.value.description.trim(),
      date: editState.value.date.trim(),
      ...(nextPhotoLinks ? { photoLinks: nextPhotoLinks } : {}),
      ...(nextCover ? { image: nextCover } : {}),
    });
    editOpen.value = false;
    toast.add({
      title: 'Альбом обновлен',
      color: 'success',
      icon: 'i-lucide-check-circle',
    });
  } catch (e: any) {
    editError.value = e?.message ?? 'Ошибка при сохранении';
  } finally {
    editSubmitting.value = false;
  }
}

async function handleDelete() {
  if (!albumId.value) return;
  deleteSubmitting.value = true;
  deleteError.value = null;
  try {
    removeAlbum(albumId.value);
    deleteConfirmOpen.value = false;
    await router.push(isKiosk.value ? '/kiosk/gallery' : '/gallery');
  } catch (e: any) {
    deleteError.value = e?.message ?? 'Ошибка при удалении';
  } finally {
    deleteSubmitting.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6 overflow-y-auto scrollbar-hide">
    <UContainer class="flex flex-col gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0 max-w-none scrollbar-hide">
      <UPageHeader :links="headerLinks" class="border-none p-0 w-full">
        <template #title>
          <div class="flex items-center gap-3 min-w-0">
            <div class="min-w-0">
              <h1 class="text-4xl font-normal font-unbounded">
                {{ albumTitle }}
              </h1>
            </div>
          </div>
        </template>
      </UPageHeader>
    </UContainer>
    <UEmpty
      v-if="!items.length"
      icon="i-lucide-image-off"
      title="Нет фотографий"
      description="В этом альбоме пока нет снимков."
      class="py-12"
    />
    <UContainer v-else class="w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 max-w-none scrollbar-hide">
      <UContainer class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 max-w-none scrollbar-hide">
        <div
          v-for="(item, index) in items"
          :key="item.id"
          class="rounded-xl overflow-hidden bg-elevated ring ring-transparent hover:ring-accented transition w-full"
        >
          <button type="button" class="block w-full" @click="openPhoto(item)">
            <img
              :src="item.thumbSrc"
              alt="Фотография"
              :loading="index > 8 ? 'lazy' : 'eager'"
              decoding="async"
              class="block w-full h-auto"
            />
          </button>
        </div>
      </UContainer>
    </UContainer>

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

    <USlideover
      v-model:open="editOpen"
      side="right"
      title="Редактирование альбома"
      description="Измените данные альбома"
    >
      <template #body>
        <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
          <UFormField label="Название" name="title" required>
            <UInput v-model="editState.title" size="lg" placeholder="Название альбома" />
          </UFormField>
          <UFormField label="Описание" name="description">
            <UTextarea v-model="editState.description" size="lg" :rows="4" placeholder="Описание альбома" />
          </UFormField>
          <UFormField label="Дата" name="date">
            <UInput v-model="editState.date" size="lg" placeholder="YYYY-MM-DD" />
          </UFormField>
          <UFormField label="Заменить фотографии" name="editFiles">
            <UFileUpload
              v-model="editFiles"
              multiple
              label="Перетащите фото сюда"
              description="Оставьте пустым, если не нужно менять текущие фотографии"
              class="w-full min-h-48"
            />
          </UFormField>
          <p v-if="editError" class="text-sm text-error">{{ editError }}</p>
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

    <UModal
      v-model:open="deleteConfirmOpen"
      title="Удалить альбом?"
      description="Подтвердите удаление альбома"
    >
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ albumTitle }}»</strong>? Это действие нельзя отменить.
        </p>
        <p v-if="deleteError" class="mt-2 text-sm text-error">{{ deleteError }}</p>
      </template>
      <template #footer>
        <div class="flex  gap-3 justify-end w-full">
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
