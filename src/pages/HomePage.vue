<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import type { TabsItem } from '@nuxt/ui';
import { useRouter } from 'vue-router';
import { useNewsData, resolveNewsImageSrc } from '../composables/useNewsData';
import { useNewsReactions } from '../composables/useNewsReactions';
import { useBirthdayColleagues } from '../composables/useBirthdayColleagues';
import { attachAbsenceStorageSync, hasActiveAbsence } from '../stores/absenceJournal';
import { currentRole } from '../stores/role';
import { useSectionAccess } from '../composables/useSectionAccess';
import { apiSessionUpload } from '../composables/useAuthSession';
import { useHeaderUser } from '../composables/useHeaderUser';
import { useOfoData } from '../composables/useOfoData';
import { useAppToast } from '../composables/useAppToast';
import LearningHomeWidget from './Courses/components/LearningHomeWidget.vue';
import HomeNewsCard from '../components/home/HomeNewsCard.vue';

const router = useRouter();
const { toast, success } = useAppToast();

const isAdmin = computed(() => currentRole.value === 'admin');

const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const canEditBirthdays = computed(() => canEditSection('birthdays'));

const { profile, headerName } = useHeaderUser();
const { ofoRows, ensureDirectoryLoaded } = useOfoData();
ensureDirectoryLoaded();

const greetingTitle = computed(() => {
  const hour = new Date().getHours();
  let prefix = 'Добрый день';
  if (hour < 6) prefix = 'Доброй ночи';
  else if (hour < 12) prefix = 'Доброе утро';
  else if (hour >= 18) prefix = 'Добрый вечер';

  const p = profile.value;
  const namePatronymic = [p?.firstname, p?.lastname].filter(Boolean).join(' ').trim();
  const name = namePatronymic || headerName.value || 'коллега';
  return `${prefix}, ${name}!`;
});

const ofoTabLabel = computed(() => {
  const id = String(profile.value?.ofo ?? '').trim();
  if (!id) return 'Моё ОФО';
  const row = ofoRows.value.find((r) => String(r.id) === id);
  return row?.title?.trim() || 'Моё ОФО';
});

type HomeService = {
  id: string;
  label: string;
  icon: string;
  to?: string;
  action?: 'sed';
};

const homeServices: HomeService[] = [
  {
    id: 'absence',
    label: 'Журнал отсутствия',
    icon: 'i-lucide-brain-circuit',
    to: '/absence-journal',
  },
  {
    id: 'applications',
    label: 'Заявки',
    icon: 'i-lucide-brain-circuit',
    to: '/applications',
  },
  {
    id: 'sed',
    label: 'СЭД',
    icon: 'i-lucide-brain-circuit',
    action: 'sed',
  },
  {
    id: 'knowledge',
    label: 'Справочник',
    icon: 'i-lucide-brain-circuit',
    to: '/knowledge-base',
  },
  {
    id: 'all',
    label: 'Все сервисы',
    icon: 'i-lucide-layout-grid',
    to: '/services',
  },
];

function onSedClick() {
  toast.add({
    title: 'СЭД',
    description: 'Внешняя ссылка на систему электронного документооборота пока не настроена.',
    color: 'neutral',
    icon: 'i-lucide-file-stack',
  });
}

type HomeEventRecord = {
  id: number;
  title: string;
  description: string;
  date: string;
  badge?: string;
  image?: string;
  image_full?: string;
};

const homeEvents = ref<HomeEventRecord[]>([]);
const eventsLoading = ref(false);
const eventsError = ref<string | null>(null);

