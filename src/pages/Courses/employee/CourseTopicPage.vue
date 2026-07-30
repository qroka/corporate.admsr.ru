<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { newsEditorHtmlClass } from '../../../composables/newsEditorHtmlClass';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const topicId = computed(() => Number(route.params.topicId));
const loading = ref(true);
const data = ref<any>(null);
const activeMaterialId = ref<number | null>(null);
const lastActivityAt = ref(Date.now());
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Мои курсы', to: { name: 'courses' } },
  {
    label: 'Курс',
    to: { name: 'course-enrollment', params: { enrollmentId: enrollmentId.value } },
  },
  { label: data.value?.topic?.title || 'Тема' },
]);

const materials = computed(() => data.value?.topic?.materials || data.value?.materials || []);
const topicTest = computed(() => data.value?.topic?.topicTest || data.value?.topic?.testLink || data.value?.testLink);
const isReview = computed(() => Boolean(data.value?.reviewMode));
const nextAction = computed(() => data.value?.nextAction || null);
const topicsList = computed(() => data.value?.topicsList || []);

const allMaterialsDone = computed(() => {
  const list = materials.value;
  if (!list.length) return true;
  return list.every((m: any) => m.isRequired === false || matStatus(m) === 'completed');
});

/** Тест темы ещё обязателен (материалы уже изучены). */
const needsTopicTest = computed(() => {
  if (isReview.value || !topicTest.value?.id) return false;
  if (topicTest.value.isRequired === false) return false;
  const a = nextAction.value;
  if (a?.type === 'topic_test' && Number(a.courseTestLinkId) === Number(topicTest.value.id)) {
    return true;
  }
  // если nextAction всё ещё на этой теме (материалы/тест) — тест не закрыт
  if (a?.topicId != null && Number(a.topicId) === topicId.value) {
    return a.type === 'topic_test' || a.type === 'complete_topic';
  }
  return false;
});

const nextTopic = computed(() => {
  const fromApi = data.value?.nextTopic;
  if (fromApi?.id) return fromApi;
  const list = topicsList.value;
  const idx = list.findIndex((t: any) => Number(t.id) === topicId.value);
  if (idx < 0 || idx >= list.length - 1) return null;
  const t = list[idx + 1];
  return { id: Number(t.id), title: String(t.title || 'Следующая тема'), status: t.progress?.status || null };
});

const canGoNext = computed(() => {
  if (!allMaterialsDone.value && !isReview.value) return false;
  if (needsTopicTest.value) return false;
  if (nextTopic.value?.id) return true;
  if (isReview.value) return false;
  const t = nextAction.value?.type;
  return t === 'final_test' || t === 'complete_course' || t === 'done';
});

const nextLabel = computed(() => {
  if (nextTopic.value?.id) return 'К следующей теме';
  if (nextAction.value?.type === 'final_test') return 'К итоговому тесту';
  if (nextAction.value?.type === 'done' || nextAction.value?.type === 'complete_course') {
    return 'К результату';
  }
  return 'Далее';
});

const showTopicTest = computed(() => {
  if (!topicTest.value?.id || isReview.value) return false;
  // после материалов — основной следующий шаг, либо пока тест ещё нужен
  return allMaterialsDone.value || needsTopicTest.value;
});

function onActivity() {
  lastActivityAt.value = Date.now();
}

function isPageActive() {
  return document.visibilityState === 'visible' && document.hasFocus();
}

async function tickHeartbeat() {
  if (!activeMaterialId.value || !isPageActive()) return;
  // считаем активным, если было взаимодействие за последние 30с
  if (Date.now() - lastActivityAt.value > 30_000) return;
  try {
    await store.heartbeat({
      enrollmentId: enrollmentId.value,
      materialId: activeMaterialId.value,
      seconds: 15,
    });
  } catch {
    /* ignore transient */
  }
}

onMounted(async () => {
  await loadTopic();
  window.addEventListener('mousemove', onActivity);
  window.addEventListener('keydown', onActivity);
  window.addEventListener('scroll', onActivity, true);
  window.addEventListener('click', onActivity);
  heartbeatTimer = setInterval(() => void tickHeartbeat(), 15_000);
});

