<script setup lang="ts">
import { computed } from 'vue'
import type { Question, QuestionType, UUID } from '../../tests/types'

const props = defineProps<{
  modelValue: Question
  mode: 'test' | 'survey'
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: Question): void
  (e: 'delete'): void
}>()

const typeItems = [
  { label: 'Один ответ', value: 'single_choice' },
  { label: 'Несколько ответов', value: 'multiple_choice' },
  { label: 'Короткий текст', value: 'short_text' },
  { label: 'Длинный текст', value: 'long_text' },
  { label: 'Шкала 1–10', value: 'rating_1_10' },
  { label: 'Выпадающий список', value: 'select' },
  { label: 'Загрузка файла', value: 'file' },
]

const hasOptions = computed(() =>
  props.modelValue.type === 'single_choice' ||
  props.modelValue.type === 'multiple_choice' ||
  props.modelValue.type === 'select',
)

function update(patch: Partial<Question>) {
  emit('update:modelValue', { ...props.modelValue, ...patch })
}

function ensureOptionsForType(nextType: QuestionType) {
  if (nextType === 'short_text' || nextType === 'long_text' || nextType === 'rating_1_10' || nextType === 'file') {
    update({ type: nextType, options: undefined })
    return
  }

  const opts = props.modelValue.options?.length
    ? props.modelValue.options
    : [
        { id: crypto.randomUUID(), label: 'Вариант 1', order: 0, isCorrect: false },
        { id: crypto.randomUUID(), label: 'Вариант 2', order: 1, isCorrect: false },
      ]
  update({ type: nextType, options: opts })
}

function addOption() {
  const base = props.modelValue.options ?? []
  const next = [
    ...base,
    { id: crypto.randomUUID(), label: `Вариант ${base.length + 1}`, order: base.length, isCorrect: false },
  ]
  update({ options: next })
}

function updateOption(id: UUID, patch: any) {
  const base = props.modelValue.options ?? []
  update({
    options: base
      .map(o => (o.id === id ? { ...o, ...patch } : o))
      .map((o, idx) => ({ ...o, order: idx })),
  })
}

function deleteOption(id: UUID) {
  const base = props.modelValue.options ?? []
  update({ options: base.filter(o => o.id !== id).map((o, idx) => ({ ...o, order: idx })) })
}

function markSingleCorrect(id: UUID) {
  const base = props.modelValue.options ?? []
  update({ options: base.map(o => ({ ...o, isCorrect: o.id === id })) })
}
</script>

<template>
  <UCard class="w-full">
    <template #header>
      <div class="flex items-start justify-between gap-3">
        <div class="flex-1 min-w-0">
          <UInput
            :model-value="modelValue.title"
            placeholder="Текст вопроса"
            @update:model-value="(v) => update({ title: String(v) })"
          />
          <div class="mt-2">
            <UInput
              :model-value="modelValue.hint"
              placeholder="Подсказка (необязательно)"
              @update:model-value="(v) => update({ hint: String(v) })"
            />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <UBadge color="neutral" variant="soft">#{{ modelValue.order + 1 }}</UBadge>
          <UButton color="red" variant="soft" icon="i-lucide-trash-2" size="sm" @click="$emit('delete')">
            Удалить
          </UButton>
        </div>
      </div>
    </template>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <UFormField label="Тип вопроса">
        <USelect
          :items="typeItems"
          :model-value="modelValue.type"
          @update:model-value="(v) => ensureOptionsForType(v as any)"
        />
      </UFormField>

      <UFormField label="Обязательный">
        <UToggle
          :model-value="modelValue.required"
          @update:model-value="(v) => update({ required: Boolean(v) })"
        />
      </UFormField>

      <UFormField label="Режим">
        <div class="text-sm text-muted">
          {{ mode === 'test' ? 'Тест (есть правильные ответы)' : 'Опрос (без проверки)' }}
        </div>
      </UFormField>
    </div>

    <div v-if="hasOptions" class="mt-5 space-y-3">
      <div class="flex items-center justify-between">
        <div class="text-sm font-medium text-highlighted">Варианты ответа</div>
        <UButton color="primary" variant="soft" icon="i-lucide-plus" size="sm" @click="addOption">
          Добавить вариант
        </UButton>
      </div>

      <div class="space-y-2">
        <div
          v-for="opt in modelValue.options"
          :key="opt.id"
          class="flex items-center gap-2"
        >
          <UInput
            class="flex-1"
            :model-value="opt.label"
            @update:model-value="(v) => updateOption(opt.id, { label: String(v) })"
          />

          <template v-if="mode === 'test'">
            <UTooltip text="Правильный вариант">
              <UButton
                v-if="modelValue.type === 'multiple_choice'"
                :color="opt.isCorrect ? 'green' : 'neutral'"
                variant="soft"
                icon="i-lucide-check"
                size="sm"
                @click="updateOption(opt.id, { isCorrect: !opt.isCorrect })"
              />
              <UButton
                v-else
                :color="opt.isCorrect ? 'green' : 'neutral'"
                variant="soft"
                icon="i-lucide-check"
                size="sm"
                @click="markSingleCorrect(opt.id)"
              />
            </UTooltip>
          </template>

          <UButton color="red" variant="ghost" icon="i-lucide-x" size="sm" @click="deleteOption(opt.id)" />
        </div>
      </div>
    </div>

    <div v-else class="mt-5 text-sm text-muted">
      Для этого типа вопроса варианты ответов не требуются.
    </div>
  </UCard>
</template>

