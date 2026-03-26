<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import type { ButtonProps } from '@nuxt/ui';
import { useRouter } from 'vue-router';
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../../composables/useNewsData';

type NewsPost = BlogPostProps & { id: string; likes: number; views: number };

const router = useRouter();

const { loading, error, sortedNews, ensureLoaded } = useNewsData();
ensureLoaded();

const links = <ButtonProps[]>([
  {
    icon: 'i-lucide-arrow-up-right',
    to: '/kiosk/news',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
])

const posts = computed<NewsPost[]>(() =>
  sortedNews.value.map((n, idx) => {
    const imageSrc = resolveNewsImageSrc(n.announceImagePath);
    const date = formatUnixDate(n.timestamp);
    const description = stripHtmlToText(n.shortHtml || n.html).slice(0, 260);
    return {
      id: n.id,
      title: n.title || `Новость #${n.id}`,
      description,
      date,
      badge: 'Новости',
      to: `/news/${n.id}`,
      image: imageSrc ? { src: imageSrc, alt: n.title } : { src: '/src/img/Logo.svg', alt: n.title },
      likes: 24 + (idx % 7) * 13,
      views: 120 + (idx % 11) * 97,
    };
  }),
);

type SliderItem = {
  id: string;
  title: string;
  imageSrc: string;
  to: string;
};

const sliderItems = computed<SliderItem[]>(() =>
  sortedNews.value
    .map((n) => ({
      id: n.id,
      title: n.title || `Новость #${n.id}`,
      imageSrc: resolveNewsImageSrc(n.announceImagePath) ?? '',
      to: `/news/${n.id}`,
    }))
    .filter((x) => x.imageSrc.length)
    .slice(0, 8),
);

const sliderRef = ref<HTMLElement | null>(null);
const sliderIndex = ref(0);
let sliderTimer: number | null = null;

const scrollAreaRef = ref<any>(null);

function setSliderIndex(next: number) {
  const len = sliderItems.value.length;
  if (!len) {
    sliderIndex.value = 0;
    return;
  }
  sliderIndex.value = ((next % len) + len) % len;
  const el = sliderRef.value;
  if (!el) return;
  const w = el.clientWidth;
  el.scrollTo({ left: sliderIndex.value * w, behavior: 'smooth' });
}

function startSliderAuto() {
  stopSliderAuto();
  if (sliderItems.value.length <= 1) return;
  sliderTimer = window.setInterval(() => setSliderIndex(sliderIndex.value + 1), 7000);
}

function stopSliderAuto() {
  if (sliderTimer) window.clearInterval(sliderTimer);
  sliderTimer = null;
}

function getScrollViewportEl(): HTMLElement | null {
  const root = (scrollAreaRef.value?.$el ?? scrollAreaRef.value) as HTMLElement | undefined;
  if (!root) return null;
  return (
    root.querySelector<HTMLElement>('[data-radix-scroll-area-viewport]') ??
    root.querySelector<HTMLElement>('[data-scroll-area-viewport]') ??
    root.querySelector<HTMLElement>('.scroll-area-viewport') ??
    root
  );
}

function handleKioskIdle() {
  const viewport = getScrollViewportEl();
  if (!viewport) return;
  if (viewport.scrollTop <= 0) return;
  viewport.scrollTo({ top: 0, behavior: 'smooth' });
}

onMounted(() => {
  startSliderAuto();
  window.addEventListener('kiosk-idle', handleKioskIdle);
});
onUnmounted(() => {
  stopSliderAuto();
  window.removeEventListener('kiosk-idle', handleKioskIdle);
});

const newsLiked = ref<Record<string, boolean>>({});
const newsLikeCounts = ref<Record<string, number>>({});

watch(
  posts,
  (items) => {
    const liked = { ...newsLiked.value };
    const counts = { ...newsLikeCounts.value };
    for (const item of items) {
      if (!(item.id in liked)) liked[item.id] = false;
      if (!(item.id in counts)) counts[item.id] = item.likes ?? 0;
    }
    newsLiked.value = liked;
    newsLikeCounts.value = counts;
  },
  { immediate: true },
);

function toggleNewsLike(newsId: string) {
  const wasLiked = !!newsLiked.value[newsId];
  const currentCount = newsLikeCounts.value[newsId] ?? 0;

  newsLiked.value = { ...newsLiked.value, [newsId]: !wasLiked };
  newsLikeCounts.value = {
    ...newsLikeCounts.value,
    [newsId]: wasLiked ? Math.max(0, currentCount - 1) : currentCount + 1,
  };
}

function formatViews(n: number) {
  return n.toLocaleString('ru-RU');
}

const items = [
  'src/img/EventsWebpFull/event_110_73.webp',
  'src/img/EventsWebpFull/event_110_73.webp',
  'src/img/EventsWebpFull/event_110_73.webp',
  'src/img/EventsWebpFull/event_110_73.webp',
]

</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col gap-6 w-full min-h-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <UPageHeader  :links="links" class="border-none p-0">
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Лента новостей</h1>
        </template>
      </UPageHeader>
      <UScrollArea ref="scrollAreaRef" class="flex-1 min-h-0 scrollbar-hide">
        <UContainer class="flex flex-col gap-6 sm:p-px md:p-px lg:p-px xl:p-px mx-0 w-full">
          <USkeleton v-if="loading" class="h-32 w-full" />
          <UBlogPost
            v-for="post in posts"
            v-else
            :key="post.id"
            v-bind="post"
            class="w-full rounded-3xl"
            :ui="{
              header: 'relative overflow-hidden w-full pointer-events-none h-96',
              image: 'object-cover object-center w-full h-full',
            }"
          >
            <template #description>
              <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
                <p class="text-base text-pretty text-muted">
                  {{ post.description }}
                </p>
                <UContainer
                  class="relative z-10 flex justify-between items-center gap-3 w-full sm:p-0 md:p-0 lg:p-0 xl:p-0"
                  @click.stop>
                  <UBadge as="button" type="button" :color="newsLiked[post.id] ? 'primary' : 'neutral'" variant="soft"
                    size="lg" leading icon="i-lucide-heart" :label="formatViews(newsLikeCounts[post.id] ?? post.likes)"
                    :class="[
                      'relative z-10 cursor-pointer shrink-0 [&_svg]:stroke-[1.75]',
                      newsLiked[post.id]
                        ? '[&_svg]:stroke-primary [&_svg_path]:fill-primary [&_svg_path]:stroke-primary'
                        : '[&_svg]:stroke-current [&_svg_path]:fill-none [&_svg_path]:stroke-current',
                    ]" @click.stop.prevent="toggleNewsLike(post.id)" />
                  <span class="shrink-0 text-sm text-muted">
                    {{ formatViews(post.views) }} просмотров
                  </span>
                </UContainer>
              </UContainer>
            </template>
          </UBlogPost>
          <UEmpty v-if="!loading && !posts.length" icon="i-lucide-newspaper" title="Новостей пока нет"
            description="Попробуйте зайти позже." />
        </UContainer>
      </UScrollArea>
    </UContainer>
  </UMain>

</template>
