<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import { useTestsStore } from '../../composables/useTestsStore';
import { type Question } from './questionTypes';
import { type TestForm } from './testForm';

const props = withDefaults(defineProps<{ form: TestForm; previewHint?: boolean; persistSession?: boolean }>(), {
  previewHint: false,
  persistSession: false,
});
const emit = defineEmits<{ (e: 'finish', payload: { answers: Record<string, unknown>; durationSec: number }): void }>();

const store = useTestsStore();

const kindLabels: Record<string, string> = { test: 'Тест', survey: 'Опрос', poll: 'Голосование' };
const ctaLabel = computed(() =>
  props.form.kind === 'poll' ? 'Участвовать в голосовании'
  : props.form.kind === 'survey' ? 'Пройти опрос'
  : 'Пройти тест',
);
const finishLabel = computed(() => (props.form.kind === 'poll' ? 'Проголосовать' : 'Завершить'));
const finishIcon = computed(() => (props.form.kind === 'poll' ? 'i-lucide-vote' : 'i-lucide-check'));

// ── Постраничная навигация: 0 = титул, 1..N = вопросы ────────────────────────
const page = ref(0);
const total = computed(() => props.form.questions.length + 1);
const currentQuestion = computed(() => props.form.questions[page.value - 1]);
const isLastPage = computed(() => page.value === props.form.questions.length && props.form.questions.length > 0);
function next() { if (!timeUp.value && page.value < total.value - 1) page.value++; }
function prev() { if (!timeUp.value && page.value > 1) page.value--; } // на титул после старта не возвращаемся
watch(total, (t) => { if (page.value > t - 1) page.value = t - 1; });

// ── Ответы ───────────────────────────────────────────────────────────────────
const answers = reactive<Record<string, unknown>>({});
const yesnoItems = [
  { label: 'Да', value: 'yes' },
  { label: 'Нет', value: 'no' },
];
function optionItems(q: Question) {
  return (q.options ?? []).map((o, i) => ({ label: o.text || `Вариант ${i + 1}`, value: o.id }));
}
function scaleNumbers(q: Question) {
  const min = Math.min(q.scaleMin, q.scaleMax);
  const max = Math.max(q.scaleMin, q.scaleMax);
  const arr: number[] = [];
  for (let n = min; n <= max && arr.length < 50; n++) arr.push(n);
  return arr;
}
function isAnswered(q: Question): boolean {
  const v = answers[q.id];
  if (q.type === 'multiple') return Array.isArray(v) && v.length > 0;
  if (q.type === 'number' || q.type === 'scale') return v !== undefined && v !== null && (v as unknown) !== '';
  if (typeof v === 'string') return v.trim() !== '';
  return v !== undefined && v !== null;
}

// ── Неотвеченные ──────────────────────────────────────────────────────────────
const unanswered = computed(() =>
  props.form.questions.map((q, i) => ({ q, page: i + 1 })).filter((x) => !isAnswered(x.q)),
);
const hasRequiredUnanswered = computed(() => unanswered.value.some((x) => x.q.required));
const nextUnansweredPage = computed(() => unanswered.value.find((x) => x.page > page.value)?.page ?? null);
const reviewActive = ref(false);

function plural(n: number, one: string, few: string, many: string): string {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
}
const unansweredLabel = computed(() =>
  unanswered.value.length === 1 ? 'Перейти к вопросу' : 'Перейти к вопросам',
);

