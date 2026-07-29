<script setup lang="ts">
/**
 * Компактные настройки теста (проходной балл, попытки, время, показ ответов).
 * Можно встроить в конструктор курса; не ломает TestBuilder.
 */
export type TestSettingsModel = {
  usePassingScore: boolean;
  passingScore: number;
  limitAttempts: boolean;
  attempts: number;
  useTimeLimit: boolean;
  timeLimit: string;
  showCorrectAnswers: boolean;
};

const model = defineModel<TestSettingsModel>({ required: true });
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <UFormField label="Проходной балл">
        <div class="flex flex-col gap-2">
          <UCheckbox v-model="model.usePassingScore" label="Использовать проходной балл" />
          <UInput
            v-model.number="model.passingScore"
            type="number"
            :min="0"
            :max="100"
            size="lg"
            class="w-full"
            :disabled="!model.usePassingScore"
            :class="model.usePassingScore ? '' : 'opacity-50'"
            placeholder="%"
          />
        </div>
      </UFormField>

      <UFormField label="Попытки">
        <div class="flex flex-col gap-2">
          <UCheckbox v-model="model.limitAttempts" label="Ограничить попытки" />
          <UInput
            v-model.number="model.attempts"
            type="number"
            :min="1"
            size="lg"
            class="w-full"
            :disabled="!model.limitAttempts"
            :class="model.limitAttempts ? '' : 'opacity-50'"
            placeholder="Количество попыток"
          />
        </div>
      </UFormField>

      <UFormField label="Лимит времени">
        <div class="flex flex-col gap-2">
          <UCheckbox v-model="model.useTimeLimit" label="Ограничить время" />
          <UInput
            v-model="model.timeLimit"
            type="time"
            :step="1"
            size="lg"
            class="w-full"
            :disabled="!model.useTimeLimit"
            :class="model.useTimeLimit ? '' : 'opacity-50'"
          />
        </div>
      </UFormField>

      <UFormField label="Ответы">
        <UCheckbox v-model="model.showCorrectAnswers" label="Показывать правильные ответы" />
      </UFormField>
    </div>
  </div>
</template>
