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
import { formatFileSize, materialKindIcon, materialKindLabel, materialViewKind } from '../courseMaterialView';

/**
 * Один материал — одна страница. PDF, видео и изображения показываются прямо
 * здесь (файл отдаёт `course_file.php` по cookie сессии), текст — как HTML,
 * ссылка открывается в новой вкладке, прочие файлы (docx, xlsx…) скачиваются.
 * Время изучения считается heartbeat'ом, пока открыта эта страница.
 */
const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const breadcrumbLabel = useBreadcrumbCurrentLabel();
const breadcrumbByRoute = useBreadcrumbLabelsByRoute();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const topicId = computed(() => Number(route.params.topicId));
const materialId = computed(() => Number(route.params.materialId));

const loading = ref(true);
const loadError = ref<string | null>(null);
const context = ref<{ courseTitle: string; topics: any[] } | null>(null);
const topicData = ref<any>(null);
const completing = ref(false);

const lastActivityAt = ref(Date.now());
const liveSeconds = ref(0);
const liveTopicSeconds = ref(0);
watch(topicData, () => {
  liveTopicSeconds.value = 0;
});
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;

const topic = computed(() => topicData.value?.topic || null);
const materials = computed<any[]>(() => topic.value?.materials || []);
const index = computed(() => materials.value.findIndex((m) => Number(m.id) === materialId.value));
const material = computed(() => (index.value >= 0 ? materials.value[index.value] : null));
const prevMaterial = computed(() => (index.value > 0 ? materials.value[index.value - 1] : null));
const nextMaterial = computed(() =>
  index.value >= 0 && index.value < materials.value.length - 1 ? materials.value[index.value + 1] : null,
);
const isReview = computed(() => Boolean(topicData.value?.reviewMode));
const nextAction = computed(() => topicData.value?.nextAction || null);
const topicsList = computed<any[]>(() => context.value?.topics || []);
const courseTitle = computed(() => context.value?.courseTitle || 'Обучение');
const topicIndex = computed(() => topicsList.value.findIndex((t) => Number(t.id) === topicId.value));

const kind = computed(() => (material.value ? materialViewKind(material.value) : 'file'));
const fileUrl = computed(() => String(material.value?.fileUrl || ''));
const fileName = computed(() => String(material.value?.originalFilename || material.value?.title || 'файл'));
const fileSize = computed(() => formatFileSize(material.value?.fileSize));
/**
 * Браузер без встроенного просмотра PDF (Android, настройка «скачивать PDF»)
 * вместо показа во фрейме скачал бы файл — тогда сразу даём кнопки.
 */
const canInlinePdf = typeof navigator === 'undefined' || (navigator as any).pdfViewerEnabled !== false;

const headline = computed(() => {
  const parts: string[] = [];
  if (topicIndex.value >= 0) parts.push(`Тема ${topicIndex.value + 1}`);
  if (index.value >= 0) parts.push(`Материал ${index.value + 1} из ${materials.value.length}`);
  return parts.join(' · ') || 'Материал';
});

watch(
  [courseTitle, () => topic.value?.title, () => material.value?.title],
  ([course, topicTitle, title]) => {
    breadcrumbByRoute.value = {
      ...breadcrumbByRoute.value,
      'course-enrollment': course,
      'course-topic': topicTitle || 'Тема',
    };
    breadcrumbLabel.value = title || 'Материал';
  },
  { immediate: true },
);

const status = computed(() => material.value?.progress?.status || 'not_started');
const isDone = computed(() => status.value === 'completed');
const minSeconds = computed(() => Math.max(0, Number(material.value?.minimumActiveSeconds || 0)));
const activeSeconds = computed(() =>
  Math.max(liveSeconds.value, Number(material.value?.progress?.activeSeconds || 0)),
);
const timeMet = computed(() => minSeconds.value === 0 || activeSeconds.value >= minSeconds.value);
const timeLine = computed(() => {
  if (!minSeconds.value || isDone.value || isReview.value) return '';
  if (timeMet.value) return 'Время набрано — можно отметить изученным';
  return `Нужно ${formatSeconds(minSeconds.value)} · засчитано ${formatSeconds(activeSeconds.value) || '0 с'}`;
});
const timePercent = computed(() =>
  minSeconds.value ? Math.min(100, Math.round((activeSeconds.value / minSeconds.value) * 100)) : 100,
);

