<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../../composables/useNewsData';
import { useAppToast } from '../../composables/useAppToast';

type NewsPost = BlogPostProps & { id: string };

const { loading, error, sortedNews, ensureLoaded } = useNewsData();
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

const searchQuery = ref('');
const sortKey = ref<'newest' | 'oldest' | 'title-asc' | 'title-desc'>('newest');
const sortOptions = [
  { value: 'newest', label: 'Сначала новые' },
  { value: 'oldest', label: 'Сначала старые' },
  { value: 'title-asc', label: 'По названию (А‑Я)' },
  { value: 'title-desc', label: 'По названию (Я‑А)' },
];

const postsBase = computed<NewsPost[]>(() =>
  sortedNews.value.map((n) => {
    const imageSrc = resolveNewsImageSrc(n.announceImagePath);
    const date = formatUnixDate(n.timestamp);
    const description = stripHtmlToText(n.shortHtml || n.html).slice(0, 240);
    return {
      id: n.id,
      title: n.title || `Новость #${n.id}`,
      description,
      date,
      badge: 'Новости',
      to: `/news/${n.id}`,
      image: imageSrc ? { src: imageSrc, alt: n.title } : { src: '/src/img/Logo.svg', alt: n.title },
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
    if (sortKey.value === 'newest') return String(b.date ?? '').localeCompare(String(a.date ?? ''), 'ru-RU');
    if (sortKey.value === 'oldest') return String(a.date ?? '').localeCompare(String(b.date ?? ''), 'ru-RU');
    if (sortKey.value === 'title-asc') return String(a.title ?? '').localeCompare(String(b.title ?? ''), 'ru-RU');
    return String(b.title ?? '').localeCompare(String(a.title ?? ''), 'ru-RU');
  });

  return items;
});
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col max-w-full w-full gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
      <UPageHeader class="border-none p-0 w-full">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Новости</h1>
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
        <USelect
          v-model="sortKey"
          :items="sortOptions"
          size="xl"
          color="neutral"
        />
      </UContainer>

      <p v-if="!loading && filtered.length !== postsBase.length" class="text-sm text-muted -mt-2">
        Найдено: {{ filtered.length }} из {{ postsBase.length }}
      </p>
    </UContainer>

    <UContainer class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UContainer class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:p-0 max-w-full w-full md:p-0 lg:p-0 xl:p-0 mx-0">
        <USkeleton v-if="loading" class="h-28 w-full" />
        <UBlogPost
          v-for="post in filtered"
          v-else
          :key="post.id"
          v-bind="post"
          class="h-full max-w-full w-full"
        />
      </UContainer>

      <UEmpty
        v-if="!loading && !filtered.length"
        icon="i-lucide-newspaper"
        title="Новостей не найдено"
        description="Попробуйте изменить запрос или сортировку."
        class="mt-3"
      />
    </UContainer>
  </UMain>
</template>

