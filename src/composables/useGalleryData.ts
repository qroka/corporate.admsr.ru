import { computed, onMounted, ref } from 'vue';

export type GalleryAlbumRecord = {
  id: string;
  title: string;
  description: string;
  date: string; // YYYY-MM-DD
  badge?: string;
  coverIndex?: number;
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

function safeTemplate(tpl: string, vars: Record<string, string | number>) {
  return tpl.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
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
        const res = await fetch('/data/gallery.json');
        if (!res.ok) throw new Error(`Не удалось загрузить gallery.json: ${res.status}`);
        const data = (await res.json()) as GalleryConfig;
        if (!data || typeof data !== 'object' || !Array.isArray((data as any).albums)) {
          throw new Error('Некорректный формат gallery.json');
        }
        sharedConfig.value = data;
        sharedLoaded.value = true;
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

  return { loading, error, albums, getAlbum, buildPhotoMeta, load, ensureLoaded };
}

