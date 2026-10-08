<script setup lang="ts">
/**
 * Комментарии на странице новости: «Популярные» / «Новые», в каждой ветке —
 * самый популярный ответ и «Развернуть». focusCommentId — переход из
 * уведомления: ветка с этим комментарием закрепляется сверху развёрнутой.
 */
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import type { TabsItem } from '@nuxt/ui';
import NewsCommentThread from './NewsCommentThread.vue';
import NewsCommentComposer from './NewsCommentComposer.vue';
import { useNewsComments, type NewsComment, type NewsCommentsSort } from '../../composables/useNewsComments';
import { useAppToast } from '../../composables/useAppToast';

const props = withDefaults(
  defineProps<{ newsId: string | number; focusCommentId?: number | null; scrollIntoView?: boolean }>(),
  { focusCommentId: null, scrollIntoView: false },
);

const { loadList, loadThread, createComment } = useNewsComments();
const { error } = useAppToast();

const sortTabs: TabsItem[] = [
  { label: 'Популярные', value: 'popular' },
  { label: 'Новые', value: 'new' },
];
const sort = ref<NewsCommentsSort>('popular');
const items = ref<NewsComment[]>([]);
const total = ref(0);
const rootsTotal = ref(0);
const loading = ref(true);
const loadingMore = ref(false);
const loadError = ref('');
const pinned = ref<{ root: NewsComment; replies: NewsComment[] } | null>(null);
const sectionEl = ref<HTMLElement | null>(null);
let seq = 0;

const listItems = computed(() => items.value.filter((c) => c.id !== pinned.value?.root.id));
const hasMore = computed(() => items.value.length < rootsTotal.value);

async function load() {
  const my = ++seq;
  loading.value = true;
  loadError.value = '';
  try {
    const res = await loadList(props.newsId, sort.value, 0);
    if (my !== seq) return;
    items.value = res.items;
    total.value = res.total;
    rootsTotal.value = res.rootsTotal;
  } catch (e) {
    if (my !== seq) return;
    loadError.value = e instanceof Error ? e.message : 'Не удалось загрузить комментарии';
  } finally {
    if (my === seq) loading.value = false;
  }
}

async function loadMore() {
  if (loadingMore.value) return;
  loadingMore.value = true;
  try {
    const res = await loadList(props.newsId, sort.value, items.value.length);
    const seen = new Set(items.value.map((c) => c.id));
    items.value = [...items.value, ...res.items.filter((c) => !seen.has(c.id))];
    total.value = res.total;
    rootsTotal.value = res.rootsTotal;
  } catch (e) {
    error('Не удалось загрузить комментарии', e instanceof Error ? e.message : undefined);
  } finally {
    loadingMore.value = false;
  }
}

async function loadPinned() {
  if (!props.focusCommentId) return;
  try {
    pinned.value = await loadThread(props.focusCommentId);
    await nextTick();
    document.getElementById(`comment-${props.focusCommentId}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  } catch (e) {
    error('Комментарий не найден', e instanceof Error ? e.message : 'Возможно, его удалили.');
  }
}

watch(sort, () => void load());
watch(
  () => props.newsId,
  () => {
    pinned.value = null;
    void load();
    void loadPinned();
  },
);

onMounted(async () => {
  void loadPinned();
  await load();
  if (props.scrollIntoView && !props.focusCommentId) {
    await nextTick();
    sectionEl.value?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }
});

function updateRoot(c: NewsComment) {
  const old = items.value.find((x) => x.id === c.id) ?? (pinned.value?.root.id === c.id ? pinned.value.root : null);
  if (old) total.value = Math.max(0, total.value + c.replyCount - old.replyCount - (c.deleted && !old.deleted ? 1 : 0));
  items.value = items.value.map((x) => (x.id === c.id ? c : x));
  if (pinned.value?.root.id === c.id) pinned.value = { ...pinned.value, root: c };
}

function removeRoot(id: number) {
  const old = items.value.find((x) => x.id === id) ?? (pinned.value?.root.id === id ? pinned.value.root : null);
  if (old && !old.deleted) total.value = Math.max(0, total.value - 1);
  if (items.value.some((x) => x.id === id)) rootsTotal.value = Math.max(0, rootsTotal.value - 1);
  items.value = items.value.filter((x) => x.id !== id);
  if (pinned.value?.root.id === id) pinned.value = null;
}

async function sendRoot(text: string) {
  try {
    const created = await createComment({ newsId: props.newsId, content: text });
    items.value = [created, ...items.value];
    total.value += 1;
    rootsTotal.value += 1;
  } catch (e) {
    error('Не удалось опубликовать комментарий', e instanceof Error ? e.message : undefined);
    throw e;
  }
}
</script>

<template>
  <section ref="sectionEl" class="flex flex-col gap-4 scroll-mt-4" aria-labelledby="news-comments-title">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 id="news-comments-title" class="text-lg font-semibold text-highlighted">
        Комментарии
        <span v-if="total" class="ml-1 font-normal text-muted tabular-nums">{{ total }}</span>
      </h2>
      <UTabs
        v-model="sort"
        :items="sortTabs"
        variant="pill"
        size="xs"
        color="neutral"
        :content="false"
        activation-mode="manual"
        aria-label="Порядок комментариев"
      />
    </div>

    <!-- Пришли по «Прокомментировать» (#comments) — курсор сразу в поле. -->
    <NewsCommentComposer :send="sendRoot" :autofocus="scrollIntoView && !focusCommentId" />

    <template v-if="pinned">
      <NewsCommentThread
        :key="`pinned-${pinned.root.id}`"
        :root="pinned.root"
        :initial-replies="pinned.replies"
        :highlight-id="focusCommentId"
        @update:root="updateRoot"
        @removed="removeRoot"
      />
      <USeparator />
    </template>

    <div v-if="loading" class="flex flex-col gap-4">
      <div v-for="n in 3" :key="n" class="flex gap-2.5">
        <USkeleton class="size-8 rounded-full shrink-0" />
        <div class="flex-1 flex flex-col gap-1.5">
          <USkeleton class="h-3.5 w-40" />
          <USkeleton class="h-3.5 w-5/6" />
        </div>
      </div>
    </div>

    <div v-else-if="loadError" class="flex flex-wrap items-center gap-3 text-sm">
      <span class="text-error">{{ loadError }}</span>
      <UButton type="button" color="neutral" variant="outline" size="xs" icon="i-lucide-refresh-cw" @click="load">
        Повторить
      </UButton>
    </div>

    <UEmpty
      v-else-if="!listItems.length && !pinned"
      variant="naked"
      icon="i-lucide-message-circle"
      title="Комментариев пока нет"
      description="Напишите первым, что думаете о новости."
      class="py-6"
    />

    <template v-else>
      <NewsCommentThread
        v-for="c in listItems"
        :key="c.id"
        :root="c"
        @update:root="updateRoot"
        @removed="removeRoot"
      />
      <UButton
        v-if="hasMore"
        type="button"
        color="neutral"
        variant="outline"
        size="sm"
        class="self-center"
        :loading="loadingMore"
        @click="loadMore"
      >
        Показать ещё
      </UButton>
    </template>
  </section>
</template>
