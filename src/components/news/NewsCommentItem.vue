<script setup lang="ts">
/**
 * Один комментарий или ответ: автор, «в ответ …», текст, время, реакции,
 * «Ответить», «Изменить» (автор, правка на месте), «Удалить» (автор или
 * редактор новостей — флаги приходят с сервера).
 */
import { computed, nextTick, ref } from 'vue';
import NewsReactions from '../NewsReactions.vue';
import {
  NEWS_COMMENT_MAX_LENGTH,
  useNewsComments,
  type NewsComment,
} from '../../composables/useNewsComments';
import { formatWallDate } from '../../composables/useProfileWall';
import { userAvatarSrc } from '../../utils/userName';
import { useAppToast } from '../../composables/useAppToast';

const props = withDefaults(defineProps<{ comment: NewsComment; reply?: boolean; highlighted?: boolean }>(), {
  reply: false,
  highlighted: false,
});

const emit = defineEmits<{
  (e: 'reply', comment: NewsComment): void;
  (e: 'delete', comment: NewsComment): void;
  (e: 'saved', comment: NewsComment): void;
}>();

const { updateComment } = useNewsComments();
const { error } = useAppToast();

const editing = ref(false);
const draft = ref('');
const saving = ref(false);
const editField = ref<{ textareaRef?: HTMLTextAreaElement } | null>(null);

const authorLink = computed(() => `/profile/${props.comment.author.id}`);
const canSave = computed(() => {
  const s = draft.value.trim();
  return !!s && s !== props.comment.content && s.length <= NEWS_COMMENT_MAX_LENGTH;
});

async function startEdit() {
  draft.value = props.comment.content;
  editing.value = true;
  await nextTick();
  editField.value?.textareaRef?.focus();
}

function cancelEdit() {
  editing.value = false;
}

async function submitEdit() {
  if (!canSave.value || saving.value) return;
  saving.value = true;
  try {
    emit('saved', await updateComment(props.comment.id, draft.value));
    editing.value = false;
  } catch (e) {
    error('Не удалось сохранить комментарий', e instanceof Error ? e.message : undefined);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <article
    :id="`comment-${comment.id}`"
    class="flex gap-2.5 rounded-md transition-shadow"
    :class="highlighted ? 'ring-2 ring-primary/60 ring-offset-4 ring-offset-default' : ''"
  >
    <RouterLink
      v-if="!comment.deleted"
      :to="authorLink"
      class="shrink-0 self-start rounded-full focus-visible:outline-2 focus-visible:outline-primary"
      :aria-label="`Профиль: ${comment.author.name}`"
    >
      <UAvatar :src="userAvatarSrc(comment.author)" :alt="comment.author.name" :size="reply ? 'sm' : 'md'" />
    </RouterLink>
    <span v-else class="shrink-0 grid place-items-center rounded-full bg-elevated" :class="reply ? 'size-7' : 'size-8'">
      <UIcon name="i-lucide-message-circle-off" class="size-4 text-dimmed" aria-hidden="true" />
    </span>

    <div class="flex-1 min-w-0 flex flex-col gap-1">
      <p v-if="comment.deleted" class="text-sm italic text-muted py-1.5">Комментарий удалён</p>

      <template v-else>
        <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-sm">
          <RouterLink :to="authorLink" class="font-semibold text-highlighted hover:underline">
            {{ comment.author.name }}
          </RouterLink>
          <span v-if="comment.replyTo" class="inline-flex items-center gap-1 text-xs text-muted">
            <UIcon name="i-lucide-reply" class="size-3.5" aria-hidden="true" />
            <span class="sr-only">в ответ</span>{{ comment.replyTo.name }}
          </span>
        </div>

        <div v-if="editing" class="flex flex-col gap-2">
          <UTextarea
            ref="editField"
            v-model="draft"
            :rows="1"
            autoresize
            :maxrows="8"
            :maxlength="NEWS_COMMENT_MAX_LENGTH"
            class="w-full"
            aria-label="Текст комментария"
            @keydown.ctrl.enter.prevent="submitEdit"
            @keydown.meta.enter.prevent="submitEdit"
            @keydown.esc.prevent="cancelEdit"
          />
          <div class="flex gap-2">
            <UButton size="xs" color="primary" :loading="saving" :disabled="!canSave" @click="submitEdit">Сохранить</UButton>
            <UButton size="xs" color="neutral" variant="ghost" :disabled="saving" @click="cancelEdit">Отмена</UButton>
          </div>
        </div>
        <p v-else class="text-sm leading-5 text-default whitespace-pre-line break-words">{{ comment.content }}</p>

        <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
            <time :datetime="comment.createdAt">{{ formatWallDate(comment.createdAt) }}</time>
            <span v-if="comment.updatedAt">изменено</span>
            <template v-if="!editing">
              <UButton size="xs" color="neutral" variant="link" class="p-0" @click="emit('reply', comment)">Ответить</UButton>
              <UButton v-if="comment.canEdit" size="xs" color="neutral" variant="link" class="p-0" @click="startEdit">Изменить</UButton>
              <UButton v-if="comment.canDelete" size="xs" color="neutral" variant="link" class="p-0" @click="emit('delete', comment)">Удалить</UButton>
            </template>
          </div>
          <NewsReactions :news-id="comment.id" target="comment" />
        </div>
      </template>
    </div>
  </article>
</template>
