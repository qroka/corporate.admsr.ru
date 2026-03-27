import { computed, onMounted, ref } from 'vue';

export type GalleryAlbumRecord = {
  id: string;
  title: string;
  description: string;
  date: string; // YYYY-MM-DD
  badge?: string;
  coverIndex?: number;
  image?: string;
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

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedConfig = ref<GalleryConfig | null>(null);
const sharedLoaded = ref(false);
let sharedPromise: Promise<void> | null = null;
const GALLERY_STORAGE_KEY = 'gallery-config-v1';

function safeTemplate(tpl: string, vars: Record<string, string | number>) {
  return tpl.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
}

function isValidGalleryConfig(data: unknown): data is GalleryConfig {
  return Boolean(data && typeof data === 'object' && Array.isArray((data as any).albums));
}

function persistConfig() {
  if (typeof window === 'undefined') return;
  if (!sharedConfig.value) return;
  try {
    window.localStorage.setItem(GALLERY_STORAGE_KEY, JSON.stringify(sharedConfig.value));
  } catch {
    // Ignore quota/storage access errors.
  }
}

function readStoredConfig(): GalleryConfig | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(GALLERY_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return isValidGalleryConfig(parsed) ? parsed : null;
  } catch {
    return null;
  }
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
        const stored = readStoredConfig();
        if (stored) {
          sharedConfig.value = stored;
          sharedLoaded.value = true;
          return;
        }

        const res = await fetch('/data/gallery.json');
        if (!res.ok) throw new Error(`Не удалось загрузить gallery.json: ${res.status}`);
        const data = (await res.json()) as GalleryConfig;
        if (!isValidGalleryConfig(data)) {
          throw new Error('Некорректный формат gallery.json');
        }
        sharedConfig.value = data;
        sharedLoaded.value = true;
        persistConfig();
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки фотогалереи';
        sharedConfig.value = null;
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
    if (!sharedConfig.value) {
      sharedConfig.value = {
        version: 1,
        albums: [],
        albumFallback: {
          titleTemplate: 'Альбом {albumId}',
          description: 'Фотографии с мероприятия.',
        },
        photoMeta: {
          every: 0,
          titleTemplate: 'Фото {n}',
          descriptionTemplate: 'Кадр {n}',
        },
      };
      sharedLoaded.value = true;
    }
    sharedConfig.value.albums = [album, ...sharedConfig.value.albums];
    persistConfig();
  }

  function updateAlbum(albumId: string, patch: Partial<GalleryAlbumRecord>) {
    if (!sharedConfig.value) return;
    const id = String(albumId ?? '');
    const idx = sharedConfig.value.albums.findIndex((a) => a.id === id);
    if (idx < 0) return;
    sharedConfig.value.albums[idx] = {
      ...sharedConfig.value.albums[idx],
      ...patch,
      id,
    };
    persistConfig();
  }

  function removeAlbum(albumId: string) {
    if (!sharedConfig.value) return;
    const id = String(albumId ?? '');
    sharedConfig.value.albums = sharedConfig.value.albums.filter((a) => a.id !== id);
    persistConfig();
  }

  function getAlbum(albumId: string) {
    const id = String(albumId ?? '');
    const found = albums.value.find((a) => a.id === id);
    if (found) return found;

    const fb = sharedConfig.value?.albumFallback;
    return {
      id,
      title: fb ? safeTemplate(fb.titleTemplate, { albumId: id }) : `Альбом ${id}`,
      description: fb?.description ?? 'Фотографии с мероприятия. Нажмите на фото, чтобы открыть в большом размере.',
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
    // no auto-load by default (lazy from pages)
  });

  return { loading, error, albums, getAlbum, buildPhotoMeta, addAlbum, updateAlbum, removeAlbum, load, ensureLoaded };
}