// ── Таймер ────────────────────────────────────────────────────────────────────
const startTs = ref<number | null>(null);
const nowTs = ref(Date.now());
const timeUp = ref(false);
let timerId: ReturnType<typeof setInterval> | null = null;
function hmsToSeconds(v: string): number {
  const [h = 0, m = 0, s = 0] = v.split(':').map(Number);
  return h * 3600 + m * 60 + s;
}
function secondsToHms(t: number): string {
  const x = Math.max(0, Math.floor(t));
  const h = Math.floor(x / 3600);
  const m = Math.floor((x % 3600) / 60);
  const s = x % 60;
  return [h, m, s].map((n) => String(n).padStart(2, '0')).join(':');
}
const elapsed = computed(() => (startTs.value ? Math.floor((nowTs.value - startTs.value) / 1000) : 0));
const timerLabel = computed(() =>
  props.form.useTimeLimit && props.form.timeLimit
    ? secondsToHms(hmsToSeconds(props.form.timeLimit) - elapsed.value)
    : secondsToHms(elapsed.value),
);
const timeLow = computed(
  () => props.form.useTimeLimit && !!props.form.timeLimit && hmsToSeconds(props.form.timeLimit) - elapsed.value <= 60,
);
function checkTimeUp() {
  if (props.form.useTimeLimit && props.form.timeLimit && startTs.value) {
    if (hmsToSeconds(props.form.timeLimit) - elapsed.value <= 0) {
      timeUp.value = true;
      finishOpen.value = false;
      unansweredOpen.value = false;
      if (timerId) { clearInterval(timerId); timerId = null; }
    }
  }
}
function startInterval() {
  nowTs.value = Date.now();
  if (timerId) clearInterval(timerId);
  timerId = setInterval(() => { nowTs.value = Date.now(); checkTimeUp(); }, 1000);
  checkTimeUp();
}
function stopTimer() {
  if (timerId) { clearInterval(timerId); timerId = null; }
}
onUnmounted(stopTimer);

// ── Сессия (для «Продолжить») ─────────────────────────────────────────────────
function saveSessionNow() {
  if (!props.persistSession || props.form.id == null || page.value < 1 || !startTs.value) return;
  store.saveSession(props.form.id, {
    page: page.value,
    answers: { ...answers },
    startTs: startTs.value,
  });
}
watch(page, saveSessionNow);
watch(answers, saveSessionNow, { deep: true });

function initSession() {
  if (props.persistSession && props.form.id != null) {
    const s = store.getSession(props.form.id);
    if (s && store.hasActiveSession(props.form)) {
      page.value = s.page;
      Object.assign(answers, s.answers);
      startTs.value = s.startTs;
      startInterval();
      return;
    }
  }
  page.value = 0;
}
initSession();

// ── Старт / завершение ───────────────────────────────────────────────────────
function start() {
  if (!props.form.questions.length) return;
  for (const k in answers) delete answers[k];
  reviewActive.value = false;
  timeUp.value = false;
  page.value = 1;
  startTs.value = Date.now();
  startInterval();
  saveSessionNow();
}

const finishOpen = ref(false);
const unansweredOpen = ref(false);

// Нажатие «Завершить»: если есть неотвеченные — спец-уведомление, иначе подтверждение
function attemptFinish() {
  if (unanswered.value.length === 0) finishOpen.value = true;
  else unansweredOpen.value = true;
}
function goToUnanswered() {
  unansweredOpen.value = false;
  reviewActive.value = true;
  const first = unanswered.value[0];
  if (first) page.value = first.page;
}
function finishNow() {
  stopTimer();
  finishOpen.value = false;
  unansweredOpen.value = false;
  const durationSec = startTs.value ? Math.floor((Date.now() - startTs.value) / 1000) : 0;
  if (props.persistSession && props.form.id != null) store.clearSession(props.form.id);
  emit('finish', { answers: { ...answers }, durationSec });
  page.value = 0; // на случай повторного использования (предпросмотр)
  reviewActive.value = false;
}
</script>

