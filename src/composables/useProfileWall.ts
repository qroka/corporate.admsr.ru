import { ref } from 'vue';
import { apiSessionFetch } from './useAuthSession';
import { seedWallReactions } from './useNewsReactions';

/**
 * Стена профиля — на сервере (/api/profile_wall.php, таблица wall_posts, V14).
 * Писать на стену может любой вошедший сотрудник; править — автор; удалять —
 * автор, владелец стены или администратор (права считает сервер: canEdit/canDelete).
 * Текст простой, без HTML — выводить как текст, не через v-html.
 */
export type WallPost = {
  id: number;
  ownerId: number;
  author: { id: number; name: string; avatar_url: string };
  content: string;
  createdAt: string;
  updatedAt: string | null;
  canEdit: boolean;
  canDelete: boolean;
};

export const WALL_POST_MAX_LENGTH = 4000;
const PAGE_SIZE = 10;

// Записи до появления сервера лежали только в браузере автора — их никто, кроме него, не видел.
if (typeof window !== 'undefined') {
  try {
    window.localStorage.removeItem('profile-wall-posts:v2');
    window.localStorage.removeItem('profile-wall-posts:v1');
  } catch {
    /* хранилище недоступно */
  }
}

function normalize(raw: any): WallPost {
  seedWallReactions(raw?.id, raw?.reactions);
  return {
    id: Number(raw?.id) || 0,
    ownerId: Number(raw?.ownerId) || 0,
    author: {
      id: Number(raw?.author?.id) || 0,
      name: String(raw?.author?.name ?? 'Сотрудник'),
      avatar_url: String(raw?.author?.avatar_url ?? ''),
    },
    content: String(raw?.content ?? ''),
    createdAt: String(raw?.createdAt ?? ''),
    updatedAt: raw?.updatedAt ? String(raw.updatedAt) : null,
    canEdit: Boolean(raw?.canEdit),
    canDelete: Boolean(raw?.canDelete),
  };
}

export function useProfileWall() {
  const ownerId = ref(0);
  const posts = ref<WallPost[]>([]);
  const total = ref(0);
  const loading = ref(false);
  const loadingMore = ref(false);
  const error = ref('');
  let seq = 0;

  async function fetchPage(offset: number) {
    const json = await apiSessionFetch<any>(
      `/api/profile_wall.php?userId=${ownerId.value}&limit=${PAGE_SIZE}&offset=${offset}`,
    );
    if (!json?.success) throw new Error(json?.message || 'Не удалось загрузить стену');
    const data = json.data as any;
    return {
      items: Array.isArray(data?.items) ? data.items.map(normalize) : [],
      total: Number(data?.total) || 0,
    };
  }

  /** Загрузить стену сотрудника с начала. */
  async function load(userId: number) {
    const my = ++seq;
    ownerId.value = userId;
    posts.value = [];
    total.value = 0;
    error.value = '';
    if (!userId) return;
    loading.value = true;
    try {
      const page = await fetchPage(0);
      if (my !== seq) return;
      posts.value = page.items;
      total.value = page.total;
    } catch (e) {
      if (my === seq) error.value = e instanceof Error ? e.message : 'Не удалось загрузить стену';
    } finally {
      if (my === seq) loading.value = false;
    }
  }

  async function loadMore() {
    if (loadingMore.value || posts.value.length >= total.value) return;
    const my = seq;
    loadingMore.value = true;
    try {
      const page = await fetchPage(posts.value.length);
      if (my !== seq) return;
      const known = new Set(posts.value.map((p) => p.id));
      posts.value = [...posts.value, ...page.items.filter((p: WallPost) => !known.has(p.id))];
      total.value = page.total;
    } finally {
      if (my === seq) loadingMore.value = false;
    }
  }

  async function create(content: string): Promise<WallPost> {
    const json = await apiSessionFetch<any>('/api/profile_wall.php', {
      method: 'POST',
      json: { action: 'create', userId: ownerId.value, content },
    });
    if (!json?.success) throw new Error(json?.message || 'Не удалось опубликовать запись');
    const post = normalize(json.data);
    posts.value = [post, ...posts.value];
    total.value += 1;
    return post;
  }

  async function update(id: number, content: string) {
    const json = await apiSessionFetch<any>('/api/profile_wall.php', {
      method: 'POST',
      json: { action: 'update', id, content },
    });
    if (!json?.success) throw new Error(json?.message || 'Не удалось сохранить запись');
    const post = normalize(json.data);
    posts.value = posts.value.map((p) => (p.id === id ? post : p));
  }

  async function remove(id: number) {
    const json = await apiSessionFetch<any>('/api/profile_wall.php', {
      method: 'POST',
      json: { action: 'delete', id },
    });
    if (!json?.success) throw new Error(json?.message || 'Не удалось удалить запись');
    posts.value = posts.value.filter((p) => p.id !== id);
    total.value = Math.max(0, total.value - 1);
  }

  return { posts, total, loading, loadingMore, error, load, loadMore, create, update, remove };
}

/** «только что», «5 мин. назад», «вчера в 23:49», «8 ноября в 20:02», с годом — если не текущий. */
export function formatWallDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const now = new Date();
  const diffMin = Math.floor((now.getTime() - d.getTime()) / 60_000);
  if (diffMin < 1) return 'только что';
  if (diffMin < 60) return `${diffMin} мин. назад`;
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(d)) / 86_400_000);
  if (days === 0) return `сегодня в ${time}`;
  if (days === 1) return `вчера в ${time}`;
  const date = d.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'long',
    ...(d.getFullYear() !== now.getFullYear() ? { year: 'numeric' } : {}),
  });
  return `${date} в ${time}`;
}
