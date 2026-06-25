<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useUsersData } from '../../composables/useUsersData';
import { useAppToast } from '../../composables/useAppToast';
import OfoMultiSelect from '../../components/OfoMultiSelect.vue';
import ModalTextField from './ModalTextField.vue';
import QuestionsBuilder from './QuestionsBuilder.vue';
import { questionTypeLabel, type Question } from './questionTypes';

const { toast } = useAppToast();

// ── Разделы конструктора ──────────────────────────────────────────────────────
const section = ref<'new' | 'drafts'>('new');
const sectionItems = [
  { label: 'Создать новый', value: 'new', icon: 'i-lucide-plus' },
  { label: 'Черновики', value: 'drafts', icon: 'i-lucide-file-clock' },
];

const todayISO = new Date().toISOString().slice(0, 10); // YYYY-MM-DD

// ── Данные формы нового опроса ────────────────────────────────────────────────
const form = reactive({
  title: '',
  description: '',
  kind: 'test' as 'test' | 'survey' | 'poll',
  visibility: 'public' as 'public' | 'private',
  recipients: [] as number[],
  // ── Параметры ──
  shuffle: false,
  shuffleOptions: false,
  showProgress: false,
  freeNavigation: false,
  anonymous: false,
  usePassingScore: false,
  passingScore: 70,
  showCorrectAnswers: false,
  allowChangeAnswer: false,
  liveResults: false,
  allowRevote: false,
  completionMessage: '',
  notifyAdmin: false,
  // ── Доступ (ограничение по ОФО) ──
  restrictByOfo: false,
  ofoIds: [] as number[],
  questions: [] as Question[],
  // ── Ограничения ──
  useTimeLimit: false,
  timeLimit: '',
  limitAttempts: false,
  attempts: 1,
  useStart: true,
  startsAt: todayISO,
  useEnd: false,
  endsAt: '',
  showResult: 'after' as 'immediate' | 'after' | 'never',
  accessByLink: false,
});

const kindItems = [
  { label: 'Тест', value: 'test' },
  { label: 'Опрос', value: 'survey' },
  { label: 'Голосование', value: 'poll' },
];
const visibilityItems = [
  { label: 'Публичный', value: 'public' },
  { label: 'Приватный', value: 'private' },
];
const showResultItems = [
  { label: 'Сразу', value: 'immediate' },
  { label: 'После прохождения', value: 'after' },
  { label: 'Не показывать результат', value: 'never' },
];
const labelOf = (items: { label: string; value: string }[], v: string) =>
  items.find((i) => i.value === v)?.label ?? v;

