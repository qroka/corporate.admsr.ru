<script setup lang="ts">
/**
 * Запись на стене профиля в духе старого ВКонтакте: аватар автора слева, имя-ссылка
 * и дата, текст, реакции. «Изменить» — правка на месте (только автор), «Удалить» —
 * автор, владелец стены или администратор (флаги приходят с сервера).
 */
import { computed, nextTick, ref } from 'vue';
import NewsReactions from '../NewsReactions.vue';
import { formatWallDate, WALL_POST_MAX_LENGTH, type WallPost } from '../../composables/useProfileWall';
import { userAvatarSrc } from '../../utils/userName';

const props = defineProps<{
  post: WallPost;
  /** Сохранение правки: бросает ошибку — тогда поле правки остаётся открытым. */
  save: (id: number, content: string) => Promise<void>;
}>();

const emit = defineEmits<{ (e: 'delete', post: WallPost): void }>();

const editing = ref(false);
const draft = ref('');
const saving = ref(false);
const editField = ref<{ textareaRef?: HTMLTextAreaElement } | null>(null);

const authorLink = computed(() => `/profile/${props.post.author.id}`);
const canSave = computed(() => {
  const text = draft.value.trim();
  return !!text && text !== props.post.content && text.length <= WALL_POST_MAX_LENGTH;
});

async function startEdit() {
  draft.value = props.post.content;
  editing.value = true;
  await nextTick();
  editField.value?.textareaRef?.focus();
}

async function submitEdit() {
  if (!canSave.value || saving.value) return;
  saving.value = true;
  try {
    await props.save(props.post.id, draft.value.trim());
    editing.value = false;
  } catch {
    /* тост показал вызывающий; правка остаётся открытой */
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <article class="flex gap-3 py-4">
    <RouterLink :to="authorLink" class="shrink-0 self-start rounded-full focus-visible:outline-2 focus-visible:outline-primary" :aria-label="`Профиль: ${post.author.name}`">
      <UAvatar :src="userAvatarSrc(post.author)" :alt="post.author.name" size="xl" class="bg-elevated" />
    </RouterLink>

    <div class="flex-1 min-w-0 flex flex-col gap-1.5">
      <div class="flex flex-wrap items-baseline gap-x-2">
        <RouterLink :to="authorLink" class="text-sm font-semibold text-primary hover:underline">
          {{ post.author.name }}
        </RouterLink>
      </div>

      <div v-if="editing" class="flex flex-col gap-2">
        <UTextarea
          ref="editField"
          v-model="draft"
          :rows="2"
          autoresize
          :maxrows="12"
          :maxlength="WALL_POST_MAX_LENGTH"
          class="w-full"
          aria-label="Текст записи"
          @keydown.ctrl.enter.prevent="submitEdit"
          @keydown.meta.enter.prevent="submitEdit"
          @keydown.esc.prevent="editing = false"
        />
        <div class="flex gap-2">
          <UButton size="sm" color="primary" :loading="saving" :disabled="!canSave" @click="submitEdit">Сохранить</UButton>
          <UButton size="sm" color="neutral" variant="ghost" :disabled="saving" @click="editing = false">Отмена</UButton>
        </div>
      </div>
      <p v-else class="text-sm leading-6 text-default whitespace-pre-line break-words">{{ post.content }}</p>

      <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 pt-1">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
          <time :datetime="post.createdAt">{{ formatWallDate(post.createdAt) }}</time>
          <span v-if="post.updatedAt">изменено</span>
          <template v-if="!editing">
            <UButton v-if="post.canEdit" size="xs" color="neutral" variant="link" class="p-0" @click="startEdit">Изменить</UButton>
            <UButton v-if="post.canDelete" size="xs" color="neutral" variant="link" class="p-0" @click="emit('delete', post)">Удалить</UButton>
          </template>
        </div>
        <NewsReactions :news-id="post.id" target="wall" />
      </div>
    </div>
  </article>
</template>