<template>
  <div class="flex flex-col h-full min-h-0">
    <p v-if="previewHint" class="text-xs text-dimmed mb-2 flex items-center justify-center gap-1.5 shrink-0">
      <UIcon name="i-lucide-eye" class="size-3.5" />
      Так тест увидит сотрудник
    </p>

    <!-- «Сцена»: вписываем окно 16:10 по высоте и ширине, без скролла -->
    <div class="flex-1 min-h-0 w-full grid place-items-center [container-type:size]">
      <div class="aspect-[16/10] w-[min(100%,160cqh)] max-w-4xl rounded-2xl ring-1 ring-default bg-default shadow-2xl overflow-hidden flex flex-col">

        <!-- Прогресс-бар -->
        <div v-if="form.showProgress && page > 0" class="h-1 bg-elevated shrink-0">
          <div class="h-full bg-primary transition-all" :style="{ width: `${(page / Math.max(form.questions.length, 1)) * 100}%` }" />
        </div>

        <div class="flex-1 min-h-0 overflow-hidden">
          <!-- Титульник -->
          <div v-if="page === 0" class="h-full flex flex-col items-center justify-center text-center gap-4 px-10 py-6">
            <UBadge color="primary" variant="subtle" size="lg">{{ kindLabels[form.kind] }}</UBadge>
            <h2 class="text-3xl font-semibold text-highlighted leading-tight line-clamp-2">{{ form.title || 'Без названия' }}</h2>
            <p v-if="form.description" class="text-muted max-w-xl line-clamp-4 whitespace-pre-line">{{ form.description }}</p>

            <div class="flex flex-wrap gap-1.5 justify-center">
              <UBadge v-if="form.anonymous" color="neutral" variant="subtle">Анонимно</UBadge>
              <UBadge v-if="form.useTimeLimit && form.timeLimit" color="neutral" variant="subtle">⏱ {{ form.timeLimit }}</UBadge>
              <UBadge v-if="form.limitAttempts" color="neutral" variant="subtle">Попыток: {{ form.attempts }}</UBadge>
              <UBadge v-if="form.usePassingScore && form.kind === 'test'" color="neutral" variant="subtle">Проходной: {{ form.passingScore }}%</UBadge>
              <UBadge color="neutral" variant="subtle">Вопросов: {{ form.questions.length }}</UBadge>
            </div>

            <UButton size="xl" class="mt-2" trailing-icon="i-lucide-arrow-right" :disabled="!form.questions.length" @click="start">
              {{ ctaLabel }}
            </UButton>
            <p v-if="!form.questions.length" class="text-xs text-dimmed">В форме нет вопросов.</p>
          </div>

          <!-- Страница вопроса -->
          <div v-else-if="currentQuestion" class="h-full flex flex-col px-10 py-6">
            <div class="flex items-center justify-between gap-3 shrink-0">
              <p class="text-sm text-dimmed tabular-nums">Вопрос {{ page }} из {{ form.questions.length }}</p>
              <div class="flex items-center gap-1.5 text-sm tabular-nums" :class="timeUp || timeLow ? 'text-error font-medium' : 'text-dimmed'">
                <UIcon name="i-lucide-timer" class="size-4" />
                {{ timerLabel }}
              </div>
            </div>

            <div class="flex-1 min-h-0 flex flex-col justify-center gap-5 overflow-hidden">
              <div class="flex flex-col gap-2">
                <h3 class="text-2xl font-medium text-highlighted">
                  {{ currentQuestion.title || 'Без названия' }}
                  <span v-if="currentQuestion.required" class="text-error">*</span>
                </h3>
                <p v-if="currentQuestion.hint" class="text-muted">{{ currentQuestion.hint }}</p>
              </div>

              <div class="w-full max-w-lg overflow-y-auto">
                <URadioGroup
                  v-if="currentQuestion.type === 'single'"
                  v-model="answers[currentQuestion.id]"
                  :items="optionItems(currentQuestion)" value-key="value" label-key="label" size="lg"
                />
                <UCheckboxGroup
                  v-else-if="currentQuestion.type === 'multiple'"
                  v-model="answers[currentQuestion.id]"
                  :items="optionItems(currentQuestion)" value-key="value" label-key="label" size="lg"
                />
                <USelect
                  v-else-if="currentQuestion.type === 'dropdown'"
                  v-model="answers[currentQuestion.id]"
                  :items="optionItems(currentQuestion)" value-key="value" label-key="label"
                  size="lg" class="w-full max-w-sm" placeholder="Выберите вариант"
                />
                <div v-else-if="currentQuestion.type === 'scale'" class="flex flex-col gap-2">
                  <div class="flex flex-wrap gap-2">
                    <button
                      v-for="n in scaleNumbers(currentQuestion)"
                      :key="n"
                      type="button"
                      class="size-10 rounded-lg ring-1 text-sm font-medium transition-colors"
                      :class="answers[currentQuestion.id] === n ? 'bg-primary text-inverted ring-primary' : 'ring-default text-muted hover:bg-elevated'"
                      @click="answers[currentQuestion.id] = n"
                    >
                      {{ n }}
                    </button>
                  </div>
                  <div v-if="currentQuestion.scaleMinLabel || currentQuestion.scaleMaxLabel" class="flex justify-between text-xs text-dimmed">
                    <span>{{ currentQuestion.scaleMinLabel }}</span>
                    <span>{{ currentQuestion.scaleMaxLabel }}</span>
                  </div>
                </div>
                <URadioGroup
                  v-else-if="currentQuestion.type === 'yesno'"
                  v-model="answers[currentQuestion.id]"
                  :items="yesnoItems" value-key="value" label-key="label" orientation="horizontal" size="lg"
                />
                <UInput v-else-if="currentQuestion.type === 'text'" v-model="answers[currentQuestion.id]" size="lg" class="w-full" placeholder="Короткий ответ" />
                <UTextarea v-else-if="currentQuestion.type === 'textarea'" v-model="answers[currentQuestion.id]" :rows="3" size="lg" class="w-full" placeholder="Развёрнутый ответ" />
                <UInput v-else-if="currentQuestion.type === 'number'" v-model="answers[currentQuestion.id]" type="number" size="lg" class="w-44" placeholder="0" />
                <UInput v-else-if="currentQuestion.type === 'date'" v-model="answers[currentQuestion.id]" type="date" size="lg" class="w-52" />
              </div>
            </div>
          </div>
        </div>

        <!-- Сообщение о тайм-ауте -->
        <div v-if="page > 0 && timeUp" class="shrink-0 px-5 pt-2 text-center text-xs text-error font-medium">
          Время вышло — ответ на текущий вопрос можно дать, затем форма закроется
        </div>

        <!-- Пейджер -->
        <div v-if="page > 0" class="shrink-0 border-t border-default px-5 py-3 flex items-center justify-between gap-3">
          <UButton v-if="page > 1" color="neutral" variant="ghost" size="md" leading-icon="i-lucide-arrow-left" :disabled="timeUp" @click="prev">Назад</UButton>
          <span v-else />

          <span class="text-xs text-dimmed tabular-nums">{{ page }} / {{ form.questions.length }}</span>

          <UButton v-if="timeUp" color="primary" size="md" :icon="finishIcon" @click="finishNow">Завершить</UButton>
          <UButton v-else-if="reviewActive && nextUnansweredPage" color="neutral" variant="ghost" size="md" trailing-icon="i-lucide-arrow-right" @click="page = nextUnansweredPage">Далее</UButton>
          <UButton v-else-if="isLastPage || reviewActive" color="primary" size="md" :icon="finishIcon" @click="attemptFinish">{{ finishLabel }}</UButton>
          <UButton v-else color="neutral" variant="ghost" size="md" trailing-icon="i-lucide-arrow-right" @click="next">Далее</UButton>
        </div>
      </div>
    </div>

    <!-- Подтверждение завершения (все вопросы отвечены) -->
    <UModal v-model:open="finishOpen" :title="`${finishLabel}?`" description="">
      <template #body>
        <p class="text-default">
          {{ form.kind === 'poll'
            ? 'Вы уверены, что хотите отдать голос? После отправки изменить его будет нельзя.'
            : 'Вы уверены, что хотите завершить и отправить ответы? После отправки изменить их будет нельзя.' }}
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" @click="finishOpen = false">Отмена</UButton>
          <UButton color="primary" :icon="finishIcon" @click="finishNow">{{ finishLabel }}</UButton>
        </div>
      </template>
    </UModal>

    <!-- Есть неотвеченные вопросы -->
    <UModal v-model:open="unansweredOpen" title="Остались неотвеченные вопросы" description="">
      <template #body>
        <p class="text-default">
          Вы не ответили на {{ unanswered.length }}
          {{ plural(unanswered.length, 'вопрос', 'вопроса', 'вопросов') }}.
          <template v-if="hasRequiredUnanswered"> Среди них есть обязательные — без ответа на них завершить нельзя.</template>
          <template v-else> Можно вернуться к ним или завершить как есть.</template>
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="primary" icon="i-lucide-list-checks" @click="goToUnanswered">{{ unansweredLabel }}</UButton>
          <UButton v-if="!hasRequiredUnanswered" color="neutral" variant="outline" :icon="finishIcon" @click="finishNow">{{ finishLabel }}</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
