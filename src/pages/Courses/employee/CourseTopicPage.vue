<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import {
  useBreadcrumbCurrentLabel,
  useBreadcrumbLabelsByRoute,
} from '../../../composables/usePortalNavigation';
import { newsEditorHtmlClass } from '../../../composables/newsEditorHtmlClass';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';
import { followCourseNextAction, isActionableStep } from '../followNextAction';
import { formatSeconds } from '../courseDuration';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const breadcrumbLabel = useBreadcrumbCurrentLabel();
const breadcrumbByRoute = useBreadcrumbLabelsByRoute();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const topicId = computed(() => Number(route.params.topicId));

const loading = ref(true);
const loadError = ref<string | null>(null);
/**
 * Два независимых источника. `context` — курс и список тем (из записи на курс),
 * `topicData` — ответ темы. Раньше они жили в одном объекте, и обновление темы
 * после отметки материала стирало список тем: пропадала кнопка «Дальше».
 */
const context = ref<{ courseTitle: string; topics: any[] } | null>(null);
const topicData = ref<any>(null);
const completingId = ref<number | null>(null);

const activeMaterialId = ref<number | null>(null);
const lastActivityAt = ref(Date.now());
/** Засчитанные секунды из ответов heartbeat — показываем без перезагрузки темы. */
const liveSeconds = ref<Record<number, number>>({});
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;

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

function minSeconds(m: any) {
  return Math.max(0, Number(m.minimumActiveSeconds || 0));
}

function activeSeconds(m: any) {
  return Math.max(Number(liveSeconds.value[m.id] || 0), Number(m.progress?.activeSeconds || 0));
}

function timeMet(m: any) {
  return minSeconds(m) === 0 || activeSeconds(m) >= minSeconds(m);
}

function isExternal(m: any) {
  return m.type !== 'rich_text';
}

/** Строка про время — только у материалов с требованием по времени. */
function timeLine(m: any) {
  const min = minSeconds(m);
  if (!min || isDone(m) || isReview.value) return '';
  const got = activeSeconds(m);
  if (got >= min) return 'Время набрано — можно отметить изученным';
  return `Нужно ${formatSeconds(min)} · засчитано ${formatSeconds(got) || '0 с'}`;
}

function timePercent(m: any) {
  const min = minSeconds(m);
  return min ? Math.min(100, Math.round((activeSeconds(m) / min) * 100)) : 100;
}

function openLabel(m: any) {
  if (m.type === 'rich_text') {
    if (activeMaterialId.value === m.id) return 'Свернуть';
    return isReview.value || isDone(m) ? 'Читать снова' : 'Читать';
  }
  return m.type === 'link' ? 'Открыть ссылку' : 'Открыть файл';
}

function openIcon(m: any) {
  if (m.type === 'rich_text') return activeMaterialId.value === m.id ? 'i-lucide-chevron-up' : 'i-lucide-book-open';
  return m.type === 'link' ? 'i-lucide-external-link' : 'i-lucide-file-down';
}

