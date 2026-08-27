import { onBeforeUnmount, onMounted, type Ref, type MaybeRefOrGetter, toValue } from 'vue';

export const COLOR_MODE_KEY = 'ui-color-mode';

export function applyColorModeToDocument(mode: 'light' | 'dark') {
  document.documentElement.classList.toggle('dark', mode === 'dark');
}

export function readMainColorMode(): 'light' | 'dark' {
  const saved = localStorage.getItem(COLOR_MODE_KEY);
  return saved === 'dark' ? 'dark' : 'light';
}

export function applyMainColorModeFromStorage() {
  applyColorModeToDocument(readMainColorMode());
}

type UseColorModeOptions = {
  enabled?: MaybeRefOrGetter<boolean>;
};

export function useColorMode(isDark: Ref<boolean>, options: UseColorModeOptions = {}) {
  function isEnabled() {
    return options.enabled === undefined ? true : Boolean(toValue(options.enabled));
  }

  function syncFromStorage() {
    if (!isEnabled()) return;
    const mode = readMainColorMode();
    isDark.value = mode === 'dark';
    applyColorModeToDocument(mode);
  }

  function toggleColorMode() {
    const next: 'light' | 'dark' = isDark.value ? 'light' : 'dark';
    isDark.value = next === 'dark';
    localStorage.setItem(COLOR_MODE_KEY, next);
    applyColorModeToDocument(next);
    window.dispatchEvent(new CustomEvent('ui-color-mode-change', { detail: { mode: next } }));
  }

  function onExternalMode(e: Event) {
    if (!isEnabled()) return;
    const next = (e as CustomEvent<{ mode?: string }>).detail?.mode;
    if (next !== 'light' && next !== 'dark') return;
    isDark.value = next === 'dark';
    applyColorModeToDocument(next);
  }

  onMounted(() => {
    syncFromStorage();
    window.addEventListener('ui-color-mode-change', onExternalMode);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('ui-color-mode-change', onExternalMode);
  });

  return { syncFromStorage, toggleColorMode };
}
