<script setup lang="ts">
import { ref } from 'vue';
import { QUESTION_TYPE_ITEMS, type Question } from './questionTypes';

const questions = defineModel<Question[]>({ default: () => [] });

function uid(): string {
  return (crypto as any)?.randomUUID?.() ?? `q_${Date.now()}_${Math.random().toString(36).slice(2)}`;
}
function addQuestion() {
  questions.value.push({ id: uid(), title: '', hint: '', type: 'single', required: true });
}

// ── Удаление с подтверждением ────────────────────────────────────────────────
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
</script>

<template>
  <div class="flex flex-col gap-4">
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
        />
        <UCheckbox v-model="q.required" label="Обязательный вопрос" />
      </div>
    </div>

    <UButton
      color="neutral"
      variant="outline"
      size="lg"
      icon="i-lucide-plus"
      class="w-fit"
      @click="addQuestion"
    >
      Добавить вопрос
    </UButton>

    <!-- Подтверждение удаления -->
    <UModal v-model:open="confirmOpen" title="Удалить вопрос?" description="">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить вопрос №{{ (deleteIndex ?? 0) + 1 }}?
        </p>
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" @click="confirmOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-trash-2" @click="confirmDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