/** Сервер держит сотрудника на этой теме только из-за времени темы. */
const topicTimeLeft = computed(() => {
  const a = nextAction.value;
  if (isReview.value || a?.type !== 'topic' || Number(a?.topicId) !== topicId.value) return 0;
  const min = Number(topic.value?.minimumActiveSeconds || 0);
  const got = Number(topic.value?.progress?.activeSeconds || 0) + liveTopicSeconds.value;
  return Math.max(0, min - got);
});

function materialRoute(m: any) {
  return {
    name: 'course-material',
    params: { enrollmentId: enrollmentId.value, topicId: topicId.value, materialId: m.id },
  };
}

/**
 * Куда ведёт «Дальше». Сначала — следующий материал темы; после последнего —
 * то, что говорит сервер: пропущенный обязательный материал, тест темы,
 * следующая тема, итоговый тест или итоги.
 */
const forward = computed<null | { label: string; icon: string; run: () => void }>(() => {
  if (nextMaterial.value) {
    const m = nextMaterial.value;
    return { label: 'Следующий материал', icon: 'i-lucide-arrow-right', run: () => void router.push(materialRoute(m)) };
  }
  if (isReview.value) {
    const t = topicsList.value[topicIndex.value + 1];
    if (!t) return null;
    return {
      label: `К теме «${t.title}»`,
      icon: 'i-lucide-arrow-right',
      run: () => void router.push({ name: 'course-topic', params: { enrollmentId: enrollmentId.value, topicId: t.id } }),
    };
  }
  const a = nextAction.value;
  if (!isActionableStep(a)) return null;
  const here = Number(a.topicId || 0) === topicId.value;
  if (a.type === 'material' && here) {
    if (Number(a.materialId) === materialId.value) return null;
    const m = materials.value.find((x) => Number(x.id) === Number(a.materialId));
    if (!m) return null;
    return { label: `Не изучен: «${m.title}»`, icon: 'i-lucide-arrow-left', run: () => void router.push(materialRoute(m)) };
  }
  if (here && (a.type === 'topic' || a.type === 'complete_topic')) return null;
  let label = 'Продолжить';
  if (a.type === 'topic_test') label = here ? 'Пройти тест темы' : 'К тесту';
  else if (a.type === 'final_test') label = 'К итоговому тесту';
  else if (a.type === 'complete_course') label = 'К итогам';
  else if (a.topicId) {
    const t = topicsList.value.find((x) => Number(x.id) === Number(a.topicId));
    if (t) label = `К теме «${t.title}»`;
  }
  return {
    label,
    icon: a.type === 'topic_test' || a.type === 'final_test' ? 'i-lucide-clipboard-check' : 'i-lucide-arrow-right',
    run: () => void followCourseNextAction(router, enrollmentId.value, a),
  };
});

const footerNote = computed(() => {
  if (isReview.value) return 'Повторный просмотр — прогресс уже засчитан.';
  if (!isDone.value && !timeMet.value) {
    return `Отметить изученным можно через ${formatSeconds(minSeconds.value - activeSeconds.value)}.`;
  }
  if (isDone.value && topicTimeLeft.value > 0) {
    return `В теме нужно провести ещё ${formatSeconds(topicTimeLeft.value)} — время идёт, пока материал открыт.`;
  }
  return '';
});

async function loadAll() {
  loading.value = true;
  loadError.value = null;
  liveSeconds.value = 0;
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
    if (!material.value) {
      loadError.value = 'Материал не найден в этой теме — возможно, автор курса его удалил.';
      return;
    }
    lastActivityAt.value = Date.now();
    const opened = material.value;
    void store
      .openMaterial(enrollmentId.value, materialId.value)
      .then(() => {
        // Сервер перевёл материал в «открыт» — показываем это без перезагрузки темы.
        if (opened.progress?.status !== 'completed') opened.progress = { ...opened.progress, status: 'in_progress' };
      })
      .catch((e: any) => {
        toast.add({ title: 'Не удалось открыть материал', description: e?.message, color: 'error', icon: 'i-lucide-x' });
      });
  } catch (e: any) {
    topicData.value = null;
    loadError.value = e?.message || 'Материал недоступен';
  } finally {
    loading.value = false;
  }
}

async function refreshTopic() {
  try {
    topicData.value = await store.getTopic(enrollmentId.value, topicId.value);
  } catch {
    /* действие уже прошло на сервере — оставляем текущее состояние */
  }
}

function onActivity() {
  lastActivityAt.value = Date.now();
}

