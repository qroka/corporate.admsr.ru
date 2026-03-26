<script setup lang="ts">
import { computed, ref } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { useGalleryData } from '../../composables/useGalleryData';

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

const { loading, error, albums: albumRecords, ensureLoaded } = useGalleryData();
ensureLoaded();

const albums = computed<Album[]>(() =>
  albumRecords.value.map((a, idx) => ({
    id: a.id,
    title: a.title,
    description: a.description,
    date: a.date,
    to: `/gallery/${a.id}`,
    image: { src: coverAt(a.coverIndex ?? idx), alt: 'Обложка альбома' },
    badge: a.badge,
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
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <UPageHeader title="" class="border-none p-0 w-full">
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
        <USelect
          v-model="sortKey"
          :items="sortOptions"
          size="xl"
          color="neutral"
        />
      </UContainer>
    </UContainer>

    <UContainer v-if="error" class="max-w-full w-full mx-0">
      <UAlert
        color="error"
        variant="subtle"
        icon="i-lucide-alert-circle"
        :title="error"
      />
    </UContainer>

    <UContainer class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UContainer class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:p-0 max-w-full w-full md:p-0 lg:p-0 xl:p-0 mx-0">
        <UBlogPost
          v-for="album in filteredAlbums"
          :key="album.id"
          v-bind="album"
          class="h-full max-w-full w-full"
        />
      </UContainer>
    </UContainer>
  </UMain>
</template>

