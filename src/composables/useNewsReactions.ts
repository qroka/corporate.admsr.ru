import { ref } from 'vue';
import { apiSessionFetch } from './useAuthSession';
import { useAppToast } from './useAppToast';

/**
 * Реакции — на сервере, по одной каждого вида на сотрудника. Один набор ключей
 * на новости (news_reactions) и записи стены профиля (wall_post_reactions).
 * Порядок и ключи дублируются в Go:
 * backend/internal/handlers/news_reactions.go (NewsReactionKeys) — менять парами.
 */
export const NEWS_REACTIONS = [
  { key: 'like', emoji: '👍', label: 'Нравится' },
  { key: 'love', emoji: '❤️', label: 'Люблю' },
  { key: 'haha', emoji: '😄', label: 'Смешно' },
  { key: 'wow', emoji: '😮', label: 'Удивительно' },
  { key: 'sad', emoji: '😢', label: 'Грустно' },
  { key: 'fire', emoji: '🔥', label: 'Огонь' },
  { key: 'clap', emoji: '👏', label: 'Браво' },
  { key: 'party', emoji: '🎉', label: 'Праздник' },
] as const;

export type NewsReactionKey = (typeof NEWS_REACTIONS)[number]['key'];
export type NewsReactionSummary = { key: NewsReactionKey; count: number; mine: boolean };
export type NewsReactor = { id: number; name: string; avatar_url: string };
/** Чему ставим реакции: новости или записи на стене профиля. */
export type ReactionTarget = 'news' | 'wall';

const KNOWN = new Set<string>(NEWS_REACTIONS.map((r) => r.key));

// Раньше «лайкнул ли я» хранилось в браузере — теперь это знает сервер.
if (typeof window !== 'undefined') {
  try {
    window.localStorage.removeItem('news-likes:v1');
  } catch {
    /* хранилище недоступно */
  }
}

function normalize(raw: unknown): NewsReactionSummary[] {
  if (!Array.isArray(raw)) return [];
  const out: NewsReactionSummary[] = [];
  for (const r of raw) {
    const key = String((r as any)?.key ?? '');
    const count = Math.max(0, Number((r as any)?.count) || 0);
    if (!KNOWN.has(key) || count <= 0) continue;
    out.push({ key: key as NewsReactionKey, count, mine: Boolean((r as any)?.mine) });
  }
  return out;
}

function isLoggedIn(): boolean {
  try {
    const u = JSON.parse(localStorage.getItem('auth-user') || 'null');
    return Number(u?.id ?? 0) > 0;
  } catch {
    return false;
  }
}

/** Как сохранить реакцию и кого спросить «кто отреагировал» — у каждой цели свой эндпоинт. */
type ReactionApi = {
  react: (id: string, key: NewsReactionKey, active: boolean) => Promise<any>;
  reactorsUrl: (id: string, key: NewsReactionKey) => string;
};