function isArchivedBadge(value: unknown) {
  return String(value ?? '').trim().toLowerCase().includes('архив');
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

const upcomingEvents = computed(() =>
  homeEvents.value
    .filter((e) => !isArchivedBadge(e.badge))
    .sort((a, b) => String(a.date ?? '').localeCompare(String(b.date ?? ''), 'ru-RU') || (a.id ?? 0) - (b.id ?? 0))
    .slice(0, 5),
);

function openEventDetails(eventId: number) {
  void router.push(`/events/${eventId}`);
}

type NewsFeedItem = {
  id: string;
  likes: number;
  views: number;
  title: string;
  description: string;
  imageSrc: string;
  to: string;
  date?: string;
  createdAt?: string | null;
  category: string;
};

const { sortedNews, ensureLoaded: ensureNewsLoaded } = useNewsData();
ensureNewsLoaded();

const newsPageSize = 6;
const visibleNewsCount = ref(newsPageSize);
const newsTab = ref<'feed' | 'ofo'>('feed');

function newsPreviewText(html: string, maxLen: number): string {
  const plain = String(html ?? '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return plain.length > maxLen ? `${plain.slice(0, maxLen)}…` : plain;
}

const allNewsItems = computed<NewsFeedItem[]>(() =>
  sortedNews.value.map((n) => {
    const imageSrc = resolveNewsImageSrc(n.imagePath);
    return {
      id: n.id,
      likes: n.likes ?? 0,
      views: n.views ?? 0,
      title: n.title || `Новость #${n.id}`,
      description: newsPreviewText(n.description, 220),
      imageSrc: imageSrc || '/src/img/Logo.svg',
      to: `/news/${n.id}`,
      date: n.date || undefined,
      createdAt: n.createdAt,
      category: n.category || 'Новости',
    };
  }),
);

/** У новостей нет поля ОФО — вкладка показывает совпадение категории с названием ОФО, иначе пусто. */
const ofoFilteredNews = computed(() => {
  const label = ofoTabLabel.value.trim().toLowerCase();
  if (!label || label === 'моё офо') return [] as NewsFeedItem[];
  return allNewsItems.value.filter((n) => n.category.toLowerCase().includes(label));
});

const activeNewsPool = computed(() =>
  newsTab.value === 'ofo' ? ofoFilteredNews.value : allNewsItems.value,
);

const newsItems = computed(() => activeNewsPool.value.slice(0, visibleNewsCount.value));
const hasMoreNews = computed(() => visibleNewsCount.value < activeNewsPool.value.length);

watch(newsTab, () => {
  visibleNewsCount.value = newsPageSize;
});

const newsTabItems = computed<TabsItem[]>(() => [
  { label: 'Лента новостей', value: 'feed' },
  { label: ofoTabLabel.value, value: 'ofo' },
]);

const { isLiked: isNewsLiked, toggleLike: toggleNewsLike } = useNewsReactions();

function showMoreNews() {
  visibleNewsCount.value = Math.min(activeNewsPool.value.length, visibleNewsCount.value + newsPageSize);
}

const {
  birthdayGroups,
  loading: birthdaysLoading,
  error: birthdaysError,
  ensureLoaded: ensureBirthdaysLoaded,
  reload: reloadBirthdays,
} = useBirthdayColleagues();
ensureBirthdaysLoaded();

const visibleBirthdayGroups = computed(() =>
  birthdayGroups.value.filter((g) => g.people.length > 0),
);

function congratulate(name: string) {
  success('Поздравление', `Открыть карточку «${name}» можно на странице дней рождения.`);
  void router.push('/birthdays');
}

// ── Админ: загрузка дат рождений из xlsx ──────────────────────────────────────
const MONTH_NAMES = [
  'январь',
  'февраль',
  'март',
  'апрель',
  'май',
  'июнь',
  'июль',
  'август',
  'сентябрь',
  'октябрь',
  'ноябрь',
  'декабрь',
];
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
  } catch {
    /* молча */
  }
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

onMounted(() => {
  attachAbsenceStorageSync();
  void fetchHomeEvents();
});
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 max-h-full overflow-hidden">
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

    <div class="flex w-full flex-1 flex-col gap-2 min-h-0 overflow-y-auto scrollbar-hide">
      <UPageHeader
        :title="greetingTitle"
        class="border-none py-4 px-0"
        :ui="{ title: 'text-2xl font-bold leading-8 text-highlighted' }"
      />

      <div class="flex flex-col xl:flex-row gap-4 min-h-0 items-start">
        <div class="flex min-w-0 flex-1 flex-col gap-4">
      <section class="flex flex-col gap-4" aria-labelledby="home-services-title">
        <div class="flex items-center justify-between gap-1">
          <div class="flex items-center gap-1 min-w-0">
            <h2 id="home-services-title" class="text-lg font-bold leading-7 text-highlighted">
              Сервисы
            </h2>
            <UTooltip text="Быстрый доступ к корпоративным сервисам">
              <UButton
                type="button"
                color="neutral"
                variant="ghost"
                size="xs"
                icon="i-lucide-info"
                square
                aria-label="О сервисах"
              />
            </UTooltip>
          </div>
          <UButton
            to="/services"
            color="neutral"
            variant="ghost"
            size="xs"
            icon="i-lucide-pencil"
            square
            aria-label="Все сервисы"
          />
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
          <UPageCard
            v-for="svc in homeServices"
            :key="svc.id"
            :title="svc.label"
            :icon="svc.icon"
            :to="svc.to"
            :on-click="svc.action === 'sed' ? onSedClick : undefined"
            variant="soft"
            class="bg-elevated/50"
            :ui="{
              root: 'h-[92px] cursor-pointer rounded-[10px] ring-0 border-0 bg-elevated/50',
              container: 'items-center justify-center text-center gap-2 p-4 h-full',
              wrapper: 'items-center',
              leading: 'mb-0',
              leadingIcon: svc.id === 'all' ? 'size-8 text-muted' : 'size-8 text-primary',
              title: 'text-sm font-semibold leading-5 text-highlighted',
            }"
          />
        </div>
      </section>

        <!-- Центр: лента -->
        <section class="flex flex-col gap-4 min-w-0 w-full" aria-label="Лента новостей">
          <UTabs
            v-model="newsTab"
            :items="newsTabItems"
            variant="link"
            color="primary"
            size="md"
            :content="false"
            class="w-full border-b border-default"
            :ui="{
              list: 'h-12 gap-1.5 overflow-x-auto',
              trigger: 'h-12 shrink-0 justify-start focus-visible:outline-none focus-visible:ring-0',
              label: 'truncate max-w-[min(100%,28rem)]',
            }"
          />

          <div class="flex flex-col gap-4">
            <template v-if="newsItems.length">
              <HomeNewsCard
                v-for="item in newsItems"
                :key="item.id"
                :id="item.id"
                :title="item.title"
                :description="item.description"
                :image-src="item.imageSrc"
                :image-alt="item.title"
                :to="item.to"
                :date="item.date"
                :created-at="item.createdAt"
                :likes="item.likes"
                :views="item.views"
                :liked="isNewsLiked(item.id)"
                :author-role="item.category"
                @toggle-like="toggleNewsLike(item.id)"
              />

              <UButton
                v-if="hasMoreNews"
                type="button"
                color="neutral"
                variant="outline"
                size="lg"
                class="w-full justify-center"
                icon="i-lucide-chevron-down"
                @click="showMoreNews"
              >
                Показать ещё
              </UButton>
            </template>

            <UEmpty
              v-else-if="newsTab === 'ofo'"
              variant="naked"
              icon="i-lucide-building-2"
              title="Нет новостей ОФО"
              description="У новостей пока нет привязки к ОФО. Вкладка покажет материалы, если категория совпадёт с названием вашего подразделения."
              class="py-10"
            />
            <UEmpty
              v-else
              variant="naked"
              icon="i-lucide-newspaper"
              title="Новостей пока нет"
              description="Как только появятся публикации, они отобразятся здесь."
              class="py-10"
            />
          </div>
        </section>
        </div>

        <!-- Правая колонка -->
        <aside class="w-full xl:w-[420px] shrink-0 flex flex-col gap-4">
          <LearningHomeWidget v-if="!isAdmin" />

          <UCard
            variant="soft"
            class="w-full rounded-[10px]"
            :ui="{
              root: 'rounded-[10px] bg-elevated/50 ring-0 border-0 divide-y-0',
              header: 'px-4 py-4 sm:px-4',
              body: 'flex flex-col gap-2 px-4 pb-4 pt-0 sm:px-4 sm:pb-4 sm:pt-0',
            }"
          >
            <template #header>
              <div class="flex items-center justify-between gap-1">
                <h2 class="text-lg font-bold leading-7 text-highlighted truncate">Мероприятия</h2>
                <UButton
                  to="/events"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-arrow-up-right"
                  square
                  aria-label="Все мероприятия"
                />
              </div>
            </template>
            <div v-if="eventsLoading" class="flex flex-col gap-2">
              <USkeleton v-for="n in 2" :key="n" class="h-12 w-full rounded-lg" />
            </div>
            <p v-else-if="eventsError" class="text-sm text-error">{{ eventsError }}</p>
            <p v-else-if="!upcomingEvents.length" class="text-sm text-muted">
              Ближайших мероприятий нет
            </p>
            <button
              v-for="evt in upcomingEvents"
              :key="evt.id"
              type="button"
              class="flex flex-col gap-0.5 rounded-lg p-2 text-left hover:bg-elevated transition-colors"
              @click="openEventDetails(evt.id)"
            >
              <span class="text-sm font-medium text-highlighted line-clamp-2">{{ evt.title }}</span>
              <span class="text-xs text-dimmed">{{ evt.date }}</span>
            </button>
          </UCard>

          <UCard
            variant="soft"
            class="w-full rounded-[10px]"
            :ui="{
              root: 'rounded-[10px] bg-elevated/50 ring-0 border-0 divide-y-0',
              header: 'px-4 py-4 sm:px-4',
              body: 'flex flex-col gap-3 px-4 pb-4 pt-0 sm:px-4 sm:pb-4 sm:pt-0',
            }"
          >
            <template #header>
              <div class="flex items-center justify-between gap-1">
                <h2 class="text-lg font-bold leading-7 text-highlighted truncate">Дни рождения коллег</h2>
                <UButton
                  to="/birthdays"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-arrow-up-right"
                  square
                  aria-label="Календарь дней рождения"
                />
              </div>
            </template>
            <UButton
              v-if="canEditBirthdays"
              label="Загрузить даты xlsx"
              icon="i-lucide-upload"
              color="neutral"
              variant="outline"
              size="sm"
              block
              @click="openBirthdayUpload"
            />
            <div v-if="birthdaysLoading" class="flex flex-col gap-2">
              <USkeleton v-for="n in 3" :key="n" class="h-12 w-full rounded-lg" />
            </div>
            <p v-else-if="birthdaysError" class="text-sm text-error">{{ birthdaysError }}</p>
            <p v-else-if="!visibleBirthdayGroups.length" class="text-sm text-muted">
              В ближайшие дни именинников нет
            </p>
            <div
              v-for="group in visibleBirthdayGroups"
              :key="group.id"
              class="flex flex-col gap-2"
            >
              <p class="text-xs font-medium leading-4 text-muted">
                {{ group.dayLabel }}
              </p>
              <div
                v-for="person in group.people"
                :key="person.id"
                class="flex items-center gap-2"
              >
                <UUser
                  :name="person.name"
                  :description="person.role || undefined"
                  :avatar="{ src: person.avatar, alt: person.name }"
                  size="md"
                  class="min-w-0 flex-1"
                />
                <UTooltip text="Поздравить">
                  <UButton
                    type="button"
                    color="neutral"
                    variant="outline"
                    size="sm"
                    icon="i-lucide-gift"
                    square
                    aria-label="Поздравить"
                    @click="congratulate(person.name)"
                  />
                </UTooltip>
              </div>
            </div>
          </UCard>

          <UCard
            variant="soft"
            class="w-full rounded-[10px]"
            :ui="{
              root: 'rounded-[10px] bg-elevated/50 ring-0 border-0 divide-y-0',
              header: 'px-4 py-4 sm:px-4',
              body: 'flex flex-col gap-2 px-4 pb-4 pt-0 sm:px-4 sm:pb-4 sm:pt-0',
            }"
          >
            <template #header>
              <div class="flex items-center justify-between gap-1">
                <h2 class="text-lg font-bold leading-7 text-highlighted truncate">Новые сотрудники</h2>
                <UButton
                  to="/newcomers"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-arrow-up-right"
                  square
                  aria-label="Открыть «Новичкам»"
                />
              </div>
            </template>
            <p class="text-sm text-muted">
              Раздел для новых сотрудников в разработке.
            </p>
          </UCard>
        </aside>
      </div>
    </div>

    <USlideover
      v-model:open="birthdayUploadOpen"
      side="right"
      title="Загрузка дат рождений (xlsx)"
      description=""
    >
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