// ── Сотрудники (получатели приватного теста) ─────────────────────────────────
const { users, ensureLoaded: ensureUsersLoaded } = useUsersData();
ensureUsersLoaded();
const recipientItems = computed(() =>
  users.value
    .map((u) => ({ label: u.fullName, value: u.id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru')),
);

// ── Анонимность по типу ──
const anonymousLocked = computed(() => form.kind !== 'survey');
watch(
  () => form.kind,
  (k) => {
    if (k === 'test') form.anonymous = false;
    else if (k === 'poll') form.anonymous = true;
  },
  { immediate: true },
);

// ОФО-ограничение доступно только при приватности
watch(
  () => form.visibility,
  (v) => { if (v !== 'private') form.restrictByOfo = false; },
);

// «Показывать правильные ответы» — нельзя при «не показывать результат»
watch(
  () => form.showResult,
  (r) => { if (r === 'never') form.showCorrectAnswers = false; },
);
function onToggleShowCorrect(value: boolean) {
  if (value && form.showResult === 'never') {
    toast.add({
      title: 'Сначала включите показ результата',
      description: 'В «Показать результат» выберите «Сразу» или «После прохождения».',
      color: 'warning',
      icon: 'i-lucide-info',
    });
    return;
  }
  form.showCorrectAnswers = value;
}

// Подсветка «Доступ по ссылке»: приватный без получателей
const suggestLink = computed(
  () => form.visibility === 'private' && form.recipients.length === 0 && !form.accessByLink,
);

// ── Шаги мастера ──────────────────────────────────────────────────────────────
const STEPS = [
  { key: 'settings', label: 'Настройка' },
  { key: 'questions', label: 'Вопросы' },
  { key: 'preview', label: 'Предпросмотр' },
] as const;
type StepKey = (typeof STEPS)[number]['key'];
const step = ref<StepKey>('settings');
const stepIdx = computed(() => STEPS.findIndex((s) => s.key === step.value));
function goStep(i: number) { step.value = STEPS[i].key; }
function nextStep() { if (stepIdx.value < STEPS.length - 1) step.value = STEPS[stepIdx.value + 1].key; }
function prevStep() { if (stepIdx.value > 0) step.value = STEPS[stepIdx.value - 1].key; }

const recipientNames = computed(() =>
  form.recipients.map((id) => users.value.find((u) => u.id === id)?.fullName).filter(Boolean) as string[],
);

// ── Предпросмотр: постраничный просмотр (титул + по вопросу на страницу) ──────
const previewPage = ref(0); // 0 = титульник, 1..N = вопросы
const previewTotal = computed(() => form.questions.length + 1);
const ctaLabel = computed(() =>
  form.kind === 'poll' ? 'Участвовать в голосовании' : form.kind === 'survey' ? 'Пройти опрос' : 'Пройти тест',
);
const currentPreviewQuestion = computed(() => form.questions[previewPage.value - 1]);
function previewNext() { if (previewPage.value < previewTotal.value - 1) previewPage.value++; }
function previewPrev() { if (previewPage.value > 0) previewPage.value--; }
watch(step, (s) => { if (s === 'preview') previewPage.value = 0; });
watch(previewTotal, (t) => { if (previewPage.value > t - 1) previewPage.value = t - 1; });

const isLastPreviewPage = computed(() => previewPage.value === form.questions.length && form.questions.length > 0);
const finishOpen = ref(false);
function confirmFinish() {
  finishOpen.value = false;
  previewPage.value = 0; // в предпросмотре возвращаемся на титул
  toast.add({ title: 'Это предпросмотр', description: 'В рабочем тесте здесь ответы отправятся и зачтутся.', color: 'info', icon: 'i-lucide-info' });
}

function saveDraft() {
  toast.add({ title: 'Черновик', description: 'Сохранение черновика будет позже (нужен бэкенд).', color: 'info', icon: 'i-lucide-file-clock' });
}
function publish() {
  toast.add({ title: 'Публикация', description: 'Публикация теста будет позже (нужен бэкенд).', color: 'info', icon: 'i-lucide-send' });
}
</script>

<template>
  <div class="flex flex-col flex-1 min-h-0 w-full gap-3 pt-1">
    <!-- Тулбар: разделы + шаги в одну строку -->
    <div class="flex items-center gap-3 flex-wrap">
      <UTabs v-model="section" :items="sectionItems" size="md" class="w-fit" />

      <template v-if="section === 'new'">
        <USeparator orientation="vertical" class="h-7 hidden md:block" />
        <div class="flex items-center gap-1.5 text-sm flex-wrap">
          <template v-for="(s, i) in STEPS" :key="s.key">
            <button
              type="button"
              class="px-3 py-1.5 rounded-lg transition-colors"
              :class="step === s.key ? 'bg-primary/10 text-primary ring-1 ring-primary/40' : 'text-muted hover:bg-elevated'"
              @click="goStep(i)"
            >
              {{ i + 1 }}. {{ s.label }}
            </button>
            <UIcon v-if="i < STEPS.length - 1" name="i-lucide-chevron-right" class="size-4 text-dimmed" />
          </template>
        </div>
      </template>
    </div>

    <!-- Создать новый: мастер по шагам -->
    <div v-if="section === 'new'" class="flex flex-col flex-1 min-h-0 gap-4">
      <!-- Прокручиваемая область текущего шага -->
      <div class="flex-1 min-h-0 overflow-y-auto px-0.5 pt-0.5">
      <!-- ШАГ 1: Настройка -->
      <div v-show="step === 'settings'" class="flex flex-col lg:flex-row gap-4 items-start">
        <!-- Основные поля -->
        <div class="flex-1 min-w-0 flex flex-col gap-3 w-full">
          <UFormField label="Название" name="title">
            <UInput v-model="form.title" :maxlength="150" size="lg" class="w-full" placeholder="Название (до 150 символов)" />
            <template #hint><span class="text-xs text-muted tabular-nums">{{ form.title.length }}/150</span></template>
          </UFormField>

          <UFormField label="Описание" name="description">
            <ModalTextField v-model="form.description" label="Описание" :maxlength="500" placeholder="Описание (до 500 символов)" />
            <template #hint><span class="text-xs text-muted tabular-nums">{{ form.description.length }}/500</span></template>
          </UFormField>

          <div class="flex flex-col sm:flex-row gap-3">
            <UFormField label="Тип" name="kind" class="flex-1">
              <USelect v-model="form.kind" :items="kindItems" size="lg" class="w-full" />
            </UFormField>
            <UFormField label="Доступ" name="visibility" class="flex-1">
              <USelect v-model="form.visibility" :items="visibilityItems" size="lg" class="w-full" />
            </UFormField>
          </div>

          <!-- Получатели приватного теста -->
          <UFormField v-if="form.visibility === 'private'" label="Первоначальные получатели" name="recipients">
            <USelectMenu
              v-model="form.recipients"
              :items="recipientItems"
              value-key="value"
              label-key="label"
              multiple
              searchable
              size="xl"
              class="w-full"
              placeholder="Выбрать первоначальных получателей"
              :content="{ align: 'start', sideOffset: 8 }"
            />
            <template #hint><span class="text-xs text-muted">Можно добавить ещё людей после создания теста</span></template>
          </UFormField>

          <!-- Ограничение по ОФО (только при приватности) -->
          <div v-if="form.visibility === 'private'" class="flex flex-col gap-2">
            <div class="flex items-center justify-between gap-3 flex-wrap">
              <UCheckbox v-model="form.restrictByOfo" label="Ограничить по ОФО" />
              <span v-if="form.restrictByOfo" class="text-xs text-muted text-right">
                Выбор родителя отмечает все дочерние ОФО. Доступ получают сотрудники выбранных ОФО и/или первоначальные получатели.
              </span>
            </div>
            <OfoMultiSelect v-if="form.restrictByOfo" v-model="form.ofoIds" />
          </div>

          <!-- Параметры -->
          <div class="flex flex-col gap-2">
            <h4 class="text-sm font-medium text-highlighted">Параметры</h4>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-2 items-start">
              <UCheckbox v-model="form.shuffle" label="Перемешивать вопросы" />
              <UCheckbox v-model="form.shuffleOptions" label="Перемешивать варианты ответов" />
              <UCheckbox v-model="form.showProgress" label="Показывать прогресс-бар" />
              <UCheckbox v-model="form.freeNavigation" label="Свободный переход между вопросами" />
              <UCheckbox v-model="form.anonymous" :disabled="anonymousLocked" label="Анонимные ответы" />
              <UCheckbox
                v-if="form.kind === 'test'"
                :model-value="form.showCorrectAnswers"
                :class="form.showResult === 'never' ? 'opacity-60' : ''"
                label="Показывать правильные ответы"
                @update:model-value="onToggleShowCorrect"
              />
              <UCheckbox v-if="form.kind === 'test' || form.kind === 'survey'" v-model="form.allowChangeAnswer" label="Разрешить изменять ответ до завершения" />
              <UCheckbox v-if="form.kind === 'poll'" v-model="form.liveResults" label="Промежуточные результаты в реальном времени" />
              <UCheckbox v-if="form.kind === 'poll'" v-model="form.allowRevote" label="Разрешить переголосовать до закрытия" />
            </div>

            <USeparator class="my-0.5" />

            <UFormField label="Сообщение после завершения" name="completionMessage">
              <ModalTextField v-model="form.completionMessage" label="Сообщение после завершения" :maxlength="500" placeholder="Например: Спасибо за прохождение!" />
            </UFormField>
            <UCheckbox v-model="form.notifyAdmin" label="Уведомлять админа о новом прохождении" />
          </div>
        </div>

        <!-- Зона ограничений -->
        <div class="w-full lg:w-80 shrink-0 min-w-0 rounded-xl ring-1 ring-default bg-elevated/40 p-4 flex flex-col gap-4">
          <div class="flex items-center gap-2 text-highlighted font-medium">
            <UIcon name="i-lucide-sliders-horizontal" class="size-5 text-primary" />
            Ограничения
          </div>

          <div class="flex flex-col gap-2 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <UCheckbox v-model="form.useTimeLimit" label="Время для прохождения" />
              <span class="text-xs text-dimmed">чч:мм:сс</span>
            </div>
            <UInput v-model="form.timeLimit" type="time" :step="1" size="lg" class="w-full min-w-0 transition-opacity" :class="form.useTimeLimit ? '' : 'opacity-50'" :disabled="!form.useTimeLimit" />
          </div>

          <div class="flex flex-col gap-2 min-w-0">
            <UCheckbox v-model="form.limitAttempts" label="Ограничить попытки" />
            <UInput v-model.number="form.attempts" type="number" :min="1" size="lg" class="w-full min-w-0 transition-opacity" :class="form.limitAttempts ? '' : 'opacity-50'" :disabled="!form.limitAttempts" placeholder="Количество попыток" />
          </div>

          <div v-if="form.kind === 'test'" class="flex flex-col gap-2 min-w-0">
            <UCheckbox v-model="form.usePassingScore" label="Проходной балл" />
            <div class="flex items-center gap-2">
              <UInput
                v-model.number="form.passingScore"
                type="number" :min="0" :max="100" size="lg"
                class="flex-1 min-w-0 transition-opacity"
                :class="form.usePassingScore ? '' : 'opacity-50'"
                :disabled="!form.usePassingScore"
              />
              <span class="text-sm text-muted shrink-0">%</span>
            </div>
          </div>

          <div class="flex flex-col gap-2 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <UCheckbox v-model="form.useStart" label="Начинается" />
              <span class="text-xs text-dimmed">дд:мм:гггг</span>
            </div>
            <UInput v-model="form.startsAt" type="date" size="lg" class="w-full min-w-0 transition-opacity" :class="form.useStart ? '' : 'opacity-50'" :disabled="!form.useStart" />
          </div>

          <div class="flex flex-col gap-2 min-w-0">
            <div class="flex items-center justify-between gap-2">
              <UCheckbox v-model="form.useEnd" label="Завершается" />
              <span class="text-xs text-dimmed">дд:мм:гггг</span>
            </div>
            <UInput v-model="form.endsAt" type="date" size="lg" class="w-full min-w-0 transition-opacity" :class="form.useEnd ? '' : 'opacity-50'" :disabled="!form.useEnd" />
          </div>

          <div class="flex flex-col gap-1.5 min-w-0 transition-colors" :class="suggestLink ? 'rounded-lg ring-1 ring-primary/50 bg-primary/5 p-2' : ''">
            <UCheckbox v-model="form.accessByLink" label="Доступ по ссылке" />
            <p v-if="suggestLink" class="text-xs text-primary">
              Приватный тест без выбранных получателей — включите доступ по ссылке, иначе его никто не откроет.
            </p>
          </div>

          <UFormField label="Показать результат" name="showResult">
            <USelect v-model="form.showResult" :items="showResultItems" size="lg" class="w-full min-w-0" />
          </UFormField>
        </div>
      </div>

      <!-- ШАГ 2: Вопросы -->
      <div v-show="step === 'questions'">
        <QuestionsBuilder v-model="form.questions" />
      </div>

      <!-- ШАГ 3: Предпросмотр (слайдер 16:10, одна страница = один вопрос) -->
      <div v-show="step === 'preview'" class="flex flex-col h-full min-h-0">
        <p class="text-xs text-dimmed mb-2 flex items-center gap-1.5 shrink-0">
          <UIcon name="i-lucide-eye" class="size-3.5" />
          Так тест увидит сотрудник
        </p>

        <!-- «Сцена»: вписываем окно 16:10 по высоте и ширине, без скролла -->
        <div class="flex-1 min-h-0 w-full grid place-items-center [container-type:size]">
          <div class="aspect-[16/10] w-[min(100%,160cqh)] max-w-4xl rounded-2xl ring-1 ring-default bg-default shadow-2xl overflow-hidden flex flex-col">

            <!-- Прогресс-бар (на страницах вопросов, если включён) -->
            <div v-if="form.showProgress && previewPage > 0" class="h-1 bg-elevated shrink-0">
              <div class="h-full bg-primary transition-all" :style="{ width: `${(previewPage / Math.max(form.questions.length, 1)) * 100}%` }" />
            </div>

            <!-- Область страницы -->
            <div class="flex-1 min-h-0 overflow-hidden">
              <!-- Страница 0: титульник -->
              <div v-if="previewPage === 0" class="h-full flex flex-col items-center justify-center text-center gap-4 px-10 py-6">
                <UBadge color="primary" variant="subtle" size="lg">{{ labelOf(kindItems, form.kind) }}</UBadge>
                <h2 class="text-3xl font-semibold text-highlighted leading-tight line-clamp-2">{{ form.title || 'Без названия' }}</h2>
                <p v-if="form.description" class="text-muted max-w-xl line-clamp-4 whitespace-pre-line">{{ form.description }}</p>

                <div class="flex flex-wrap gap-1.5 justify-center">
                  <UBadge v-if="form.anonymous" color="neutral" variant="subtle">Анонимно</UBadge>
                  <UBadge v-if="form.useTimeLimit && form.timeLimit" color="neutral" variant="subtle">⏱ {{ form.timeLimit }}</UBadge>
                  <UBadge v-if="form.limitAttempts" color="neutral" variant="subtle">Попыток: {{ form.attempts }}</UBadge>
                  <UBadge v-if="form.usePassingScore && form.kind === 'test'" color="neutral" variant="subtle">Проходной: {{ form.passingScore }}%</UBadge>
                  <UBadge color="neutral" variant="subtle">Вопросов: {{ form.questions.length }}</UBadge>
                </div>

                <UButton
                  size="xl"
                  class="mt-2"
                  trailing-icon="i-lucide-arrow-right"
                  :disabled="!form.questions.length"
                  @click="previewNext"
                >
                  {{ ctaLabel }}
                </UButton>
                <p v-if="!form.questions.length" class="text-xs text-dimmed">Добавьте вопросы на шаге «Вопросы».</p>
              </div>

              <!-- Страница вопроса -->
              <div v-else-if="currentPreviewQuestion" class="h-full flex flex-col px-10 py-6">
                <p class="text-sm text-dimmed tabular-nums shrink-0">Вопрос {{ previewPage }} из {{ form.questions.length }}</p>

                <div class="flex-1 min-h-0 flex flex-col justify-center gap-5 overflow-hidden">
                  <div class="flex flex-col gap-2">
                    <h3 class="text-2xl font-medium text-highlighted">
                      {{ currentPreviewQuestion.title || 'Без названия' }}
                      <span v-if="currentPreviewQuestion.required" class="text-error">*</span>
                    </h3>
                    <p v-if="currentPreviewQuestion.hint" class="text-muted">{{ currentPreviewQuestion.hint }}</p>
                  </div>

                  <!-- Заглушка области ответа по типу -->
                  <div>
                    <UInput v-if="currentPreviewQuestion.type === 'text'" disabled size="lg" class="w-full max-w-lg" placeholder="Короткий ответ" />
                    <UTextarea v-else-if="currentPreviewQuestion.type === 'textarea'" disabled :rows="3" size="lg" class="w-full max-w-lg" placeholder="Развёрнутый ответ" />
                    <UInput v-else-if="currentPreviewQuestion.type === 'number'" disabled type="number" size="lg" class="w-44" placeholder="0" />
                    <UInput v-else-if="currentPreviewQuestion.type === 'date'" disabled type="date" size="lg" class="w-52" />
                    <div v-else-if="currentPreviewQuestion.type === 'yesno'" class="flex gap-3">
                      <UBadge color="neutral" variant="outline" size="lg">Да</UBadge>
                      <UBadge color="neutral" variant="outline" size="lg">Нет</UBadge>
                    </div>
                    <p v-else class="text-sm text-dimmed">Тип: {{ questionTypeLabel(currentPreviewQuestion.type) }} — варианты ответа добавим позже.</p>
                  </div>
                </div>
              </div>
            </div>

            <!-- Пейджер окна (только на страницах вопросов; титульник не считается) -->
            <div v-if="previewPage > 0" class="shrink-0 border-t border-default px-5 py-3 flex items-center justify-between gap-3">
              <UButton color="neutral" variant="ghost" size="md" leading-icon="i-lucide-arrow-left" @click="previewPrev">Назад</UButton>
              <span class="text-xs text-dimmed tabular-nums">{{ previewPage }} / {{ form.questions.length }}</span>
              <UButton v-if="isLastPreviewPage" color="primary" size="md" icon="i-lucide-check" @click="finishOpen = true">Завершить</UButton>
              <UButton v-else color="neutral" variant="ghost" size="md" trailing-icon="i-lucide-arrow-right" @click="previewNext">Далее</UButton>
            </div>
          </div>
        </div>

        <!-- Подтверждение завершения -->
        <UModal v-model:open="finishOpen" title="Завершить?" description="">
          <template #body>
            <p class="text-default">
              Вы уверены, что хотите завершить и отправить ответы? После отправки изменить их будет нельзя.
            </p>
          </template>
          <template #footer>
            <div class="flex justify-end gap-3 w-full">
              <UButton color="neutral" variant="outline" @click="finishOpen = false">Отмена</UButton>
              <UButton color="primary" icon="i-lucide-check" @click="confirmFinish">Завершить</UButton>
            </div>
          </template>
        </UModal>
      </div>

      </div>
      <!-- Навигация по шагам (неподвижный футер) -->
      <div class="shrink-0 flex justify-between items-center gap-3 pt-3 border-t border-default">
        <UButton v-if="stepIdx > 0" color="neutral" variant="outline" size="xl" leading-icon="i-lucide-arrow-left" @click="prevStep">
          Назад
        </UButton>
        <span v-else />

        <div class="flex gap-3">
          <template v-if="step === 'preview'">
            <UButton color="neutral" variant="outline" size="xl" icon="i-lucide-save" @click="saveDraft">Сохранить черновик</UButton>
            <UButton size="xl" icon="i-lucide-send" @click="publish">Опубликовать</UButton>
          </template>
          <UButton v-else size="xl" trailing-icon="i-lucide-arrow-right" @click="nextStep">Дальше</UButton>
        </div>
      </div>
    </div>

    <!-- Черновики -->
    <div v-else-if="section === 'drafts'">
      <UEmpty
        icon="i-lucide-file-clock"
        title="Черновиков пока нет"
        description="Сохранённые, но не опубликованные тесты появятся здесь."
        class="py-12"
      />
    </div>
  </div>
</template>
