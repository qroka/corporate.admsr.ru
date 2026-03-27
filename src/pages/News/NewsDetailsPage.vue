<script setup lang="ts">
import { computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../../composables/useNewsData';
import { useAppToast } from '../../composables/useAppToast';
import UContentSurround from '../../components/UContentSurround.vue';

const route = useRoute();
const router = useRouter();
const { loading, error, getById, ensureLoaded, sortedNews } = useNewsData();
ensureLoaded();

const { toast } = useAppToast();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить новость',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const newsId = computed(() => String(route.params.id ?? '').trim());
const item = computed(() => (newsId.value ? getById(newsId.value) : undefined));

const title = computed(() => item.value?.title || (newsId.value ? `Новость #${newsId.value}` : 'Новость'));
const date = computed(() => formatUnixDate(item.value?.timestamp ?? null));
const imageSrc = computed(() => resolveNewsImageSrc(item.value?.announceImagePath ?? null));
const html = computed(() => item.value?.html || item.value?.shortHtml || '');
const summary = computed(() => (item.value ? stripHtmlToText(item.value.shortHtml || item.value.html).slice(0, 180) : ''));

async function copyNewsLink() {
  if (typeof window === 'undefined') return;
  try {
    await navigator.clipboard.writeText(window.location.href);
    toast.add({
      title: 'Ссылка скопирована',
      description: 'Можно отправить или открыть на другом устройстве.',
      color: 'success',
      icon: 'i-lucide-circle-check',
    });
  } catch {
    toast.add({
      title: 'Не удалось скопировать ссылку',
      description: 'Браузер мог запретить доступ к буферу обмена.',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  }
}

const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const surround = computed(() => {
  const id = item.value?.id;
  if (!id) return { prev: null, next: null };
  const list = sortedNews.value;
  const idx = list.findIndex((x) => x.id === id);
  if (idx < 0) return { prev: null, next: null };

  const prev = idx > 0 ? list[idx - 1] : null;
  const next = idx >= 0 && idx < list.length - 1 ? list[idx + 1] : null;
  const prefix = isKiosk.value ? '/kiosk/news/' : '/news/';

  return {
    prev: prev
      ? {
        title: prev.title || `Новость #${prev.id}`,
        description: stripHtmlToText(prev.shortHtml || prev.html).slice(0, 140),
        to: `${prefix}${prev.id}`,
      }
      : null,
    next: next
      ? {
        title: next.title || `Новость #${next.id}`,
        description: stripHtmlToText(next.shortHtml || next.html).slice(0, 140),
        to: `${prefix}${next.id}`,
      }
      : null,
  };
});
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-0">

    <UContainer
      class="flex-1 min-h-0 overflow-y-auto max-w-full w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 scrollbar-hide mx-0">
      <!-- Loading skeleton -->
      <div v-if="loading" class="space-y-6">
        <USkeleton class="h-96 rounded-3xl" />
        <USkeleton class="h-64 rounded-3xl" />
      </div>

      <!-- Content -->
      <div v-else-if="item" class="flex flex-col gap-6 p-px">
        <!-- Hero -->
        <div class="relative overflow-hidden rounded-3xl ring ring-default bg-default">
          <div class="relative h-96">
            <img v-if="imageSrc" :src="imageSrc" :alt="title" class="absolute inset-0 h-full w-full object-cover"
              loading="lazy" />
            <div class="absolute inset-0"
              :class="imageSrc ? 'bg-linear-to-t from-black/85 via-black/35 to-transparent' : 'bg-linear-to-br from-primary/12 via-transparent to-primary/8'" />
            <div class="absolute inset-0 p-5 sm:p-8 flex flex-col justify-end">
              <div class="flex flex-wrap items-center gap-2 mb-3">
                <UBadge color="primary" variant="solid" size="md">Новости</UBadge>
                <UBadge v-if="date" color="neutral" variant="soft" size="md" class="backdrop-blur">
                  {{ date }}
                </UBadge>
              </div>
              <h1 class="text-2xl sm:text-4xl font-semibold tracking-tight"
                :class="imageSrc ? 'text-white' : 'text-highlighted'">
                {{ title }}
              </h1>
            </div>
          </div>
        </div>

        <!-- Body -->
        <UCard class="rounded-3xl">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-file-text" class="size-5 text-primary" />
              <span class="text-sm font-semibold text-highlighted">Текст новости</span>
            </div>
          </template>

          <div class="prose prose-neutral dark:prose-invert max-w-none" v-html="html" />
        </UCard>

        <UContentSurround v-if="surround.prev || surround.next" :prev="surround.prev" :next="surround.next" />
      </div>

      <!-- Not found -->
      <div v-else class="py-10">
        <UEmpty icon="i-lucide-file-question" title="Новость не найдена"
          description="Возможно, она была удалена или ссылка неверная." />
      </div>
    </UContainer>
  </UMain>
</template>
