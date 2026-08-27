<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import { useRouter } from 'vue-router';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';

type NewsPost = BlogPostProps & { id: string; likes: number; views: number };

const router = useRouter();

const { loading, error, sortedNews, ensureLoaded } = useNewsData();
ensureLoaded();

// Промо-ролик киоска. Файл лежит в web-корне (public/) и не бандлится Vite —
// поэтому путь абсолютный и передаётся через :src, а не статичным src.
const heroVideoSrc = '/kiosk-hero.mp4';
const heroVideoRef = ref<HTMLVideoElement | null>(null);

function restartHeroVideo() {
  const video = heroVideoRef.value;
  if (!video) return;
  video.currentTime = 0;
  void video.play().catch(() => {});
}

function setupHeroVideo() {
  const video = heroVideoRef.value;
  if (!video) return;
  video.loop = true;
  video.muted = true;
  void video.play().catch(() => {});
}

function stripHtml(html: string, maxLen: number): string {
  const plain = String(html ?? '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return plain.length > maxLen ? `${plain.slice(0, maxLen)}…` : plain;
}

const posts = computed<NewsPost[]>(() =>
  sortedNews.value.map((n) => {
    const imageSrc = resolveNewsImageSrc(n.imagePath);
    return {
      id: n.id,
      title: n.title || `Новость #${n.id}`,
      description: stripHtml(n.description, 260),
      date: formatNewsDate(n.date),
      badge: n.category || 'Новости',
      to: `/kiosk/news/${n.id}`,
      image: imageSrc ? { src: imageSrc, alt: n.title } : { src: '/src/img/Logo.svg', alt: n.title },
      likes: n.likes ?? 0,
      views: n.views ?? 0,
    };
  }),
);

function formatCountRu(n: number): string {
  const v = Number.isFinite(Number(n)) ? Number(n) : 0;
  return Math.max(0, Math.round(v)).toLocaleString('ru-RU');
}

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
      imageSrc: resolveNewsImageSrc(n.imagePath) ?? '',
      to: `/kiosk/news/${n.id}`,
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
  setupHeroVideo();
  startSliderAuto();
  window.addEventListener('kiosk-idle', handleKioskIdle);
});
onUnmounted(() => {
  stopSliderAuto();
  window.removeEventListener('kiosk-idle', handleKioskIdle);
});

</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col gap-6 w-full min-h-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <div class="relative w-full aspect-video rounded-3xl overflow-hidden bg-black">
        <video
          ref="heroVideoRef"
          class="absolute inset-0 w-full h-full object-cover"
          autoplay
          muted
          loop
          playsinline
          preload="auto"
          :src="heroVideoSrc"
          @ended="restartHeroVideo"
        />
      </div>
      <div class="flex items-center justify-between gap-4 w-full">
        <h1 class="text-4xl font-normal font-unbounded">Лента новостей</h1>
        <UButton
          to="/kiosk/news"
          size="xl"
          color="neutral"
          variant="outline"
          trailing-icon="i-lucide-arrow-right"
        >
          Смотреть все новости
        </UButton>
      </div>
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
              header: 'relative overflow-hidden w-full pointer-events-none',
              image: 'object-cover object-center w-full h-full',
            }"
          >
            <template #description>
              <p class="text-xl leading-relaxed text-pretty text-muted">
                {{ post.description }}
              </p>
            </template>
          </UBlogPost>
          <UEmpty v-if="!loading && !posts.length" icon="i-lucide-newspaper" title="Новостей пока нет"
            description="Попробуйте зайти позже." />
        </UContainer>
      </UScrollArea>
    </UContainer>
  </UMain>

</template>
