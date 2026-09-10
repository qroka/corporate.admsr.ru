import { reactive, watch } from 'vue';
import type { UiThemeSelection } from './useUiTheme';
import {
  getFontOption,
  getRadiusOption,
  getSavedUiTheme,
  randomUiTheme,
  resetUiTheme,
  setSavedUiTheme,
} from './useUiTheme';

type AppConfig = {
  ui: {
    colors: {
      primary: UiThemeSelection['primary'];
      neutral: UiThemeSelection['neutral'];
    };
    font: UiThemeSelection['font'];
    radius: UiThemeSelection['radius'];
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
      font: theme.font,
      radius: theme.radius,
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
    const merged = clampFromTheme({
      primary: parsed?.ui?.colors?.primary ?? fallback.ui.colors.primary,
      neutral: parsed?.ui?.colors?.neutral ?? fallback.ui.colors.neutral,
      font: parsed?.ui?.font ?? theme.font ?? fallback.ui.font,
      radius: parsed?.ui?.radius ?? theme.radius ?? fallback.ui.radius,
    });
    return merged;
  } catch {
    return fallback;
  }
}

function syncFromTheme(theme: UiThemeSelection) {
  appConfig.ui.colors.primary = theme.primary;
  appConfig.ui.colors.neutral = theme.neutral;
  appConfig.ui.font = theme.font;
  appConfig.ui.radius = theme.radius;
}

const appConfig = reactive<AppConfig>(readStored());
let watchersInitialized = false;
let syncingFromTheme = false;

function persistAppConfig() {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(appConfig));
  } catch {
    // ignore
  }
}

function ensureWatchers() {
  if (watchersInitialized) return;
  watchersInitialized = true;

  watch(
    () =>
      [
        appConfig.ui.colors.primary,
        appConfig.ui.colors.neutral,
        appConfig.ui.font,
        appConfig.ui.radius,
      ] as const,
    ([primary, neutral, font, radius]) => {
      if (syncingFromTheme) {
        persistAppConfig();
        return;
      }
      setSavedUiTheme({ primary, neutral, font, radius });
      persistAppConfig();
    },
    { immediate: true },
  );
}

export function useAppConfig() {
  ensureWatchers();
  return appConfig;
}

export function applyThemeSelection(theme: UiThemeSelection) {
  ensureWatchers();
  const next = setSavedUiTheme(theme);
  syncingFromTheme = true;
  syncFromTheme(next);
  persistAppConfig();
  syncingFromTheme = false;
  return next;
}

export function patchAppTheme(patch: Partial<UiThemeSelection>) {
  ensureWatchers();
  return applyThemeSelection({
    primary: patch.primary ?? appConfig.ui.colors.primary,
    neutral: patch.neutral ?? appConfig.ui.colors.neutral,
    font: patch.font ?? appConfig.ui.font,
    radius: patch.radius ?? appConfig.ui.radius,
  });
}

export function randomAppTheme() {
  ensureWatchers();
  const theme = randomUiTheme({
    primary: appConfig.ui.colors.primary,
    neutral: appConfig.ui.colors.neutral,
    font: appConfig.ui.font,
    radius: appConfig.ui.radius,
  });
  syncingFromTheme = true;
  syncFromTheme(theme);
  persistAppConfig();
  syncingFromTheme = false;
  return theme;
}

export function resetAppTheme() {
  ensureWatchers();
  const theme = resetUiTheme();
  syncingFromTheme = true;
  syncFromTheme(theme);
  persistAppConfig();
  syncingFromTheme = false;
  return theme;
}

export function currentFontLabel() {
  ensureWatchers();
  return getFontOption(appConfig.ui.font).label;
}

export function currentRadiusLabel() {
  ensureWatchers();
  return getRadiusOption(appConfig.ui.radius).label;
}
