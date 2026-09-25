<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useBreadcrumbCurrentLabel } from '../../../composables/usePortalNavigation';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';
import { describeDeadline, formatDateTime } from '../courseDeadline';
import { followCourseNextAction, isActionableStep } from '../followNextAction';
import { plural } from '../courseDuration';

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

const enrollment = computed(() => data.value?.enrollment || null);
const title = computed(() => enrollment.value?.course?.title || 'Обучение');
const status = computed(() => String(enrollment.value?.status || ''));
const progress = computed(() => enrollment.value?.progress || data.value?.progress || {});
const percent = computed(() => Number(progress.value?.percent ?? 0));
const topicsDone = computed(() => Number(progress.value?.topicsCompleted ?? 0));
const topicsTotal = computed(() => Number(progress.value?.topicsTotal ?? 0));
const next = computed(() => data.value?.nextAction || progress.value?.nextAction || null);
const topics = computed<any[]>(() => data.value?.version?.topics || []);

const isCompleted = computed(() => status.value === 'completed');
/** Пройденный курс открывается на повтор — темы не блокируются. */
const isReview = computed(() => isCompleted.value || status.value === 'failed');
const notStarted = computed(() => status.value === 'not_started');

const deadline = computed(() => describeDeadline(enrollment.value?.deadlineAt, isCompleted.value));
const deadlineClass = computed(() => {
  if (deadline.value?.tone === 'error') return 'text-error';
  if (deadline.value?.tone === 'warning') return 'text-warning';
  return 'text-muted';
});
const completedAtLabel = computed(() => formatDateTime(enrollment.value?.completedAt));

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

