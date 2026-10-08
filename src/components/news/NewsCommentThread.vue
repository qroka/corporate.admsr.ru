<script setup lang="ts">
/**
 * Ветка: комментарий первого уровня и ответы. Свёрнута — виден самый
 * популярный ответ (и ваши новые ответы), «Развернуть» грузит всю ветку по времени.
 * Ответить можно на любой комментарий ветки — ответ встаёт в эту же ветку.
 */
import { computed, ref } from 'vue';
import NewsCommentItem from './NewsCommentItem.vue';
import NewsCommentComposer from './NewsCommentComposer.vue';
import { useNewsComments, type NewsComment } from '../../composables/useNewsComments';
import { useAppToast } from '../../composables/useAppToast';
import { isProfanityCancelled } from '../../composables/useProfanityGate';
import { plural } from '../../pages/Courses/courseDuration';

const props = withDefaults(
  defineProps<{
    root: NewsComment;
    /** Вся ветка уже загружена (переход из уведомления) — показать развёрнутой */
    initialReplies?: NewsComment[] | null;
    highlightId?: number | null;
  }>(),
  { initialReplies: null, highlightId: null },
);

const emit = defineEmits<{
  (e: 'update:root', root: NewsComment): void;
  (e: 'removed', id: number): void;
}>();

const { loadReplies, createComment, deleteComment } = useNewsComments();
const { success, error } = useAppToast();

const replies = ref<NewsComment[] | null>(props.initialReplies ? [...props.initialReplies] : null);
const expanded = ref(!!props.initialReplies);
/** Ваши ответы, пока ветка свёрнута, — чтобы сразу видеть отправленное */
const extra = ref<NewsComment[]>([]);
const loadingReplies = ref(false);
const replyTarget = ref<NewsComment | null>(null);

const confirmOpen = ref(false);
const confirmTarget = ref<NewsComment | null>(null);
const deleting = ref(false);

const visibleReplies = computed<NewsComment[]>(() => {
  if (expanded.value && replies.value) return replies.value;
  const top = props.root.topReply;
  const list = top && !extra.value.some((x) => x.id === top.id) ? [top] : [];
  return [...list, ...extra.value];
});

const hiddenCount = computed(() =>
  expanded.value ? 0 : Math.max(0, props.root.replyCount - visibleReplies.value.length),
);

const showBranch = computed(
  () => visibleReplies.value.length > 0 || hiddenCount.value > 0 || !!replyTarget.value,
);

async function expand() {
  if (loadingReplies.value) return;
  loadingReplies.value = true;
  try {
    replies.value = await loadReplies(props.root.id);
    extra.value = [];
    expanded.value = true;
  } catch (e) {
    error('Не удалось загрузить ответы', e instanceof Error ? e.message : undefined);
  } finally {
    loadingReplies.value = false;
  }
}

function collapse() {
  expanded.value = false;
  extra.value = [];
}

function startReply(c: NewsComment) {
  replyTarget.value = c;
}

function cancelReply() {
  replyTarget.value = null;
}

async function sendReply(text: string) {
  const target = replyTarget.value;
  if (!target) return;
  try {
    const created = await createComment({ replyToId: target.id, content: text });
    if (expanded.value && replies.value) replies.value = [...replies.value, created];
    else extra.value = [...extra.value, created];
    emit('update:root', { ...props.root, replyCount: props.root.replyCount + 1 });
    replyTarget.value = null;
  } catch (e) {
    if (!isProfanityCancelled(e)) error('Не удалось ответить', e instanceof Error ? e.message : undefined);
    throw e;
  }
}

function replaceReply(list: NewsComment[], c: NewsComment) {
  return list.map((x) => (x.id === c.id ? c : x));
}

function onSaved(c: NewsComment) {
  if (c.id === props.root.id) {
    emit('update:root', { ...c, topReply: props.root.topReply, replyCount: props.root.replyCount });
    return;
  }
  if (replies.value) replies.value = replaceReply(replies.value, c);
  extra.value = replaceReply(extra.value, c);
  if (props.root.topReply?.id === c.id) emit('update:root', { ...props.root, topReply: c });
}

function closeConfirm() {
  confirmOpen.value = false;
}

function askDelete(c: NewsComment) {
  confirmTarget.value = c;
  confirmOpen.value = true;
}

async function confirmDelete() {
  const c = confirmTarget.value;
  if (!c || deleting.value) return;
  deleting.value = true;
  try {
    const { soft } = await deleteComment(c.id);
    confirmOpen.value = false;
    if (c.id === props.root.id) {
      if (soft) emit('update:root', { ...props.root, deleted: true, content: '', canEdit: false, canDelete: false });
      else emit('removed', props.root.id);
    } else {
      if (replies.value) replies.value = replies.value.filter((x) => x.id !== c.id);
      extra.value = extra.value.filter((x) => x.id !== c.id);
      const replyCount = Math.max(0, props.root.replyCount - 1);
      if (props.root.deleted && replyCount === 0) emit('removed', props.root.id);
      else
        emit('update:root', {
          ...props.root,
          replyCount,
          topReply: props.root.topReply?.id === c.id ? null : props.root.topReply,
        });
    }
    success('Комментарий удалён');
  } catch (e) {
    error('Не удалось удалить комментарий', e instanceof Error ? e.message : undefined);
  } finally {
    deleting.value = false;
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <NewsCommentItem
      :comment="root"
      :highlighted="highlightId === root.id"
      @reply="startReply"
      @delete="askDelete"
      @saved="onSaved"
    />

    <div v-if="showBranch" class="ml-4 flex flex-col gap-3 border-l border-default pl-4 sm:ml-[42px]">
      <NewsCommentItem
        v-for="r in visibleReplies"
        :key="r.id"
        :comment="r"
        reply
        :highlighted="highlightId === r.id"
        @reply="startReply"
        @delete="askDelete"
        @saved="onSaved"
      />

      <UButton
        v-if="hiddenCount"
        type="button"
        color="neutral"
        variant="link"
        size="xs"
        icon="i-lucide-chevron-down"
        class="self-start p-0"
        :loading="loadingReplies"
        @click="expand"
      >
        Развернуть ещё {{ hiddenCount }} {{ plural(hiddenCount, ['ответ', 'ответа', 'ответов']) }}
      </UButton>
      <UButton
        v-else-if="expanded && root.replyCount > 1"
        type="button"
        color="neutral"
        variant="link"
        size="xs"
        icon="i-lucide-chevron-up"
        class="self-start p-0"
        @click="collapse"
      >
        Свернуть
      </UButton>

      <div v-if="replyTarget" class="flex flex-col gap-1.5">
        <p class="text-xs text-muted">Ответ: {{ replyTarget.author.name }}</p>
        <NewsCommentComposer
          :key="replyTarget.id"
          :send="sendReply"
          autofocus
          cancelable
          placeholder="Ваш ответ…"
          @cancel="cancelReply"
        />
      </div>
    </div>

    <UModal
      v-model:open="confirmOpen"
      title="Удалить комментарий?"
      :description="
        confirmTarget && !confirmTarget.rootId && root.replyCount > 0
          ? 'Ответы останутся, вместо текста будет «Комментарий удалён». Отменить это нельзя.'
          : 'Комментарий исчезнет вместе с реакциями. Отменить это нельзя.'
      "
    >
      <template #body>
        <p class="text-sm text-default whitespace-pre-line line-clamp-4 break-words">{{ confirmTarget?.content }}</p>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="deleting" @click="closeConfirm">Отмена</UButton>
          <UButton color="error" :loading="deleting" @click="confirmDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