/**
 * Текст читается на странице: время идёт, если вкладка активна и было действие
 * за 30 с. PDF и видео живут во встроенном просмотрщике, куда события мыши не
 * доходят, ссылка — в другой вкладке: для них фокус не требуем (ADR-024).
 * От накрутки защищает сервер: паузы длиннее 90 с он не засчитывает.
 */
function shouldBeat() {
  if (kind.value !== 'text') return true;
  const focused = document.visibilityState === 'visible' && document.hasFocus();
  return focused && Date.now() - lastActivityAt.value <= 30_000;
}

async function tickHeartbeat() {
  const m = material.value;
  if (!m || isReview.value || loading.value) return;
  // Изученный материал тоже считает время, пока не набран минимум темы.
  if ((isDone.value && topicTimeLeft.value <= 0) || !shouldBeat()) return;
  try {
    const res = (await store.heartbeat({ enrollmentId: enrollmentId.value, materialId: m.id })) as any;
    if (res?.activeSeconds != null) liveSeconds.value = Number(res.activeSeconds);
    liveTopicSeconds.value += Number(res?.addedSeconds || 0);
    if (res?.topicCompleted) {
      toast.add({ title: 'Тема пройдена', color: 'success', icon: 'i-lucide-check' });
      await refreshTopic();
    }
  } catch {
    /* сеть мигнула — следующий тик досчитает */
  }
}

