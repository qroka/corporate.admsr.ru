<script setup lang="ts">
import { computed } from 'vue';
import type { WallPost } from '../../composables/useProfileWall';
import { formatWallDate } from '../../composables/useProfileWall';
import { newsEditorHtmlClass } from '../../composables/newsEditorHtmlClass';

const props = defineProps<{
  post: WallPost;
  isOwner: boolean;
}>();

const emit = defineEmits<{
  delete: [id: string];
  edit: [post: WallPost];
}>();

const menuItems = computed(() => {
  if (!props.isOwner) return [];
  return [[
    {
      label: 'Редактировать',
      icon: 'i-lucide-pencil',
      onSelect: () => emit('edit', props.post),
    },
    {
      label: 'Удалить',
      icon: 'i-lucide-trash-2',
      color: 'error' as const,
      onSelect: () => emit('delete', props.post.id),
    },
  ]];
});

const hasContent = computed(() => {
  const html = props.post.content || '';
  const text = html.replace(/<[^>]*>/g, '').replace(/&nbsp;/g, ' ').trim();
  return Boolean(text) || html.includes('<img');
});
</script>

<template>
  <article class="profile-wall-post">
    <header class="profile-wall-post__header">
      <div class="profile-wall-post__author">
        <UAvatar
          :src="post.authorAvatar"
          :alt="post.authorName"
          size="md"
          :ui="{ root: '!bg-accented shrink-0' }"
        />
        <div class="min-w-0">
          <span class="text-sm font-semibold text-highlighted truncate block">{{ post.authorName }}</span>
          <time class="text-xs text-muted" :datetime="post.createdAt">{{ formatWallDate(post.createdAt) }}</time>
        </div>
      </div>

      <UDropdownMenu v-if="isOwner && menuItems[0]?.length" :items="menuItems">
        <UButton
          type="button"
          color="neutral"
          variant="ghost"
          size="sm"
          icon="i-lucide-ellipsis"
          class="rounded-full"
          aria-label="Действия с постом"
        />
      </UDropdownMenu>
    </header>

    <div
      v-if="hasContent"
      class="profile-wall-post__content"
      :class="newsEditorHtmlClass"
      v-html="post.content"
    />
  </article>
</template>

<style scoped>
.profile-wall-post {
  background: var(--ui-bg-elevated);
  border: 1px solid color-mix(in srgb, var(--ui-border) 60%, transparent);
  border-radius: 1rem;
  padding: 1rem 1.125rem;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.profile-wall-post:hover {
  border-color: color-mix(in srgb, var(--ui-border) 100%, transparent);
  box-shadow: 0 4px 24px -8px color-mix(in srgb, var(--ui-text) 8%, transparent);
}

.profile-wall-post__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.profile-wall-post__author {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  min-width: 0;
}

.profile-wall-post__content {
  font-size: 0.9375rem;
  color: var(--ui-text-highlighted);
}

.profile-wall-post__content :deep(*:first-child) {
  margin-top: 0;
}

.profile-wall-post__content :deep(*:last-child) {
  margin-bottom: 0;
}

.profile-wall-post__content :deep(img) {
  margin-top: 0.5rem;
  margin-bottom: 0;
}
</style>
