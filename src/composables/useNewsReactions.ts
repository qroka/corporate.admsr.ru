import { ref } from 'vue';
import { useNewsData } from './useNewsData';

const LIKES_KEY = 'news-likes:v1';
const VIEW_SESSION_KEY = 'news-viewed:v1';

// Хранилище лайков (localStorage) — инициализируется один раз
const localLikes = ref<Record<string, boolean>>({});
const submitting = ref<Record<string, boolean>>({});
let initialized = false;

function initLikes() {
  if (initialized || typeof window === 'undefined') return;
  initialized = true;
  try {
    const stored = window.localStorage.getItem(LIKES_KEY);
    if (stored) localLikes.value = JSON.parse(stored);
  } catch { /* ignore */ }
}

function saveLikes() {
  if (typeof window === 'undefined') return;
  try { window.localStorage.setItem(LIKES_KEY, JSON.stringify(localLikes.value)); } catch { /* ignore */ }
}

function safeParseJson(raw: string | null): any {
  if (!raw) return null;
  try { return JSON.parse(raw); } catch { return null; }
}

function readViewedMap(): Record<string, boolean> {
  if (typeof window === 'undefined') return {};
  return safeParseJson(window.sessionStorage.getItem(VIEW_SESSION_KEY)) ?? {};
}

function writeViewedMap(map: Record<string, boolean>) {
  if (typeof window === 'undefined') return;
  window.sessionStorage.setItem(VIEW_SESSION_KEY, JSON.stringify(map));
}

function mapApiToNewsRecord(d: any) {
  return {
    id:          String(d?.id ?? ''),
    title:       String(d?.title ?? ''),
    category:    String(d?.category ?? ''),
    description: String(d?.description ?? ''),
    date:        String(d?.date ?? ''),
    imagePath:   d?.image_path ?? null,
    createdAt:   d?.created_at ?? null,
    likes: Number(d?.likes ?? 0) || 0,
    views: Number(d?.views ?? 0) || 0,
  };
}

export function useNewsReactions() {
  const { getById, patchItem } = useNewsData();

  if (!initialized) initLikes();

  function isLiked(id: string | number): boolean {
    return !!localLikes.value[String(id)];
  }

  async function toggleLike(id: string | number) {
    const key = String(id);
    if (submitting.value[key]) return;

    const item = getById(key);
    if (!item) return;

    const nextLiked = !isLiked(key);
    const before = { ...item };
    submitting.value[key] = true;

    // Если лайкаем и новость ещё не просматривалась в этой сессии — засчитать просмотр
    const viewed = readViewedMap();
    const needsView = nextLiked && !viewed[key];

    try {
      const [likeRes, viewRes] = await Promise.all([
        fetch(`/api/news.php?id=${key}&action=like`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ liked: nextLiked }),
        }),
        needsView
          ? fetch(`/api/news.php?id=${key}&action=view`, { method: 'POST' })
          : Promise.resolve(null),
      ]);

      const json = await likeRes.json();
      if (!json?.success) throw new Error(json?.message || 'Ошибка обновления лайка');

      let patched = mapApiToNewsRecord(json.data);

      if (viewRes) {
        const viewJson = await viewRes.json();
        if (viewJson?.success) {
          patched = { ...patched, views: mapApiToNewsRecord(viewJson.data).views };
          viewed[key] = true;
          writeViewedMap(viewed);
        }
      }

      patchItem(patched);
      localLikes.value[key] = nextLiked;
      saveLikes();
    } catch {
      patchItem(before);
    } finally {
      submitting.value[key] = false;
    }
  }

  return { isLiked, toggleLike };
}