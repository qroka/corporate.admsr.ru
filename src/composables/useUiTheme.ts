export type UiThemeSelection = {
  primary: keyof typeof PRIMARY_PALETTES;
  neutral: keyof typeof NEUTRAL_PALETTES;
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

type Palette = Record<Shade, string>;

// Tailwind-ish palettes (hex). We only need a curated set for UX.
export const PRIMARY_PALETTES = {
  sky: {
    '50': '#f0f9ff',
    '100': '#e0f2fe',
    '200': '#bae6fd',
    '300': '#7dd3fc',
    '400': '#38bdf8',
    '500': '#0ea5e9',
    '600': '#0284c7',
    '700': '#0369a1',
    '800': '#075985',
    '900': '#0c4a6e',
    '950': '#082f49',
  },
  emerald: {
    '50': '#ecfdf5',
    '100': '#d1fae5',
    '200': '#a7f3d0',
    '300': '#6ee7b7',
    '400': '#34d399',
    '500': '#10b981',
    '600': '#059669',
    '700': '#047857',
    '800': '#065f46',
    '900': '#064e3b',
    '950': '#022c22',
  },
  violet: {
    '50': '#f5f3ff',
    '100': '#ede9fe',
    '200': '#ddd6fe',
    '300': '#c4b5fd',
    '400': '#a78bfa',
    '500': '#8b5cf6',
    '600': '#7c3aed',
    '700': '#6d28d9',
    '800': '#5b21b6',
    '900': '#4c1d95',
    '950': '#2e1065',
  },
  rose: {
    '50': '#fff1f2',
    '100': '#ffe4e6',
    '200': '#fecdd3',
    '300': '#fda4af',
    '400': '#fb7185',
    '500': '#f43f5e',
    '600': '#e11d48',
    '700': '#be123c',
    '800': '#9f1239',
    '900': '#881337',
    '950': '#4c0519',
  },
  amber: {
    '50': '#fffbeb',
    '100': '#fef3c7',
    '200': '#fde68a',
    '300': '#fcd34d',
    '400': '#fbbf24',
    '500': '#f59e0b',
    '600': '#d97706',
    '700': '#b45309',
    '800': '#92400e',
    '900': '#78350f',
    '950': '#451a03',
  },
} as const satisfies Record<string, Palette>;

export const NEUTRAL_PALETTES = {
  slate: {
    '50': '#f8fafc',
    '100': '#f1f5f9',
    '200': '#e2e8f0',
    '300': '#cbd5e1',
    '400': '#94a3b8',
    '500': '#64748b',
    '600': '#475569',
    '700': '#334155',
    '800': '#1e293b',
    '900': '#0f172a',
    '950': '#020617',
  },
  zinc: {
    '50': '#fafafa',
    '100': '#f4f4f5',
    '200': '#e4e4e7',
    '300': '#d4d4d8',
    '400': '#a1a1aa',
    '500': '#71717a',
    '600': '#52525b',
    '700': '#3f3f46',
    '800': '#27272a',
    '900': '#18181b',
    '950': '#09090b',
  },
  gray: {
    '50': '#f9fafb',
    '100': '#f3f4f6',
    '200': '#e5e7eb',
    '300': '#d1d5db',
    '400': '#9ca3af',
    '500': '#6b7280',
    '600': '#4b5563',
    '700': '#374151',
    '800': '#1f2937',
    '900': '#111827',
    '950': '#030712',
  },
  neutral: {
    '50': '#fafafa',
    '100': '#f5f5f5',
    '200': '#e5e5e5',
    '300': '#d4d4d4',
    '400': '#a3a3a3',
    '500': '#737373',
    '600': '#525252',
    '700': '#404040',
    '800': '#262626',
    '900': '#171717',
    '950': '#0a0a0a',
  },
} as const satisfies Record<string, Palette>;

const STORAGE_KEY = 'ui-theme-selection:v1';

const DEFAULT_THEME: UiThemeSelection = { primary: 'sky', neutral: 'slate' };

function clampTheme(input: any): UiThemeSelection {
  const primary = input?.primary;
  const neutral = input?.neutral;

  return {
    primary: (primary in PRIMARY_PALETTES ? primary : DEFAULT_THEME.primary) as UiThemeSelection['primary'],
    neutral: (neutral in NEUTRAL_PALETTES ? neutral : DEFAULT_THEME.neutral) as UiThemeSelection['neutral'],
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
  const primary = PRIMARY_PALETTES[theme.primary];
  const neutral = NEUTRAL_PALETTES[theme.neutral];
  const el = document.documentElement;

  (Object.keys(primary) as Shade[]).forEach((shade) => {
    el.style.setProperty(`--ui-color-primary-${shade}`, primary[shade]);
  });
  el.style.setProperty('--ui-primary', primary['500']);

  (Object.keys(neutral) as Shade[]).forEach((shade) => {
    el.style.setProperty(`--ui-color-neutral-${shade}`, neutral[shade]);
  });
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
    PRIMARY_PALETTES,
    NEUTRAL_PALETTES,
    getSavedUiTheme,
    applyUiTheme,
    setSavedUiTheme,
    current,
  };
}

