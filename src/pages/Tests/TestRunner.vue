<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import { useTestsStore } from '../../composables/useTestsStore';
import { type Question } from './questionTypes';
import { type TestForm } from './testForm';

const props = withDefaults(defineProps<{
  form: TestForm;
  previewHint?: boolean;
  persistSession?: boolean;
  record?: boolean;
  submit?: (answers: Record<string, unknown>, durationSec: number) => Promise<unknown>;
}>(), {
  previewHint: false,
  persistSession: false,
  record: false,
  submit: undefined,
});
const emit = defineEmits<{ (e: 'finish', payload: { answers: Record<string, unknown>; durationSec: number }): void }>();

const store = useTestsStore();

const kindLabels: Record<string, string> = { test: 'Тест', survey: 'Опрос', poll: 'Голосование' };
const isTest = computed(() => props.form.kind === 'test');
const isPoll = computed(() => props.form.kind === 'poll');
const ctaLabel = computed(() =>
  isPoll.value ? 'Участвовать в голосовании' : props.form.kind === 'survey' ? 'Пройти опрос' : 'Пройти тест',
);
const finishLabel = computed(() => (isPoll.value ? 'Проголосовать' : 'Завершить'));
const finishIcon = computed(() => (isPoll.value ? 'i-lucide-vote' : 'i-lucide-check'));

type ViewItem = { q: Question; options: { id: string; text: string }[] };

function shuffled<T>(arr: T[]): T[] {
  const a = [...arr];
  for (let i = a.length - 1; i > 0; i--) { const j = Math.floor(Math.random() * (i + 1)); [a[i], a[j]] = [a[j], a[i]]; }
  return a;
}

// ── Навигация ─────────────────────────────────────────────────────────────────
const view = ref<ViewItem[]>([]);
const page = ref(0); // 0 = титул
const total = computed(() => view.value.length + 1);
const currentItem = computed(() => view.value[page.value - 1]);
const currentQuestion = computed(() => currentItem.value?.q);
const currentOptions = computed(() => currentItem.value?.options ?? []);
const isLastPage = computed(() => page.value === view.value.length && view.value.length > 0);

const canGoBack = computed(() => props.form.freeNavigation && page.value > 1 && !timeUp.value && !revealing.value);

// ── Ответы ───────────────────────────────────────────────────────────────────
const answers = reactive<Record<string, unknown>>({});
const lockedIds = ref<Set<string>>(new Set()); // заблокированные (при выкл. «изменять ответ»)
const currentLocked = computed(() => {
  const q = currentQuestion.value;
  return !!q && (lockedIds.value.has(q.id) || revealing.value || resultsView.value !== null);
});

function isAnswered(q: Question): boolean {
  const v = answers[q.id];
  if (q.type === 'multiple') return Array.isArray(v) && v.length > 0;
  if (q.type === 'number' || q.type === 'scale') return v !== undefined && v !== null && (v as unknown) !== '';
  if (typeof v === 'string') return v.trim() !== '';
  return v !== undefined && v !== null;
}
function scaleNumbers(q: Question) {
  const min = Math.min(q.scaleMin, q.scaleMax), max = Math.max(q.scaleMin, q.scaleMax);
  const arr: number[] = [];
  for (let n = min; n <= max && arr.length < 50; n++) arr.push(n);
  return arr;
}

// выбор варианта
function onPick(optId: string) {
  if (currentLocked.value) return;
  const q = currentQuestion.value!;
  if (q.type === 'multiple') {
    const arr = Array.isArray(answers[q.id]) ? [...(answers[q.id] as string[])] : [];
    const i = arr.indexOf(optId);
    if (i >= 0) arr.splice(i, 1); else arr.push(optId);
    answers[q.id] = arr;
  } else {
    answers[q.id] = optId;
  }
}
function isSelected(optId: string): boolean {
  const q = currentQuestion.value!;
  const v = answers[q.id];
  return Array.isArray(v) ? v.includes(optId) : v === optId;
}

