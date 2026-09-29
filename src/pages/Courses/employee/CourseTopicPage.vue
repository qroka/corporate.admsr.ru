<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import {
  useBreadcrumbCurrentLabel,
  useBreadcrumbLabelsByRoute,
} from '../../../composables/usePortalNavigation';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';
import { followCourseNextAction, isActionableStep } from '../followNextAction';
import { formatSeconds } from '../courseDuration';
import { materialKindIcon, materialKindLabel, materialViewKind } from '../courseMaterialView';

/**
 * Оглавление темы: список материалов (каждый открывается на своей странице —
 * `CourseMaterialPage`), прогресс и следующий шаг. Время здесь не считается —
 * heartbeat живёт на странице материала.
 */
const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const breadcrumbLabel = useBreadcrumbCurrentLabel();
const breadcrumbByRoute = useBreadcrumbLabelsByRoute();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const topicId = computed(() => Number(route.params.topicId));

const loading = ref(true);
const loadError = ref<string | null>(null);
/**
 * Два независимых источника. `context` — курс и список тем (из записи на курс),
 * `topicData` — ответ темы.
 */
const context = ref<{ courseTitle: string; topics: any[] } | null>(null);
const topicData = ref<any>(null);

const topic = computed(() => topicData.value?.topic || null);
const materials = computed<any[]>(() => topic.value?.materials || []);
const isReview = computed(() => Boolean(topicData.value?.reviewMode));
const nextAction = computed(() => topicData.value?.nextAction || null);
const topicsList = computed<any[]>(() => context.value?.topics || []);
const courseTitle = computed(() => context.value?.courseTitle || 'Обучение');

const topicIndex = computed(() => topicsList.value.findIndex((t) => Number(t.id) === topicId.value));
const positionLabel = computed(() =>
  topicIndex.value >= 0 ? `Тема ${topicIndex.value + 1} из ${topicsList.value.length}` : 'Обучение',
);

watch(
  [courseTitle, () => topic.value?.title],
  ([course, title]) => {
    breadcrumbByRoute.value = { ...breadcrumbByRoute.value, 'course-enrollment': course };
    breadcrumbLabel.value = title || 'Тема';
  },
  { immediate: true },
);

function matStatus(m: any): string {
  return m.progress?.status || 'not_started';
}

function isDone(m: any) {
  return matStatus(m) === 'completed';
}

function materialRoute(m: any) {
  return {
    name: 'course-material',
    params: { enrollmentId: enrollmentId.value, topicId: topicId.value, materialId: m.id },
  };
}

/** Строка про время — только у материалов с требованием по времени. */
function timeLine(m: any) {
  const min = Math.max(0, Number(m.minimumActiveSeconds || 0));
  if (!min || isDone(m) || isReview.value) return '';
  return `Не меньше ${formatSeconds(min)}`;
}

const requiredMaterials = computed(() => materials.value.filter((m) => m.isRequired !== false));
const requiredDone = computed(() => requiredMaterials.value.filter(isDone).length);
const materialsLabel = computed(() => {
  if (!materials.value.length) return '';
  if (!requiredMaterials.value.length) {
    return `Изучено ${materials.value.filter(isDone).length} из ${materials.value.length}`;
  }
  return `Изучено ${requiredDone.value} из ${requiredMaterials.value.length} обязательных`;
});

/** Сервер всё ещё держит сотрудника на этой теме — значит, «дальше» пока нельзя. */
const stuckHere = computed(() => {
  const a = nextAction.value;
  return !isReview.value && Number(a?.topicId || 0) === topicId.value && a?.type !== 'complete_topic';
});

/** Не хватает только времени темы: материалы и тест уже закрыты. */
const topicTimeLeft = computed(() => {
  if (!stuckHere.value || nextAction.value?.type !== 'topic') return 0;
  const min = Number(topic.value?.minimumActiveSeconds || 0);
  const got = Number(topic.value?.progress?.activeSeconds || 0);
  return Math.max(0, min - got);
});

