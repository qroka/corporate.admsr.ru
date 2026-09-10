import { computed, onUnmounted, ref } from 'vue';
import {
  mapApiRow,
  type NewsRecord,
  useNewsData,
} from './useNewsData';
import { useNewsFeedStore, type NewsFeedTab } from '../stores/newsFeed';

export { useFeedSentinel } from './useFeedSentinel';

export const NEWS_FEED_LIMIT = 8;

type FeedPageResponse = {
  items?: unknown[];
  nextCursor?: string | null;
  hasMore?: boolean;
};

type LoadOpts = {
  category?: string | null;
  force?: boolean;
};

function isAbortError(e: unknown): boolean {
  if (!e || typeof e !== 'object') return false;
  const name = (e as { name?: string }).name;
  return name === 'AbortError';
}

/**
 * Infinite-scroll news feed for the home page.
 * Keeps pagination state in Pinia so returning from /news/:id restores items + scroll.
 * Does not replace useNewsData (list/detail/admin/kiosk).
 */
export function useNewsFeed() {
  const store = useNewsFeedStore();
  const { upsertItems, getById } = useNewsData();

  const loading = ref(false);
  const initialLoading = ref(false);
  const error = ref<string | null>(null);

  let abortCtrl: AbortController | null = null;
  let inFlightKey: string | null = null;

  const items = computed(() => store.items);
  const hasMore = computed(() => store.hasMore);
  const cursor = computed(() => store.cursor);

  function cancelInFlight() {
    if (abortCtrl) {
      abortCtrl.abort();
      abortCtrl = null;
    }
    inFlightKey = null;
  }

  function dedupeAppend(existing: NewsRecord[], batch: NewsRecord[]): NewsRecord[] {
    const seen = new Set(existing.map((x) => x.id));
    const out = existing.slice();
    for (const row of batch) {
      if (seen.has(row.id)) continue;
      seen.add(row.id);
      out.push(row);
    }
    return out;
  }

  async function fetchPage(opts: {
    cursor: string | null;
    category?: string | null;
    signal: AbortSignal;
  }): Promise<FeedPageResponse> {
    const params = new URLSearchParams();
    params.set('limit', String(NEWS_FEED_LIMIT));
    if (opts.cursor) params.set('cursor', opts.cursor);
    const cat = (opts.category ?? '').trim();
    if (cat) params.set('category', cat);

    const res = await fetch(`/api/news.php?${params.toString()}`, {
      cache: 'no-store',
      signal: opts.signal,
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = (await res.json()) as {
      success?: boolean;
      message?: string;
      data?: FeedPageResponse;
    };
    if (!json?.success || !json.data || typeof json.data !== 'object') {
      throw new Error(json?.message || 'Не удалось загрузить новости');
    }
    return json.data;
  }

  function mapItems(raw: unknown[]): NewsRecord[] {
    return raw
      .map((x) => mapApiRow(x as Record<string, unknown>))
      .filter(Boolean) as NewsRecord[];
  }

  async function loadInitial(opts?: LoadOpts) {
    const category = opts?.category ?? null;
    const catKey = (category ?? '').trim() || null;

    if (
      !opts?.force &&
      store.hydrated &&
      store.items.length > 0 &&
      (store.categoryFilter ?? null) === catKey
    ) {
      return;
    }

    cancelInFlight();
    const ctrl = new AbortController();
    abortCtrl = ctrl;
    const key = `init:${catKey ?? ''}`;
    inFlightKey = key;

    loading.value = true;
    initialLoading.value = true;
    error.value = null;

    store.items = [];
    store.cursor = null;
    store.hasMore = true;
    store.categoryFilter = catKey;
    store.hydrated = false;

    try {
      const data = await fetchPage({
        cursor: null,
        category: catKey,
        signal: ctrl.signal,
      });
      if (inFlightKey !== key) return;

      const batch = mapItems(Array.isArray(data.items) ? data.items : []);
      store.items = batch;
      store.cursor = data.nextCursor ?? null;
      store.hasMore = Boolean(data.hasMore) && Boolean(data.nextCursor);
      store.hydrated = true;
      upsertItems(batch);
    } catch (e: unknown) {
      if (isAbortError(e) || ctrl.signal.aborted) return;
      error.value = 'Не удалось загрузить новости';
      store.hasMore = false;
    } finally {
      if (inFlightKey === key) {
        loading.value = false;
        initialLoading.value = false;
        inFlightKey = null;
        if (abortCtrl === ctrl) abortCtrl = null;
      }
    }
  }

  async function loadMore() {
    if (loading.value || !store.hasMore) return;
    if (!store.hydrated) return;

    const catKey = store.categoryFilter;
    const cur = store.cursor;
    if (!cur) {
      store.hasMore = false;
      return;
    }

    const key = `more:${catKey ?? ''}:${cur}`;
    if (inFlightKey === key) return;

    cancelInFlight();
    const ctrl = new AbortController();
    abortCtrl = ctrl;
    inFlightKey = key;

    loading.value = true;
    error.value = null;

    try {
      const data = await fetchPage({
        cursor: cur,
        category: catKey,
        signal: ctrl.signal,
      });
      if (inFlightKey !== key) return;

      const batch = mapItems(Array.isArray(data.items) ? data.items : []);
      store.items = dedupeAppend(store.items, batch);
      store.cursor = data.nextCursor ?? null;
      store.hasMore = Boolean(data.hasMore) && Boolean(data.nextCursor);
      upsertItems(batch);
    } catch (e: unknown) {
      if (isAbortError(e) || ctrl.signal.aborted) return;
      error.value = 'Не удалось загрузить новости';
    } finally {
      if (inFlightKey === key) {
        loading.value = false;
        inFlightKey = null;
        if (abortCtrl === ctrl) abortCtrl = null;
      }
    }
  }

  async function refresh(opts?: LoadOpts) {
    await loadInitial({
      ...opts,
      force: true,
      category: opts?.category ?? store.categoryFilter,
    });
  }

  function setActiveTab(tab: NewsFeedTab) {
    store.activeTab = tab;
  }

  function saveScrollTop(y: number) {
    store.setScrollTop(y);
  }

  function resolveLikesViews(id: string): { likes: number; views: number } {
    const fromShared = getById(id);
    const fromFeed = store.items.find((x) => x.id === id);
    return {
      likes: fromShared?.likes ?? fromFeed?.likes ?? 0,
      views: fromShared?.views ?? fromFeed?.views ?? 0,
    };
  }

  onUnmounted(() => {
    cancelInFlight();
  });

  return {
    items,
    cursor,
    hasMore,
    loading,
    initialLoading,
    error,
    scrollTop: computed(() => store.scrollTop),
    activeTab: computed({
      get: () => store.activeTab,
      set: (v: NewsFeedTab) => {
        store.activeTab = v;
      },
    }),
    categoryFilter: computed(() => store.categoryFilter),
    hydrated: computed(() => store.hydrated),
    loadInitial,
    loadMore,
    refresh,
    setActiveTab,
    saveScrollTop,
    resolveLikesViews,
    cancelInFlight,
  };
}