/** «Изучено — дальше»: отмечаем и сразу ведём к следующему шагу. */
async function completeAndContinue() {
  const m = material.value;
  if (!m) return;
  completing.value = true;
  try {
    await store.completeMaterial(enrollmentId.value, m.id);
    await refreshTopic();
    toast.add({ title: 'Материал изучен', color: 'success', icon: 'i-lucide-check' });
    forward.value?.run();
  } catch (e: any) {
    toast.add({ title: 'Не удалось отметить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    completing.value = false;
  }
}

function openLink() {
  const url = material.value?.externalUrl;
  if (url) window.open(url, '_blank', 'noopener');
}

onMounted(async () => {
  await loadAll();
  window.addEventListener('mousemove', onActivity);
  window.addEventListener('keydown', onActivity);
  window.addEventListener('scroll', onActivity, true);
  window.addEventListener('click', onActivity);
  heartbeatTimer = setInterval(() => void tickHeartbeat(), 15_000);
});

watch([topicId, materialId], ([t, m], [pt, pm]) => {
  if (t && m && (t !== pt || m !== pm)) void loadAll();
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
  delete next['course-topic'];
  breadcrumbByRoute.value = next;
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-5xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
      <div v-if="loading" class="flex flex-col gap-4" aria-busy="true" aria-label="Загрузка материала">
        <USkeleton class="h-16 w-2/3 rounded-lg" />
        <USkeleton class="h-[60dvh] w-full rounded-panel" />
      </div>

      <UAlert
        v-else-if="loadError || !material"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Материал недоступен"
        :description="loadError || 'Не удалось открыть материал.'"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="loadAll">Повторить</UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'course-topic', params: { enrollmentId, topicId } }">
            К теме
          </UButton>
        </template>
      </UAlert>

      <template v-else>
        <UPageHeader :headline="headline" :title="material.title" :description="material.description || undefined" />

        <div class="flex items-center gap-x-3 gap-y-1 flex-wrap text-sm text-muted -mt-2">
          <span class="inline-flex items-center gap-1.5">
            <UIcon :name="materialKindIcon(kind)" class="size-4" aria-hidden="true" />
            {{ materialKindLabel(kind, material) }}<template v-if="fileSize"> · {{ fileSize }}</template>
          </span>
          <CourseStatusBadge v-if="!isReview" :status="status" />
          <span v-if="material.isRequired === false">Необязательный</span>
          <span v-if="timeLine" :class="timeMet ? 'text-success' : ''">{{ timeLine }}</span>
        </div>
        <UProgress
          v-if="timeLine && !timeMet"
          :model-value="timePercent"
          size="xs"
          color="neutral"
          :aria-label="timeLine"
          class="-mt-3"
        />

        <!-- Текст -->
        <div
          v-if="kind === 'text'"
          :class="['rounded-panel bg-elevated p-4 sm:p-6 text-default min-w-0 overflow-x-auto', newsEditorHtmlClass]"
          v-html="material.contentHtml || '<p>Текст материала пуст.</p>'"
        />

        <!-- PDF: встроенный просмотрщик браузера -->
        <div v-else-if="kind === 'pdf' && canInlinePdf" class="flex flex-col gap-2">
          <iframe
            :src="fileUrl"
            :title="material.title"
            class="w-full h-[75dvh] min-h-96 rounded-panel border border-default bg-elevated"
          />
          <p class="text-xs text-muted">
            Документ не отображается? Откройте его
            <a :href="fileUrl" target="_blank" rel="noopener" class="text-primary underline underline-offset-2">в новой вкладке</a>
            или
            <a :href="fileUrl" :download="fileName" class="text-primary underline underline-offset-2">скачайте</a>.
          </p>
        </div>

        <!-- Видео -->
        <video
          v-else-if="kind === 'video'"
          :key="fileUrl"
          :src="fileUrl"
          controls
          playsinline
          preload="metadata"
          class="w-full max-h-[75dvh] rounded-panel bg-elevated"
          @play="onActivity"
          @timeupdate="onActivity"
        >
          Браузер не может воспроизвести видео —
          <a :href="fileUrl" :download="fileName">скачайте файл</a>.
        </video>

        <!-- Изображение -->
        <div v-else-if="kind === 'image'" class="rounded-panel bg-elevated p-2 flex justify-center">
          <img :src="fileUrl" :alt="material.title" class="max-w-full max-h-[75dvh] object-contain rounded-lg" />
        </div>

        <!-- Ссылка -->
        <div v-else-if="kind === 'link'" class="rounded-panel bg-elevated p-4 sm:p-6 flex flex-col gap-3 items-start">
          <p class="text-sm text-muted break-all">{{ material.externalUrl }}</p>
          <UButton color="neutral" variant="soft" icon="i-lucide-external-link" @click="openLink">Открыть ссылку</UButton>
          <p v-if="minSeconds && !isDone && !isReview" class="text-xs text-muted">
            Время засчитывается, пока эта страница открыта.
          </p>
        </div>

        <!-- Прочие файлы (и PDF там, где браузер его не покажет) -->
        <div v-else class="rounded-panel bg-elevated p-4 sm:p-6 flex items-center gap-4 flex-wrap">
          <UIcon name="i-lucide-file-down" class="size-8 text-muted shrink-0" aria-hidden="true" />
          <div class="flex-1 min-w-0">
            <p class="font-medium text-highlighted break-words">{{ fileName }}</p>
            <p class="text-sm text-muted">
              {{ kind === 'pdf' ? 'Этот браузер не показывает PDF на странице — откройте или скачайте документ.' : 'Этот формат открывается на компьютере — скачайте файл.' }}
            </p>
          </div>
          <div class="flex gap-2 flex-wrap">
            <UButton
              v-if="kind === 'pdf'"
              color="neutral"
              variant="soft"
              icon="i-lucide-external-link"
              :href="fileUrl"
              external
              target="_blank"
              rel="noopener"
            >
              Открыть
            </UButton>
            <!-- external: иначе ULink отдаёт «/api/...» во Vue Router и открывается 404 SPA -->
            <UButton color="neutral" variant="soft" icon="i-lucide-download" :href="fileUrl" external :download="fileName">
              Скачать
            </UButton>
          </div>
        </div>

        <div class="sticky bottom-0 z-10 pt-3 pb-1 bg-default/95 backdrop-blur border-t border-default flex flex-col gap-2">
          <p v-if="footerNote" class="text-sm text-muted">{{ footerNote }}</p>
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              v-if="!isReview && !isDone"
              color="primary"
              size="lg"
              :trailing-icon="forward ? 'i-lucide-arrow-right' : 'i-lucide-check'"
              :loading="completing"
              :disabled="!timeMet"
              @click="completeAndContinue"
            >
              {{ forward && nextMaterial ? 'Изучено, дальше' : 'Отметить изученным' }}
            </UButton>
            <UButton v-else-if="forward" color="primary" size="lg" :trailing-icon="forward.icon" @click="forward.run()">
              {{ forward.label }}
            </UButton>
            <UButton
              v-if="prevMaterial"
              color="neutral"
              variant="ghost"
              size="lg"
              icon="i-lucide-arrow-left"
              :to="materialRoute(prevMaterial)"
            >
              Предыдущий
            </UButton>
            <UButton
              color="neutral"
              variant="ghost"
              size="lg"
              :to="{ name: 'course-topic', params: { enrollmentId, topicId } }"
            >
              К теме
            </UButton>
          </div>
        </div>
      </template>
    </div>
  </UMain>
</template>
