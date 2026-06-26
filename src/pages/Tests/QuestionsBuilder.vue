<script setup lang="ts">
import { computed, ref } from 'vue';
import {
  QUESTION_TYPE_ITEMS,
  typeHasOptions,
  applyTypeDefaults,
  createQuestion,
  createOption,
  questionTypeHint,
  questionTypeLabel,
  type QType,
  type Question,
} from './questionTypes';

const props = defineProps<{ kind: 'test' | 'survey' | 'poll' }>();
const questions = defineModel<Question[]>({ default: () => [] });

// В голосовании — единственный вопрос с кандидатами
const pollQuestion = computed(() => questions.value[0]);

function addQuestion() {
  questions.value.push(createQuestion());
}

function onTypeChange(q: Question) {
  applyTypeDefaults(q);
  q.correct = null; // правильный ответ сбрасываем при смене типа
}

// ── Варианты / кандидаты ─────────────────────────────────────────────────────
function addOption(q: Question) {
  if (!Array.isArray(q.options)) q.options = [];
  q.options.push(createOption());
}
function removeOption(q: Question, index: number) {
  if (q.options.length > 2) q.options.splice(index, 1);
}
function optionIcon(t: QType): string {
  if (t === 'single') return 'i-lucide-circle';
  if (t === 'multiple') return 'i-lucide-square';
  return 'i-lucide-chevron-down'; // dropdown
}

// ── Удаление вопроса с подтверждением ────────────────────────────────────────
const confirmOpen = ref(false);
const deleteIndex = ref<number | null>(null);
function askDelete(index: number) {
  deleteIndex.value = index;
  confirmOpen.value = true;
}
function confirmDelete() {
  if (deleteIndex.value !== null) questions.value.splice(deleteIndex.value, 1);
  confirmOpen.value = false;
  deleteIndex.value = null;
}

// ── Правильный ответ (только тесты) ──────────────────────────────────────────
function hasCorrect(q: Question): boolean {
  const c = q.correct;
  if (c == null) return false;
  if (Array.isArray(c)) return c.length > 0;
  if (typeof c === 'string') return c.trim() !== '';
  return true;
}

const correctOpen = ref(false);
const correctQ = ref<Question | null>(null);
const correctDraft = ref<string | number | string[] | null>(null);

function openCorrect(q: Question) {
  correctQ.value = q;
  correctDraft.value = q.type === 'multiple'
    ? (Array.isArray(q.correct) ? [...q.correct] : [])
    : (q.correct ?? null);
  correctOpen.value = true;
}
function isCorrectSelected(val: string | number): boolean {
  if (Array.isArray(correctDraft.value)) return correctDraft.value.includes(val as string);
  return correctDraft.value === val;
}
function pickCorrect(val: string | number) {
  if (correctQ.value?.type === 'multiple') {
    const arr = Array.isArray(correctDraft.value) ? [...correctDraft.value] : [];
    const i = arr.indexOf(val as string);
    if (i >= 0) arr.splice(i, 1); else arr.push(val as string);
    correctDraft.value = arr;
  } else {
    correctDraft.value = val;
  }
}
function saveCorrect() {
  if (correctQ.value) correctQ.value.correct = correctDraft.value;
  correctOpen.value = false;
}
const yesnoItems = [
  { label: 'Да', value: 'yes' },
  { label: 'Нет', value: 'no' },
];
function scaleNumbers(q: Question): number[] {
  const min = Math.min(q.scaleMin, q.scaleMax);
  const max = Math.max(q.scaleMin, q.scaleMax);
  const arr: number[] = [];
  for (let n = min; n <= max && arr.length < 50; n++) arr.push(n);
  return arr;
}
</script>

