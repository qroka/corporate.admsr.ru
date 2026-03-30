import { reactive, watch } from 'vue';
import type { UiThemeSelection } from './useUiTheme';
import { getSavedUiTheme, setSavedUiTheme } from './useUiTheme';

type AppConfig = {
  ui: {
    colors: {
      primary: UiThemeSelection['primary'];
      neutral: UiThemeSelection['neutral'];
    };
  };
};

const STORAGE_KEY = 'ui-app-config:v1';

function clampFromTheme(theme: UiThemeSelection): AppConfig {
  return {
    ui: {
      colors: {
        primary: theme.primary,
        neutral: theme.neutral,
      },
    },
  };
}

function readStored(): AppConfig {
  const theme = getSavedUiTheme();
  const fallback = clampFromTheme(theme);
  if (typeof window === 'undefined') return fallback;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;
    const parsed = JSON.parse(raw);
    const p = parsed?.ui?.colors?.primary ?? fallback.ui.colors.primary;
    const n = parsed?.ui?.colors?.neutral ?? fallback.ui.colors.neutral;
    return {
      ui: {
        colors: {
          primary: p,
          neutral: n,
        },
      },
    };
  } catch {
    return fallback;
  }
}

const appConfig = reactive<AppConfig>(readStored());
let watchersInitialized = false;

function ensureWatchers() {
  if (watchersInitialized) return;
  watchersInitialized = true;

  watch(
    () => [appConfig.ui.colors.primary, appConfig.ui.colors.neutral] as const,
    ([primary, neutral]) => {
      setSavedUiTheme({ primary, neutral });
      if (typeof window !== 'undefined') {
        try {
          window.localStorage.setItem(STORAGE_KEY, JSON.stringify(appConfig));
        } catch {
          // ignore
        }
      }
    },
    { immediate: true },
  );
}

export function useAppConfig() {
  ensureWatchers();
  return appConfig;
}

