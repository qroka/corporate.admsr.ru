import { computed, ref, watch } from 'vue';

type NewsStatsState = {
  likes: Record<string, number>;
  views: Record<string, number>;
  liked: Record<string, boolean>;
};

const STORAGE_KEY = 'news-stats:v1';

const sharedLoaded = ref(false);
const sharedLikes = ref<Record<string, number>>({});
const sharedViews = ref<Record<string, number>>({});
const sharedLiked = ref<Record<string, boolean>>({});

function safeParseJson(raw: string | null): any {
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

function normalizeId(id: unknown): string {
  return String(id ?? '').trim();
}

function ensureInMap<T>(map: Record<string, T>, key: string, initial: T): Record<string, T> {
  if (!key) return map;
  if (Object.prototype.hasOwnProperty.call(map, key)) return map;
  return { ...map, [key]: initial };
}

function loadFromStorageOnce() {
  if (sharedLoaded.value) return;
  sharedLoaded.value = true;
  if (typeof window === 'undefined') return;

  const parsed = safeParseJson(window.localStorage.getItem(STORAGE_KEY)) as NewsStatsState | null;
  if (!parsed || typeof parsed !== 'object') return;

  sharedLikes.value = parsed.likes && typeof parsed.likes === 'object' ? parsed.likes : {};
  sharedViews.value = parsed.views && typeof parsed.views === 'object' ? parsed.views : {};
  sharedLiked.value = parsed.liked && typeof parsed.liked === 'object' ? parsed.liked : {};
}

function persistToStorage() {
  if (typeof window === 'undefined') return;
  const state: NewsStatsState = {
    likes: sharedLikes.value,
    views: sharedViews.value,
    liked: sharedLiked.value,
  };
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // ignore quota / private mode issues
  }
}

export function formatCountRu(n: number): string {
  const v = Number(n);
  if (!Number.isFinite(v)) return '0';
  return Math.max(0, Math.round(v)).toLocaleString('ru-RU');
}

export function useNewsStats() {
  loadFromStorageOnce();

  // Persist on change (lightweight for our small maps)
  watch([sharedLikes, sharedViews, sharedLiked], persistToStorage, { deep: true });

  function ensure(id: unknown) {
    const key = normalizeId(id);
    if (!key) return;
    sharedLikes.value = ensureInMap(sharedLikes.value, key, 0);
    sharedViews.value = ensureInMap(sharedViews.value, key, 0);
    sharedLiked.value = ensureInMap(sharedLiked.value, key, false);
  }

  function isLiked(id: unknown): boolean {
    const key = normalizeId(id);
    if (!key) return false;
    return Boolean(sharedLiked.value[key]);
  }

  function likesCount(id: unknown): number {
    const key = normalizeId(id);
    if (!key) return 0;
    return Number(sharedLikes.value[key] ?? 0) || 0;
  }

  function viewsCount(id: unknown): number {
    const key = normalizeId(id);
    if (!key) return 0;
    return Number(sharedViews.value[key] ?? 0) || 0;
  }

  function toggleLike(id: unknown) {
    const key = normalizeId(id);
    if (!key) return;
    ensure(key);
    const was = Boolean(sharedLiked.value[key]);
    const cur = likesCount(key);
    sharedLiked.value = { ...sharedLiked.value, [key]: !was };
    sharedLikes.value = { ...sharedLikes.value, [key]: was ? Math.max(0, cur - 1) : cur + 1 };
  }

  function incrementView(id: unknown) {
    const key = normalizeId(id);
    if (!key) return;
    ensure(key);
    const cur = viewsCount(key);
    sharedViews.value = { ...sharedViews.value, [key]: cur + 1 };
  }

  const state = computed(() => ({
    likes: sharedLikes.value,
    views: sharedViews.value,
    liked: sharedLiked.value,
  }));

  return { state, ensure, isLiked, likesCount, viewsCount, toggleLike, incrementView };
}