const forward = computed<null | { label: string; icon: string; run: () => void }>(() => {
  if (isReview.value) {
    const next = topicsList.value[topicIndex.value + 1];
    if (!next) return null;
    return {
      label: `К теме «${next.title}»`,
      icon: 'i-lucide-arrow-right',
      run: () => router.push({ name: 'course-topic', params: { enrollmentId: enrollmentId.value, topicId: next.id } }),
    };
  }
  const a = nextAction.value;
  if (!isActionableStep(a)) return null;
  // Следующий шаг — материал этой темы: ведём прямо на него.
  if (stuckHere.value && a.type === 'material') {
    const m = materials.value.find((x) => Number(x.id) === Number(a.materialId));
    if (!m) return null;
    const started = materials.value.some(isDone);
    return {
      label: started ? `Продолжить: «${m.title}»` : 'Начать изучение',
      icon: 'i-lucide-arrow-right',
      run: () => void router.push(materialRoute(m)),
    };
  }
  if (stuckHere.value && a.type !== 'topic_test') return null;
  // complete_topic на этой же теме — сервер вот-вот её закроет; ссылка «на себя» бессмысленна.
  if (a.type === 'complete_topic' && Number(a.topicId) === topicId.value) return null;
  let label = a.label || 'Продолжить';
  if (a.type === 'topic_test' && stuckHere.value) label = 'Пройти тест темы';
  else if (a.type === 'final_test') label = 'К итоговому тесту';
  else if (a.type === 'complete_course') label = 'К итогам';
  else if (a.topicId) {
    const t = topicsList.value.find((x) => Number(x.id) === Number(a.topicId));
    if (t && a.type !== 'topic_test') label = `К теме «${t.title}»`;
  }
  return {
    label,
    icon: a.type === 'topic_test' || a.type === 'final_test' ? 'i-lucide-clipboard-check' : 'i-lucide-arrow-right',
    run: () => void followCourseNextAction(router, enrollmentId.value, a),
  };
});

const footerNote = computed(() => {
  if (isReview.value) return 'Повторный просмотр — прогресс уже засчитан.';
  const a = nextAction.value;
  if (stuckHere.value && a?.type === 'topic_test') return 'Материалы изучены — осталось пройти тест темы.';
  if (topicTimeLeft.value > 0) {
    return `В теме нужно провести ещё ${formatSeconds(topicTimeLeft.value)}. Откройте любой материал — время засчитывается, пока он открыт.`;
  }
  if (stuckHere.value) return '';
  if (a?.type === 'complete_course') return 'Все темы пройдены.';
  return forward.value ? 'Тема пройдена.' : '';
});

const showFooter = computed(() => !loading.value && !loadError.value && Boolean(forward.value || footerNote.value));

async function loadAll() {
  loading.value = true;
  loadError.value = null;
  try {
    const [t, enrollment] = await Promise.all([
      store.getTopic(enrollmentId.value, topicId.value),
      store.getEnrollment(enrollmentId.value).catch(() => null) as Promise<any>,
    ]);
    topicData.value = t;
    context.value = {
      courseTitle: enrollment?.enrollment?.course?.title || 'Обучение',
      topics: enrollment?.version?.topics || [],
    };
  } catch (e: any) {
    topicData.value = null;
    loadError.value = e?.message || 'Тема недоступна';
  } finally {
    loading.value = false;
  }
}

onMounted(loadAll);

watch(topicId, (id, prev) => {
  if (id && id !== prev) void loadAll();
});

