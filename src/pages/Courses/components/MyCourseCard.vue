<script setup lang="ts">
/**
 * Одна карточка назначения — общая для всех статусов.
 * Слева цветная кромка для требующих внимания (просрочен / не сдан).
 */
import { computed } from 'vue';
import type { EnrollmentSummary } from '../../../composables/useCoursesStore';
import { describeDeadline, formatDateTime } from '../courseDeadline';
import { isActionableStep } from '../followNextAction';
import CourseStatusBadge from './CourseStatusBadge.vue';

const props = defineProps<{ enrollment: EnrollmentSummary }>();
const emit = defineEmits<{ open: [enrollment: EnrollmentSummary] }>();

const status = computed(() => String(props.enrollment.status || ''));
const isCompleted = computed(() => status.value === 'completed');
const needsAttention = computed(() => status.value === 'overdue' || status.value === 'failed');

const percent = computed(() => Math.max(0, Math.min(100, Number(props.enrollment.progressPercent ?? 0))));
const topicsDone = computed(() => Number(props.enrollment.topicsCompleted ?? 0));
const topicsTotal = computed(() => Number(props.enrollment.topicsTotal ?? 0));
const started = computed(() => percent.value > 0 || topicsDone.value > 0);

const deadline = computed(() => describeDeadline(props.enrollment.deadlineAt, isCompleted.value));
const deadlineClass = computed(() => {
  if (deadline.value?.tone === 'error') return 'text-error';
  if (deadline.value?.tone === 'warning') return 'text-warning';
  return 'text-muted';
});

const completedLabel = computed(() =>
  props.enrollment.completedAt ? `Завершено ${formatDateTime(props.enrollment.completedAt)}` : '',
);

/**
 * Подсказка о следующем шаге — только пока курс в работе и только для шага,
 * который можно сделать. Иначе на карточке появлялось «Дальше: Просрочен дедлайн».
 */
const nextStep = computed(() => {
  if (isCompleted.value || !started.value) return '';
  if (!isActionableStep(props.enrollment.nextAction)) return '';
  const label = props.enrollment.nextAction?.label?.trim();
  return label ? `Дальше: ${label}` : '';
});

const cta = computed(() => {
  if (isCompleted.value) return { label: 'Смотреть', icon: 'i-lucide-eye', color: 'neutral' as const, variant: 'soft' as const };
  if (status.value === 'failed') return { label: 'Открыть', icon: 'i-lucide-arrow-right', color: 'primary' as const, variant: 'solid' as const };
  if (!started.value) return { label: 'Начать', icon: 'i-lucide-play', color: 'primary' as const, variant: 'solid' as const };
  return { label: 'Продолжить', icon: 'i-lucide-play', color: 'primary' as const, variant: 'solid' as const };
});

const rootClass = computed(() => {
  if (status.value === 'overdue') return 'rounded-panel bg-error/5 ring-1 ring-inset ring-error/25 border-0 divide-y-0';
  if (status.value === 'failed') return 'rounded-panel bg-warning/5 ring-1 ring-inset ring-warning/25 border-0 divide-y-0';
  if (isCompleted.value) return 'rounded-panel bg-elevated/60 ring-0 border-0 divide-y-0';
  return 'rounded-panel bg-elevated ring-0 border-0 divide-y-0';
});

const ariaLabel = computed(() => `Открыть обучение «${props.enrollment.courseTitle}»`);
</script>

<template>
  <UCard
    variant="soft"
    class="w-full rounded-panel transition-shadow hover:shadow-md focus-within:ring-2 focus-within:ring-primary/40"
    :ui="{ root: rootClass, body: 'flex flex-col md:flex-row md:items-center gap-3 p-4 sm:p-4' }"
  >
    <div class="flex-1 min-w-0 flex flex-col gap-1.5">
      <button
        type="button"
        class="font-medium text-highlighted text-left break-words hover:text-primary focus:outline-none focus-visible:underline"
        :aria-label="ariaLabel"
        @click="emit('open', enrollment)"
      >
        {{ enrollment.courseTitle }}
      </button>

      <div class="flex items-center gap-x-2 gap-y-1 flex-wrap text-xs text-muted">
        <CourseStatusBadge :status="status" />
        <span v-if="deadline" class="inline-flex items-center gap-1" :class="deadlineClass" :title="deadline.full">
          <UIcon :name="deadline.icon" class="size-3.5 shrink-0" />
          {{ deadline.label }}
        </span>
        <span v-if="deadline && topicsTotal > 0" aria-hidden="true">·</span>
        <span v-if="topicsTotal > 0">Темы: {{ topicsDone }} из {{ topicsTotal }}</span>
        <span v-if="isCompleted && completedLabel">{{ completedLabel }}</span>
      </div>

      <UProgress
        v-if="started && !isCompleted"
        :model-value="percent"
        size="sm"
        class="mt-0.5"
        :color="needsAttention ? 'error' : 'primary'"
        :aria-label="`Пройдено ${percent} процентов, ${topicsDone} из ${topicsTotal} тем`"
      />

      <p v-if="nextStep" class="text-xs text-toned break-words">{{ nextStep }}</p>
    </div>

    <UButton
      :color="cta.color"
      :variant="cta.variant"
      :icon="cta.icon"
      class="shrink-0 self-start md:self-center"
      :aria-label="ariaLabel"
      @click="emit('open', enrollment)"
    >
      {{ cta.label }}
    </UButton>
  </UCard>
</template>
