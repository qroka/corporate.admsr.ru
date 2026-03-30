import { computed, onMounted, ref } from 'vue';

export type GalleryAlbumRecord = {
  id: string;
  title: string;
  description: string;
  date: string; // YYYY-MM-DD
  badge?: string;
  coverIndex?: number;
  image?: string;
  /** Полные URL (FullPic или data:); превью в UI — SmallPic при необходимости */
  photoLinks?: string[];
};

export type GalleryConfig = {
  version: number;
  albums: GalleryAlbumRecord[];
  albumFallback: {
    titleTemplate: string;
    description: string;
  };
  photoMeta: {
    every: number;
    titleTemplate: string;
    descriptionTemplate: string;
  };
};

type GalleryJsonFile = {
  version?: number;
  source?: string;
  albums?: Array<{
    id?: string;
    title?: string;
    description?: string;
    count?: string;
    date?: string;
    timestamp?: string;
    hided?: string;
    html?: string;
  }>;
  albumFallback: GalleryConfig['albumFallback'];
  photoMeta: GalleryConfig['photoMeta'];
};

type GalleryAlbumsJsonFile = {
  version?: number;
  albums?: Array<{
    albumId?: string;
    files?: Array<{
      urls?: { fullWebp?: string; smallWebp?: string };
      filename?: string;
    }>;
  }>;
};

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedConfig = ref<GalleryConfig | null>(null);
const sharedLoaded = ref(false);
let sharedPromise: Promise<void> | null = null;

/** Кэш альбомов из gallery.json + gallery_albums.json (без админских правок) */
let baseMergedAlbums: GalleryAlbumRecord[] = [];

const ADMIN_EXTRAS_KEY = 'gallery-admin-extras-v1';
const REMOVED_MERGED_IDS_KEY = 'gallery-removed-merged-ids-v1';
const OVERLAYS_KEY = 'gallery-overlays-v1';

function safeTemplate(tpl: string, vars: Record<string, string | number>) {
  return tpl.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
}

