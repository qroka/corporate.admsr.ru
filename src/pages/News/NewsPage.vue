<script setup lang="ts">
import { computed, ref } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../../composables/useNewsData';

type NewsPost = BlogPostProps & { id: string };

const { loading, error, sortedNews, ensureLoaded } = useNewsData();
ensureLoaded();

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
  <UMain class="flex flex-1 min-h-0">
    <UContainer class="flex flex-col gap-4 w-full min-h-0">
      <UPageHeader title="Новости" description="Все новости корпоративного портала" class="border-none p-0" />

      <UAlert v-if="error" color="error" variant="soft" :title="error" />

      <UContainer class="flex flex-col sm:flex-row gap-3 sm:items-center sm:justify-between p-0">
        <UInput v-model="searchQuery" placeholder="Поиск по новостям..." class="w-full sm:max-w-[520px]" />
        <USelectMenu v-model="sortKey" :items="sortOptions" value-key="value" label-key="label" class="w-full sm:w-64" />
      </UContainer>

      <UScrollArea class="min-h-0 flex-1 rounded-lg border border-[var(--ui-border)]">
        <UContainer class="flex flex-col gap-3 p-3">
          <USkeleton v-if="loading" class="h-28 w-full" />
          <UBlogPost
            v-for="post in filtered"
            v-else
            :key="post.id"
            v-bind="post"
            class="w-full"
          />
          <UEmpty
            v-if="!loading && !filtered.length"
            icon="i-lucide-newspaper"
            title="Новостей не найдено"
            description="Попробуйте изменить запрос или сортировку."
          />
        </UContainer>
      </UScrollArea>
    </UContainer>
  </UMain>
</template>

