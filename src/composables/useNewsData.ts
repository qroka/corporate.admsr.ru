import { computed, ref } from 'vue';
import { formatUnixSecondsRuDate } from '../utils/date';

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
  /** из БД (для PUT /api/news.php) */
  likes?: number;
  views?: number;
};

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedRows = ref<NewsRecord[]>([]);
const sharedLoaded = ref(false);
let sharedLoadPromise: Promise<void> | null = null;

function mapApiRow(r: Record<string, unknown>): NewsRecord | null {
  const id = String(r?.id ?? '').trim();
  if (!id) return null;
  const ts = Number(r?.timestamp);
  const aid = r?.author_id;
  const authorNum = aid != null && aid !== '' ? Number(aid) : NaN;
  return {
    id,
    title: String(r?.title ?? '').trim(),
    html: String(r?.html ?? ''),
    shortHtml: String(r?.short_html ?? ''),
    authorId: Number.isFinite(authorNum) && authorNum > 0 ? String(authorNum) : null,
    timestamp: Number.isFinite(ts) ? ts : null,
    announceImagePath: asNonEmptyString(r?.announce_image_path),
    likes: typeof r?.likes === 'number' ? r.likes : Number(r?.likes ?? 0) || 0,
    views: typeof r?.views === 'number' ? r.views : Number(r?.views ?? 0) || 0,
  };
}

async function tryLoadFromApi(): Promise<boolean> {
  try {
    const res = await fetch('/api/news.php', { cache: 'no-store' });
    if (!res.ok) return false;
    const json = (await res.json()) as { success?: boolean; data?: unknown };
    if (!json?.success || !Array.isArray(json.data)) return false;
    const rows = json.data
      .map((x) => mapApiRow(x as Record<string, unknown>))
      .filter(Boolean) as NewsRecord[];
    sharedRows.value = rows;
    return true;
  } catch {
    return false;
  }
}

async function loadFromJsonExport(): Promise<void> {
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
}

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
  return formatUnixSecondsRuDate(unixSeconds);
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

  async function load(opts?: { force?: boolean }) {
    if (!opts?.force) {
      if (sharedLoaded.value) return;
      if (sharedLoadPromise) return sharedLoadPromise;
    } else {
      sharedLoadPromise = null;
      sharedLoaded.value = false;
    }

    sharedLoadPromise = (async () => {
      sharedLoading.value = true;
      sharedError.value = null;
      try {
        const apiOk = await tryLoadFromApi();
        if (!apiOk) {
          await loadFromJsonExport();
        }
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
    items.sort((a, b) => (b.timestamp ?? 0) - (a.timestamp ?? 0) || b.id.localeCompare(a.id, 'ru-RU'));
    return items;
  });

  function getById(id: string): NewsRecord | undefined {
    const key = String(id ?? '').trim();
    if (!key) return undefined;
    return news.value.find((x) => x.id === key);
  }

  return { loading, error, news, sortedNews, getById, load, reload, ensureLoaded };
}
