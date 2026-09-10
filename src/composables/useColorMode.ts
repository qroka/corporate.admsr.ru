import { onBeforeUnmount, onMounted, ref, type Ref, type MaybeRefOrGetter, toValue } from 'vue';

export const COLOR_MODE_KEY = 'ui-color-mode';

export type ColorModePreference = 'light' | 'dark' | 'system';
export type ColorModeResolved = 'light' | 'dark';

export function applyColorModeToDocument(mode: ColorModeResolved) {
  document.documentElement.classList.toggle('dark', mode === 'dark');
}

export function getSystemColorMode(): ColorModeResolved {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function readMainColorModePreference(): ColorModePreference {
  if (typeof localStorage === 'undefined') return 'light';
  const saved = localStorage.getItem(COLOR_MODE_KEY);
  if (saved === 'dark' || saved === 'light' || saved === 'system') return saved;
  return 'light';
}

export function resolveMainColorMode(
  pref: ColorModePreference = readMainColorModePreference(),
): ColorModeResolved {
  return pref === 'system' ? getSystemColorMode() : pref;
}

export function readMainColorMode(): ColorModeResolved {
  return resolveMainColorMode();
}

export function applyMainColorModeFromStorage() {
  applyColorModeToDocument(readMainColorMode());
}

type UseColorModeOptions = {
  enabled?: MaybeRefOrGetter<boolean>;
};

export function useColorMode(isDark: Ref<boolean>, options: UseColorModeOptions = {}) {
  const preference = ref<ColorModePreference>(
    typeof localStorage !== 'undefined' ? readMainColorModePreference() : 'light',
  );

  function isEnabled() {
    return options.enabled === undefined ? true : Boolean(toValue(options.enabled));
  }

  function applyResolved(pref: ColorModePreference, persist: boolean) {
    preference.value = pref;
    if (persist && typeof localStorage !== 'undefined') {
      localStorage.setItem(COLOR_MODE_KEY, pref);
    }
    const resolved = resolveMainColorMode(pref);
    isDark.value = resolved === 'dark';
    applyColorModeToDocument(resolved);
    window.dispatchEvent(
      new CustomEvent('ui-color-mode-change', {
        detail: { mode: resolved, preference: pref },
      }),
    );
  }

  function syncFromStorage() {
    if (!isEnabled()) return;
    applyResolved(readMainColorModePreference(), false);
  }

  function setColorMode(next: ColorModePreference) {
    if (!isEnabled()) return;
    if (next !== 'light' && next !== 'dark' && next !== 'system') return;
    applyResolved(next, true);
  }

  function toggleColorMode() {
    setColorMode(isDark.value ? 'light' : 'dark');
  }

  function onExternalMode(e: Event) {
    if (!isEnabled()) return;
    const detail = (e as CustomEvent<{ mode?: string; preference?: string }>).detail;
    const nextPref = detail?.preference;
    if (nextPref === 'light' || nextPref === 'dark' || nextPref === 'system') {
      preference.value = nextPref;
    }
    const next = detail?.mode;
    if (next !== 'light' && next !== 'dark') return;
    isDark.value = next === 'dark';
    applyColorModeToDocument(next);
  }

  function onSystemChange() {
    if (!isEnabled() || preference.value !== 'system') return;
    const resolved = getSystemColorMode();
    isDark.value = resolved === 'dark';
    applyColorModeToDocument(resolved);
  }

  let mediaQuery: MediaQueryList | null = null;

  onMounted(() => {
    syncFromStorage();
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', onSystemChange);
    window.addEventListener('ui-color-mode-change', onExternalMode);
  });

  onBeforeUnmount(() => {
    mediaQuery?.removeEventListener('change', onSystemChange);
    window.removeEventListener('ui-color-mode-change', onExternalMode);
  });

  return { preference, syncFromStorage, setColorMode, toggleColorMode };
}
