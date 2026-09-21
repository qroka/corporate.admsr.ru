<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useBreadcrumbCurrentLabel } from '../../../composables/usePortalNavigation';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const breadcrumbLabel = useBreadcrumbCurrentLabel();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const loading = ref(true);
const loadError = ref<string | null>(null);
const acting = ref(false);
const data = ref<any>(null);

const title = computed(() => data.value?.enrollment?.course?.title || data.value?.version?.title || 'Обучение');

watch(
  title,
  (t) => {
    breadcrumbLabel.value = t.trim() || null;
  },
  { immediate: true },
);

onUnmounted(() => {
  breadcrumbLabel.value = null;
});
const progress = computed(() => data.value?.enrollment?.progress || data.value?.progress || {});
const percent = computed(() => progress.value?.percent ?? 0);
const topicsDone = computed(() => progress.value?.topicsCompleted ?? 0);
const topicsTotal = computed(() => progress.value?.topicsTotal ?? 0);
const next = computed(() => data.value?.nextAction || progress.value?.nextAction);
const topics = computed(() => data.value?.version?.topics || []);
const enrollmentStatus = computed(() => String(data.value?.enrollment?.status || ''));
const isReview = computed(() => ['completed', 'failed'].includes(enrollmentStatus.value));
const completedAt = computed(() => data.value?.enrollment?.completedAt || null);

