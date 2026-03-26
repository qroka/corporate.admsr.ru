<script setup lang="ts">
import { computed, ref } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';

type Album = BlogPostProps & {
  id: string;
  date: string; // YYYY-MM-DD для сортировки
};

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

const albums = ref<Album[]>([
  {
    id: 'album-01',
    title: 'Зимний корпоратив 2026',
    description: 'Лучшие моменты с зимнего корпоратива: награждения, выступления и afterparty.',
    date: '2026-01-25',
    to: '/gallery/album-01',
    image: { src: coverAt(0), alt: 'Обложка альбома' },
    badge: 'Корпоратив',
  },
  {
    id: 'album-02',
    title: 'Новый год в отделах',
    description: 'Как мы встречали Новый год по подразделениям: уютно и по‑домашнему.',
    date: '2026-01-05',
    to: '/gallery/album-02',
    image: { src: coverAt(1), alt: 'Обложка альбома' },
    badge: 'Праздники',
  },
  {
    id: 'album-03',
    title: 'Спартакиада сотрудников',
    description: 'Командный дух, спортивные эмоции и победы.',
    date: '2025-09-10',
    to: '/gallery/album-03',
    image: { src: coverAt(2), alt: 'Обложка альбома' },
    badge: 'Спорт',
  },
  {
    id: 'album-04',
    title: 'День города',
    description: 'Участие команды в городских активностях и волонтёрских инициативах.',
    date: '2025-09-01',
    to: '/gallery/album-04',
    image: { src: coverAt(3), alt: 'Обложка альбома' },
    badge: 'Город',
  },
  {
    id: 'album-05',
    title: 'Летний тимбилдинг',
    description: 'Квесты, игры и много общения вне офиса.',
    date: '2025-07-18',
    to: '/gallery/album-05',
    image: { src: coverAt(4), alt: 'Обложка альбома' },
    badge: 'Тимбилдинг',
  },
  {
    id: 'album-06',
    title: 'Экскурсия для новичков',
    description: 'Знакомство с командой, офисом и внутренними сервисами.',
    date: '2025-06-06',
    to: '/gallery/album-06',
    image: { src: coverAt(5), alt: 'Обложка альбома' },
    badge: 'Новичкам',
  },
  {
    id: 'album-07',
    title: 'День донора',
    description: 'Добрые дела вместе: корпоративная донорская акция.',
    date: '2025-05-22',
    to: '/gallery/album-07',
    image: { src: coverAt(6), alt: 'Обложка альбома' },
    badge: 'Волонтёрство',
  },
  {
    id: 'album-08',
    title: 'Субботник',
    description: 'Навели порядок и зарядились весенним настроением.',
    date: '2025-04-26',
    to: '/gallery/album-08',
    image: { src: coverAt(7), alt: 'Обложка альбома' },
    badge: 'Команда',
  },
  {
    id: 'album-09',
    title: 'Культурная программа',
    description: 'Посещение музея и небольшая прогулка по городу.',
    date: '2025-03-15',
    to: '/gallery/album-09',
    image: { src: coverAt(8), alt: 'Обложка альбома' },
    badge: 'Культура',
  },
  {
    id: 'album-10',
    title: 'Обучение и развитие',
    description: 'Интенсивы, воркшопы и внутренние лекции.',
    date: '2025-02-20',
    to: '/gallery/album-10',
    image: { src: coverAt(9), alt: 'Обложка альбома' },
    badge: 'Обучение',
  },
  {
    id: 'album-11',
    title: 'Пятничные встречи',
    description: 'Кофе‑брейки, обсуждения и мини‑презентации проектов.',
    date: '2025-02-07',
    to: '/gallery/album-11',
    image: { src: coverAt(10), alt: 'Обложка альбома' },
    badge: 'Офис',
  },
  {
    id: 'album-12',
    title: 'День рождения компании',
    description: 'Праздничная атмосфера, поздравления и торт.',
    date: '2025-01-30',
    to: '/gallery/album-12',
    image: { src: coverAt(11), alt: 'Обложка альбома' },
    badge: 'Праздники',
  },
]);

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

    <UContainer class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UContainer class="grid grid-cols-3 md:grid-cols-3 xl:grid-cols-3 gap-3 sm:p-0 max-w-full w-full md:p-0 lg:p-0 xl:p-0 mx-0">
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