function createReactionStore(api: ReactionApi) {
  /** Сводка: id → реакции с count > 0. */
  const summaries = ref<Record<string, NewsReactionSummary[]>>({});
  /** Записи, по которым идёт запрос: ответ ленты не должен затирать оптимистичное состояние. */
  const pending = new Set<string>();
  /** Кто отреагировал: `${id}:${key}` → список (грузится при наведении). */
  const reactorsCache = ref<Record<string, NewsReactor[]>>({});
  const reactorsLoading = ref<Record<string, boolean>>({});

  function seed(targetId: string | number, raw: unknown) {
    const id = String(targetId);
    if (!id || pending.has(id) || !Array.isArray(raw)) return;
    summaries.value = { ...summaries.value, [id]: normalize(raw) };
  }

  function use() {
    const { toast } = useAppToast();

    function reactionsOf(targetId: string | number): NewsReactionSummary[] {
      return summaries.value[String(targetId)] ?? [];
    }

    function countOf(targetId: string | number, key: NewsReactionKey): number {
      return reactionsOf(targetId).find((r) => r.key === key)?.count ?? 0;
    }

    function isMine(targetId: string | number, key: NewsReactionKey): boolean {
      return Boolean(reactionsOf(targetId).find((r) => r.key === key)?.mine);
    }

    /** Поставить / снять реакцию. Сразу меняем счётчик, при ошибке — откат. */
    async function toggle(targetId: string | number, key: NewsReactionKey) {
      const id = String(targetId);
      if (pending.has(id)) return;
      if (!isLoggedIn()) {
        toast.add({ title: 'Войдите, чтобы ставить реакции', color: 'neutral', icon: 'i-lucide-log-in' });
        return;
      }

      const before = reactionsOf(id);
      const active = !isMine(id, key);
      const next = NEWS_REACTIONS.map(({ key: k }) => {
        const cur = before.find((r) => r.key === k);
        if (k !== key) return cur ?? null;
        const count = Math.max(0, (cur?.count ?? 0) + (active ? 1 : -1));
        return count > 0 ? { key: k, count, mine: active } : null;
      }).filter(Boolean) as NewsReactionSummary[];

      summaries.value = { ...summaries.value, [id]: next };
      pending.add(id);
      try {
        const json = await api.react(id, key, active);
        if (!json?.success) throw new Error(json?.message || 'Не удалось сохранить реакцию');
        pending.delete(id);
        seed(id, json.data?.reactions);
        // Список «кто отреагировал» устарел — перезагрузим при следующем наведении.
        const cacheKey = `${id}:${key}`;
        if (reactorsCache.value[cacheKey]) {
          const { [cacheKey]: _drop, ...rest } = reactorsCache.value;
          reactorsCache.value = rest;
        }
      } catch (e) {
        pending.delete(id);
        summaries.value = { ...summaries.value, [id]: before };
        toast.add({
          title: 'Реакция не сохранилась',
          description: e instanceof Error ? e.message : undefined,
          color: 'error',
          icon: 'i-lucide-alert-circle',
        });
      }
    }

    async function loadReactors(targetId: string | number, key: NewsReactionKey) {
      const cacheKey = `${targetId}:${key}`;
      if (reactorsCache.value[cacheKey] || reactorsLoading.value[cacheKey] || !isLoggedIn()) return;
      reactorsLoading.value = { ...reactorsLoading.value, [cacheKey]: true };
      try {
        const json = await apiSessionFetch<any>(api.reactorsUrl(String(targetId), key));
        if (json?.success && Array.isArray(json.data)) {
          reactorsCache.value = { ...reactorsCache.value, [cacheKey]: json.data as NewsReactor[] };
        }
      } catch {
        /* подсказка просто останется без имён */
      } finally {
        reactorsLoading.value = { ...reactorsLoading.value, [cacheKey]: false };
      }
    }

    function reactorsOf(targetId: string | number, key: NewsReactionKey): NewsReactor[] | null {
      return reactorsCache.value[`${targetId}:${key}`] ?? null;
    }

    return { reactionsOf, countOf, isMine, toggle, loadReactors, reactorsOf };
  }

  return { seed, use };
}

const newsStore = createReactionStore({
  react: (id, key, active) =>
    apiSessionFetch<any>(`/api/news.php?id=${encodeURIComponent(id)}&action=react`, {
      method: 'POST',
      json: { reaction: key, active },
    }),
  reactorsUrl: (id, key) => `/api/news.php?id=${encodeURIComponent(id)}&action=reactors&reaction=${key}`,
});

const wallStore = createReactionStore({
  react: (id, key, active) =>
    apiSessionFetch<any>('/api/profile_wall.php', {
      method: 'POST',
      json: { action: 'react', id: Number(id), reaction: key, active },
    }),
  reactorsUrl: (id, key) => `/api/profile_wall.php?action=reactors&id=${encodeURIComponent(id)}&reaction=${key}`,
});

/** Принять реакции из любого ответа news.php (лента, список, карточка). */
export const seedNewsReactions = newsStore.seed;
/** Принять реакции из ответа profile_wall.php. */
export const seedWallReactions = wallStore.seed;

export function useNewsReactions() {
  return newsStore.use();
}

export function useReactions(target: ReactionTarget) {
  return (target === 'wall' ? wallStore : newsStore).use();
}