// ── Правильные ответы (тесты) ─────────────────────────────────────────────────
function hasCorrect(q: Question): boolean {
  const c = q.correct;
  if (c == null || c === '') return false;
  return Array.isArray(c) ? c.length > 0 : true;
}
function isCorrectOption(q: Question, optId: string): boolean {
  if (q.type === 'multiple') return Array.isArray(q.correct) && q.correct.includes(optId);
  return q.correct === optId;
}
const shouldRevealImmediate = computed(
  () => isTest.value && props.form.showCorrectAnswers && !props.form.allowChangeAnswer && props.form.showResult === 'immediate',
);
const revealing = ref(false);

function optionRowClass(optId: string, boxed = true): string {
  const q = currentQuestion.value!;
  if (revealing.value && hasCorrect(q)) {
    if (isCorrectOption(q, optId)) return boxed ? 'ring-green-500 bg-green-500/10 text-green-600 dark:text-green-400' : 'text-green-600 dark:text-green-400';
    if (isSelected(optId)) return boxed ? 'ring-red-500 bg-red-500/10 text-red-600 dark:text-red-400' : 'text-red-600 dark:text-red-400';
    return boxed ? 'ring-default' : 'text-default';
  }
  if (isSelected(optId)) return boxed ? 'ring-primary bg-primary/10 text-highlighted' : 'text-highlighted';
  return boxed ? 'ring-default hover:bg-elevated' : 'text-default hover:text-highlighted';
}

// ── Таймер ────────────────────────────────────────────────────────────────────
const startTs = ref<number | null>(null);
const nowTs = ref(Date.now());
const timeUp = ref(false);
let timerId: ReturnType<typeof setInterval> | null = null;
function hmsToSeconds(v: string): number { const [h = 0, m = 0, s = 0] = v.split(':').map(Number); return h * 3600 + m * 60 + s; }
function secondsToHms(t: number): string {
  const x = Math.max(0, Math.floor(t));
  return [Math.floor(x / 3600), Math.floor((x % 3600) / 60), x % 60].map((n) => String(n).padStart(2, '0')).join(':');
}
const elapsed = computed(() => (startTs.value ? Math.floor((nowTs.value - startTs.value) / 1000) : 0));
const timerLabel = computed(() =>
  props.form.useTimeLimit && props.form.timeLimit ? secondsToHms(hmsToSeconds(props.form.timeLimit) - elapsed.value) : secondsToHms(elapsed.value),
);
const timeLow = computed(() => props.form.useTimeLimit && !!props.form.timeLimit && hmsToSeconds(props.form.timeLimit) - elapsed.value <= 60);
function checkTimeUp() {
  if (props.form.useTimeLimit && props.form.timeLimit && startTs.value && hmsToSeconds(props.form.timeLimit) - elapsed.value <= 0) {
    timeUp.value = true; finishOpen.value = false; unansweredOpen.value = false;
    if (timerId) { clearInterval(timerId); timerId = null; }
  }
}
function startInterval() {
  nowTs.value = Date.now();
  if (timerId) clearInterval(timerId);
  timerId = setInterval(() => { nowTs.value = Date.now(); checkTimeUp(); }, 1000);
  checkTimeUp();
}
function stopTimer() { if (timerId) { clearInterval(timerId); timerId = null; } }
onUnmounted(stopTimer);

// ── Сессия (для «Продолжить») ─────────────────────────────────────────────────
function saveSessionNow() {
  if (!props.persistSession || props.form.id == null || page.value < 1 || !startTs.value) return;
  store.saveSession(props.form.id, { page: page.value, answers: { ...answers }, startTs: startTs.value });
}
watch(page, saveSessionNow);
watch(answers, saveSessionNow, { deep: true });

