<script setup lang="ts">
import { ref } from 'vue';

const props = defineProps<{
  label?: string;
  placeholder?: string;
  maxlength?: number;
  rows?: number;
}>();

const model = defineModel<string>({ default: '' });

const open = ref(false);
const draft = ref('');

function openModal() {
  draft.value = model.value;
  open.value = true;
}
function save() {
  model.value = draft.value.slice(0, props.maxlength ?? undefined);
  open.value = false;
}
</script>

<template>
  <div class="relative cursor-pointer" @click="openModal">
    <UInput
      :model-value="model"
      :placeholder="placeholder"
      readonly
      tabindex="-1"
      size="lg"
      class="w-full pointer-events-none"
      trailing-icon="i-lucide-pen-line"
    />
  </div>

  <UModal v-model:open="open" :title="label || 'Редактирование'" :ui="{ content: 'max-w-2xl' }">
    <template #body>
      <UTextarea
        v-model="draft"
        :rows="rows ?? 10"
        :maxlength="maxlength"
        :placeholder="placeholder"
        size="lg"
        class="w-full"
        autofocus
      />
      <div v-if="maxlength" class="mt-1.5 text-xs text-muted text-right tabular-nums">
        {{ draft.length }}/{{ maxlength }}
      </div>
    </template>
    <template #footer>
      <div class="flex justify-end gap-3 w-full">
        <UButton color="neutral" variant="outline" @click="open = false">Отмена</UButton>
        <UButton icon="i-lucide-check" @click="save">Готово</UButton>
      </div>
    </template>
  </UModal>
</template>
