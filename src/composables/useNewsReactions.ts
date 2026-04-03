import { ref } from 'vue';
import { useNewsData } from './useNewsData';

const STORAGE_KEY = 'news-likes:v1';
const localLikes = ref<Record<string, boolean>>({});
const submitting = ref<Record<string, boolean>>({});
let initialized = false;

function initLikes() {
  if (initialized || typeof window === 'undefined') return;
  initialized = true;
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) {
      localLikes.value = JSON.parse(stored);
    }
  } catch {
    // ignore
  }
}

function saveLikes() {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(localLikes.value));
  } catch {
    // ignore
  }
}

function mapApiToNewsRecord(d: any) {
  return {
    id: String(d?.id ?? ''),
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

  if (!initialized) {
    initLikes();
  }

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

    try {
      const res = await fetch(`/api/news.php?id=${key}&action=like`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ liked: nextLiked }),
      });
      const json = await res.json();
      if (!json?.success) throw new Error(json?.message || 'Ошибка обновления лайка');

      patchItem(mapApiToNewsRecord(json.data));
      localLikes.value[key] = nextLiked;
      saveLikes();
    } catch (e) {
      patchItem(before);
    } finally {
      submitting.value[key] = false;
    }
  }

  return { isLiked, toggleLike };
}
