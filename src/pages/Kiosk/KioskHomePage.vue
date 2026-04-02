<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import type { BlogPostProps } from '@nuxt/ui';
import type { ButtonProps } from '@nuxt/ui';
import { useRouter } from 'vue-router';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';

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
  sortedNews.value.map((n) => {
    const imageSrc = resolveNewsImageSrc(n.imagePath);
    return {
      id: n.id,
      title: n.title || `Новость #${n.id}`,
      description: n.description.slice(0, 260),
      date: formatNewsDate(n.date),
      badge: n.category || 'Новости',
      to: `/news/${n.id}`,
      image: imageSrc ? { src: imageSrc, alt: n.title } : { src: '/src/img/Logo.svg', alt: n.title },
      likes: 0,
      views: 0,
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
      imageSrc: resolveNewsImageSrc(n.imagePath) ?? '',
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

const items = [
  'src/img/EventsWebpFull/event_110_1.webp',
  'src/img/EventsWebpFull/event_110_50.webp',
  'src/img/EventsWebpFull/event_110_40.webp',
  'src/img/EventsWebpFull/event_110_70.webp',
]

</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col gap-6 w-full min-h-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <UCarousel v-slot="{ item }" :autoplay="{ delay: 5000 }" :items="items" class="w-full mx-0">
        <img :src="item" class="rounded-3xl object-cover w-full h-96" loading="lazy">
      </UCarousel>
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
              header: 'relative overflow-hidden w-full pointer-events-none',
              image: 'object-cover object-center w-full h-full',
            }"
          >
            <template #description>
              <p class="text-base text-pretty text-muted">
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
