import { computed, ref } from 'vue';

export type GalleryAlbumRecord = {
  id: string;
  title: string;       // gallery.name
  description: string;
  date: string;        // YYYY-MM-DD
  image?: string;      // обложка — image_small_url первого фото
};

// ── Shared module-level state ─────────────────────────────────────────────────
const albums = ref<GalleryAlbumRecord[]>([]);
const loading = ref(false);
const error   = ref<string | null>(null);
let loaded     = false;
let loadPromise: Promise<void> | null = null;

async function doLoad(): Promise<void> {
  loading.value = true;
  error.value   = null;
  try {
    const res  = await fetch('/api/gallery.php');
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки');
    albums.value = (json.data ?? []).map((a: any) => ({
      id:          String(a.id),
      title:       String(a.name        ?? ''),
      description: String(a.description ?? ''),
      date:        String(a.date        ?? '').slice(0, 10),
      image:       a.cover ?? undefined,
    }));
    loaded = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки галереи';
  } finally {
    loading.value = false;
    loadPromise   = null;
  }
}

// ── Composable ────────────────────────────────────────────────────────────────
export function useGalleryData() {
  function ensureLoaded() {
    if (loaded) return;
    if (!loadPromise) loadPromise = doLoad();
  }

  async function reload(): Promise<void> {
    loaded = false;
    if (!loadPromise) loadPromise = doLoad();
    return loadPromise;
  }

  function getAlbum(id: string): GalleryAlbumRecord {
    return (
      albums.value.find((a) => a.id === id) ?? {
        id,
        title:       `Альбом ${id}`,
        description: '',
        date:        '',
      }
    );
  }

  return { albums, loading, error, ensureLoaded, reload, getAlbum };
}