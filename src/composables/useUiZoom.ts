import { computed, ref } from 'vue';

export type UiZoomLevel = 0.8 | 0.9 | 1 | 1.1 | 1.25;

export const ZOOM_OPTIONS: ReadonlyArray<{ id: UiZoomLevel; label: string }> = [
  { id: 0.8, label: '80%' },
  { id: 0.9, label: '90%' },
  { id: 1, label: '100%' },
  { id: 1.1, label: '110%' },
  { id: 1.25, label: '125%' },
] as const;

const STORAGE_KEY = 'ui-zoom:v1';
const DEFAULT_ZOOM: UiZoomLevel = 1;

function clampZoom(value: unknown): UiZoomLevel {
  const n = Number(value);
  const allowed = ZOOM_OPTIONS.map((o) => o.id);
  return (allowed.includes(n as UiZoomLevel) ? n : DEFAULT_ZOOM) as UiZoomLevel;
}

function readStored(): UiZoomLevel {
  if (typeof window === 'undefined') return DEFAULT_ZOOM;
  try {
    return clampZoom(window.localStorage.getItem(STORAGE_KEY));
  } catch {
    return DEFAULT_ZOOM;
  }
}

function persist(zoom: UiZoomLevel) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, String(zoom));
  } catch {
    // ignore
  }
}

/** Применяем CSS zoom — ближайший аналог масштаба страницы в браузере. */
export function applyUiZoom(zoom: UiZoomLevel) {
  if (typeof document === 'undefined') return;
  const el = document.documentElement;
  const value = clampZoom(zoom);
  if (value === 1) {
    el.style.removeProperty('zoom');
  } else {
    el.style.setProperty('zoom', String(value));
  }
  el.dataset.uiZoom = String(value);
}

const zoomLevel = ref<UiZoomLevel>(DEFAULT_ZOOM);
let initialized = false;

function ensureInit() {
  if (initialized) return;
  initialized = true;
  const stored = readStored();
  zoomLevel.value = stored;
  applyUiZoom(stored);
}

export function getSavedUiZoom(): UiZoomLevel {
  return readStored();
}

export function setUiZoom(next: UiZoomLevel) {
  ensureInit();
  const value = clampZoom(next);
  zoomLevel.value = value;
  persist(value);
  applyUiZoom(value);
  return value;
}

export function useUiZoom() {
  ensureInit();

  const zoom = computed({
    get: () => zoomLevel.value,
    set: (v) => {
      setUiZoom(v);
    },
  });

  const zoomLabel = computed(
    () => ZOOM_OPTIONS.find((o) => o.id === zoomLevel.value)?.label ?? '100%',
  );

  return {
    zoom,
    zoomLabel,
    zoomOptions: ZOOM_OPTIONS,
    setZoom: setUiZoom,
  };
}
