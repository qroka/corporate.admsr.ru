<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useUsersData } from '../../composables/useUsersData';
import { useAppToast } from '../../composables/useAppToast';
import { useTestsStore } from '../../composables/useTestsStore';
import OfoMultiSelect from '../../components/OfoMultiSelect.vue';
import ModalTextField from './ModalTextField.vue';
import QuestionsBuilder from './QuestionsBuilder.vue';
import TestRunner from './TestRunner.vue';
import { createQuestion, applyTypeDefaults } from './questionTypes';
import { createEmptyForm, cloneForm, type TestForm } from './testForm';

const emit = defineEmits<{ (e: 'published'): void }>();
const { toast } = useAppToast();
const store = useTestsStore();
onMounted(() => { store.ensureLoaded(); });

// ── Разделы конструктора ──────────────────────────────────────────────────────
const section = ref<'new' | 'drafts'>('new');
const sectionItems = [
  { label: 'Создать новый', value: 'new', icon: 'i-lucide-plus' },
  { label: 'Черновики', value: 'drafts', icon: 'i-lucide-file-clock' },
];

// ── Данные формы ──────────────────────────────────────────────────────────────
const form = reactive<TestForm>(createEmptyForm());
const editingId = ref<number | null>(null); // != null → редактируем существующий черновик

const kindItems = [
  { label: 'Тест', value: 'test' },
  { label: 'Опрос', value: 'survey' },
  { label: 'Голосование', value: 'poll' },
];
const visibilityItems = [
  { label: 'Публичный', value: 'public' },
  { label: 'Приватный', value: 'private' },
];
const showResultItems = computed(() => [
  { label: 'Сразу', value: 'immediate', disabled: form.allowChangeAnswer },
  { label: 'После прохождения', value: 'after' },
  { label: 'Не показывать результат', value: 'never' },
]);
const kindLabel = (k: string) => kindItems.find((i) => i.value === k)?.label ?? k;

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

// «Показывать правильные ответы» — нельзя при «не показывать результат»;
// «Сразу» взаимоисключимо с «изменять ответ до завершения»
watch(
  () => form.showResult,
  (r) => {
    if (r === 'never') form.showCorrectAnswers = false;
    if (r === 'immediate') form.allowChangeAnswer = false;
  },
);

