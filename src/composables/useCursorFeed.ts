import { computed, onUnmounted, ref, type Ref } from 'vue';

export type CursorPageResponse = {
  items?: unknown[];
  nextCursor?: string | null;
  hasMore?: boolean;
};

function isAbortError(e: unknown): boolean {
  if (!e || typeof e !== 'object') return false;
  return (e as { name?: string }).name === 'AbortError';
}

export type UseCursorFeedOptions<T> = {
  /** Build request URL for a page (cursor null = first page). */
  buildUrl: (cursor: string | null) => string;
  mapItem: (raw: unknown) => T | null;
  getId: (item: T) => string;
  /** Optional external items ref (e.g. Pinia). If omitted, internal ref is used. */
  items?: Ref<T[]>;
  cursor?: Ref<string | null>;
  hasMore?: Ref<boolean>;
  hydrated?: Ref<boolean>;
};

/**
 * Generic cursor-paginated infinite list (same pattern as home news feed).
 */
export function useCursorFeed<T>(opts: UseCursorFeedOptions<T>) {
  const items = opts.items ?? ref<T[]>([]) as Ref<T[]>;
  const cursor = opts.cursor ?? ref<string | null>(null);
  const hasMore = opts.hasMore ?? ref(true);
  const hydrated = opts.hydrated ?? ref(false);

  const loading = ref(false);
  const initialLoading = ref(false);
  const error = ref<string | null>(null);

  let abortCtrl: AbortController | null = null;
  let inFlightKey: string | null = null;

  function cancelInFlight() {
    if (abortCtrl) {
      abortCtrl.abort();
      abortCtrl = null;
    }
    inFlightKey = null;
  }

  function dedupeAppend(existing: T[], batch: T[]): T[] {
    const seen = new Set(existing.map((x) => opts.getId(x)));
    const out = existing.slice();
    for (const row of batch) {
      const id = opts.getId(row);
      if (seen.has(id)) continue;
      seen.add(id);
      out.push(row);
    }
    return out;
  }

  async function fetchPage(pageCursor: string | null, signal: AbortSignal): Promise<CursorPageResponse> {
    const res = await fetch(opts.buildUrl(pageCursor), {
      cache: 'no-store',
      signal,
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = (await res.json()) as {
      success?: boolean;
      message?: string;
      data?: CursorPageResponse | unknown[];
    };
    if (!json?.success) {
      throw new Error(json?.message || 'Не удалось загрузить данные');
    }
    // Cursor mode returns { items, nextCursor, hasMore }
    if (json.data && typeof json.data === 'object' && !Array.isArray(json.data)) {
      return json.data as CursorPageResponse;
    }
    // Legacy full array — treat as single page
    const arr = Array.isArray(json.data) ? json.data : [];
    return { items: arr, nextCursor: null, hasMore: false };
  }

  function mapBatch(raw: unknown[]): T[] {
    return raw.map((x) => opts.mapItem(x)).filter(Boolean) as T[];
  }

  async function loadInitial(force = false) {
    if (!force && hydrated.value && items.value.length > 0) return;

    cancelInFlight();
    const ctrl = new AbortController();
    abortCtrl = ctrl;
    const key = 'init';
    inFlightKey = key;

    loading.value = true;
    initialLoading.value = true;
    error.value = null;
    items.value = [];
    cursor.value = null;
    hasMore.value = true;
    hydrated.value = false;

    try {
      const data = await fetchPage(null, ctrl.signal);
      if (inFlightKey !== key) return;
      const batch = mapBatch(Array.isArray(data.items) ? data.items : []);
      items.value = batch;
      cursor.value = data.nextCursor ?? null;
      hasMore.value = Boolean(data.hasMore) && Boolean(data.nextCursor);
      hydrated.value = true;
    } catch (e: unknown) {
      if (isAbortError(e) || ctrl.signal.aborted) return;
      error.value = e instanceof Error ? e.message : 'Не удалось загрузить данные';
      hasMore.value = false;
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
    if (loading.value || !hasMore.value || !hydrated.value) return;
    const cur = cursor.value;
    if (!cur) {
      hasMore.value = false;
      return;
    }

    const key = `more:${cur}`;
    if (inFlightKey === key) return;

    cancelInFlight();
    const ctrl = new AbortController();
    abortCtrl = ctrl;
    inFlightKey = key;
    loading.value = true;
    error.value = null;

    try {
      const data = await fetchPage(cur, ctrl.signal);
      if (inFlightKey !== key) return;
      const batch = mapBatch(Array.isArray(data.items) ? data.items : []);
      items.value = dedupeAppend(items.value, batch);
      cursor.value = data.nextCursor ?? null;
      hasMore.value = Boolean(data.hasMore) && Boolean(data.nextCursor);
    } catch (e: unknown) {
      if (isAbortError(e) || ctrl.signal.aborted) return;
      error.value = e instanceof Error ? e.message : 'Не удалось загрузить данные';
    } finally {
      if (inFlightKey === key) {
        loading.value = false;
        inFlightKey = null;
        if (abortCtrl === ctrl) abortCtrl = null;
      }
    }
  }

  async function refresh() {
    await loadInitial(true);
  }

  onUnmounted(() => {
    cancelInFlight();
  });

  return {
    items,
    cursor,
    hasMore,
    hydrated,
    loading,
    initialLoading,
    error,
    loadInitial,
    loadMore,
    refresh,
    cancelInFlight,
    sentinelEnabled: computed(
      () => hydrated.value && hasMore.value && !loading.value && !error.value,
    ),
  };
}
