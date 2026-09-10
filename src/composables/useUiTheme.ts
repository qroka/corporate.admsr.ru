export const PRIMARY_COLORS = [
  'white',
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
] as const;

export const NEUTRAL_COLORS = [
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

export const PRIMARY_COLOR_LABELS: Record<(typeof PRIMARY_COLORS)[number], string> = {
  white: 'Белый / чёрный',
  red: 'Красный',
  orange: 'Оранжевый',
  amber: 'Янтарный',
  yellow: 'Жёлтый',
  lime: 'Лаймовый',
  green: 'Зелёный',
  emerald: 'Изумрудный',
  teal: 'Бирюзовый',
  cyan: 'Циан',
  sky: 'Небесный',
  blue: 'Синий',
  indigo: 'Индиго',
  violet: 'Фиолетовый',
  purple: 'Пурпурный',
  fuchsia: 'Фуксия',
  pink: 'Розовый',
  rose: 'Алый',
};

export const NEUTRAL_COLOR_LABELS: Record<(typeof NEUTRAL_COLORS)[number], string> = {
  slate: 'Сланец',
  gray: 'Серый',
  zinc: 'Цинк',
  neutral: 'Нейтральный',
  stone: 'Камень',
  taupe: 'Тауп',
  mauve: 'Мов',
  mist: 'Туман',
  olive: 'Олива',
};

export const FONT_OPTIONS = [
  {
    id: 'inter',
    label: 'Inter',
    value: "'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  },
  {
    id: 'unbounded',
    label: 'Unbounded',
    value: "'Unbounded', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  },
  {
    id: 'comic-sans',
    label: 'Comic Sans',
    value: "'Comic Sans MS', 'Comic Sans', 'Chalkboard SE', 'Comic Neue', cursive",
  },
  {
    id: 'montserrat',
    label: 'Montserrat',
    value: "'Montserrat', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  },
] as const;

export const RADIUS_OPTIONS = [
  { id: 'none', label: 'Нет', value: '0' },
  { id: 'sm', label: 'Маленький', value: '0.125rem' },
  { id: 'md', label: 'Средний', value: '0.25rem' },
  { id: 'lg', label: 'Большой', value: '0.375rem' },
  { id: 'xl', label: 'Очень большой', value: '0.5rem' },
] as const;

export type UiThemeSelection = {
  primary: (typeof PRIMARY_COLORS)[number];
  neutral: (typeof NEUTRAL_COLORS)[number];
  font: (typeof FONT_OPTIONS)[number]['id'];
  radius: (typeof RADIUS_OPTIONS)[number]['id'];
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

const STORAGE_KEY = 'ui-theme-selection:v1';

export const DEFAULT_THEME: UiThemeSelection = {
  primary: 'emerald',
  neutral: 'zinc',
  font: 'inter',
  radius: 'md',
};

// Контур фавикона (тот же путь, что у логотипа). Перекрашивается под палитру.
const FAVICON_PATH =
  'M29.6904 0C30.9641 0 31.9979 1.02996 31.998 2.29883V3.24121L32.002 16.0791C32.0018 24.8707 24.8442 32.001 16.0195 32.001H0L27.2412 4.8623L22.3604 0H29.6904ZM16.0596 15.9971L0.00195312 31.9971V26.5889H4.25195V22.3545H0.00195312V18.1172H4.25195V13.8799H0.00195312V9.64258H4.25195V5.40918H0.00195312V0L16.0596 15.9971ZM4.25879 18.1143V22.3516H8.51172V18.1143H4.25879ZM8.51758 13.8799V18.1172H12.7715V13.8799H8.51758ZM4.25879 9.64258V13.8799H8.51172V9.64258H4.25879Z';

const EMERALD_500 = '#00DC82';

function pickRandom<T>(items: readonly T[]): T {
  return items[Math.floor(Math.random() * items.length)]!;
}

/** Перекрашивает favicon вкладки в переданный цвет (data-URI SVG). */
function applyFavicon(color: string) {
  if (typeof document === 'undefined') return;
  let link = document.querySelector<HTMLLinkElement>("link[rel~='icon']");
  if (!link) {
    link = document.createElement('link');
    link.rel = 'icon';
    document.head.appendChild(link);
  }
  link.type = 'image/svg+xml';
  const svg =
    `<svg width="32" height="32" viewBox="0 0 32 32" xmlns="http://www.w3.org/2000/svg">` +
    `<path d="${FAVICON_PATH}" fill="${color || EMERALD_500}"/></svg>`;
  link.href = `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

function clampTheme(input: any): UiThemeSelection {
  const primary = input?.primary;
  const neutral = input?.neutral;
  const font = input?.font;
  const radius = input?.radius;
  const fontIds = FONT_OPTIONS.map((f) => f.id) as readonly string[];
  const radiusIds = RADIUS_OPTIONS.map((r) => r.id) as readonly string[];

  return {
    primary: (PRIMARY_COLORS.includes(primary) ? primary : DEFAULT_THEME.primary) as UiThemeSelection['primary'],
    neutral: (NEUTRAL_COLORS.includes(neutral) ? neutral : DEFAULT_THEME.neutral) as UiThemeSelection['neutral'],
    font: (fontIds.includes(font) ? font : DEFAULT_THEME.font) as UiThemeSelection['font'],
    radius: (radiusIds.includes(radius) ? radius : DEFAULT_THEME.radius) as UiThemeSelection['radius'],
  };
}

export function getFontOption(id: UiThemeSelection['font']) {
  return FONT_OPTIONS.find((f) => f.id === id) ?? FONT_OPTIONS[0];
}

export function getRadiusOption(id: UiThemeSelection['radius']) {
  return RADIUS_OPTIONS.find((r) => r.id === id) ?? RADIUS_OPTIONS[2];
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

  const font = getFontOption(theme.font);
  el.style.setProperty('--font-sans', font.value);
  el.dataset.uiFont = theme.font;

  const radius = getRadiusOption(theme.radius);
  el.style.setProperty('--ui-radius', radius.value);

  // Favicon под выбранную палитру (вызывается и на старте, и при смене цвета)
  applyFavicon(primary500 || EMERALD_500);
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
  return theme;
}

export function randomUiTheme(current?: UiThemeSelection): UiThemeSelection {
  const base = current ?? getSavedUiTheme();
  let next: UiThemeSelection = {
    primary: pickRandom(PRIMARY_COLORS),
    neutral: pickRandom(NEUTRAL_COLORS),
    font: pickRandom(FONT_OPTIONS).id,
    radius: pickRandom(RADIUS_OPTIONS).id,
  };

  // Avoid identical result when possible.
  if (
    next.primary === base.primary
    && next.neutral === base.neutral
    && next.font === base.font
    && next.radius === base.radius
  ) {
    next = {
      ...next,
      primary: pickRandom(PRIMARY_COLORS.filter((c) => c !== base.primary)),
    };
  }

  return setSavedUiTheme(next);
}

export function resetUiTheme(): UiThemeSelection {
  return setSavedUiTheme({ ...DEFAULT_THEME });
}

export function useUiTheme() {
  const current = getSavedUiTheme();
  return {
    PRIMARY_COLORS,
    NEUTRAL_COLORS,
    FONT_OPTIONS,
    RADIUS_OPTIONS,
    DEFAULT_THEME,
    getSavedUiTheme,
    applyUiTheme,
    setSavedUiTheme,
    randomUiTheme,
    resetUiTheme,
    getFontOption,
    getRadiusOption,
    current,
  };
}
