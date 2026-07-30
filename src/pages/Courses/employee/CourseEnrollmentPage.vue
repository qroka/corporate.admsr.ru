<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const loading = ref(true);
const acting = ref(false);
const data = ref<any>(null);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Мои курсы', to: { name: 'courses' } },
  { label: title.value },
]);

const title = computed(() => data.value?.enrollment?.course?.title || data.value?.version?.title || 'Курс');
const progress = computed(() => data.value?.enrollment?.progress || data.value?.progress || {});
const percent = computed(() => progress.value?.percent ?? 0);
const topicsDone = computed(() => progress.value?.topicsCompleted ?? 0);
const topicsTotal = computed(() => progress.value?.topicsTotal ?? 0);
const next = computed(() => data.value?.nextAction || progress.value?.nextAction);
const topics = computed(() => data.value?.version?.topics || []);
const enrollmentStatus = computed(() => String(data.value?.enrollment?.status || ''));
const isReview = computed(() => ['completed', 'failed'].includes(enrollmentStatus.value));

onMounted(async () => {
  loading.value = true;
  try {
    data.value = await store.getEnrollment(enrollmentId.value);
  } catch (e: any) {
    toast.add({ title: 'Курс недоступен', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
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
    <UBreadcrumb :items="crumbs" />

    <div v-if="loading" class="flex flex-col gap-3 p-1">
      <USkeleton class="h-10 w-2/3 rounded-lg" />
      <USkeleton class="h-24 w-full rounded-xl" />
    </div>

    <template v-else-if="data">
      <div class="flex flex-col gap-2 min-w-0 p-1">
        <h1 class="text-2xl font-medium text-highlighted break-words">{{ title }}</h1>
        <p v-if="data.version?.shortDescription" class="text-sm text-muted break-words whitespace-pre-wrap">
          {{ data.version.shortDescription }}
        </p>
        <p v-if="isReview" class="text-sm text-muted">
          Можно снова открыть темы и материалы курса.
        </p>
      </div>

      <div class="flex flex-col gap-2 p-1">
        <UProgress
          :model-value="percent"
          size="md"
          color="primary"
          :aria-label="`Завершено ${topicsDone} из ${topicsTotal} тем, ${percent} процентов`"
        />
        <p class="text-sm text-dimmed">
          Завершено {{ topicsDone }} из {{ topicsTotal }} тем, {{ percent }}%
        </p>
      </div>

      <div class="flex flex-wrap gap-2 p-1">
        <UButton
          color="primary"
          size="lg"
          class="w-fit"
          :loading="acting"
          :icon="isReview ? 'i-lucide-book-open' : (data.enrollment?.status === 'not_started' ? 'i-lucide-play' : 'i-lucide-arrow-right')"
          @click="continueLearning"
        >
          <template v-if="isReview">Смотреть материалы</template>
          <template v-else-if="data.enrollment?.status === 'not_started'">Начать курс</template>
          <template v-else>{{ next?.label || 'Продолжить' }}</template>
        </UButton>
        <UButton
          v-if="isReview"
          color="neutral"
          variant="soft"
          size="lg"
          icon="i-lucide-award"
          :to="{ name: 'course-result', params: { enrollmentId } }"
        >
          Результат
        </UButton>
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
  </UMain>
</template>
