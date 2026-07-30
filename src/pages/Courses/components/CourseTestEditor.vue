<script setup lang="ts">
/**
 * Конструктор теста курса: шаги «Параметры → Вопросы → Превью»,
 * визуально в духе модуля «Тесты» (TestBuilder).
 */
import { computed, ref } from 'vue';
import type { TestForm } from '../../Tests/testForm';
import QuestionsBuilder from '../../Tests/QuestionsBuilder.vue';
import TestRunner from '../../Tests/TestRunner.vue';
import TestSettingsForm, { type TestSettingsModel } from '../../Tests/components/TestSettingsForm.vue';
import ModalTextField from '../../Tests/ModalTextField.vue';

const props = withDefaults(
  defineProps<{
    form: TestForm;
    settings: TestSettingsModel;
    saving?: boolean;
    titlePlaceholder?: string;
    headline?: string;
    subtitle?: string;
  }>(),
  {
    saving: false,
    titlePlaceholder: 'Название теста',
    headline: 'Тест',
    subtitle: '',
  },
);

const emit = defineEmits<{
  save: [];
}>();

type StepKey = 'settings' | 'questions' | 'preview';

const STEPS: { key: StepKey; label: string; icon: string }[] = [
  { key: 'settings', label: 'Параметры', icon: 'i-lucide-sliders-horizontal' },
  { key: 'questions', label: 'Вопросы', icon: 'i-lucide-list-checks' },
  { key: 'preview', label: 'Превью', icon: 'i-lucide-eye' },
];

const step = ref<StepKey>('settings');
const stepIdx = computed(() => STEPS.findIndex((s) => s.key === step.value));
const questionsCount = computed(() => props.form.questions?.length ?? 0);
const withCorrect = computed(
  () => (props.form.questions || []).filter((q) => {
    const c = q.correct;
    if (c == null) return false;
    if (Array.isArray(c)) return c.length > 0;
    if (typeof c === 'string') return c.trim() !== '';
    return true;
  }).length,
);

function goStep(i: number) {
  if (i < 0 || i >= STEPS.length) return;
  step.value = STEPS[i].key;
}

function nextStep() {
  if (stepIdx.value < STEPS.length - 1) goStep(stepIdx.value + 1);
}

function prevStep() {
  if (stepIdx.value > 0) goStep(stepIdx.value - 1);
}

function onSave() {
  emit('save');
}
</script>

