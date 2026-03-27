<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { ButtonProps } from '@nuxt/ui'
import type { BlogPostProps } from '@nuxt/ui'
import { useRouter } from 'vue-router';
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../composables/useNewsData';
import { useEventsData } from '../composables/useEventsData';
import { useAppToast } from '../composables/useAppToast';

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
  eventId: number;
  post: BlogPostProps;
};

const router = useRouter();

const eventCoverModules = (import.meta as any).glob('../img/EventsWebp/*.webp', {
  eager: true,
  import: 'default',
});
const eventCoverSrcs = Object.entries(eventCoverModules)
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([, src]) => src as string);
function eventCoverAt(index: number): string {
  return eventCoverSrcs.length ? eventCoverSrcs[index % eventCoverSrcs.length] : '/src/img/Logo.svg';
}

const { events, ensureLoaded: ensureEventsLoaded } = useEventsData();
ensureEventsLoaded();
const { toast } = useAppToast();

const actualEvents = computed<ActualEventItem[]>(() =>
  events.value
    .filter((e) => String(e.badge ?? '').trim().toLowerCase().includes('нов'))
    .sort((a, b) => String(b.date ?? '').localeCompare(String(a.date ?? ''), 'ru-RU') || (b.id ?? 0) - (a.id ?? 0))
    .map((e, idx) => ({
      id: `evt-${e.id}`,
      eventId: e.id,
      post: {
        title: e.title,
        description: e.description,
        image: eventCoverAt(e.coverIndex ?? idx),
        date: e.date,
        badge: e.badge,
      },
    })),
);

const eventParticipation = ref<Record<string, boolean>>({});

watch(
  actualEvents,
  (items) => {
    const next = { ...eventParticipation.value };
    for (const item of items) {
      if (!(item.id in next)) next[item.id] = false;
    }
    eventParticipation.value = next;
  },
  { immediate: true },
);

function toggleEventParticipation(eventId: string) {
  const wasJoined = !!eventParticipation.value[eventId];
  eventParticipation.value = { ...eventParticipation.value, [eventId]: !wasJoined };

  toast.add({
    title: wasJoined ? 'Запись отменена' : 'Вы записались на мероприятие',
    description: wasJoined ? 'Вы больше не в списке участников (демо).' : 'Добавили вас в список участников (демо).',
    color: 'success',
    icon: 'i-lucide-circle-check',
  });
}

function openEventDetails(eventId: number) {
  void router.push(`/events/${eventId}`);
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
  sortedNews.value.map((n) => {
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
            class="w-full cursor-pointer"
            @click="openEventDetails(item.eventId)"
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
                  @click.stop.prevent="toggleEventParticipation(item.id)"
                >
                  Записаться на мероприятие
                </UButton>
                <UButton
                  v-else
                  color="primary"
                  variant="soft"
                  size="lg"
                  class="w-full justify-center rounded-full"
                  @click.stop.prevent="toggleEventParticipation(item.id)"
                >
                  Отписаться от мероприятия
                </UButton>
              </UContainer>
            </template>
          </UBlogPost>
        </UContainer>
      </UContainer>
    </UContainer>
    <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 min-h-0">
      <UPageHeader title="" :links="newsLinks" class="border-none p-0">
        <template #title>
          <h1 class="text-2xl font-medium">Лента новостей</h1>
        </template>
      </UPageHeader>
      <UScrollArea class="flex-1 min-h-0 sm:p-px md:p-px lg:p-px xl:p-px scrollbar-hide">
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
      </UScrollArea>
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