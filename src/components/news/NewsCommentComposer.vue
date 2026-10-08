<script setup lang="ts">
/**
 * Поле «Прокомментировать» / «Ответить». Отправка — Ctrl+Enter или кнопкой;
 * send бросает ошибку — текст остаётся в поле (тост показывает вызывающий).
 */
import { computed, nextTick, onMounted, ref } from 'vue';
import { NEWS_COMMENT_MAX_LENGTH } from '../../composables/useNewsComments';

const props = withDefaults(
  defineProps<{
    send: (text: string) => Promise<void>;
    placeholder?: string;
    autofocus?: boolean;
    /** Кнопка «Отмена» (для ответа) */
    cancelable?: boolean;
  }>(),
  { placeholder: 'Прокомментировать…', autofocus: false, cancelable: false },
);

const emit = defineEmits<{ (e: 'cancel'): void }>();

const text = ref('');
const sending = ref(false);
const field = ref<{ textareaRef?: HTMLTextAreaElement } | null>(null);

const canSend = computed(() => {
  const s = text.value.trim();
  return !!s && s.length <= NEWS_COMMENT_MAX_LENGTH && !sending.value;
});

async function submit() {
  if (!canSend.value) return;
  sending.value = true;
  try {
    await props.send(text.value.trim());
    text.value = '';
  } catch {
    /* текст остаётся, тост показал вызывающий */
  } finally {
    sending.value = false;
  }
}

function cancel() {
  text.value = '';
  emit('cancel');
}

onMounted(async () => {
  if (!props.autofocus) return;
  await nextTick();
  field.value?.textareaRef?.focus();
});
</script>

<template>
  <div class="flex items-end gap-2">
    <UTextarea
      ref="field"
      v-model="text"
      :rows="1"
      autoresize
      :maxrows="8"
      :maxlength="NEWS_COMMENT_MAX_LENGTH"
      :placeholder="placeholder"
      :aria-label="placeholder"
      :disabled="sending"
      class="w-full min-w-0 flex-1"
      @keydown.ctrl.enter.prevent="submit"
      @keydown.meta.enter.prevent="submit"
      @keydown.esc.prevent="cancelable && cancel()"
    />
    <UTooltip v-if="cancelable" text="Отмена">
      <UButton
        type="button"
        color="neutral"
        variant="ghost"
        icon="i-lucide-x"
        square
        aria-label="Отменить ответ"
        :disabled="sending"
        @click="cancel"
      />
    </UTooltip>
    <UTooltip text="Отправить (Ctrl+Enter)">
      <UButton
        type="button"
        color="primary"
        icon="i-lucide-send-horizontal"
        square
        aria-label="Отправить комментарий"
        :loading="sending"
        :disabled="!canSend"
        @click="submit"
      />
    </UTooltip>
  </div>
</template>