<template>
  <div class="flex flex-col flex-1 min-h-0 w-full gap-4">
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-3 min-w-0">
      <div class="flex flex-col gap-1 min-w-0">
        <h1 class="text-2xl font-medium text-highlighted break-words">{{ headline }}</h1>
        <p v-if="subtitle" class="text-sm text-muted break-words">{{ subtitle }}</p>
      </div>

      <div class="flex items-center gap-2 flex-wrap shrink-0">
        <UBadge color="neutral" variant="subtle" class="tabular-nums">
          Вопросов: {{ questionsCount }}
        </UBadge>
        <UBadge
          v-if="questionsCount"
          :color="withCorrect === questionsCount ? 'success' : 'warning'"
          variant="subtle"
          class="tabular-nums"
        >
          С ответом: {{ withCorrect }}/{{ questionsCount }}
        </UBadge>
      </div>
    </div>

    <!-- Шаги -->
    <div class="flex items-center gap-1.5 text-sm flex-wrap min-w-0 p-1">
      <template v-for="(s, i) in STEPS" :key="s.key">
        <button
          type="button"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg transition-colors"
          :class="step === s.key
            ? 'bg-primary/10 text-primary ring-1 ring-primary/40'
            : 'text-muted hover:bg-elevated'"
          @click="goStep(i)"
        >
          <UIcon :name="s.icon" class="size-4 shrink-0" />
          <span>{{ i + 1 }}. {{ s.label }}</span>
        </button>
        <UIcon
          v-if="i < STEPS.length - 1"
          name="i-lucide-chevron-right"
          class="size-4 text-dimmed shrink-0"
        />
      </template>
    </div>

    <!-- Контент шага -->
    <div class="flex-1 min-h-0 overflow-y-auto overflow-x-hidden p-1">
      <!-- Шаг 1: параметры -->
      <div
        v-show="step === 'settings'"
        class="flex flex-col lg:flex-row gap-4 items-start min-w-0"
      >
        <div class="flex-1 min-w-0 w-full flex flex-col gap-4 rounded-xl ring-1 ring-default bg-elevated/30 p-4 sm:p-5">
          <div class="flex items-center gap-2 text-highlighted font-medium">
            <UIcon name="i-lucide-file-text" class="size-5 text-primary" />
            Основное
          </div>

          <UFormField label="Название" name="title" required>
            <UInput
              v-model="form.title"
              :maxlength="150"
              size="lg"
              class="w-full"
              :placeholder="titlePlaceholder"
            />
            <template #hint>
              <span class="text-xs text-muted tabular-nums">{{ (form.title || '').length }}/150</span>
            </template>
          </UFormField>

          <UFormField label="Описание" name="description">
            <ModalTextField
              v-model="form.description"
              label="Описание"
              :maxlength="500"
              placeholder="Кратко опишите, что проверяет тест"
            />
            <template #hint>
              <span class="text-xs text-muted tabular-nums">{{ (form.description || '').length }}/500</span>
            </template>
          </UFormField>

          <USeparator />

          <div class="flex flex-col gap-2">
            <h3 class="text-sm font-medium text-highlighted">Поведение</h3>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2">
              <UCheckbox v-model="form.shuffle" label="Перемешивать вопросы" />
              <UCheckbox v-model="form.shuffleOptions" label="Перемешивать варианты ответов" />
              <UCheckbox v-model="form.showProgress" label="Показывать прогресс-бар" />
              <UCheckbox v-model="form.freeNavigation" label="Свободный переход между вопросами" />
            </div>
          </div>
        </div>

        <aside class="w-full lg:w-80 shrink-0 min-w-0 rounded-xl ring-1 ring-default bg-elevated/40 p-4 flex flex-col gap-3 [&_.grid]:!grid-cols-1">
          <div class="flex items-center gap-2 text-highlighted font-medium">
            <UIcon name="i-lucide-sliders-horizontal" class="size-5 text-primary" />
            Ограничения
          </div>
          <TestSettingsForm :model-value="settings" />
        </aside>
      </div>

      <!-- Шаг 2: вопросы -->
      <div v-show="step === 'questions'" class="flex flex-col gap-3 min-w-0">
        <div class="flex items-center justify-between gap-2 flex-wrap">
          <div class="flex flex-col gap-0.5">
            <h2 class="text-lg font-medium text-highlighted">Вопросы</h2>
            <p class="text-sm text-muted">
              Добавьте вопросы и укажите правильные ответы.
            </p>
          </div>
        </div>

        <UEmpty
          v-if="!questionsCount"
          icon="i-lucide-list-checks"
          title="Вопросов пока нет"
          description="Добавьте первый вопрос — без них тест нельзя полноценно пройти."
          class="py-10 rounded-xl ring-1 ring-default"
        />

        <QuestionsBuilder v-model="form.questions" kind="test" />
      </div>

      <!-- Шаг 3: превью -->
      <div v-show="step === 'preview'" class="flex flex-col gap-3 min-w-0 min-h-[28rem]">
        <div class="flex flex-col gap-0.5">
          <h2 class="text-lg font-medium text-highlighted">Превью</h2>
          <p class="text-sm text-muted">
            Так тест увидит сотрудник. Ответы не сохраняются.
          </p>
        </div>

        <div
          v-if="!questionsCount"
          class="rounded-xl ring-1 ring-warning/30 bg-warning/5 p-4 text-sm text-muted"
        >
          Добавьте хотя бы один вопрос на шаге «Вопросы», чтобы увидеть превью.
        </div>

        <div
          v-else
          class="rounded-xl ring-1 ring-default bg-elevated/20 p-3 sm:p-4 min-h-[24rem] flex flex-col"
        >
          <TestRunner
            v-if="step === 'preview'"
            :form="form"
            preview-hint
            class="flex-1 min-h-0"
          />
        </div>
      </div>
    </div>

    <!-- Футер навигации -->
    <div class="shrink-0 flex flex-wrap justify-between items-center gap-3 pt-3 border-t border-default">
      <UButton
        v-if="stepIdx > 0"
        color="neutral"
        variant="outline"
        size="lg"
        leading-icon="i-lucide-arrow-left"
        @click="prevStep"
      >
        Назад
      </UButton>
      <span v-else />

      <div class="flex gap-2 flex-wrap">
        <UButton
          color="primary"
          variant="soft"
          size="lg"
          icon="i-lucide-check"
          :loading="saving"
          @click="onSave"
        >
          Сохранить тест
        </UButton>
        <UButton
          v-if="stepIdx < STEPS.length - 1"
          color="primary"
          size="lg"
          trailing-icon="i-lucide-arrow-right"
          @click="nextStep"
        >
          Дальше
        </UButton>
      </div>
    </div>
  </div>
</template>