onUnmounted(() => {
  breadcrumbLabel.value = null;
  const next = { ...breadcrumbByRoute.value };
  delete next['course-enrollment'];
  breadcrumbByRoute.value = next;
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
      <div v-if="loading" class="flex flex-col gap-4" aria-busy="true" aria-label="Загрузка темы">
        <USkeleton class="h-16 w-2/3 rounded-lg" />
        <USkeleton v-for="n in 3" :key="n" class="h-28 w-full rounded-panel" />
      </div>

      <UAlert
        v-else-if="loadError || !topic"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Тема недоступна"
        :description="loadError || 'Не удалось открыть тему.'"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="loadAll">Повторить</UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'course-enrollment', params: { enrollmentId } }">
            К курсу
          </UButton>
        </template>
      </UAlert>

      <template v-else>
        <UPageHeader :headline="positionLabel" :title="topic.title" :description="topic.description || undefined" />

        <UAlert
          v-if="isReview"
          color="neutral"
          variant="subtle"
          icon="i-lucide-book-open"
          title="Повторный просмотр"
          description="Курс уже пройден. Материалы можно открыть снова — на результат это не влияет."
        />

        <section class="flex flex-col gap-3 min-w-0" aria-labelledby="topic-materials-title">
          <div class="flex items-end justify-between gap-2 flex-wrap">
            <h2 id="topic-materials-title" class="text-lg font-bold leading-7 text-highlighted">Материалы</h2>
            <p v-if="materialsLabel && !isReview" class="text-sm text-muted tabular-nums">{{ materialsLabel }}</p>
          </div>
          <UProgress
            v-if="requiredMaterials.length && !isReview"
            :model-value="Math.round((requiredDone / requiredMaterials.length) * 100)"
            size="sm"
            color="primary"
            :aria-label="materialsLabel"
          />

          <UEmpty
            v-if="!materials.length"
            variant="naked"
            icon="i-lucide-file"
            title="В теме нет материалов"
            description="Автор курса не добавил материалов — переходите к следующему шагу."
            class="py-8"
          />

          <ul v-else class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
            <li v-for="(m, i) in materials" :key="m.id" class="min-w-0">
              <RouterLink
                :to="materialRoute(m)"
                class="group rounded-panel bg-elevated p-4 flex items-center gap-3 min-w-0 transition-colors hover:bg-accented/60 focus-visible:outline-2 focus-visible:outline-primary"
              >
                <span
                  class="size-9 shrink-0 rounded-full flex items-center justify-center"
                  :class="isDone(m) ? 'bg-success/15 text-success' : 'bg-default text-muted'"
                  aria-hidden="true"
                >
                  <UIcon :name="isDone(m) ? 'i-lucide-check' : materialKindIcon(materialViewKind(m))" class="size-5" />
                </span>
                <div class="flex-1 min-w-0 flex flex-col gap-1">
                  <p class="font-medium text-highlighted break-words">
                    <span class="text-muted tabular-nums">{{ i + 1 }}.</span> {{ m.title }}
                  </p>
                  <div class="flex items-center gap-x-2 gap-y-1 flex-wrap text-xs text-muted">
                    <span>{{ materialKindLabel(materialViewKind(m), m) }}</span>
                    <CourseStatusBadge v-if="!isReview" :status="matStatus(m)" />
                    <span v-if="m.isRequired === false">Необязательный</span>
                    <span v-if="timeLine(m)">{{ timeLine(m) }}</span>
                  </div>
                </div>
                <UIcon
                  name="i-lucide-chevron-right"
                  class="size-5 shrink-0 text-dimmed group-hover:text-default"
                  aria-hidden="true"
                />
              </RouterLink>
            </li>
          </ul>
        </section>

        <div
          v-if="showFooter"
          class="sticky bottom-0 z-10 pt-3 pb-1 bg-default/95 backdrop-blur border-t border-default flex flex-col gap-2"
        >
          <p v-if="footerNote" class="text-sm text-muted">{{ footerNote }}</p>
          <div class="flex flex-wrap gap-2">
            <UButton v-if="forward" color="primary" size="lg" :trailing-icon="forward.icon" @click="forward.run()">
              {{ forward.label }}
            </UButton>
            <UButton
              color="neutral"
              variant="ghost"
              size="lg"
              :to="{ name: 'course-enrollment', params: { enrollmentId } }"
            >
              К курсу
            </UButton>
          </div>
        </div>
      </template>
    </div>
  </UMain>
</template>
