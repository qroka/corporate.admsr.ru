<script setup lang="ts">
/**
 * Окно задачи для текста с матом (ADR-053). Одно на приложение (App.vue); открывает
 * его postWithProfanityGate, когда сервер вернул задачу. Закрыли — публикация
 * отменена, текст остаётся в поле.
 */
import { ref, watch } from 'vue';
import { cancelProfanityChallenge, profanityState as state, submitProfanityAnswer } from '../composables/useProfanityGate';

const answer = ref('');

// Новая задача (в том числе после неверного ответа) — поле очищаем.
watch(
  () => state.question,
  () => {
    answer.value = '';
  },
);

function submit() {
  if (!answer.value.trim() || state.busy) return;
  submitProfanityAnswer(answer.value.trim());
}

function onOpenChange(value: boolean) {
  if (!value && !state.busy) cancelProfanityChallenge();
}
</script>

<template>
  <UModal
    :open="state.open"
    title="Нецензурные выражения"
    description="Чтобы опубликовать текст, решите задачу. Найденные слова всё равно скроем звёздочками."
    :ui="{ content: 'sm:max-w-md', body: 'flex flex-col gap-4' }"
    @update:open="onOpenChange"
  >
    <template #body>
      <p v-if="state.retry && state.message" class="flex items-center gap-1.5 text-sm text-error">
        <UIcon name="i-lucide-circle-x" class="size-4 shrink-0" aria-hidden="true" />
        {{ state.message }}
      </p>

      <div class="rounded-panel bg-elevated px-4 py-3">
        <p class="text-xs text-muted">Задача</p>
        <p class="mt-1 whitespace-pre-line text-lg font-semibold text-highlighted tabular-nums">{{ state.question }}</p>
      </div>

      <form id="profanity-challenge-form" @submit.prevent="submit">
        <UFormField label="Ответ" hint="Целое число">
          <UInput
            :key="state.question"
            v-model="answer"
            autofocus
            inputmode="numeric"
            autocomplete="off"
            class="w-full"
            placeholder="Например, 42"
          />
        </UFormField>
      </form>

      <div class="flex flex-col gap-1">
        <p class="text-xs text-muted">Опубликуется так:</p>
        <p class="line-clamp-4 whitespace-pre-line break-words text-sm text-default">{{ state.masked }}</p>
      </div>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :disabled="state.busy" @click="cancelProfanityChallenge">Отмена</UButton>
        <UButton
          color="primary"
          type="submit"
          form="profanity-challenge-form"
          :loading="state.busy"
          :disabled="!answer.trim()"
        >
          Опубликовать
        </UButton>
      </div>
    </template>
  </UModal>
</template>