async function loadTopic() {
  loading.value = true;
  activeMaterialId.value = null;
  try {
    const [topicData, enrollment] = await Promise.all([
      store.getTopic(enrollmentId.value, topicId.value),
      store.getEnrollment(enrollmentId.value).catch(() => null),
    ]);
    data.value = {
      ...topicData,
      topicsList: enrollment?.version?.topics || topicData?.topicsList || [],
    };
  } catch (e: any) {
    data.value = null;
    toast.add({ title: 'Тема недоступна', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
}

watch(topicId, () => {
  void loadTopic();
});

onUnmounted(() => {
  if (heartbeatTimer) clearInterval(heartbeatTimer);
  window.removeEventListener('mousemove', onActivity);
  window.removeEventListener('keydown', onActivity);
  window.removeEventListener('scroll', onActivity, true);
  window.removeEventListener('click', onActivity);
});

async function openMaterial(m: any) {
  try {
    await store.openMaterial(enrollmentId.value, m.id);
    activeMaterialId.value = m.id;
    lastActivityAt.value = Date.now();
    if (m.type === 'link' && m.externalUrl) {
      window.open(m.externalUrl, '_blank', 'noopener');
    } else if (m.fileUrl) {
      window.open(m.fileUrl, '_blank', 'noopener');
    }
  } catch (e: any) {
    toast.add({ title: 'Не удалось открыть', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  }
}

async function completeMaterial(m: any) {
  try {
    await store.completeMaterial(enrollmentId.value, m.id);
    toast.add({ title: 'Материал отмечен', color: 'success', icon: 'i-lucide-check' });
    data.value = await store.getTopic(enrollmentId.value, topicId.value);
  } catch (e: any) {
    toast.add({ title: 'Не удалось завершить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  }
}

function matStatus(m: any) {
  return m.progress?.status || 'not_started';
}

function goTest() {
  if (!topicTest.value?.id) return;
  router.push({
    name: 'course-test',
    params: {
      enrollmentId: enrollmentId.value,
      courseTestLinkId: topicTest.value.id,
    },
  });
}

function goNext() {
  if (needsTopicTest.value && topicTest.value?.id) {
    goTest();
    return;
  }
  if (nextTopic.value?.id) {
    router.push({
      name: 'course-topic',
      params: {
        enrollmentId: enrollmentId.value,
        topicId: nextTopic.value.id,
      },
    });
    return;
  }
  const action = nextAction.value;
  if (action?.type === 'final_test' && action.courseTestLinkId) {
    router.push({
      name: 'course-test',
      params: {
        enrollmentId: enrollmentId.value,
        courseTestLinkId: action.courseTestLinkId,
      },
    });
    return;
  }
  if (action?.type === 'done' || action?.type === 'complete_course') {
    router.push({
      name: 'course-result',
      params: { enrollmentId: enrollmentId.value },
    });
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <UBreadcrumb :items="crumbs" />

    <div v-if="loading" class="flex flex-col gap-3 p-1">
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <template v-else-if="data">
      <div class="min-w-0 p-1 flex flex-col gap-4">
      <h1 class="text-2xl font-medium text-highlighted break-words">
        {{ data.topic?.title || 'Тема' }}
      </h1>
      <p v-if="data.topic?.description" class="text-sm text-muted break-words whitespace-pre-wrap">
        {{ data.topic.description }}
      </p>
      <UAlert
        v-if="isReview"
        color="neutral"
        variant="subtle"
        icon="i-lucide-book-open"
        title="Повторный просмотр"
        description="Можно снова открыть материалы. Прохождение и тесты уже завершены."
      />

      <section class="flex flex-col gap-2 min-w-0">
        <h2 class="text-lg font-medium">Материалы</h2>
        <UEmpty
          v-if="!materials.length"
          icon="i-lucide-file"
          title="Нет материалов"
          class="py-8"
        />
        <ul v-else class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
          <li
            v-for="m in materials"
            :key="m.id"
            class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-3 min-w-0"
            :class="activeMaterialId === m.id ? 'ring-primary' : ''"
          >
            <div class="flex items-start justify-between gap-2 min-w-0">
              <div class="min-w-0">
                <p class="font-medium break-words">{{ m.title }}</p>
                <CourseStatusBadge :status="matStatus(m)" class="mt-1" />
              </div>
            </div>

            <div
              v-if="m.type === 'rich_text' && m.contentHtml && activeMaterialId === m.id"
              :class="['rounded-lg bg-elevated/40 p-3 sm:p-4 text-default min-w-0 overflow-x-auto', newsEditorHtmlClass]"
              v-html="m.contentHtml"
            />

            <div class="flex flex-wrap gap-2">
              <UButton
                color="primary"
                variant="soft"
                size="sm"
                icon="i-lucide-book-open"
                @click="openMaterial(m)"
              >
                {{ isReview || matStatus(m) === 'completed' ? 'Смотреть снова' : 'Открыть' }}
              </UButton>
              <UButton
                v-if="!isReview && matStatus(m) !== 'completed'"
                color="neutral"
                variant="outline"
                size="sm"
                icon="i-lucide-check"
                @click="completeMaterial(m)"
              >
                Отметить изученным
              </UButton>
            </div>
          </li>
        </ul>
      </section>

      <div
        v-if="allMaterialsDone || isReview"
        class="flex flex-col gap-3 pt-2 border-t border-default"
      >
        <p v-if="!isReview && allMaterialsDone" class="text-sm text-muted">
          <template v-if="needsTopicTest">Материалы изучены — пройдите тест темы, чтобы открыть следующую.</template>
          <template v-else-if="nextTopic">Материалы изучены — можно перейти к следующей теме.</template>
          <template v-else>Материалы изучены.</template>
        </p>
        <div class="flex flex-wrap gap-2">
          <UButton
            v-if="showTopicTest"
            color="primary"
            size="lg"
            class="w-fit"
            icon="i-lucide-clipboard-check"
            @click="goTest"
          >
            Пройти тест темы
          </UButton>
          <UButton
            v-if="canGoNext"
            color="primary"
            size="lg"
            class="w-fit"
            trailing-icon="i-lucide-arrow-right"
            @click="goNext"
          >
            {{ nextLabel }}
          </UButton>
        </div>
      </div>
      </div>
    </template>
  </UMain>
</template>
