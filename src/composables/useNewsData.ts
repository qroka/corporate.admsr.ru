import { computed, ref } from 'vue';
import { formatDateRuLong } from '../utils/date';

export type NewsRecord = {
  id: string;
  title: string;
  category: string;
  description: string;
  date: string; // YYYY-MM-DD
  imagePath: string | null;
  createdAt: string | null;
};

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedRows = ref<NewsRecord[]>([]);
const sharedLoaded = ref(false);
let sharedLoadPromise: Promise<void> | null = null;

function mapApiRow(r: Record<string, unknown>): NewsRecord | null {
  const id = String(r?.id ?? '').trim();
  if (!id) return null;
  return {
    id,
    title:       String(r?.title       ?? '').trim(),
    category:    String(r?.category    ?? '').trim(),
    description: String(r?.description ?? ''),
    date:        String(r?.date        ?? '').trim(),
    imagePath:   asNonEmptyString(r?.image_path),
    createdAt:   asNonEmptyString(r?.created_at),
  };
}

async function tryLoadFromApi(): Promise<boolean> {
  try {
    const res = await fetch('/api/news.php', { cache: 'no-store' });
    if (!res.ok) return false;
    const json = (await res.json()) as { success?: boolean; data?: unknown };
    if (!json?.success || !Array.isArray(json.data)) return false;
    sharedRows.value = json.data
      .map((x) => mapApiRow(x as Record<string, unknown>))
      .filter(Boolean) as NewsRecord[];
    return true;
  } catch {
    return false;
  }
}

function asNonEmptyString(v: unknown): string | null {
  const s = typeof v === 'string' ? v.trim() : '';
  return s.length ? s : null;
}

/** Форматирует дату YYYY-MM-DD в русский формат «12 апреля 2025» */
export function formatNewsDate(date: string | null | undefined): string | undefined {
  if (!date) return undefined;
  return formatDateRuLong(date) || date;
}

/** Возвращает src для изображения новости */
export function resolveNewsImageSrc(path: string | null): string | undefined {
  const raw = (path ?? '').trim();
  if (!raw) return undefined;
  return raw;
}

export function useNewsData() {
  const loading = sharedLoading;
  const error   = sharedError;
  const news    = sharedRows;

  async function load(opts?: { force?: boolean }) {
    if (!opts?.force) {
      if (sharedLoaded.value) return;
      if (sharedLoadPromise) return sharedLoadPromise;
    } else {
      sharedLoadPromise  = null;
      sharedLoaded.value = false;
    }

    sharedLoadPromise = (async () => {
      sharedLoading.value = true;
      sharedError.value   = null;
      try {
        const ok = await tryLoadFromApi();
        if (!ok) throw new Error('Не удалось загрузить новости');
        sharedLoaded.value = true;
      } catch (e: unknown) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки новостей';
      } finally {
        sharedLoading.value = false;
      }
    })();

    return sharedLoadPromise;
  }

  async function reload() {
    await load({ force: true });
  }

  function ensureLoaded() {
    void load();
  }

  const sortedNews = computed(() => {
    const items = news.value.slice();
    items.sort((a, b) => {
      const dc = String(b.date ?? '').localeCompare(String(a.date ?? ''));
      return dc !== 0 ? dc : b.id.localeCompare(a.id, 'ru-RU');
    });
    return items;
  });

  function getById(id: string): NewsRecord | undefined {
    const key = String(id ?? '').trim();
    if (!key) return undefined;
    return news.value.find((x) => x.id === key);
  }

  return { loading, error, news, sortedNews, getById, load, reload, ensureLoaded };
}