const completedAtLabel = computed(() => {
  const raw = completedAt.value;
  if (!raw) return '';
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return '';
  const date = d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${date}, ${time}`;
});

const continueLabel = computed(() => {
  if (isReview.value) return 'Смотреть материалы';
  if (data.value?.enrollment?.status === 'not_started') return 'Начать курс';
  return next.value?.label || 'Продолжить';
});

const nextHint = computed(() => {
  if (enrollmentStatus.value === 'completed' && completedAtLabel.value) {
    return `Обучение завершено ${completedAtLabel.value}. Можно снова открыть темы.`;
  }
  if (isReview.value) return 'Курс завершён — можно снова открыть темы.';
  if (data.value?.enrollment?.status === 'not_started') {
    return 'Нажмите «Начать курс», чтобы открыть первую тему.';
  }
  const t = next.value?.type;
  if (t === 'topic' || t === 'topic_material') return 'Следующий шаг — продолжить тему.';
  if (t === 'topic_test') return 'Следующий шаг — тест по теме.';
  if (t === 'final_test') return 'Следующий шаг — итоговый тест.';
  if (t === 'done' || t === 'complete_course') return 'Курс почти завершён — откройте итоги.';
  return next.value?.label || 'Продолжите с того места, где остановились.';
});

onMounted(async () => {
  loading.value = true;
  loadError.value = null;
  try {
    data.value = await store.getEnrollment(enrollmentId.value);
  } catch (e: any) {
    loadError.value = e?.message || 'Курс недоступен';
    toast.add({ title: 'Курс недоступен', description: loadError.value, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

function topicStatus(t: any) {
  return t.progress?.status || 'locked';
}

function isLocked(t: any) {
  if (isReview.value) return false;
  return topicStatus(t) === 'locked';
}

async function continueLearning() {
  if (isReview.value) {
    const first = topics.value[0];
    if (first) openTopic(first);
    return;
  }
  acting.value = true;
  try {
    const status = data.value?.enrollment?.status;
    if (status === 'not_started') {
      await store.startCourse(enrollmentId.value);
      data.value = await store.getEnrollment(enrollmentId.value);
    }
    const action = data.value?.nextAction || (await store.nextAction(enrollmentId.value) as any)?.nextAction;
    if (!action) return;
    if (action.type === 'topic_test' || action.type === 'final_test') {
      await router.push({
        name: 'course-test',
        params: {
          enrollmentId: enrollmentId.value,
          courseTestLinkId: action.courseTestLinkId,
        },
      });
    } else if (action.topicId) {
      await router.push({
        name: 'course-topic',
        params: { enrollmentId: enrollmentId.value, topicId: action.topicId },
      });
    } else if (action.type === 'done' || action.type === 'complete_course') {
      await router.push({ name: 'course-result', params: { enrollmentId: enrollmentId.value } });
    }
  } catch (e: any) {
    toast.add({ title: 'Не удалось продолжить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    acting.value = false;
  }
}

function openTopic(t: any) {
  if (isLocked(t)) return;
  router.push({
    name: 'course-topic',
    params: { enrollmentId: enrollmentId.value, topicId: t.id },
  });
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <div v-if="loading" class="flex flex-col gap-3 p-1">
      <USkeleton class="h-10 w-2/3 rounded-lg" />
      <USkeleton class="h-24 w-full rounded-xl" />
    </div>

    <template v-else-if="data">
      <div class="flex flex-col gap-3 min-w-0 rounded-xl ring-1 ring-default bg-elevated/40 p-4">
        <div class="flex flex-col gap-2 min-w-0">
          <h1 class="text-2xl font-medium text-highlighted break-words">{{ title }}</h1>
          <p v-if="data.version?.shortDescription" class="text-sm text-muted break-words whitespace-pre-wrap">
            {{ data.version.shortDescription }}
          </p>
        </div>

        <div class="flex flex-col gap-2">
          <UProgress
            :model-value="percent"
            size="md"
            color="primary"
            :aria-label="`Завершено ${topicsDone} из ${topicsTotal} тем, ${percent} процентов`"
          />
          <p class="text-sm text-dimmed">
            Завершено {{ topicsDone }} из {{ topicsTotal }} тем · {{ percent }}%
          </p>
          <p v-if="enrollmentStatus === 'completed' && completedAtLabel" class="text-sm text-muted">
            Дата завершения: {{ completedAtLabel }}
          </p>
        </div>

        <p class="text-sm text-muted">{{ nextHint }}</p>

        <div class="flex flex-wrap gap-2">
          <UButton
            color="primary"
            size="lg"
            class="w-fit"
            :loading="acting"
            :icon="isReview ? 'i-lucide-book-open' : (data.enrollment?.status === 'not_started' ? 'i-lucide-play' : 'i-lucide-arrow-right')"
            @click="continueLearning"
          >
            {{ continueLabel }}
          </UButton>
          <UButton
            v-if="isReview"
            color="neutral"
            variant="soft"
            size="lg"
            icon="i-lucide-award"
            :to="{ name: 'course-result', params: { enrollmentId } }"
          >
            Итоги
          </UButton>
        </div>
      </div>

      <section class="flex flex-col gap-2 min-w-0 p-1">
        <h2 class="text-lg font-medium">Темы</h2>
        <ul class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
          <li
            v-for="(t, idx) in topics"
            :key="t.id"
            class="rounded-xl ring-1 ring-default p-4 flex items-center gap-3 min-w-0"
            :class="isLocked(t) ? 'opacity-60' : 'cursor-pointer hover:bg-elevated/40'"
            @click="openTopic(t)"
          >
            <span class="text-dimmed tabular-nums text-sm w-6 shrink-0">{{ idx + 1 }}</span>
            <div class="flex-1 min-w-0">
              <p class="font-medium break-words">{{ t.title }}</p>
            </div>
            <UIcon
              :name="isLocked(t) ? 'i-lucide-lock' : (topicStatus(t) === 'completed' ? 'i-lucide-check-circle' : 'i-lucide-chevron-right')"
              class="size-5 text-dimmed shrink-0"
            />
          </li>
        </ul>
      </section>
    </template>

    <UAlert
      v-else
      color="error"
      variant="subtle"
      icon="i-lucide-alert-circle"
      title="Курс не загрузился"
      :description="loadError || 'Не удалось открыть запись на курс.'"
    >
      <template #actions>
        <UButton color="neutral" variant="outline" :to="{ name: 'courses' }">К моему обучению</UButton>
      </template>
    </UAlert>
  </UMain>
</template>