async function load() {
  loading.value = true;
  loadError.value = null;
  try {
    data.value = await store.getEnrollment(enrollmentId.value);
  } catch (e: any) {
    data.value = null;
    loadError.value = e?.message || 'Курс недоступен';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function topicTitleById(id: unknown) {
  return topics.value.find((t) => Number(t.id) === Number(id))?.title || '';
}

/** Главная кнопка: подпись говорит, куда именно она ведёт. */
const primary = computed<null | { label: string; icon: string }>(() => {
  if (isReview.value) return null;
  if (notStarted.value) return { label: 'Начать курс', icon: 'i-lucide-play' };
  const a = next.value;
  if (!isActionableStep(a)) return null;
  switch (a.type) {
    case 'topic_test':
      return { label: a.label || 'Пройти тест темы', icon: 'i-lucide-clipboard-check' };
    case 'final_test':
      return { label: 'Пройти итоговый тест', icon: 'i-lucide-clipboard-check' };
    case 'complete_course':
      return { label: 'Завершить и посмотреть итоги', icon: 'i-lucide-award' };
    default: {
      const t = topicTitleById(a.topicId);
      return { label: t ? `Продолжить: «${t}»` : 'Продолжить', icon: 'i-lucide-arrow-right' };
    }
  }
});

const hint = computed(() => {
  if (isCompleted.value) {
    return completedAtLabel.value
      ? `Обучение завершено ${completedAtLabel.value}. Материалы можно открыть снова.`
      : 'Обучение завершено. Материалы можно открыть снова.';
  }
  if (notStarted.value) return 'Нажмите «Начать курс» — откроется первая тема.';
  const a = next.value;
  if (a?.type === 'material') return 'Следующий шаг — изучить материал.';
  if (a?.type === 'topic_test') return 'Следующий шаг — тест по теме.';
  if (a?.type === 'topic') return 'Тема почти пройдена — в ней осталось провести немного времени.';
  if (a?.type === 'final_test') return 'Все темы пройдены — остался итоговый тест.';
  if (a?.type === 'complete_course') return 'Все шаги выполнены — осталось подвести итоги.';
  if (a?.type === 'locked') return a.label || 'Следующая тема пока закрыта.';
  return '';
});

async function onPrimary() {
  acting.value = true;
  try {
    let action = next.value;
    if (notStarted.value) {
      await store.startCourse(enrollmentId.value);
      data.value = await store.getEnrollment(enrollmentId.value);
      action = next.value;
    }
    if (!isActionableStep(action)) {
      toast.add({ title: 'Следующий шаг пока недоступен', description: action?.label, color: 'warning', icon: 'i-lucide-info' });
      return;
    }
    await followCourseNextAction(router, enrollmentId.value, action);
  } catch (e: any) {
    toast.add({ title: 'Не удалось продолжить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    acting.value = false;
  }
}

function topicStatus(t: any): string {
  return t.progress?.status || 'locked';
}

function isLocked(t: any) {
  return !isReview.value && topicStatus(t) === 'locked';
}

/** Статус словами — не только иконкой (правило доступности «не только цветом»). */
function topicStatusText(t: any, idx: number) {
  const s = topicStatus(t);
  if (s === 'completed') return 'Пройдена';
  if (isReview.value) return 'Доступна для просмотра';
  if (s === 'in_progress') return 'В процессе';
  if (s === 'available') return 'Доступна';
  return idx > 0 ? `Откроется после темы ${idx}` : 'Пока закрыта';
}

function topicIcon(t: any) {
  const s = topicStatus(t);
  if (s === 'completed') return 'i-lucide-circle-check';
  if (isLocked(t)) return 'i-lucide-lock';
  if (s === 'in_progress') return 'i-lucide-circle-dot';
  return 'i-lucide-circle';
}

function topicMeta(t: any) {
  const n = Number(t.materialsCount ?? t.materials?.length ?? 0);
  const parts = [`${n} ${plural(n, ['материал', 'материала', 'материалов'])}`];
  if (t.topicTest || t.testLink) parts.push('тест');
  return parts.join(' · ');
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <div v-if="loading" class="flex flex-col gap-4" aria-busy="true" aria-label="Загрузка курса">
        <USkeleton class="h-16 w-2/3 rounded-lg" />
        <USkeleton class="h-36 w-full rounded-panel" />
        <USkeleton v-for="n in 3" :key="n" class="h-16 w-full rounded-panel" />
      </div>

      <UAlert
        v-else-if="loadError || !enrollment"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Курс не загрузился"
        :description="loadError || 'Не удалось открыть назначение.'"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="load">Повторить</UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'courses' }">К моему обучению</UButton>
        </template>
      </UAlert>

      <template v-else>
        <UPageHeader headline="Обучение" :title="title" :description="data.version?.shortDescription || undefined">
          <template #title>
            <div class="flex items-center gap-2 flex-wrap min-w-0">
              <span class="break-words">{{ title }}</span>
              <CourseStatusBadge :status="status" />
            </div>
          </template>
        </UPageHeader>

        <UAlert
          v-if="status === 'overdue'"
          color="error"
          variant="subtle"
          icon="i-lucide-calendar-x"
          title="Срок прохождения истёк"
          :description="`${deadline?.label || 'Срок прошёл'}. Курс всё ещё можно пройти до конца — продолжайте с того места, где остановились.`"
        />

        <section
          class="rounded-panel bg-elevated p-4 sm:p-5 flex flex-col gap-4 min-w-0"
          aria-labelledby="enrollment-progress-title"
        >
          <h2 id="enrollment-progress-title" class="sr-only">Прогресс</h2>
          <div class="flex flex-col gap-2">
            <div class="flex items-baseline justify-between gap-3 flex-wrap">
              <p class="text-sm text-highlighted">
                Пройдено {{ topicsDone }} из {{ topicsTotal }}
                {{ plural(topicsTotal, ['темы', 'тем', 'тем']) }}
              </p>
              <p class="text-2xl font-semibold text-highlighted tabular-nums">{{ percent }}%</p>
            </div>
            <UProgress
              :model-value="percent"
              size="md"
              :color="status === 'overdue' ? 'error' : 'primary'"
              :aria-label="`Пройдено ${percent} процентов`"
            />
            <p v-if="deadline" class="text-xs inline-flex items-center gap-1" :class="deadlineClass" :title="deadline.full">
              <UIcon :name="deadline.icon" class="size-3.5 shrink-0" aria-hidden="true" />
              {{ deadline.label }}
            </p>
          </div>

          <p v-if="hint" class="text-sm text-muted">{{ hint }}</p>

          <div class="flex flex-wrap gap-2">
            <UButton
              v-if="primary"
              color="primary"
              size="lg"
              :icon="primary.icon"
              :loading="acting"
              @click="onPrimary"
            >
              {{ primary.label }}
            </UButton>
            <UButton
              v-if="isCompleted"
              color="primary"
              size="lg"
              icon="i-lucide-award"
              :to="{ name: 'course-result', params: { enrollmentId } }"
            >
              Итоги и сертификат
            </UButton>
          </div>
        </section>

        <section class="flex flex-col gap-3 min-w-0" aria-labelledby="enrollment-topics-title">
          <h2 id="enrollment-topics-title" class="text-lg font-bold leading-7 text-highlighted">Темы</h2>

          <UEmpty
            v-if="!topics.length"
            variant="naked"
            icon="i-lucide-list"
            title="В курсе пока нет тем"
            description="Автор ещё наполняет обучение. Загляните позже."
            class="py-8"
          />

          <ol v-else class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
            <li v-for="(t, idx) in topics" :key="t.id">
              <component
                :is="isLocked(t) ? 'div' : RouterLink"
                :to="isLocked(t) ? undefined : { name: 'course-topic', params: { enrollmentId, topicId: t.id } }"
                class="rounded-panel bg-elevated p-3 sm:px-4 flex items-center gap-3 min-w-0 transition-shadow"
                :class="isLocked(t)
                  ? 'opacity-70'
                  : 'hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-primary'"
                :aria-disabled="isLocked(t) || undefined"
              >
                <span class="text-xs text-dimmed tabular-nums shrink-0 w-5 text-right">{{ idx + 1 }}</span>
                <div class="flex-1 min-w-0 flex flex-col gap-0.5">
                  <p class="font-medium text-highlighted break-words">{{ t.title }}</p>
                  <p class="text-xs text-muted">{{ topicStatusText(t, idx) }} · {{ topicMeta(t) }}</p>
                </div>
                <UIcon
                  :name="topicIcon(t)"
                  class="size-5 shrink-0"
                  :class="topicStatus(t) === 'completed' ? 'text-success' : 'text-dimmed'"
                  aria-hidden="true"
                />
              </component>
            </li>
          </ol>
        </section>
      </template>
    </div>
  </UMain>
</template>