function typeIcon(m: any) {
  if (m.type === 'rich_text') return 'i-lucide-file-text';
  if (m.type === 'link') return 'i-lucide-link';
  return 'i-lucide-paperclip';
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
  activeMaterialId.value = null;
  liveSeconds.value = {};
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

/** Обновляем только тему — контекст курса не трогаем. */
async function refreshTopic() {
  try {
    topicData.value = await store.getTopic(enrollmentId.value, topicId.value);
  } catch {
    /* оставляем текущее состояние: действие уже прошло на сервере */
  }
}

function onActivity() {
  lastActivityAt.value = Date.now();
}

/**
 * Файл и ссылка открываются в другой вкладке, поэтому для них фокус портала
 * не требуем — иначе время не засчитывалось вовсе. Текст читается здесь же:
 * для него прежнее правило «вкладка активна и было действие за 30 с».
 * От накрутки защищает сервер: паузы длиннее 90 с он не засчитывает.
 */
function shouldBeat(m: any) {
  if (isExternal(m)) return true;
  const focused = document.visibilityState === 'visible' && document.hasFocus();
  return focused && Date.now() - lastActivityAt.value <= 30_000;
}

async function tickHeartbeat() {
  const id = activeMaterialId.value;
  if (!id || isReview.value) return;
  const m = materials.value.find((x) => x.id === id);
  if (!m || isDone(m) || !shouldBeat(m)) return;
  try {
    const res = (await store.heartbeat({ enrollmentId: enrollmentId.value, materialId: id, seconds: 15 })) as any;
    if (res?.activeSeconds != null) {
      liveSeconds.value = { ...liveSeconds.value, [id]: Number(res.activeSeconds) };
    }
    if (res?.topicCompleted) await refreshTopic();
  } catch {
    /* сеть мигнула — следующий тик досчитает */
  }
}

async function openMaterial(m: any) {
  if (m.type === 'rich_text' && activeMaterialId.value === m.id) {
    activeMaterialId.value = null;
    return;
  }
  // Окно открываем сразу, до запроса: после await браузер может счесть его всплывающим.
  const url = m.type === 'link' ? m.externalUrl : m.fileUrl;
  if (isExternal(m) && url) window.open(url, '_blank', 'noopener');
  activeMaterialId.value = m.id;
  lastActivityAt.value = Date.now();
  try {
    await store.openMaterial(enrollmentId.value, m.id);
  } catch (e: any) {
    toast.add({ title: 'Не удалось открыть материал', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  }
}

async function completeMaterial(m: any) {
  completingId.value = m.id;
  try {
    await store.completeMaterial(enrollmentId.value, m.id);
    toast.add({ title: 'Материал изучен', color: 'success', icon: 'i-lucide-check' });
    await refreshTopic();
  } catch (e: any) {
    toast.add({ title: 'Не удалось отметить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    completingId.value = null;
  }
}

onMounted(async () => {
  await loadAll();
  window.addEventListener('mousemove', onActivity);
  window.addEventListener('keydown', onActivity);
  window.addEventListener('scroll', onActivity, true);
  window.addEventListener('click', onActivity);
  heartbeatTimer = setInterval(() => void tickHeartbeat(), 15_000);
});

watch(topicId, (id, prev) => {
  if (id && id !== prev) void loadAll();
});

onUnmounted(() => {
  if (heartbeatTimer) clearInterval(heartbeatTimer);
  window.removeEventListener('mousemove', onActivity);
  window.removeEventListener('keydown', onActivity);
  window.removeEventListener('scroll', onActivity, true);
  window.removeEventListener('click', onActivity);
  breadcrumbLabel.value = null;
  const next = { ...breadcrumbByRoute.value };
  delete next['course-enrollment'];
  breadcrumbByRoute.value = next;
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
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
            <li
              v-for="m in materials"
              :key="m.id"
              class="rounded-panel bg-elevated p-4 flex flex-col gap-3 min-w-0"
              :class="activeMaterialId === m.id ? 'ring-2 ring-inset ring-primary/40' : ''"
            >
              <div class="flex items-start gap-3 min-w-0">
                <UIcon :name="typeIcon(m)" class="size-5 mt-0.5 shrink-0 text-muted" aria-hidden="true" />
                <div class="flex-1 min-w-0 flex flex-col gap-1">
                  <p class="font-medium text-highlighted break-words">{{ m.title }}</p>
                  <p v-if="m.description" class="text-sm text-muted break-words">{{ m.description }}</p>
                  <div class="flex items-center gap-x-2 gap-y-1 flex-wrap text-xs text-muted">
                    <CourseStatusBadge :status="matStatus(m)" />
                    <span v-if="m.isRequired === false">Необязательный</span>
                    <span v-if="timeLine(m)" :class="timeMet(m) ? 'text-success' : ''">{{ timeLine(m) }}</span>
                  </div>
                </div>
              </div>

              <UProgress
                v-if="timeLine(m) && !timeMet(m)"
                :model-value="timePercent(m)"
                size="xs"
                color="neutral"
                :aria-label="timeLine(m)"
              />

              <div
                v-if="m.type === 'rich_text' && m.contentHtml && activeMaterialId === m.id"
                :class="['rounded-lg bg-default p-3 sm:p-4 text-default min-w-0 overflow-x-auto', newsEditorHtmlClass]"
                v-html="m.contentHtml"
              />

              <p
                v-if="isExternal(m) && activeMaterialId === m.id && !isDone(m) && minSeconds(m) && !isReview"
                class="text-xs text-muted"
              >
                Время засчитывается, пока материал открыт и эта вкладка не закрыта.
              </p>

              <div class="flex flex-wrap gap-2">
                <UButton color="neutral" variant="soft" size="sm" :icon="openIcon(m)" @click="openMaterial(m)">
                  {{ openLabel(m) }}
                </UButton>
                <UButton
                  v-if="!isReview && !isDone(m)"
                  color="primary"
                  size="sm"
                  icon="i-lucide-check"
                  :loading="completingId === m.id"
                  :disabled="!timeMet(m)"
                  :title="timeMet(m) ? undefined : 'Сначала наберите нужное время изучения'"
                  @click="completeMaterial(m)"
                >
                  Отметить изученным
                </UButton>
              </div>
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