function buildView() {
  const base: ViewItem[] = props.form.questions.map((q) => ({
    q,
    options: props.form.shuffleOptions ? shuffled(q.options ?? []).map((o) => ({ id: o.id, text: o.text })) : (q.options ?? []).map((o) => ({ id: o.id, text: o.text })),
  }));
  view.value = props.form.shuffle && !isPoll.value ? shuffled(base) : base;
}

function initSession() {
  if (props.persistSession && props.form.id != null) {
    const s = store.getSession(props.form.id);
    if (s && store.hasActiveSession(props.form)) {
      buildView();
      Object.assign(answers, s.answers);
      startTs.value = s.startTs;
      page.value = s.page;
      startInterval();
      return;
    }
  }
  page.value = 0;
}
initSession();

// ── Старт / навигация ─────────────────────────────────────────────────────────
function start() {
  if (!props.form.questions.length) return;
  for (const k in answers) delete answers[k];
  lockedIds.value = new Set();
  revealing.value = false;
  resultsView.value = null;
  timeUp.value = false;
  buildView();
  page.value = 1;
  startTs.value = Date.now();
  startInterval();
  saveSessionNow();
}
function lockCurrentIfNeeded() {
  const q = currentQuestion.value;
  if (q && !props.form.allowChangeAnswer) lockedIds.value.add(q.id);
}
function advance() { if (page.value < total.value - 1) { lockCurrentIfNeeded(); page.value++; } }
function prev() { if (canGoBack.value) page.value--; }

// показ правильных «сразу»: 0.7с подсветки перед переходом
function withReveal(action: () => void) {
  if (timeUp.value || !shouldRevealImmediate.value || !currentQuestion.value || !hasCorrect(currentQuestion.value) || revealing.value) {
    action();
    return;
  }
  revealing.value = true;
  setTimeout(() => { revealing.value = false; action(); }, 700);
}
// «Ответить» — подтверждает ответ (с подсветкой правильных) и ведёт дальше
function onAnswer() { withReveal(advance); }

// ── Завершение ────────────────────────────────────────────────────────────────
const finishOpen = ref(false);
const unansweredOpen = ref(false);
const reviewActive = ref(false); // прогон по неотвеченным
const resultsView = ref<null | 'review' | 'poll' | 'score'>(null); // финальные экраны
const pollResults = ref<Record<string, { count: number; percent: number }>>({});
let lastDuration = 0;

const unanswered = computed(() => view.value.map((it, i) => ({ q: it.q, page: i + 1 })).filter((x) => !isAnswered(x.q)));
const hasRequiredUnanswered = computed(() => unanswered.value.some((x) => x.q.required));
const nextUnansweredPage = computed(() => unanswered.value.find((x) => x.page > page.value)?.page ?? null);
const unansweredLabel = computed(() => (unanswered.value.length === 1 ? 'Перейти к вопросу' : 'Перейти к вопросам'));
function plural(n: number, one: string, few: string, many: string): string {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
}

// Подтверждение нужно только если ответы можно было менять
// (тест/опрос — «изменять ответ»; голосование — когда нельзя переголосовать)
const needsFinishConfirm = computed(() => (isPoll.value ? !props.form.allowRevote : props.form.allowChangeAnswer));

function onFinishClick() { withReveal(attemptFinish); }
function attemptFinish() {
  if (unanswered.value.length > 0) { unansweredOpen.value = true; return; }
  if (needsFinishConfirm.value) finishOpen.value = true;
  else finalize();
}
function goToUnanswered() {
  unansweredOpen.value = false;
  reviewActive.value = true;
  const first = unanswered.value[0];
  if (first) page.value = first.page;
}

// Результат теста (по правильным ответам)
const scoreInfo = computed(() => {
  let scorable = 0, correct = 0;
  for (const it of view.value) if (hasCorrect(it.q)) { scorable++; if (answeredCorrectly(it)) correct++; }
  return { scorable, correct, percent: scorable > 0 ? Math.round((correct / scorable) * 100) : 0 };
});
const passed = computed(() => (props.form.usePassingScore ? scoreInfo.value.percent >= props.form.passingScore : null));

