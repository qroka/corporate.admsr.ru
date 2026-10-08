<script setup lang="ts">
/**
 * Окошко девблога при первом заходе после публикации (ADR-052): последний
 * опубликованный девблог, который сотрудник ещё не закрыл. Реакции и комментарий —
 * прямо здесь (это обычная новость «Девблог»). Закрыли любым способом — больше
 * не показываем (отметка на сервере, на всех устройствах).
 */
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import NewsReactions from './NewsReactions.vue';
import DevblogVersionBadge from './DevblogVersionBadge.vue';
import DevblogCover from './DevblogCover.vue';
import NewsCommentComposer from './news/NewsCommentComposer.vue';
import { dismissDevblog, loadPendingDevblog, DEVBLOG_DEFAULT_COVER, type DevblogPending } from '../composables/useDevblog';
import { useNewsComments } from '../composables/useNewsComments';
import { newsEditorHtmlClass } from '../composables/newsEditorHtmlClass';
import { formatNewsDate } from '../composables/useNewsData';
import { useAppToast } from '../composables/useAppToast';
import { isProfanityCancelled } from '../composables/useProfanityGate';

const router = useRouter();
const { createComment } = useNewsComments();
const { success, error } = useAppToast();

const post = ref<DevblogPending | null>(null);
const open = ref(false);
const commented = ref(false);
let dismissed = false;

const cover = computed(() => post.value?.imagePath || DEVBLOG_DEFAULT_COVER);
const dateLabel = computed(() => (post.value?.date ? formatNewsDate(post.value.date) : ''));

onMounted(async () => {
  try {
    post.value = await loadPendingDevblog();
    open.value = !!post.value;
  } catch {
    /* без окошка портал работает как обычно */
  }
});

function markDismissed() {
  if (dismissed || !post.value) return;
  dismissed = true;
  void dismissDevblog(post.value.newsId).catch(() => {
    // Не отметилось — окошко покажется при следующем входе, это не страшно.
    dismissed = false;
  });
}

function onOpenChange(value: boolean) {
  open.value = value;
  if (!value) markDismissed();
}

function close() {
  onOpenChange(false);
}

function openComments() {
  if (!post.value) return;
  const id = post.value.newsId;
  onOpenChange(false);
  void router.push(`/news/${id}#comments`);
}

async function sendComment(text: string) {
  if (!post.value) return;
  try {
    await createComment({ newsId: post.value.newsId, content: text });
    commented.value = true;
    success('Комментарий опубликован');
  } catch (e) {
    if (!isProfanityCancelled(e)) error('Не удалось опубликовать комментарий', e instanceof Error ? e.message : undefined);
    throw e;
  }
}
</script>

<template>
  <UModal
    v-if="post"
    :open="open"
    :title="post.title"
    description="Девблог — что нового на портале"
    :ui="{ content: 'sm:max-w-2xl', body: 'flex flex-col gap-5' }"
    @update:open="onOpenChange"
  >
    <template v-if="post.version" #description>
      <DevblogVersionBadge :version="post.version" />
    </template>

    <template #body>
      <!-- Обложка — между версией (в шапке окна) и текстом, целиком, как на странице новости. -->
      <DevblogCover :src="cover" :alt="post.title" />

      <div class="flex flex-wrap items-center gap-2 text-sm text-muted">
        <UBadge color="primary" variant="subtle">Девблог</UBadge>
        <span v-if="dateLabel">{{ dateLabel }}</span>
      </div>

      <!-- HTML построен из Markdown редактором на странице девблога (только его схема — без произвольного HTML). -->
      <div :class="[newsEditorHtmlClass, 'sm:px-0 text-base leading-relaxed text-default']" v-html="post.html" />

      <USeparator />

      <NewsReactions :news-id="post.newsId" />

      <div class="flex flex-col gap-2">
        <p v-if="commented" class="flex items-center gap-1.5 text-sm text-success">
          <UIcon name="i-lucide-check" class="size-4" aria-hidden="true" />
          Комментарий опубликован — ответы придут в колокольчик.
        </p>
        <NewsCommentComposer v-else :send="sendComment" />
      </div>
    </template>

    <template #footer>
      <div class="flex w-full flex-wrap justify-end gap-2">
        <UButton color="neutral" variant="outline" icon="i-lucide-message-circle" @click="openComments">
          Все комментарии
        </UButton>
        <UButton color="primary" @click="close">Закрыть</UButton>
      </div>
    </template>
  </UModal>
</template>