// Голосование — единственный вопрос с кандидатами (тип single, минимум 2)
watch(
  () => form.kind,
  (k) => {
    if (k !== 'poll') return;
    if (form.questions.length === 0) form.questions.push(createQuestion());
    if (form.questions.length > 1) form.questions.splice(1);
    const q = form.questions[0];
    q.type = 'single';
    q.correct = null;
    applyTypeDefaults(q);
  },
  { immediate: true },
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

// ── Сохранение / публикация ───────────────────────────────────────────────────
function resetForm() {
  Object.assign(form, createEmptyForm());
  editingId.value = null;
  step.value = 'settings';
}
async function onSaveDraft() {
  try {
    if (editingId.value != null) {
      await store.updateDraft(editingId.value, form);
      toast.add({ title: 'Черновик обновлён', color: 'success', icon: 'i-lucide-check' });
    } else {
      await store.saveDraft(form);
      toast.add({ title: 'Черновик сохранён', description: 'Форма перенесена в «Черновики».', color: 'success', icon: 'i-lucide-file-check' });
    }
    resetForm();
    section.value = 'drafts';
  } catch (e) {
    toast.add({ title: 'Не удалось сохранить', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' });
  }
}
async function onPublish() {
  try {
    await store.publish(form, editingId.value);
    toast.add({ title: 'Опубликовано', description: 'Форма доступна во вкладке «Список».', color: 'success', icon: 'i-lucide-send' });
    resetForm();
    emit('published');
  } catch (e) {
    toast.add({ title: 'Не удалось опубликовать', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' });
  }
}

// ── Черновики: действия ───────────────────────────────────────────────────────
function editDraft(d: TestForm) {
  Object.assign(form, cloneForm(d));
  editingId.value = d.id;
  section.value = 'new';
  step.value = 'settings';
}
async function publishDraft(d: TestForm) {
  try {
    await store.publish(d, d.id);
    toast.add({ title: 'Опубликовано', description: 'Форма доступна во вкладке «Список».', color: 'success', icon: 'i-lucide-send' });
    emit('published');
  } catch (e) {
    toast.add({ title: 'Не удалось опубликовать', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' });
  }
}

// Удаление черновика
const deleteOpen = ref(false);
const deleteTarget = ref<TestForm | null>(null);
function askDeleteDraft(d: TestForm) { deleteTarget.value = d; deleteOpen.value = true; }
async function confirmDeleteDraft() {
  const id = deleteTarget.value?.id;
  deleteOpen.value = false;
  deleteTarget.value = null;
  if (id != null) {
    try { await store.removeDraft(id); }
    catch (e) { toast.add({ title: 'Не удалось удалить', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' }); }
  }
}

// Прохождение черновика
const runOpen = ref(false);
const runForm = ref<TestForm | null>(null);
function openRun(d: TestForm) {
  runForm.value = cloneForm(d);
  runOpen.value = true;
}

// Благодарность (после закрытия окна прохождения / предпросмотра)
const completionOpen = ref(false);
const completionForm = ref<TestForm | null>(null);
const completionText = computed(() => {
  const f = completionForm.value;
  if (!f) return '';
  return f.completionMessage.trim() || (f.kind === 'poll' ? 'Спасибо! Ваш голос учтён.' : 'Спасибо за прохождение!');
});
function onPreviewFinish() {
  completionForm.value = cloneForm(form);
  completionOpen.value = true;
}
function onDraftRunFinish() {
  completionForm.value = runForm.value;
  runOpen.value = false;
  completionOpen.value = true;
}

function fmtDate(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}
</script>

<template>
  <div class="flex flex-col flex-1 min-h-0 w-full gap-3 pt-1">
    <!-- Тулбар: разделы + шаги в одну строку -->
    <div class="flex items-center gap-3 flex-wrap">
      <UTabs v-model="section" :items="sectionItems" size="md" class="w-fit" />

      <div v-if="section === 'new'" class="flex items-center gap-1.5 text-sm flex-wrap ml-auto pr-1">
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
    </div>

    <!-- Создать новый / редактирование: мастер по шагам -->
    <div v-if="section === 'new'" class="flex flex-col flex-1 min-h-0 gap-4">
      <div v-if="editingId != null" class="shrink-0 -mb-1 flex items-center gap-2 text-sm text-primary">
        <UIcon name="i-lucide-pencil" class="size-4" />
        Редактирование черновика #{{ editingId }}
      </div>

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
              <UCheckbox v-if="form.kind === 'survey' || (form.kind === 'test' && form.showResult !== 'immediate')" v-model="form.allowChangeAnswer" label="Разрешить изменять ответ до завершения" />
              <UCheckbox v-if="form.kind === 'poll'" v-model="form.liveResults" label="Промежуточные результаты в реальном времени" />
              <UCheckbox v-if="form.kind === 'poll'" v-model="form.allowRevote" label="Разрешить переголосовать до закрытия" />
            </div>

            <USeparator class="my-0.5" />

            <UFormField label="Сообщение после завершения" name="completionMessage">
              <ModalTextField v-model="form.completionMessage" label="Сообщение после завершения" :maxlength="500" placeholder="Например: Спасибо за прохождение!" />
            </UFormField>
            <UCheckbox v-model="form.notifyAdmin" label="Уведомлять создателя о новом прохождении" />
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

          <div v-if="form.kind !== 'poll'" class="flex flex-col gap-2 min-w-0">
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

          <UFormField v-if="form.kind === 'test'" label="Показать результат" name="showResult">
            <USelect v-model="form.showResult" :items="showResultItems" size="lg" class="w-full min-w-0" />
          </UFormField>
        </div>
      </div>

      <!-- ШАГ 2: Вопросы -->
      <div v-show="step === 'questions'">
        <QuestionsBuilder v-model="form.questions" :kind="form.kind" />
      </div>

      <!-- ШАГ 3: Предпросмотр -->
      <TestRunner v-if="step === 'preview'" :form="form" preview-hint class="h-full" @finish="onPreviewFinish" />
      </div>

      <!-- Навигация по шагам (неподвижный футер) -->
      <div class="shrink-0 flex justify-between items-center gap-3 pt-3 border-t border-default">
        <UButton v-if="stepIdx > 0" color="neutral" variant="outline" size="xl" leading-icon="i-lucide-arrow-left" @click="prevStep">
          Назад
        </UButton>
        <span v-else />

        <div class="flex gap-3">
          <template v-if="step === 'preview'">
            <UButton color="neutral" variant="outline" size="xl" icon="i-lucide-save" @click="onSaveDraft">
              {{ editingId != null ? 'Обновить черновик' : 'Сохранить черновик' }}
            </UButton>
            <UButton size="xl" icon="i-lucide-send" @click="onPublish">Опубликовать</UButton>
          </template>
          <UButton v-else size="xl" trailing-icon="i-lucide-arrow-right" @click="nextStep">Дальше</UButton>
        </div>
      </div>
    </div>

    <!-- Черновики -->
    <div v-else-if="section === 'drafts'" class="flex-1 min-h-0 overflow-y-auto pt-0.5">
      <UEmpty
        v-if="!store.drafts.value.length"
        icon="i-lucide-file-clock"
        title="Черновиков пока нет"
        description="Сохранённые, но не опубликованные формы появятся здесь."
        class="py-12"
      />

      <div v-else class="flex flex-col gap-3">
        <div
          v-for="d in store.drafts.value"
          :key="d.id ?? 0"
          class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 bg-elevated/30"
        >
          <div class="flex-1 min-w-0 flex flex-col gap-1">
            <div class="flex items-center gap-2 flex-wrap">
              <UBadge v-if="d.listId != null" color="neutral" variant="subtle" class="tabular-nums">#{{ d.listId }}</UBadge>
              <UBadge v-else color="neutral" variant="outline">Черновик</UBadge>
              <UBadge color="primary" variant="subtle">{{ kindLabel(d.kind) }}</UBadge>
              <span class="font-medium text-highlighted truncate">{{ d.title || 'Без названия' }}</span>
            </div>
            <p class="text-xs text-muted">
              Вопросов: {{ d.questions.length }}<span v-if="d.updatedAt"> · обновлён {{ fmtDate(d.updatedAt) }}</span>
            </p>
          </div>

          <div class="flex items-center gap-2 shrink-0 flex-wrap">
            <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-pencil" @click="editDraft(d)">Редактировать</UButton>
            <UButton color="neutral" variant="soft" size="sm" :icon="store.hasActiveSession(d) ? 'i-lucide-rotate-ccw' : 'i-lucide-play'" @click="openRun(d)">
              {{ store.hasActiveSession(d) ? 'Продолжить' : 'Пройти' }}
            </UButton>
            <UButton color="primary" size="sm" icon="i-lucide-send" @click="publishDraft(d)">Опубликовать</UButton>
            <UButton color="error" variant="ghost" size="sm" icon="i-lucide-trash-2" @click="askDeleteDraft(d)" />
          </div>
        </div>
      </div>
    </div>

    <!-- Прохождение черновика (без записи данных) -->
    <UModal
      v-model:open="runOpen"
      fullscreen
      :title="runForm?.title || 'Прохождение'"
      :ui="{ content: 'flex flex-col', body: 'flex-1 min-h-0 p-4 sm:p-6' }"
    >
      <template #body>
        <TestRunner v-if="runOpen && runForm" :form="runForm" persist-session class="h-full" @finish="onDraftRunFinish" />
      </template>
    </UModal>

    <!-- Сообщение после завершения (после закрытия окна прохождения) -->
    <UModal v-model:open="completionOpen" :title="completionForm?.kind === 'poll' ? 'Голос учтён' : 'Готово'" description="" :dismissible="false">
      <template #body>
        <div class="flex flex-col items-center text-center gap-3 py-2">
          <div class="size-12 rounded-full bg-success/10 flex items-center justify-center">
            <UIcon name="i-lucide-check" class="size-7 text-success" />
          </div>
          <p class="text-default whitespace-pre-line">{{ completionText }}</p>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end w-full">
          <UButton color="primary" @click="completionOpen = false">Закрыть</UButton>
        </div>
      </template>
    </UModal>

    <!-- Подтверждение удаления черновика -->
    <UModal v-model:open="deleteOpen" title="Удалить черновик?" description="">
      <template #body>
        <p class="text-default">Удалить черновик #{{ deleteTarget?.id }} «{{ deleteTarget?.title || 'Без названия' }}»?</p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-trash-2" @click="confirmDeleteDraft">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