async function finalize() {
  stopTimer();
  finishOpen.value = false;
  unansweredOpen.value = false;
  lastDuration = startTs.value ? Math.floor((Date.now() - startTs.value) / 1000) : 0;
  if (props.persistSession && props.form.id != null) store.clearSession(props.form.id);

  let recorded = false;
  try {
    if (props.submit) { await props.submit({ ...answers }, lastDuration); recorded = true; }
    else if (props.record && props.form.id != null) { await store.submitAttempt(props.form.id, { ...answers }, lastDuration); recorded = true; }
  } catch { recorded = false; /* напр. лимит попыток */ }

  // Живые результаты голосования
  if (isPoll.value && props.form.liveResults && recorded) {
    try {
      const stats = await store.loadStats(props.form.id!);
      const q = (stats?.questions ?? []).find((x: any) => x.id === currentQuestion.value?.id) ?? stats?.questions?.[0];
      const map: Record<string, { count: number; percent: number }> = {};
      for (const o of (q?.options ?? [])) if (o.id) map[o.id] = { count: o.count, percent: o.percent };
      pollResults.value = map;
      resultsView.value = 'poll';
      return;
    } catch { /* без живых — просто завершаем */ }
  }

  // Результат теста в конце (если показ результата включён)
  if (isTest.value && props.form.showResult !== 'never' && scoreInfo.value.scorable > 0) {
    resultsView.value = (props.form.showCorrectAnswers && props.form.showResult === 'after') ? 'review' : 'score';
    return;
  }

  emitFinish();
}
function emitFinish() {
  emit('finish', { answers: { ...answers }, durationSec: lastDuration });
  page.value = 0;
  resultsView.value = null;
  reviewActive.value = false;
}

// форматирование для экрана разбора
function optText(it: ViewItem, id: string): string {
  return it.options.find((o) => o.id === id)?.text || '—';
}
function userAnswerText(it: ViewItem): string {
  const q = it.q; const v = answers[q.id];
  if (v == null || v === '' || (Array.isArray(v) && !v.length)) return '— нет ответа';
  if (q.type === 'single' || q.type === 'dropdown') return optText(it, v as string);
  if (q.type === 'multiple') return (v as string[]).map((id) => optText(it, id)).join(', ');
  if (q.type === 'yesno') return v === 'yes' ? 'Да' : 'Нет';
  return String(v);
}
function correctAnswerText(it: ViewItem): string {
  const q = it.q; const c = q.correct;
  if (c == null || c === '') return '—';
  if (q.type === 'single' || q.type === 'dropdown') return optText(it, c as string);
  if (q.type === 'multiple') return (c as string[]).map((id) => optText(it, id)).join(', ');
  if (q.type === 'yesno') return c === 'yes' ? 'Да' : 'Нет';
  return String(c);
}
function answeredCorrectly(it: ViewItem): boolean {
  const q = it.q;
  if (!hasCorrect(q)) return false;
  const v = answers[q.id];
  if (q.type === 'multiple') {
    const a = Array.isArray(v) ? [...(v as string[])].sort() : [];
    const c = Array.isArray(q.correct) ? [...q.correct].sort() : [];
    return a.length === c.length && a.every((x, i) => x === c[i]);
  }
  if (q.type === 'text' || q.type === 'textarea' || q.type === 'date') {
    return String(v ?? '').trim().toLowerCase() === String(q.correct).trim().toLowerCase();
  }
  return String(v ?? '') === String(q.correct);
}

function pollPercent(id: string): number { return pollResults.value[id]?.percent ?? 0; }
</script>

