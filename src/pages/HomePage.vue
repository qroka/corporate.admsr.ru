<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import type { ButtonProps } from '@nuxt/ui'
import type { BlogPostProps } from '@nuxt/ui'
import { useRouter } from 'vue-router';
import { useNewsData, resolveNewsImageSrc } from '../composables/useNewsData';
import { useNewsReactions } from '../composables/useNewsReactions';
import { useBirthdayColleagues } from '../composables/useBirthdayColleagues';
import { attachAbsenceStorageSync, hasActiveAbsence } from '../stores/absenceJournal';
import { currentRole } from '../stores/role';
import { useSectionAccess } from '../composables/useSectionAccess';
import { apiSessionUpload } from '../composables/useAuthSession';
import LearningHomeWidget from './Courses/components/LearningHomeWidget.vue';

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

const isAdmin = computed(() => currentRole.value === 'admin');

const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const canEditBirthdays = computed(() => canEditSection('birthdays'));

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
  attachAbsenceStorageSync();
  void fetchHomeEvents();
});

const actualEvents = computed<ActualEventItem[]>(() =>
  homeEvents.value
    .filter((e) => !isArchivedBadge(e.badge))
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

/** Превью без HTML: иначе обрезка по 220 символов рвёт теги и даёт мусор в тексте */
function newsPreviewText(html: string, maxLen: number): string {
  const plain = String(html ?? '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return plain.length > maxLen ? `${plain.slice(0, maxLen)}…` : plain;
}

function formatCountRu(n: number): string {
  const v = Number.isFinite(Number(n)) ? Number(n) : 0;
  return Math.max(0, Math.round(v)).toLocaleString('ru-RU');
}

const allNewsItems = computed<NewsItem[]>(() =>
  sortedNews.value.map((n) => {
    const imageSrc = resolveNewsImageSrc(n.imagePath);
    return {
      id: n.id,
      likes: n.likes ?? 0,
      views: n.views ?? 0,
      post: {
        title: n.title || `Новость #${n.id}`,
        description: newsPreviewText(n.description, 220),
        image: imageSrc ? { src: imageSrc, alt: n.title || 'Новость' } : { src: '/src/img/Logo.svg', alt: n.title || 'Новость' },
        to: `/news/${n.id}`,
        // ISO YYYY-MM-DD — UBlogPost сам форматирует дату; русская строка даёт Invalid Date и ломает рендер
        date: n.date || undefined,
        badge: n.category || 'Новости',
      },
    };
  }),
);

const newsItems = computed<NewsItem[]>(() => allNewsItems.value.slice(0, visibleNewsCount.value));
const hasMoreNews = computed(() => visibleNewsCount.value < allNewsItems.value.length);

const { isLiked: isNewsLiked, toggleLike: toggleNewsLike } = useNewsReactions();

function showMoreNews() {
  visibleNewsCount.value = Math.min(allNewsItems.value.length, visibleNewsCount.value + newsPageSize);
}

const { birthdayGroups, loading: birthdaysLoading, error: birthdaysError, ensureLoaded: ensureBirthdaysLoaded, reload: reloadBirthdays } =
  useBirthdayColleagues();
ensureBirthdaysLoaded();

// ── Админ: загрузка дат рождений из xlsx ──────────────────────────────────────
const MONTH_NAMES = ['январь', 'февраль', 'март', 'апрель', 'май', 'июнь', 'июль', 'август', 'сентябрь', 'октябрь', 'ноябрь', 'декабрь'];
const birthdayUploadOpen = ref(false);
const birthdayFile = ref<File | File[] | undefined>(undefined);
const birthdayUploading = ref(false);
const birthdayUploadError = ref<string | null>(null);
const birthdayManifest = ref<Record<string, { filename?: string; year?: number }>>({});

const birthdayMonths = computed(() =>
  MONTH_NAMES.map((name, i) => ({
    month: i + 1,
    name,
    filename: birthdayManifest.value[String(i + 1)]?.filename ?? null,
  })),
);

async function loadBirthdayManifest() {
  try {
    const res = await fetch('/api/birthdays.php?manifest=1');
    const json = await res.json();
    if (json.success) birthdayManifest.value = json.data ?? {};
  } catch { /* молча */ }
}

function openBirthdayUpload() {
  birthdayUploadError.value = null;
  birthdayFile.value = undefined;
  birthdayUploadOpen.value = true;
  void loadBirthdayManifest();
}

watch(birthdayFile, async (val) => {
  const file = Array.isArray(val) ? val[0] : val;
  if (!file) return;
  birthdayUploading.value = true;
  birthdayUploadError.value = null;
  try {
    const fd = new FormData();
    fd.append('file', file);
    const json = await apiSessionUpload('/api/birthdays.php', fd);
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки');
    birthdayManifest.value = (json.data as any) ?? birthdayManifest.value;
    await reloadBirthdays();
  } catch (e: any) {
    birthdayUploadError.value = e?.message ?? 'Ошибка загрузки';
  } finally {
    birthdayUploading.value = false;
    birthdayFile.value = undefined;
  }
});

</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full gap-4 min-h-0 max-h-full overflow-hidden">
    <UAlert
      v-if="hasActiveAbsence"
      color="primary"
      variant="solid"
      icon="i-lucide-timer"
      title="Есть незавершённое отсутствие"
      orientation="horizontal"
      description="Завершите запись в журнале отсутствия, чтобы убрать индикатор."
      :actions="[
        {
          label: 'Открыть журнал',
          color: 'neutral',
          variant: 'solid',
          size: 'md',
          onClick: () => void router.push({ name: 'absence-journal' }),
        },
      ]"
    />

    <UContainer class="flex flex-1 flex-row w-full gap-6 min-h-0 max-h-full overflow-hidden max-w-none">
    <UContainer
      v-if="eventsLoading || hasActualEvents"
      class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-96 shrink-0"
    >
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
      </UContainer>
    </UContainer>
    <UContainer class="flex flex-1 flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 min-h-0 min-w-0">
      <UPageHeader title="" :links="newsLinks" class="border-none p-0">
        <template #title>
          <h1 class="text-2xl font-medium">Лента новостей</h1>
        </template>
      </UPageHeader>

      <UScrollArea class="flex-1 min-h-0 min-w-0 sm:p-px md:p-px lg:p-px xl:p-px scrollbar-hide">
        <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0">
          <UBlogPost
            v-for="item in newsItems"
            :key="item.id"
            v-bind="item.post"
            class="w-full"
          >
            <template #description>
              <p class="text-base text-pretty text-muted">
                {{ item.post.description }}
              </p>
              <UContainer
                class="relative z-10 flex justify-between items-center gap-3 w-full mt-2"
                @click.stop
              >
                <UBadge
                  as="button"
                  type="button"
                  :color="isNewsLiked(item.id) ? 'primary' : 'neutral'"
                  variant="soft"
                  size="lg"
                  leading
                  icon="i-lucide-heart"
                  :label="formatCountRu(item.likes)"
                  :class="[
                    'relative z-10 cursor-pointer shrink-0 [&_svg]:stroke-[1.75]',
                    isNewsLiked(item.id)
                      ? '[&_svg]:stroke-primary [&_svg_path]:fill-primary [&_svg_path]:stroke-primary'
                      : '[&_svg]:stroke-current [&_svg_path]:fill-none [&_svg_path]:stroke-current',
                  ]"
                  @click.stop.prevent="toggleNewsLike(item.id)"
                />
                <span class="shrink-0 text-sm text-muted">
                  {{ formatCountRu(item.views) }} просмотров
                </span>
              </UContainer>
            </template>

          </UBlogPost>

          <UButton
            v-if="hasMoreNews"
            type="button"
            color="neutral"
            variant="outline"
            size="xl"
            class="relative z-10 w-full shrink-0 justify-center"
            icon="i-lucide-chevron-down"
            @click="showMoreNews"
          >
            Показать ещё
          </UButton>
        </UContainer>
      </UScrollArea>
    </UContainer>
    <UContainer class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-96 shrink-0">
      <LearningHomeWidget v-if="!isAdmin" class="shrink-0" />
      <UPageHeader title="" :links="birthdayLinks" class="border-none p-0">
        <template #title>
          <h1 class="text-2xl font-medium">Дни рождения коллег</h1>
        </template>
      </UPageHeader>
      <UButton
        v-if="canEditBirthdays"
        label="Загрузить даты xlsx"
        icon="i-lucide-upload"
        color="neutral"
        variant="outline"
        size="xl"
        block
        class="shrink-0"
        @click="openBirthdayUpload"
      />
      <UContainer class="overflow-y-auto sm:p-px md:p-px lg:p-px xl:p-px scrollbar-hide">
        <UContainer v-if="birthdaysLoading" class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-full">
          <USkeleton v-for="n in 3" :key="n" class="h-32 w-full rounded-lg" />
        </UContainer>
        <p v-else-if="birthdaysError" class="text-sm text-error w-full">
          {{ birthdaysError }}
        </p>
        <UContainer v-else class="flex flex-col gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-full">
          <UCard
            v-for="group in birthdayGroups"
            :key="group.id"
          >
            <UContainer class="flex flex-col gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-full">
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
              <p v-if="!group.people.length" class="text-sm text-muted w-full">
                В этот день никого нет
              </p>
              <UContainer
                v-for="person in group.people"
                :key="person.id"
                class="flex items-center gap-3 sm:p-0 md:p-0 lg:p-0 xl:p-0 w-full"
              >
                <UUser
                  :name="person.name"
                  :description="person.role"
                  :avatar="{ src: person.avatar }"
                  size="xl"
                  class="w-full"
                />
              </UContainer>
            </UContainer>
          </UCard>
        </UContainer>
      </UContainer>
    </UContainer>
  </UContainer>
  <!-- Админ: окно загрузки дат рождений (xlsx) -->
  <USlideover v-model:open="birthdayUploadOpen" side="right" title="Загрузка дат рождений (xlsx)" description="">
    <template #body>
      <div class="space-y-4">
        <UFileUpload
          v-model="birthdayFile"
          accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          label="Перетащите xlsx сюда"
          description="Один файл = один месяц. A1 — месяц, B1 — год, далее ФИО и дата."
          class="w-full min-h-32"
        />
        <UAlert
          v-if="birthdayUploadError"
          color="error"
          variant="subtle"
          icon="i-lucide-alert-circle"
          :description="birthdayUploadError"
        />
        <p v-if="birthdayUploading" class="text-sm text-muted">Загрузка…</p>

        <div class="flex flex-col divide-y divide-default rounded-lg ring ring-default">
          <div
            v-for="m in birthdayMonths"
            :key="m.month"
            class="flex items-center justify-between gap-3 px-3 py-2"
          >
            <span class="text-sm capitalize">{{ m.name }}</span>
            <span
              class="text-sm truncate max-w-[60%]"
              :class="m.filename ? 'text-default' : 'text-muted italic'"
              :title="m.filename || 'не загружено'"
            >
              {{ m.filename || 'не загружено' }}
            </span>
          </div>
        </div>
      </div>
    </template>
  </USlideover>
  </UMain>
</template>