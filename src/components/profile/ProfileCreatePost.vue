<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { wallEditorExtensions, newsEditorEmojiMenuItems } from '../../composables/wallEditorExtensions';
import { newsEditorHandlers } from '../../composables/newsEditorHandlers';
import { newsEditorSlideoverUi } from '../../composables/newsEditorSlideoverUi';
import { wallEditorToolbarItems } from '../../composables/wallEditorToolbar';
import { isWallContentEmpty } from '../../composables/useProfileWall';

const props = defineProps<{
  open: boolean;
  postId?: string | null;
  initialContent?: string;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  submit: [payload: { content: string; postId?: string }];
}>();

let emojiPortalEl: HTMLDivElement | null = null;

function appendEditorEmojiTo() {
  if (typeof document === 'undefined') return document.body;
  if (!emojiPortalEl) {
    emojiPortalEl = document.createElement('div');
    emojiPortalEl.setAttribute('data-profile-emoji-portal', '');
    emojiPortalEl.style.position = 'relative';
    emojiPortalEl.style.zIndex = '100';
    document.body.appendChild(emojiPortalEl);
  }
  return emojiPortalEl;
}

onBeforeUnmount(() => {
  emojiPortalEl?.remove();
  emojiPortalEl = null;
});

const formState = reactive({ content: '' });
const submitting = ref(false);

const isEditMode = computed(() => Boolean(props.postId));
const editorKey = computed(() => (isEditMode.value ? `edit-${props.postId}` : 'create'));
const slideoverTitle = computed(() =>
  isEditMode.value ? 'Редактировать запись' : 'Новая запись',
);
const submitLabel = computed(() =>
  isEditMode.value ? 'Сохранить' : 'Опубликовать',
);

function reset() {
  formState.content = '';
  submitting.value = false;
}

function close() {
  emit('update:open', false);
}

function onOpenChange(val: boolean) {
  if (!val) reset();
  emit('update:open', val);
}

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    formState.content = isEditMode.value ? (props.initialContent ?? '') : '';
  },
);

function submit() {
  if (isWallContentEmpty(formState.content)) return;
  submitting.value = true;
  emit('submit', {
    content: formState.content,
    postId: props.postId ?? undefined,
  });
  reset();
  close();
  submitting.value = false;
}
</script>

<template>
  <USlideover
    :open="open"
    side="right"
    :title="slideoverTitle"
    :ui="{
      content: '!max-w-full sm:!max-w-2xl lg:!max-w-4xl xl:!max-w-5xl',
    }"
    @update:open="onOpenChange"
  >
    <template #body>
      <UForm v-if="open" :state="formState" class="space-y-4" @submit.prevent="submit">
        <UFormField label="Текст записи" name="content">
          <UEditor
            :key="editorKey"
            v-slot="{ editor }"
            v-model="formState.content"
            content-type="html"
            :extensions="wallEditorExtensions"
            :handlers="newsEditorHandlers"
            :ui="newsEditorSlideoverUi"
            placeholder="Что у вас нового? Введите : для смайликов"
            class="w-full min-h-56 rounded-lg border border-default overflow-hidden"
          >
            <UEditorEmojiMenu
              :editor="editor"
              :items="newsEditorEmojiMenuItems"
              :append-to="appendEditorEmojiTo"
              :suggestion="{ allowedPrefixes: null }"
              :options="{
                strategy: 'fixed',
                placement: 'bottom-start',
                offset: 8,
              }"
            />
            <UEditorToolbar
              :editor="editor"
              :items="wallEditorToolbarItems"
              class="sticky top-0 z-10 border-b border-default bg-default/95 backdrop-blur-sm px-2 py-1.5 overflow-x-auto"
            />
          </UEditor>
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <div class="flex justify-between gap-3 items-center w-full">
        <UButton
          type="button"
          color="neutral"
          variant="outline"
          size="xl"
          class="w-full justify-center"
          @click="close"
        >
          Отмена
        </UButton>
        <UButton
          type="button"
          size="xl"
          class="w-full justify-center"
          :loading="submitting"
          :disabled="isWallContentEmpty(formState.content)"
          @click="submit"
        >
          {{ submitLabel }}
        </UButton>
      </div>
    </template>
  </USlideover>
</template>

<style>
/* Меню смайликов поверх USlideover */
[data-profile-emoji-portal] [role='listbox'] {
  z-index: 100 !important;
}
</style>
