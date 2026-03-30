export type UiThemeSelection = {
  primary: (typeof PRIMARY_COLORS)[number];
  neutral: (typeof NEUTRAL_COLORS)[number];
};

type Shade =
  | '50'
  | '100'
  | '200'
  | '300'
  | '400'
  | '500'
  | '600'
  | '700'
  | '800'
  | '900'
  | '950';

export const PRIMARY_COLORS = [
  'red',
  'orange',
  'amber',
  'yellow',
  'lime',
  'green',
  'emerald',
  'teal',
  'cyan',
  'sky',
  'blue',
  'indigo',
  'violet',
  'purple',
  'fuchsia',
  'pink',
  'rose',
  'slate',
  'gray',
  'zinc',
  'neutral',
  'stone',
  'taupe',
  'mauve',
  'mist',
  'olive',  
] as const;

export const NEUTRAL_COLORS = [
  'red',
  'orange',
  'amber',
  'yellow',
  'lime',
  'green',
  'emerald',
  'teal',
  'cyan',
  'sky',
  'blue',
  'indigo',
  'violet',
  'purple',
  'fuchsia',
  'pink',
  'rose',
  'slate',
  'gray',
  'zinc',
  'neutral',
  'stone',
  'taupe',
  'mauve',
  'mist',
  'olive',
] as const;

const STORAGE_KEY = 'ui-theme-selection:v1';

const DEFAULT_THEME: UiThemeSelection = { primary: 'sky', neutral: 'slate' };

function clampTheme(input: any): UiThemeSelection {
  const primary = input?.primary;
  const neutral = input?.neutral;

  return {
    primary: (PRIMARY_COLORS.includes(primary) ? primary : DEFAULT_THEME.primary) as UiThemeSelection['primary'],
    neutral: (NEUTRAL_COLORS.includes(neutral) ? neutral : DEFAULT_THEME.neutral) as UiThemeSelection['neutral'],
  };
}

export function getSavedUiTheme(): UiThemeSelection {
  if (typeof window === 'undefined') return DEFAULT_THEME;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_THEME;
    return clampTheme(JSON.parse(raw));
  } catch {
    return DEFAULT_THEME;
  }
}

export function applyUiTheme(selection: UiThemeSelection) {
  if (typeof document === 'undefined') return;

  const theme = clampTheme(selection);
  const el = document.documentElement;

  // Tailwind v4 exposes palette steps as CSS vars: `--color-sky-500`, etc.
  // We mirror them into Nuxt UI vars: `--ui-color-primary-500` and `--ui-primary`.
  const computed = getComputedStyle(el);
  (['50', '100', '200', '300', '400', '500', '600', '700', '800', '900', '950'] as Shade[]).forEach((shade) => {
    const p = computed.getPropertyValue(`--color-${theme.primary}-${shade}`).trim();
    const n = computed.getPropertyValue(`--color-${theme.neutral}-${shade}`).trim();
    if (p) el.style.setProperty(`--ui-color-primary-${shade}`, p);
    if (n) el.style.setProperty(`--ui-color-neutral-${shade}`, n);
  });
  const primary500 = computed.getPropertyValue(`--color-${theme.primary}-500`).trim();
  if (primary500) el.style.setProperty('--ui-primary', primary500);
}

export function setSavedUiTheme(selection: UiThemeSelection) {
  const theme = clampTheme(selection);
  if (typeof window !== 'undefined') {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(theme));
    } catch {
      // ignore
    }
  }
  applyUiTheme(theme);
}

export function useUiTheme() {
  const current = getSavedUiTheme();
  return {
    PRIMARY_COLORS,
    NEUTRAL_COLORS,
    getSavedUiTheme,
    applyUiTheme,
    setSavedUiTheme,
    current,
  };
}

