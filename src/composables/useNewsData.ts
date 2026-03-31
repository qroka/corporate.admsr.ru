import { computed, ref } from 'vue';

type NewsExportRow = {
  id: string;
  title: string;
  html: string;
  short_html: string;
  author: string;
  timestamp: string;
  anounce: string;
};

export type NewsRecord = {
  id: string;
  title: string;
  html: string;
  shortHtml: string;
  authorId: string | null;
  timestamp: number | null;
  announceImagePath: string | null;
};

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedRows = ref<NewsRecord[]>([]);
const sharedLoaded = ref(false);
let sharedLoadPromise: Promise<void> | null = null;

function extractTableData<T = unknown>(raw: unknown, tableName: string): T[] {
  if (!Array.isArray(raw)) return [];
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue;
    const rec = item as any;
    if (rec.type === 'table' && rec.name === tableName && Array.isArray(rec.data)) return rec.data as T[];
  }
  return [];
}

function asNonEmptyString(v: unknown): string | null {
  const s = typeof v === 'string' ? v.trim() : '';
  return s.length ? s : null;
}

export function stripHtmlToText(html: string): string {
  return String(html ?? '')
    .replace(/<style[\s\S]*?<\/style>/gi, ' ')
    .replace(/<script[\s\S]*?<\/script>/gi, ' ')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/\s+/g, ' ')
    .trim();
}

export function formatUnixDate(unixSeconds: number | null): string | undefined {
  if (!unixSeconds || !Number.isFinite(unixSeconds)) return undefined;
  const dt = new Date(unixSeconds * 1000);
  if (Number.isNaN(dt.getTime())) return undefined;
  return dt.toISOString().slice(0, 10);
}

/**
 * В `news.json` картинки часто лежат как `"/uploads/news/....jpg"`.
 * Для главной/ленты используем локальные превью из `public/img/SmallPic`.
 * Если в данных лежит старый путь `/uploads/news/<name>.<ext>`, мапим его в `/img/SmallPic/<name>.webp`.
 */
export function resolveNewsImageSrc(path: string | null): string | undefined {
  const raw = (path ?? '').trim();
  if (!raw) return undefined;

  // Already local (our uploads endpoint returns /img/SmallPic/... and /img/FullPic/...)
  if (raw.startsWith('/img/')) return raw;

  // Legacy: phpmyadmin export stored "/uploads/news/<name>.<ext>" (we keep previews in /img/SmallPic/<name>.webp)
  const legacy = raw.startsWith('/uploads/news/') ? raw : null;
  if (legacy) {
    const file = legacy.split('/').pop() ?? '';
    const base = file.replace(/\.(jpg|jpeg|png|gif|webp)$/i, '').trim();
    if (base) return `/img/SmallPic/news/${base}.webp`;
  }

  // Sometimes legacy path can be embedded into absolute URL - still map it.
  const m = /\/uploads\/news\/([^/?#]+)\.(jpg|jpeg|png|gif|webp)(?:[?#].*)?$/i.exec(raw);
  if (m?.[1]) return `/img/SmallPic/news/${m[1]}.webp`;

  // Fallback: leave as-is for other cases (can be absolute URL, or a local absolute path).
  return raw;
}

export function useNewsData() {
  const loading = sharedLoading;
  const error = sharedError;
  const news = sharedRows;

  async function load() {
    if (sharedLoaded.value) return;
    if (sharedLoadPromise) return sharedLoadPromise;

    sharedLoadPromise = (async () => {
      sharedLoading.value = true;
      sharedError.value = null;
      try {
        const res = await fetch('/data/news.json', { cache: 'force-cache' });
        if (!res.ok) throw new Error(`Не удалось загрузить news.json (${res.status})`);
        const raw = await res.json();

        const rows = extractTableData<NewsExportRow>(raw, 'news');
        sharedRows.value = rows
          .map((r) => {
            const id = String(r?.id ?? '').trim();
            if (!id) return null;
            const ts = Number(String(r?.timestamp ?? '').trim());
            return {
              id,
              title: String(r?.title ?? '').trim(),
              html: String(r?.html ?? ''),
              shortHtml: String(r?.short_html ?? ''),
              authorId: asNonEmptyString(r?.author),
              timestamp: Number.isFinite(ts) ? ts : null,
              announceImagePath: asNonEmptyString(r?.anounce),
            } satisfies NewsRecord;
          })
          .filter(Boolean) as NewsRecord[];

        sharedLoaded.value = true;
      } catch (e: any) {
        sharedError.value = e?.message ? String(e.message) : 'Ошибка загрузки новостей';
      } finally {
        sharedLoading.value = false;
      }
    })();

    return sharedLoadPromise;
  }

  function ensureLoaded() {
    void load();
  }

  const sortedNews = computed(() => {
    const items = news.value.slice();
    items.sort((a, b) => (b.timestamp ?? 0) - (a.timestamp ?? 0) || b.id.localeCompare(a.id, 'ru-RU'));
    return items;
  });

  function getById(id: string): NewsRecord | undefined {
    const key = String(id ?? '').trim();
    if (!key) return undefined;
    return news.value.find((x) => x.id === key);
  }

  return { loading, error, news, sortedNews, getById, load, ensureLoaded };
}
