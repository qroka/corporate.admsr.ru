import { apiSessionFetch } from './useAuthSession';
import { seedCommentReactions } from './useNewsReactions';

/**
 * Комментарии к новостям — /api/news_comments.php, таблицы news_comments и
 * news_comment_reactions (V18). Два уровня, как во ВКонтакте: комментарий и
 * ответы в его ветке; ответ на ответ — в той же ветке с «в ответ …».
 * Популярность — число всех реакций, при равенстве выше более новый (считает сервер).
 * Текст простой, выводить как текст (не v-html).
 */
export const NEWS_COMMENT_MAX_LENGTH = 2000;

export type NewsComment = {
  id: number;
  newsId: number;
  /** null — комментарий к новости; иначе — ответ в ветке rootId */
  rootId: number | null;
  /** «В ответ …» — только для ответов на ответы */
  replyTo: { id: number | null; userId: number; name: string } | null;
  author: { id: number; name: string; avatar_url: string };
  content: string;
  /** Удалён, но в ветке остались ответы */
  deleted: boolean;
  createdAt: string;
  updatedAt: string | null;
  score: number;
  /** Ответов в ветке (для комментария первого уровня) */
  replyCount: number;
  /** Самый популярный ответ (для комментария первого уровня) */
  topReply: NewsComment | null;
  canEdit: boolean;
  canDelete: boolean;
};

export type NewsCommentsSort = 'popular' | 'new';

export function normalizeComment(raw: any): NewsComment {
  seedCommentReactions(raw?.id, raw?.reactions);
  return {
    id: Number(raw?.id) || 0,
    newsId: Number(raw?.newsId) || 0,
    rootId: raw?.rootId != null ? Number(raw.rootId) : null,
    replyTo: raw?.replyTo
      ? {
          id: raw.replyTo.id != null ? Number(raw.replyTo.id) : null,
          userId: Number(raw.replyTo.userId) || 0,
          name: String(raw.replyTo.name ?? ''),
        }
      : null,
    author: {
      id: Number(raw?.author?.id) || 0,
      name: String(raw?.author?.name ?? 'Сотрудник'),
      avatar_url: String(raw?.author?.avatar_url ?? ''),
    },
    content: String(raw?.content ?? ''),
    deleted: Boolean(raw?.deleted),
    createdAt: String(raw?.createdAt ?? ''),
    updatedAt: raw?.updatedAt ? String(raw.updatedAt) : null,
    score: Number(raw?.score) || 0,
    replyCount: Number(raw?.replyCount) || 0,
    topReply: raw?.topReply ? normalizeComment(raw.topReply) : null,
    canEdit: Boolean(raw?.canEdit),
    canDelete: Boolean(raw?.canDelete),
  };
}

function fail(json: { message?: string } | null | undefined, fallback: string): never {
  throw new Error(json?.message || fallback);
}

const API_URL = '/api/news_comments.php';


async function loadList(newsId: string | number, sort: NewsCommentsSort, offset = 0) {
  const json = await apiSessionFetch<{ total?: number; rootsTotal?: number; items?: unknown[] }>(
    `${API_URL}?newsId=${encodeURIComponent(String(newsId))}&sort=${sort}&offset=${offset}`,
  );
  if (!json?.success) fail(json, 'Не удалось загрузить комментарии');
  return {
    total: Number(json.data?.total) || 0,
    rootsTotal: Number(json.data?.rootsTotal) || 0,
    items: Array.isArray(json.data?.items) ? json.data!.items!.map(normalizeComment) : [],
  };
}

async function loadReplies(rootId: number): Promise<NewsComment[]> {
  const json = await apiSessionFetch<unknown[]>(`${API_URL}?action=replies&rootId=${rootId}`);
  if (!json?.success) fail(json, 'Не удалось загрузить ответы');
  return Array.isArray(json.data) ? json.data.map(normalizeComment) : [];
}

async function loadThread(id: number): Promise<{ root: NewsComment; replies: NewsComment[] }> {
  const json = await apiSessionFetch<{ root?: unknown; replies?: unknown[] }>(`${API_URL}?action=thread&id=${id}`);
  if (!json?.success || !json.data?.root) fail(json, 'Комментарий не найден');
  return {
    root: normalizeComment(json.data!.root),
    replies: Array.isArray(json.data!.replies) ? json.data!.replies!.map(normalizeComment) : [],
  };
}

async function createComment(input: { newsId?: number | string; replyToId?: number; content: string }) {
  const json = await apiSessionFetch(API_URL, {
    method: 'POST',
    json: {
      action: 'create',
      newsId: input.newsId != null ? Number(input.newsId) : undefined,
      replyToId: input.replyToId,
      content: input.content.trim(),
    },
  });
  if (!json?.success || !json.data) fail(json, 'Не удалось опубликовать комментарий');
  return normalizeComment(json.data);
}

async function updateComment(id: number, content: string) {
  const json = await apiSessionFetch(API_URL, { method: 'POST', json: { action: 'update', id, content: content.trim() } });
  if (!json?.success || !json.data) fail(json, 'Не удалось сохранить комментарий');
  return normalizeComment(json.data);
}

/** soft — у комментария остались ответы, он показывается как «Комментарий удалён». */
async function deleteComment(id: number): Promise<{ soft: boolean }> {
  const json = await apiSessionFetch<{ soft?: boolean }>(API_URL, { method: 'POST', json: { action: 'delete', id } });
  if (!json?.success) fail(json, 'Не удалось удалить комментарий');
  return { soft: Boolean(json.data?.soft) };
}

export function useNewsComments() {
  return {
    loadList,
    loadReplies,
    loadThread,
    createComment,
    updateComment,
    deleteComment,
  };
}
