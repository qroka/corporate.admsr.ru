<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { ButtonProps } from '@nuxt/ui'
import type { BlogPostProps } from '@nuxt/ui'
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../composables/useNewsData';

const eventsLinks = <ButtonProps[]>([
  {
    icon: 'i-lucide-arrow-up-right',
    to: '/events',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
])

const newsLinks = <ButtonProps[]>([
  {
    icon: 'i-lucide-arrow-up-right',
    to: '/news',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
])

const birthdayLinks = <ButtonProps[]>([
  {
    icon: 'i-lucide-arrow-up-right',
    to: '/profile',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
])

/** Карточки «Актуальные события»: без `to` на карточке — кнопки записи в слоте описания (body) */
type ActualEventItem = {
  id: string;
  post: BlogPostProps;
};

const actualEvents = <ActualEventItem[]>([
  {
    id: 'evt-1',
    post: {
      title: 'Nuxt Icon v1',
      description: 'Discover Nuxt Icon v1!',
      image: 'https://nuxt.com/assets/blog/nuxt-icon/cover.png',
    },
  },
  {
    id: 'evt-2',
    post: {
      title: 'Nuxt 3.14',
      description: 'Nuxt 3.14 is out!',
      image: 'https://nuxt.com/assets/blog/v3.14.png',
    },
  },
  {
    id: 'evt-3',
    post: {
      title: 'Nuxt 3.13',
      description: 'Nuxt 3.13 is out!',
      image: 'https://nuxt.com/assets/blog/v3.13.png',
    },
  },
]);

/** Записан ли пользователь на мероприятие (локально, для демо UI) */
const eventParticipation = ref<Record<string, boolean>>({
  'evt-1': false,
  'evt-2': true,
  'evt-3': false,
});

function setParticipation(eventId: string, value: boolean) {
  eventParticipation.value = { ...eventParticipation.value, [eventId]: value };
}

/** Лента новостей: карточка + просмотры; лайк — отдельное состояние */
type NewsItem = {
  id: string;
  likes: number;
  views: number;
  post: BlogPostProps;
};

const { sortedNews, ensureLoaded: ensureNewsLoaded } = useNewsData();
ensureNewsLoaded();

const newsItems = computed<NewsItem[]>(() =>
  sortedNews.value.slice(0, 12).map((n) => {
    const imageSrc = resolveNewsImageSrc(n.announceImagePath);
    const date = formatUnixDate(n.timestamp);
    const description = stripHtmlToText(n.shortHtml || n.html).slice(0, 220);
    return {
      id: `news-${n.id}`,
      likes: 0,
      views: 0,
      post: {
        title: n.title || `Новость #${n.id}`,
        description,
        image: imageSrc ?? '/src/img/Logo.svg',
        to: `/news/${n.id}`,
        date,
        badge: 'Новости',
      },
    };
  }),
);

const newsLiked = ref<Record<string, boolean>>({});

const newsLikeCounts = ref<Record<string, number>>({});

watch(
  newsItems,
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

type BirthdayPerson = {
  id: string;
  name: string;
  role: string;
  avatar: string;
};

type BirthdayGroup = {
  id: string;
  dateLabel: string;
  dayLabel: string;
  dayColor: 'primary' | 'neutral';
  people: BirthdayPerson[];
};

const birthdayGroups = <BirthdayGroup[]>[
  {
    id: 'bday-today',
    dateLabel: '12 марта',
    dayLabel: 'Сегодня',
    dayColor: 'primary',
    people: [
      { id: 'p1', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
      { id: 'p2', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
      { id: 'p3', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
      { id: 'p4', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
    ],
  },
  {
    id: 'bday-tomorrow',
    dateLabel: '13 марта',
    dayLabel: 'Завтра',
    dayColor: 'neutral',
    people: [
      { id: 'p5', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
      { id: 'p6', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
    ],
  },
  {
    id: 'bday-after-tomorrow',
    dateLabel: '14 марта',
    dayLabel: 'Послезавтра',
    dayColor: 'neutral',
    people: [
      { id: 'p7', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
      { id: 'p8', name: 'Константинопольский Константин', role: 'Инженер', avatar: '/src/img/sticker 1.png' },
    ],
  },
];

</script>

<template>
  <UMain class="flex flex-1 flex-row w-full h-full gap-6 min-h-0 max-h-full overflow-hidden">
    <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 max-w-96">
      <UPageHeader title="" :links="eventsLinks" class="border-none p-0">
        <template #title>
          <h1 class="text-2xl font-medium">Актуальные мероприятия</h1>
        </template>
      </UPageHeader>
      <UContainer class="overflow-y-auto sm:p-px md:p-px lg:p-px xl:p-px scrollbar-hide">
        <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
          <UBlogPost
            v-for="item in actualEvents"
            :key="item.id"
            v-bind="item.post"
            class="w-full"
          >
            <template #description>
              <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
                <p class="text-base text-pretty text-muted">
                  {{ item.post.description }}
                </p>
                <UButton
                  v-if="!eventParticipation[item.id]"
                  color="primary"
                  size="lg"
                  class="w-full justify-center rounded-full"
                  @click="setParticipation(item.id, true)"
                >
                  Записаться на мероприятие
                </UButton>
                <UButton
                  v-else
                  color="primary"
                  variant="soft"
                  size="lg"
                  class="w-full justify-center rounded-full"
                  @click="setParticipation(item.id, false)"
                >
                  Отписаться от мероприятия
                </UButton>
              </UContainer>
            </template>
          </UBlogPost>
        </UContainer>
      </UContainer>
    </UContainer>
    <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
      <UPageHeader title="" :links="newsLinks" class="border-none p-0">
        <template #title>
          <h1 class="text-2xl font-medium">Лента новостей</h1>
        </template>
      </UPageHeader>
      <UContainer class="overflow-y-auto sm:p-px md:p-px lg:p-px xl:p-px scrollbar-hide">
        <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
          <UBlogPost
            v-for="item in newsItems"
            :key="item.id"
            v-bind="item.post"
            class="w-full"
          >
            <template #description>
              <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
                <p class="text-base text-pretty text-muted">
                  {{ item.post.description }}
                </p>
                <UContainer
                  class="relative z-10 flex justify-between items-center gap-3 w-full sm:p-0 md:p-0 lg:p-0 xl:p-0"
                  @click.stop
                >
                  <UBadge
                    as="button"
                    type="button"
                    :color="newsLiked[item.id] ? 'primary' : 'neutral'"
                    variant="soft"
                    size="lg"
                    leading
                    icon="i-lucide-heart"
                    :label="formatViews(newsLikeCounts[item.id] ?? item.likes)"
                    :class="[
                      'relative z-10 cursor-pointer shrink-0 [&_svg]:stroke-[1.75]',
                      newsLiked[item.id]
                        ? '[&_svg]:stroke-primary [&_svg_path]:fill-primary [&_svg_path]:stroke-primary'
                        : '[&_svg]:stroke-current [&_svg_path]:fill-none [&_svg_path]:stroke-current',
                    ]"
                    @click.stop.prevent="toggleNewsLike(item.id)"
                  />
                  <span class="shrink-0 text-sm text-muted">
                    {{ formatViews(item.views) }} просмотров
                  </span>
                </UContainer>
              </UContainer>
            </template>
          </UBlogPost>
        </UContainer>
      </UContainer>
    </UContainer>
    <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-fit">
      <UPageHeader title="" :links="birthdayLinks" class="border-none p-0">
        <template #title>
          <h1 class="text-2xl font-medium">Дни рождения коллег</h1>
        </template>
      </UPageHeader>
      <UContainer class="overflow-y-auto sm:p-px md:p-px lg:p-px xl:p-px scrollbar-hide">
        <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-fit">
          <UCard
            v-for="group in birthdayGroups"
            :key="group.id"
          >

            <UContainer class="flex flex-col gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-fit">
              <UContainer class="flex items-center gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
                <h1 class="text-2xl font-medium leading-none text-highlighted">
                  {{ group.dateLabel }}
                </h1>
                <UBadge
                  :color="group.dayColor"
                  :variant="group.dayLabel === 'Сегодня' ? 'solid' : 'subtle'"
                  size="md"
                  :label="group.dayLabel"
                />
              </UContainer>
              <UContainer
                v-for="person in group.people"
                :key="person.id"
                class="flex items-center gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-96"
              >
                <UUser
                  :name="person.name"
                  :description="person.role"
                  :avatar="{ src: person.avatar }"
                  size="xl"
                  class="w-96"
                />
              </UContainer>
            </UContainer>
          </UCard>
        </UContainer>
      </UContainer>
    </UContainer>
  </UMain>
</template>