<template>
  <!-- ═══ Режим ГОЛОСОВАНИЯ: один вопрос с кандидатами ═══ -->
  <div v-if="kind === 'poll'" class="flex flex-col gap-4">
    <div v-if="pollQuestion" class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-4 bg-elevated/30">
      <UFormField label="Вопрос голосования">
        <UInput v-model="pollQuestion.title" size="lg" class="w-full" placeholder="Например: За кого голосуете?" />
      </UFormField>

      <div class="flex flex-col gap-2">
        <p class="text-sm font-medium text-highlighted">Кандидаты</p>
        <div v-for="(opt, oi) in pollQuestion.options" :key="opt.id" class="flex items-center gap-2">
          <UIcon name="i-lucide-circle" class="size-4 shrink-0 text-dimmed" />
          <UInput v-model="opt.text" size="md" class="flex-1" :placeholder="`Кандидат ${oi + 1}`" />
          <UButton
            color="error"
            variant="ghost"
            size="sm"
            icon="i-lucide-x"
            :disabled="pollQuestion.options.length <= 2"
            @click="removeOption(pollQuestion, oi)"
          />
        </div>
        <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-plus" class="w-fit ml-6" @click="addOption(pollQuestion)">
          Добавить кандидата
        </UButton>
      </div>
    </div>
  </div>

  <!-- ═══ Обычный режим: список вопросов ═══ -->
  <div v-else class="flex flex-col gap-4">
    <div v-if="!questions.length" class="text-sm text-muted">
      Вопросов пока нет — добавьте первый.
    </div>

    <div
      v-for="(q, i) in questions"
      :key="q.id"
      class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-3 bg-elevated/30"
    >
      <div class="flex items-start gap-3">
        <span class="shrink-0 mt-1 inline-flex items-center justify-center min-w-9 h-8 px-2 rounded-lg bg-primary/10 text-primary text-sm font-medium tabular-nums">
          #{{ i + 1 }}
        </span>
        <div class="flex-1 min-w-0 flex flex-col gap-2">
          <UInput v-model="q.title" size="lg" class="w-full" placeholder="Название вопроса" />
          <UInput v-model="q.hint" size="md" class="w-full" placeholder="Подсказка (необязательно)" />
        </div>
        <UButton
          color="error"
          variant="ghost"
          size="sm"
          icon="i-lucide-trash-2"
          class="shrink-0"
          @click="askDelete(i)"
        >
          Удалить вопрос
        </UButton>
      </div>

      <div class="flex flex-col sm:flex-row sm:items-center gap-3 pl-12">
        <USelect
          v-model="q.type"
          :items="QUESTION_TYPE_ITEMS"
          size="md"
          class="w-full sm:w-64"
          :content="{ align: 'start', sideOffset: 8 }"
          @update:model-value="onTypeChange(q)"
        />
        <UCheckbox v-model="q.required" label="Обязательный вопрос" />

        <UButton
          v-if="kind === 'test'"
          :color="hasCorrect(q) ? 'success' : 'primary'"
          variant="soft"
          size="sm"
          :icon="hasCorrect(q) ? 'i-lucide-check-circle-2' : 'i-lucide-target'"
          class="sm:ml-auto"
          @click="openCorrect(q)"
        >
          {{ hasCorrect(q) ? 'Правильный ответ указан' : 'Указать правильный ответ' }}
        </UButton>
      </div>

      <!-- Конфигуратор ответа под выбранный тип -->
      <div class="pl-12 flex flex-col gap-2">
        <template v-if="typeHasOptions(q.type)">
          <div v-for="(opt, oi) in q.options" :key="opt.id" class="flex items-center gap-2">
            <UIcon :name="optionIcon(q.type)" class="size-4 shrink-0 text-dimmed" />
            <UInput v-model="opt.text" size="md" class="flex-1" :placeholder="`Вариант ${oi + 1}`" />
            <UButton
              color="error"
              variant="ghost"
              size="sm"
              icon="i-lucide-x"
              :disabled="q.options.length <= 2"
              @click="removeOption(q, oi)"
            />
          </div>
          <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-plus" class="w-fit ml-6" @click="addOption(q)">
            Добавить вариант
          </UButton>
        </template>

        <template v-else-if="q.type === 'scale'">
          <div class="flex flex-wrap items-end gap-3">
            <UFormField label="От"><UInput v-model.number="q.scaleMin" type="number" size="md" class="w-24" /></UFormField>
            <UFormField label="До"><UInput v-model.number="q.scaleMax" type="number" size="md" class="w-24" /></UFormField>
            <UFormField label="Подпись слева"><UInput v-model="q.scaleMinLabel" size="md" class="w-44" placeholder="например: Плохо" /></UFormField>
            <UFormField label="Подпись справа"><UInput v-model="q.scaleMaxLabel" size="md" class="w-44" placeholder="например: Отлично" /></UFormField>
          </div>
        </template>

        <p v-else class="text-xs text-dimmed">{{ questionTypeHint(q.type) }}</p>
      </div>
    </div>

    <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-plus" class="w-fit" @click="addQuestion">
      Добавить вопрос
    </UButton>
  </div>

  <!-- Подтверждение удаления -->
  <UModal v-model:open="confirmOpen" title="Удалить вопрос?" description="">
    <template #body>
      <p class="text-default">Вы уверены, что хотите удалить вопрос №{{ (deleteIndex ?? 0) + 1 }}?</p>
    </template>
    <template #footer>
      <div class="flex gap-3 justify-end w-full">
        <UButton color="neutral" variant="outline" @click="confirmOpen = false">Отмена</UButton>
        <UButton color="error" icon="i-lucide-trash-2" @click="confirmDelete">Удалить</UButton>
      </div>
    </template>
  </UModal>

  <!-- Установка правильного ответа -->
  <UModal v-model:open="correctOpen" title="Установить правильный ответ" description="">
    <template #body>
      <div v-if="correctQ" class="flex flex-col gap-4">
        <div class="flex flex-col gap-1">
          <p class="text-base font-medium text-highlighted">{{ correctQ.title || 'Без названия' }}</p>
          <p v-if="correctQ.hint" class="text-sm text-muted">{{ correctQ.hint }}</p>
          <p class="text-xs text-dimmed">Тип: {{ questionTypeLabel(correctQ.type) }}</p>
        </div>

        <!-- Выбор из вариантов (single / multiple / dropdown) -->
        <div v-if="typeHasOptions(correctQ.type)" class="flex flex-col gap-2">
          <button
            v-for="(opt, oi) in correctQ.options"
            :key="opt.id"
            type="button"
            class="flex items-center gap-2 w-full text-left rounded-lg ring-1 px-3 py-2 transition-colors"
            :class="isCorrectSelected(opt.id) ? 'ring-green-500 bg-green-500/10 text-green-600 dark:text-green-400' : 'ring-default hover:bg-elevated'"
            @click="pickCorrect(opt.id)"
          >
            <UIcon :name="isCorrectSelected(opt.id) ? 'i-lucide-check-circle-2' : optionIcon(correctQ.type)" class="size-4 shrink-0" />
            <span class="text-sm">{{ opt.text || `Вариант ${oi + 1}` }}</span>
          </button>
          <p v-if="correctQ.type === 'multiple'" class="text-xs text-dimmed">Можно отметить несколько правильных вариантов.</p>
        </div>

        <!-- Да / Нет -->
        <div v-else-if="correctQ.type === 'yesno'" class="flex gap-2">
          <button
            v-for="it in yesnoItems"
            :key="it.value"
            type="button"
            class="rounded-lg ring-1 px-5 py-2 text-sm transition-colors"
            :class="isCorrectSelected(it.value) ? 'ring-green-500 bg-green-500/10 text-green-600 dark:text-green-400' : 'ring-default hover:bg-elevated'"
            @click="pickCorrect(it.value)"
          >
            {{ it.label }}
          </button>
        </div>

        <!-- Шкала -->
        <div v-else-if="correctQ.type === 'scale'" class="flex flex-wrap gap-2">
          <button
            v-for="n in scaleNumbers(correctQ)"
            :key="n"
            type="button"
            class="size-10 rounded-lg ring-1 text-sm font-medium transition-colors"
            :class="isCorrectSelected(n) ? 'ring-green-500 bg-green-500/10 text-green-600 dark:text-green-400' : 'ring-default text-muted hover:bg-elevated'"
            @click="pickCorrect(n)"
          >
            {{ n }}
          </button>
        </div>

        <!-- Текст / число / дата -->
        <div v-else class="flex flex-col gap-1.5">
          <UInput v-if="correctQ.type === 'number'" v-model.number="correctDraft" type="number" size="lg" class="w-44" placeholder="Правильное число" />
          <UInput v-else-if="correctQ.type === 'date'" v-model="correctDraft" type="date" size="lg" class="w-52" />
          <UTextarea v-else-if="correctQ.type === 'textarea'" v-model="correctDraft" :rows="3" size="lg" class="w-full" placeholder="Правильный ответ" />
          <UInput v-else v-model="correctDraft" size="lg" class="w-full" placeholder="Правильный ответ" />
          <p class="text-xs text-dimmed">Сравнение с ответом сотрудника — без учёта регистра и крайних пробелов.</p>
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex gap-3 justify-end w-full">
        <UButton color="neutral" variant="outline" @click="correctOpen = false">Выйти</UButton>
        <UButton color="success" icon="i-lucide-check" @click="saveCorrect">Сохранить</UButton>
      </div>
    </template>
  </UModal>
</template>