<template>
  <div class="flex flex-col h-full min-h-0">
    <p v-if="previewHint" class="text-xs text-dimmed mb-2 flex items-center justify-center gap-1.5 shrink-0">
      <UIcon name="i-lucide-eye" class="size-3.5" />
      Так тест увидит сотрудник
    </p>

    <div class="flex-1 min-h-0 w-full grid place-items-center [container-type:size]">
      <div class="aspect-[16/10] w-[min(100%,160cqh)] max-w-4xl rounded-2xl ring-1 ring-default bg-default shadow-2xl overflow-hidden flex flex-col">

        <div v-if="form.showProgress && page > 0 && !resultsView" class="h-1 bg-elevated shrink-0">
          <div class="h-full bg-primary transition-all" :style="{ width: `${(page / Math.max(view.length, 1)) * 100}%` }" />
        </div>

        <div class="flex-1 min-h-0 overflow-hidden">
          <!-- Титульник -->
          <div v-if="page === 0 && !resultsView" class="h-full flex flex-col items-center justify-center text-center gap-4 px-10 py-6">
            <UBadge color="primary" variant="subtle" size="lg">{{ kindLabels[form.kind] }}</UBadge>
            <h2 class="text-3xl font-semibold text-highlighted leading-tight line-clamp-2">{{ form.title || 'Без названия' }}</h2>
            <p v-if="form.description" class="text-muted max-w-xl line-clamp-4 whitespace-pre-line">{{ form.description }}</p>
            <div class="flex flex-wrap gap-1.5 justify-center">
              <UBadge v-if="form.anonymous" color="neutral" variant="subtle">Анонимно</UBadge>
              <UBadge v-if="form.useTimeLimit && form.timeLimit" color="neutral" variant="subtle">⏱ {{ form.timeLimit }}</UBadge>
              <UBadge color="neutral" variant="subtle">{{ isPoll ? 'Кандидатов' : 'Вопросов' }}: {{ isPoll ? (form.questions[0]?.options.length ?? 0) : form.questions.length }}</UBadge>
            </div>
            <UButton size="xl" class="mt-2" trailing-icon="i-lucide-arrow-right" :disabled="!form.questions.length" @click="start">{{ ctaLabel }}</UButton>
            <p v-if="!form.questions.length" class="text-xs text-dimmed">В форме нет вопросов.</p>
          </div>

          <!-- Финал: разбор теста -->
          <div v-else-if="resultsView === 'review'" class="h-full flex flex-col px-8 py-6 gap-3">
            <div class="flex items-center justify-between gap-3 shrink-0">
              <p class="text-lg font-semibold text-highlighted">Ваши ответы</p>
              <UBadge :color="passed === false ? 'error' : 'success'" variant="subtle" size="lg" class="tabular-nums">{{ scoreInfo.correct }}/{{ scoreInfo.scorable }} · {{ scoreInfo.percent }}%</UBadge>
            </div>
            <div class="flex-1 min-h-0 overflow-y-auto flex flex-col gap-3 px-1 py-1">
              <div v-for="(it, i) in view" :key="it.q.id" class="rounded-xl ring-1 p-3 flex flex-col gap-1" :class="answeredCorrectly(it) ? 'ring-green-500/40 bg-green-500/5' : 'ring-red-500/40 bg-red-500/5'">
                <div class="flex items-center gap-2">
                  <UIcon :name="answeredCorrectly(it) ? 'i-lucide-check-circle-2' : 'i-lucide-x-circle'" :class="answeredCorrectly(it) ? 'text-success' : 'text-error'" class="size-4 shrink-0" />
                  <span class="text-sm text-highlighted">{{ i + 1 }}. {{ it.q.title || 'Без названия' }}</span>
                </div>
                <p class="text-sm text-muted pl-6">Ваш ответ: {{ userAnswerText(it) }}</p>
                <p v-if="!answeredCorrectly(it) && hasCorrect(it.q)" class="text-sm text-success pl-6">Правильно: {{ correctAnswerText(it) }}</p>
              </div>
            </div>
          </div>

          <!-- Финал: результат теста (процент) -->
          <div v-else-if="resultsView === 'score'" class="h-full flex flex-col items-center justify-center text-center gap-4 px-10">
            <div class="size-24 rounded-full flex items-center justify-center ring-4" :class="passed === false ? 'ring-error/30 bg-error/10' : 'ring-success/30 bg-success/10'">
              <span class="text-3xl font-bold tabular-nums" :class="passed === false ? 'text-error' : 'text-success'">{{ scoreInfo.percent }}%</span>
            </div>
            <p class="text-lg font-semibold text-highlighted">Правильных ответов: {{ scoreInfo.correct }} из {{ scoreInfo.scorable }}</p>
            <UBadge v-if="passed !== null" :color="passed ? 'success' : 'error'" variant="subtle" size="lg">{{ passed ? 'Тест пройден' : 'Тест не пройден' }} · порог {{ form.passingScore }}%</UBadge>
          </div>

          <!-- Страница вопроса -->
          <div v-else-if="currentQuestion" class="h-full flex flex-col px-10 py-6">
            <div class="flex items-center justify-between gap-3 shrink-0">
              <p class="text-sm text-dimmed tabular-nums">{{ isPoll ? 'Голосование' : `Вопрос ${page} из ${view.length}` }}</p>
              <div class="flex items-center gap-1.5 text-sm tabular-nums" :class="timeUp || timeLow ? 'text-error font-medium' : 'text-dimmed'">
                <UIcon name="i-lucide-timer" class="size-4" />
                {{ timerLabel }}
              </div>
            </div>

            <div class="flex-1 min-h-0 flex flex-col justify-center gap-5 overflow-y-auto px-1 py-1">
              <div class="flex flex-col gap-2">
                <h3 class="text-2xl font-medium text-highlighted">
                  {{ currentQuestion.title || 'Без названия' }}
                  <span v-if="currentQuestion.required" class="text-error">*</span>
                </h3>
                <p v-if="currentQuestion.hint" class="text-muted">{{ currentQuestion.hint }}</p>
                <p v-if="currentLocked && !revealing && resultsView === null" class="text-xs text-dimmed">Ответ зафиксирован — изменять нельзя.</p>
              </div>

              <!-- Варианты: single / multiple / poll — кликабельные поля -->
              <div v-if="currentQuestion.type === 'single' || currentQuestion.type === 'multiple'" class="flex flex-col gap-2 w-full p-px" :class="isPoll ? 'max-w-xl' : 'max-w-lg'">
                <button
                  v-for="opt in currentOptions"
                  :key="opt.id"
                  type="button"
                  :disabled="currentLocked"
                  class="relative overflow-hidden flex items-center gap-3 w-full text-left rounded-xl ring-1 transition-colors disabled:cursor-default"
                  :class="[optionRowClass(opt.id, true), isPoll ? 'px-4 py-3.5' : 'px-4 py-2.5']"
                  @click="onPick(opt.id)"
                >
                  <span v-if="isPoll && resultsView === 'poll'" class="absolute inset-y-0 left-0 bg-primary/15 transition-all" :style="{ width: `${pollPercent(opt.id)}%` }" />
                  <span class="relative z-10 flex items-center gap-3 w-full">
                    <span class="shrink-0 inline-flex items-center justify-center size-5" :class="currentQuestion.type === 'multiple' ? 'rounded' : 'rounded-full'">
                      <span class="size-5 ring-1 ring-current flex items-center justify-center" :class="[currentQuestion.type === 'multiple' ? 'rounded' : 'rounded-full', isSelected(opt.id) ? 'text-primary' : 'text-dimmed']">
                        <span v-if="isSelected(opt.id)" class="size-2.5 bg-primary" :class="currentQuestion.type === 'multiple' ? 'rounded-sm' : 'rounded-full'" />
                      </span>
                    </span>
                    <span class="flex-1">{{ opt.text }}</span>
                    <span v-if="isPoll && resultsView === 'poll'" class="text-sm tabular-nums text-muted shrink-0">{{ pollPercent(opt.id) }}%</span>
                    <UIcon v-else-if="revealing && isCorrectOption(currentQuestion, opt.id)" name="i-lucide-check" class="size-4 text-success shrink-0" />
                    <UIcon v-else-if="revealing && isSelected(opt.id)" name="i-lucide-x" class="size-4 text-error shrink-0" />
                  </span>
                </button>
              </div>

              <!-- Выпадающий список -->
              <div v-else-if="currentQuestion.type === 'dropdown'" class="w-full max-w-sm flex flex-col gap-1.5">
                <USelect v-model="answers[currentQuestion.id]" :items="currentOptions" value-key="id" label-key="text" :disabled="currentLocked" size="lg" class="w-full" placeholder="Выберите вариант" />
                <p v-if="revealing && hasCorrect(currentQuestion)" class="text-xs text-success">Правильно: {{ correctAnswerText({ q: currentQuestion, options: currentOptions }) }}</p>
              </div>

              <!-- Да / Нет -->
              <div v-else-if="currentQuestion.type === 'yesno'" class="flex gap-3">
                <button
                  v-for="it in [{ id: 'yes', text: 'Да' }, { id: 'no', text: 'Нет' }]"
                  :key="it.id"
                  type="button"
                  :disabled="currentLocked"
                  class="rounded-xl ring-1 px-6 py-2.5 text-sm transition-colors disabled:cursor-default"
                  :class="optionRowClass(it.id)"
                  @click="onPick(it.id)"
                >
                  {{ it.text }}
                </button>
              </div>

              <!-- Шкала -->
              <div v-else-if="currentQuestion.type === 'scale'" class="flex flex-col gap-2 max-w-lg">
                <div class="flex flex-wrap gap-2">
                  <button
                    v-for="n in scaleNumbers(currentQuestion)"
                    :key="n"
                    type="button"
                    :disabled="currentLocked"
                    class="size-10 rounded-lg ring-1 text-sm font-medium transition-colors disabled:cursor-default"
                    :class="revealing && hasCorrect(currentQuestion)
                      ? (String(currentQuestion.correct) === String(n) ? 'ring-green-500 bg-green-500/10 text-green-600 dark:text-green-400' : (answers[currentQuestion.id] === n ? 'ring-red-500 bg-red-500/10 text-red-600' : 'ring-default text-muted'))
                      : (answers[currentQuestion.id] === n ? 'bg-primary text-inverted ring-primary' : 'ring-default text-muted hover:bg-elevated')"
                    @click="!currentLocked && (answers[currentQuestion.id] = n)"
                  >
                    {{ n }}
                  </button>
                </div>
                <div v-if="currentQuestion.scaleMinLabel || currentQuestion.scaleMaxLabel" class="flex justify-between text-xs text-dimmed">
                  <span>{{ currentQuestion.scaleMinLabel }}</span>
                  <span>{{ currentQuestion.scaleMaxLabel }}</span>
                </div>
              </div>

              <!-- Свободный ответ -->
              <div v-else class="w-full max-w-lg flex flex-col gap-1.5">
                <UInput v-if="currentQuestion.type === 'text'" v-model="answers[currentQuestion.id]" :disabled="currentLocked" size="lg" class="w-full" placeholder="Короткий ответ" />
                <UTextarea v-else-if="currentQuestion.type === 'textarea'" v-model="answers[currentQuestion.id]" :disabled="currentLocked" :rows="3" size="lg" class="w-full" placeholder="Развёрнутый ответ" />
                <UInput v-else-if="currentQuestion.type === 'number'" v-model="answers[currentQuestion.id]" :disabled="currentLocked" type="number" size="lg" class="w-44" placeholder="0" />
                <UInput v-else-if="currentQuestion.type === 'date'" v-model="answers[currentQuestion.id]" :disabled="currentLocked" type="date" size="lg" class="w-52" />
                <p v-if="revealing && hasCorrect(currentQuestion)" class="text-xs text-success">Правильно: {{ correctAnswerText({ q: currentQuestion, options: currentOptions }) }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Тайм-аут -->
        <div v-if="page > 0 && timeUp && !resultsView" class="shrink-0 px-5 pt-2 text-center text-xs text-error font-medium">
          Время вышло — ответ на текущий вопрос можно дать, затем форма закроется
        </div>

        <!-- Пейджер финального экрана -->
        <div v-if="resultsView" class="shrink-0 border-t border-default px-5 py-3 flex items-center justify-end">
          <UButton color="primary" :icon="finishIcon" @click="emitFinish">Закрыть</UButton>
        </div>

        <!-- Пейджер вопроса -->
        <div v-else-if="page > 0" class="shrink-0">
          <!-- Навигация над полосой (тусклее, только для перехода) -->
          <div class="px-5 pb-2 flex items-center justify-between gap-3 min-h-9">
            <UButton v-if="canGoBack" color="neutral" variant="ghost" size="md" class="text-dimmed hover:text-default" leading-icon="i-lucide-arrow-left" @click="prev">Назад</UButton>
            <span v-else />
            <UButton v-if="!timeUp && reviewActive && nextUnansweredPage" color="neutral" variant="ghost" size="md" class="text-dimmed hover:text-default" :disabled="revealing" trailing-icon="i-lucide-arrow-right" @click="page = nextUnansweredPage">Далее</UButton>
            <UButton v-else-if="!timeUp && !isLastPage && !reviewActive" color="neutral" variant="ghost" size="md" class="text-dimmed hover:text-default" :disabled="revealing" trailing-icon="i-lucide-arrow-right" @click="advance">Далее</UButton>
            <span v-else />
          </div>
          <!-- Полоса + счётчик + основная кнопка -->
          <div class="border-t border-default px-5 py-3 flex items-center justify-between gap-3">
            <span class="text-xs text-dimmed tabular-nums">{{ page }} / {{ view.length }}</span>
            <UButton v-if="timeUp" color="primary" size="md" :icon="finishIcon" @click="finalize">Завершить</UButton>
            <UButton v-else-if="isLastPage || reviewActive" color="primary" size="md" :icon="finishIcon" :disabled="revealing" @click="onFinishClick">{{ finishLabel }}</UButton>
            <UButton v-else color="primary" size="md" icon="i-lucide-check" :disabled="!currentQuestion || !isAnswered(currentQuestion) || revealing" @click="onAnswer">Ответить</UButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Подтверждение завершения -->
    <UModal v-model:open="finishOpen" :title="`${finishLabel}?`" description="">
      <template #body>
        <p class="text-default">{{ isPoll ? 'Вы уверены, что хотите отдать голос? После отправки изменить его будет нельзя.' : 'Вы уверены, что хотите завершить и отправить ответы? После отправки изменить их будет нельзя.' }}</p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" @click="finishOpen = false">Отмена</UButton>
          <UButton color="primary" :icon="finishIcon" @click="finalize">{{ finishLabel }}</UButton>
        </div>
      </template>
    </UModal>

    <!-- Неотвеченные -->
    <UModal v-model:open="unansweredOpen" title="Остались неотвеченные вопросы" description="">
      <template #body>
        <p class="text-default">
          Вы не ответили на {{ unanswered.length }} {{ plural(unanswered.length, 'вопрос', 'вопроса', 'вопросов') }}.
          <template v-if="hasRequiredUnanswered"> Среди них есть обязательные — без ответа на них завершить нельзя.</template>
          <template v-else> Можно вернуться к ним или завершить как есть.</template>
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="primary" icon="i-lucide-list-checks" @click="goToUnanswered">{{ unansweredLabel }}</UButton>
          <UButton v-if="!hasRequiredUnanswered" color="neutral" variant="outline" :icon="finishIcon" @click="finalize">{{ finishLabel }}</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
