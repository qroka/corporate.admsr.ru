<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import type { ButtonProps } from '@nuxt/ui'
import type { BlogPostProps } from '@nuxt/ui'
import { useRouter } from 'vue-router';
import { useNewsData, formatUnixDate, resolveNewsImageSrc, stripHtmlToText } from '../composables/useNewsData';
import { useBirthdayColleagues } from '../composables/useBirthdayColleagues';

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
    to: '/birthdays',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
])

/** Карточки «Актуальные события»: клик ведёт на детальную страницу мероприятия */
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

type HomeEventRecord = {
  id: number;
  title: string;
  description: string;
  date: string;
  badge?: string;
  image?: string;
  image_full?: string;
  coverIndex?: number;
};

const homeEvents = ref<HomeEventRecord[]>([]);
const eventsLoading = ref(false);
const eventsError = ref<string | null>(null);

function todayIsoDate(): string {
  return new Date().toISOString().slice(0, 10);
}

function isArchivedBadge(value: unknown) {
  return String(value ?? '').trim().toLowerCase().includes('архив');
}

function isNewBadge(value: unknown) {
  return String(value ?? '').trim().toLowerCase().includes('нов');
}

function mapHomeEvent(raw: any): HomeEventRecord {
  return {
    id: Number(raw?.id),
    title: String(raw?.title ?? '').trim(),
    description: String(raw?.description ?? '').trim(),
    date: String(raw?.date ?? '').trim(),
    badge: raw?.badge ? String(raw.badge) : undefined,
    image: raw?.image ? String(raw.image) : undefined,
    image_full: raw?.image_full ? String(raw.image_full) : undefined,
    coverIndex: typeof raw?.coverIndex === 'number' ? raw.coverIndex : undefined,
  };
}

async function fetchHomeEvents() {
  eventsLoading.value = true;
  eventsError.value = null;
  try {
    const res = await fetch('/api/events.php');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки мероприятий');
    const arr = Array.isArray(json.data) ? json.data : [];
    homeEvents.value = arr.map(mapHomeEvent).filter((e) => e.id && e.title && e.date);
  } catch (e: any) {
    eventsError.value = e?.message ?? 'Не удалось загрузить мероприятия';
    homeEvents.value = [];
  } finally {
    eventsLoading.value = false;
  }
}

onMounted(() => {
  void fetchHomeEvents();
});

const actualEvents = computed<ActualEventItem[]>(() =>
  homeEvents.value
    .filter((e) => !isArchivedBadge(e.badge))
    .filter((e) => String(e.date ?? '') >= todayIsoDate())
    .filter((e) => isNewBadge(e.badge))
    .sort((a, b) => String(b.date ?? '').localeCompare(String(a.date ?? ''), 'ru-RU') || (b.id ?? 0) - (a.id ?? 0))
    .map((e, idx) => ({
      id: `evt-${e.id}`,
      eventId: e.id,
      post: {
        title: e.title,
        description: e.description,
        image: e.image_full || e.image || eventCoverAt(e.coverIndex ?? idx),
        date: e.date,
        badge: e.badge,
      },
    })),
);

const hasActualEvents = computed(() => !eventsLoading.value && !eventsError.value && actualEvents.value.length > 0);

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

const newsPageSize = 6;
const visibleNewsCount = ref(newsPageSize);

const allNewsItems = computed<NewsItem[]>(() =>
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

const newsItems = computed<NewsItem[]>(() => allNewsItems.value.slice(0, visibleNewsCount.value));
const hasMoreNews = computed(() => newsItems.value.length < allNewsItems.value.length);

function showMoreNews() {
  visibleNewsCount.value = Math.min(allNewsItems.value.length, visibleNewsCount.value + newsPageSize);
}

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

const { birthdayGroups, loading: birthdaysLoading, error: birthdaysError, ensureLoaded: ensureBirthdaysLoaded } =
  useBirthdayColleagues();
ensureBirthdaysLoaded();

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
        <UContainer v-if="eventsLoading" class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
          <USkeleton v-for="n in 3" :key="n" class="h-32 w-full rounded-2xl" />
        </UContainer>
        <UContainer v-else-if="hasActualEvents" class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
          <UBlogPost
            v-for="item in actualEvents"
            :key="item.id"
            v-bind="item.post"
            class="w-full cursor-pointer"
            @click="openEventDetails(item.eventId)"
          >
            <template #badge>
              <UBadge v-if="item.post.badge" color="primary" variant="solid">
                {{ item.post.badge }}
              </UBadge>
            </template>
          </UBlogPost>
        </UContainer>
        <p v-else-if="eventsError" class="px-1 py-2 text-sm text-error">
          {{ eventsError }}
        </p>
        <p v-else class="px-1 py-2 text-sm text-muted">
          Сейчас нет актуальных мероприятий.
        </p>
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

          <UButton
            v-if="hasMoreNews"
            type="button"
            color="neutral"
            variant="outline"
            size="xl"
            class="w-full justify-center"
            icon="i-lucide-chevron-down"
            @click="showMoreNews"
          >
            Показать ещё
          </UButton>
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
        <UContainer v-if="birthdaysLoading" class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-fit">
          <USkeleton v-for="n in 3" :key="n" class="h-32 w-96 rounded-lg" />
        </UContainer>
        <p v-else-if="birthdaysError" class="text-sm text-error max-w-96">
          {{ birthdaysError }}
        </p>
        <UContainer v-else class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-fit">
          <UCard
            v-for="group in birthdayGroups"
            :key="group.id"
          >
            <UContainer class="flex flex-col gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-fit">
              <UContainer class="flex items-center gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
                <h2 class="text-2xl font-medium leading-none text-highlighted">
                  {{ group.dateLabel }}
                </h2>
                <UBadge
                  :color="group.dayColor"
                  :variant="group.dayLabel === 'Сегодня' ? 'solid' : 'subtle'"
                  size="md"
                  :label="group.dayLabel"
                />
              </UContainer>
              <p v-if="!group.people.length" class="text-sm text-muted max-w-96">
                В этот день никого нет
              </p>
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