function stripHtml(html: string): string {
  return String(html ?? '')
    .replace(/<[^>]*>/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

function timestampToIsoDate(ts: string): string {
  const n = Number(ts);
  if (!Number.isFinite(n) || n <= 0) return '';
  const d = new Date(n * 1000);
  return d.toISOString().slice(0, 10);
}

/** Имя файла в БД без расширения → в public лежит WebP с тем же basename */
export function galleryFullPicUrl(filename: string): string {
  const base = String(filename ?? '').replace(/\.[^/.]+$/, '');
  return `/img/FullPic/${base}.webp`;
}

export function gallerySmallPicUrl(filename: string): string {
  const base = String(filename ?? '').replace(/\.[^/.]+$/, '');
  return `/img/SmallPic/${base}.webp`;
}

function buildMergedAlbumsFromBundled(
  gallery: GalleryJsonFile,
  galleryAlbums: GalleryAlbumsJsonFile,
): GalleryAlbumRecord[] {
  const byAlbumId = new Map<string, NonNullable<GalleryAlbumsJsonFile['albums']>[0]['files']>();
  for (const block of galleryAlbums.albums ?? []) {
    const aid = String(block.albumId ?? '').trim();
    if (!aid) continue;
    const files = Array.isArray(block.files) ? block.files : [];
    byAlbumId.set(aid, files);
  }

  const rows = gallery.albums ?? [];
  const sorted = rows.slice().sort((a, b) => Number(b.timestamp) - Number(a.timestamp));

  const albums: GalleryAlbumRecord[] = [];

  for (const cat of sorted) {
    const id = String(cat.id ?? '').trim();
    if (!id) continue;
    const files = byAlbumId.get(id) ?? [];

    const title = String(cat.title ?? '').trim() || `Альбом ${id}`;
    const html = String(cat.html ?? '');
    const countHint = String(cat.count ?? '').trim();
    const descFromField = String(cat.description ?? '').trim();
    const descFromHtml = stripHtml(html);
    const description =
      descFromField ||
      descFromHtml ||
      (files.length
        ? `Фотографий: ${files.length}.`
        : countHint
          ? `В каталоге указано фото: ${countHint}.`
          : 'Фотографии появятся после синхронизации файлов.');

    const photoLinks = files
      .map((f) => String(f.urls?.fullWebp ?? '').trim())
      .filter(Boolean);
    const firstSmall = files[0]?.urls?.smallWebp;
    const image = firstSmall ? String(firstSmall) : undefined;

    const dateStr = String(cat.date ?? '').trim();
    const date =
      dateStr ||
      timestampToIsoDate(String(cat.timestamp ?? ''));

    albums.push({
      id,
      title,
      description,
      date,
      image,
      photoLinks: photoLinks.length ? photoLinks : [],
    });
  }

  return albums;
}

function readAdminExtras(): GalleryAlbumRecord[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(ADMIN_EXTRAS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function readRemovedMergedIds(): Set<string> {
  if (typeof window === 'undefined') return new Set();
  try {
    const raw = window.localStorage.getItem(REMOVED_MERGED_IDS_KEY);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? new Set(parsed.map(String)) : new Set();
  } catch {
    return new Set();
  }
}

function readOverlays(): Record<string, Partial<GalleryAlbumRecord>> {
  if (typeof window === 'undefined') return {};
  try {
    const raw = window.localStorage.getItem(OVERLAYS_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch {
    return {};
  }
}

function persistAdminState(
  extras: GalleryAlbumRecord[],
  removed: Set<string>,
  overlays: Record<string, Partial<GalleryAlbumRecord>>,
) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(ADMIN_EXTRAS_KEY, JSON.stringify(extras));
    window.localStorage.setItem(REMOVED_MERGED_IDS_KEY, JSON.stringify([...removed]));
    window.localStorage.setItem(OVERLAYS_KEY, JSON.stringify(overlays));
  } catch {
    // quota / access
  }
}

function reapplyCombinedAlbums() {
  const extras = readAdminExtras();
  const removed = readRemovedMergedIds();
  const overlays = readOverlays();

  const mergedVisible = baseMergedAlbums
    .filter((a) => !removed.has(a.id))
    .map((a) => {
      const o = overlays[a.id];
      return o ? { ...a, ...o, id: a.id } : a;
    });

  const fb = sharedConfig.value?.albumFallback ?? {
    titleTemplate: 'Альбом {albumId}',
    description: 'Фотографии с мероприятия. Нажмите на фото, чтобы открыть в большом размере.',
  };
  const pm = sharedConfig.value?.photoMeta ?? {
    every: 0,
    titleTemplate: 'Фото {n}',
    descriptionTemplate: 'Кадр {n}',
  };

  sharedConfig.value = {
    version: 2,
    albums: [...extras, ...mergedVisible],
    albumFallback: fb,
    photoMeta: pm,
  };
}

export function useGalleryData() {
  const loading = sharedLoading;
  const error = sharedError;

  async function load() {
    if (sharedLoaded.value) return;
    if (sharedPromise) return sharedPromise;

    sharedLoading.value = true;
    sharedError.value = null;

    sharedPromise = (async () => {
      try {
        const [gRes, gaRes] = await Promise.all([
          fetch('/data/gallery.json'),
          fetch('/data/gallery_albums.json'),
        ]);
        if (!gRes.ok) throw new Error(`Не удалось загрузить gallery.json: ${gRes.status}`);
        if (!gaRes.ok) throw new Error(`Не удалось загрузить gallery_albums.json: ${gaRes.status}`);

        const galleryData = (await gRes.json()) as GalleryJsonFile;
        const galleryAlbumsData = (await gaRes.json()) as GalleryAlbumsJsonFile;

        if (!galleryData.albumFallback || !galleryData.photoMeta) {
          throw new Error('Некорректный формат gallery.json (нет albumFallback / photoMeta)');
        }

        sharedConfig.value = {
          version: Number(galleryData.version) || 2,
          albums: [],
          albumFallback: galleryData.albumFallback,
          photoMeta: galleryData.photoMeta,
        };

        baseMergedAlbums = buildMergedAlbumsFromBundled(galleryData, galleryAlbumsData);
        reapplyCombinedAlbums();
        sharedLoaded.value = true;
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки фотогалереи';
        sharedConfig.value = null;
        baseMergedAlbums = [];
      } finally {
        sharedLoading.value = false;
        sharedPromise = null;
      }
    })();

    return sharedPromise;
  }

  function ensureLoaded() {
    void load();
  }

  const albums = computed(() => sharedConfig.value?.albums ?? []);

  function addAlbum(album: GalleryAlbumRecord) {
    const extras = readAdminExtras();
    extras.unshift(album);
    persistAdminState(extras, readRemovedMergedIds(), readOverlays());
    reapplyCombinedAlbums();
    sharedLoaded.value = true;
  }

  function updateAlbum(albumId: string, patch: Partial<GalleryAlbumRecord>) {
    const id = String(albumId ?? '');
    let extras = readAdminExtras();
    const removed = readRemovedMergedIds();
    let overlays = readOverlays();

    const idx = extras.findIndex((a) => a.id === id);
    if (idx >= 0) {
      extras[idx] = { ...extras[idx], ...patch, id };
    } else {
      overlays = { ...overlays, [id]: { ...overlays[id], ...patch } };
    }

    persistAdminState(extras, removed, overlays);
    reapplyCombinedAlbums();
  }

  function removeAlbum(albumId: string) {
    const id = String(albumId ?? '');
    let extras = readAdminExtras();
    const removed = readRemovedMergedIds();
    let overlays = readOverlays();

    if (extras.some((a) => a.id === id)) {
      extras = extras.filter((a) => a.id !== id);
      persistAdminState(extras, removed, overlays);
      reapplyCombinedAlbums();
      return;
    }

    removed.add(id);
    const { [id]: _, ...rest } = overlays;
    overlays = rest;
    persistAdminState(extras, removed, overlays);
    reapplyCombinedAlbums();
  }

  function getAlbum(albumId: string) {
    const id = String(albumId ?? '');
    const found = albums.value.find((a) => a.id === id);
    if (found) return found;

    const fb = sharedConfig.value?.albumFallback;
    return {
      id,
      title: fb ? safeTemplate(fb.titleTemplate, { albumId: id }) : `Альбом ${id}`,
      description:
        fb?.description ??
        'Фотографии с мероприятия. Нажмите на фото, чтобы открыть в большом размере.',
      date: '',
      badge: undefined,
      coverIndex: undefined,
    } satisfies GalleryAlbumRecord;
  }

  function buildPhotoMeta(albumId: string, n: number) {
    const cfg = sharedConfig.value?.photoMeta;
    const every = cfg?.every ?? 0;
    const withMeta = every > 0 ? n % every === 0 : false;
    if (!withMeta) return {};
    const title = cfg?.titleTemplate
      ? safeTemplate(cfg.titleTemplate, { n, albumId })
      : `Фото ${n}`;
    const description = cfg?.descriptionTemplate
      ? safeTemplate(cfg.descriptionTemplate, { n, albumId })
      : `Кадр ${n} из альбома ${albumId}.`;
    return { title, description };
  }

  onMounted(() => {
    // lazy ensureLoaded from pages
  });

  return { loading, error, albums, getAlbum, buildPhotoMeta, addAlbum, updateAlbum, removeAlbum, load, ensureLoaded };
}
