<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { currentRole } from '../../stores/role';
import { useGalleryData } from '../../composables/useGalleryData';
import { useAppToast } from '../../composables/useAppToast';

type Album = BlogPostProps & { id: string; date: string };

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

const { loading, error, albums: albumRecords, addAlbum, ensureLoaded } = useGalleryData();
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
    date: a.date,
    to: `/gallery/${a.id}`,
    image: { src: a.image || coverAt(a.coverIndex ?? idx), alt: 'Обложка альбома' },
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

  const sorted = [...list].sort((a, b) => {
    if (sortKey.value === 'newest') return (b.date ?? '').localeCompare(a.date ?? '');
    if (sortKey.value === 'oldest') return (a.date ?? '').localeCompare(b.date ?? '');
    if (sortKey.value === 'title-asc') return (a.title ?? '').localeCompare(b.title ?? '', 'ru');
    if (sortKey.value === 'title-desc') return (b.title ?? '').localeCompare(a.title ?? '', 'ru');
    return 0;
  });

  return sorted;
});

const createOpen = ref(false);
const createSubmitting = ref(false);
const createFiles = ref<File[] | undefined>(undefined);
type SingleDateValue = { value?: any } | any | null;
const createDateValue = ref<SingleDateValue>(null);
const createErrors = reactive({
  title: '',
  date: '',
  files: '',
});

const createState = reactive({
  title: '',
  description: '',
  date: '',
});

function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ''));
    reader.onerror = () => reject(new Error('Не удалось прочитать файл'));
    reader.readAsDataURL(file);
  });
}

function resetCreateForm() {
  createState.title = '';
  createState.description = '';
  createState.date = '';
  createDateValue.value = null;
  createFiles.value = undefined;
  createErrors.title = '';
  createErrors.date = '';
  createErrors.files = '';
}

function openCreate() {
  resetCreateForm();
  createOpen.value = true;
}

const headerLinks = computed(() => {
  if (currentRole.value !== 'admin') return [];
  return [
    {
      label: 'Добавить альбом',
      icon: 'i-lucide-folder-plus',
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

function validateCreate() {
  createErrors.title = '';
  createErrors.date = '';
  createErrors.files = '';

  if (!createState.title.trim()) {
    createErrors.title = 'Заполните название альбома.';
  }
  if (!createState.date) {
    createErrors.date = 'Укажите дату альбома.';
  }
  if (!createFiles.value?.length) {
    createErrors.files = 'Добавьте хотя бы один файл.';
  }
  return !createErrors.title && !createErrors.date && !createErrors.files;
}

async function handleCreateSubmit() {
  if (!validateCreate()) return;
  createSubmitting.value = true;
  try {
    const usedIds = new Set(albumRecords.value.map((a) => Number(a.id)).filter((n) => Number.isFinite(n)));
    let nextId = 1;
    while (usedIds.has(nextId)) nextId += 1;
    const files = createFiles.value ?? [];
    const photoLinks = await Promise.all(files.map((f) => fileToDataUrl(f)));
    const coverSrc = photoLinks[0];

    addAlbum({
      id: String(nextId),
      title: createState.title.trim(),
      description:
        createState.description.trim() ||
        `В альбоме ${createFiles.value?.length ?? 0} ${createFiles.value?.length === 1 ? 'файл' : 'файлов'}.`,
      date: createState.date,
      coverIndex: nextId,
      image: coverSrc,
      photoLinks,
    });

    createOpen.value = false;
    resetCreateForm();
    toast.add({
      title: 'Альбом добавлен',
      description: 'Альбом сохранен в JSON-хранилище и отображается в списке.',
      color: 'success',
      icon: 'i-lucide-check-circle',
    });
  } catch (e: any) {
    createErrors.title = e?.message ?? 'Ошибка при создании альбома';
  } finally {
    createSubmitting.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <UPageHeader title="" :links="headerLinks" class="border-none p-0 w-full">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Фотогалерея</h1>
        </template>
      </UPageHeader>

      <UContainer
        class="flex flex-col max-w-full w-full sm:flex-row gap-3 items-stretch sm:items-center sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
        <UInput v-model="searchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline"
          placeholder="Поиск по альбомам" class="flex-1" />
        <USelect v-model="sortKey" :items="sortOptions" size="xl" color="neutral" />
      </UContainer>
    </UContainer>

    <UContainer
      class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UContainer
        class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:p-0 max-w-full w-full md:p-0 lg:p-0 xl:p-0 mx-0">
        <UBlogPost v-for="album in filteredAlbums" :key="album.id" v-bind="album" class="h-full max-w-full w-full" />
      </UContainer>
    </UContainer>

    <!-- Slideover создания альбома (только для администратора) -->
    <USlideover
      v-model:open="createOpen"
      side="right"
      title="Новый альбом"
      description="Заполните данные и загрузите фотографии альбома"
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

          <UFormField label="Дата альбома" name="date" :error="createErrors.date || undefined" required>
            <UInputDate v-model="createDateValue" size="xl" class="w-full">
              <template #trailing>
                <UPopover>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="sm"
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

          <UFormField label="Файлы альбома" name="files" :error="createErrors.files || undefined" required>
            <UFileUpload
              v-model="createFiles"
              multiple
              label="Перетащите фото сюда"
              description="JPG, PNG, WEBP или GIF. Можно выбрать несколько файлов."
              :class="[
                'w-full min-h-48 rounded-lg',
                createErrors.files ? 'ring-1 ring-error border-error' : '',
              ]"
            />
          </UFormField>

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
