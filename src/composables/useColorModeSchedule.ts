import { onBeforeUnmount, onMounted, type Ref } from 'vue';
import { applyColorModeToDocument, applyMainColorModeFromStorage } from './useColorMode';

export const KIOSK_COLOR_MODE_KEY = 'ui-kiosk-color-mode';
export type KioskColorModePreference = 'auto' | 'light' | 'dark';

/** 00:00–14:59 — светлая, 15:00–23:59 — тёмная */
export function getScheduledColorMode(now = new Date()): 'light' | 'dark' {
  return now.getHours() >= 15 ? 'dark' : 'light';
}

export function readKioskColorModePreference(): KioskColorModePreference {
  const saved = localStorage.getItem(KIOSK_COLOR_MODE_KEY);
  if (saved === 'light' || saved === 'dark' || saved === 'auto') return saved;
  return 'auto';
}

export function resolveKioskEffectiveColorMode(pref = readKioskColorModePreference()): 'light' | 'dark' {
  if (pref === 'light' || pref === 'dark') return pref;
  return getScheduledColorMode();
}

export function applyKioskColorModeFromStorage() {
  applyColorModeToDocument(resolveKioskEffectiveColorMode());
}

function msUntilNextScheduleBoundary(now = new Date()): number {
  const next = new Date(now);
  if (now.getHours() < 15) {
    next.setHours(15, 0, 0, 0);
  } else {
    next.setDate(next.getDate() + 1);
    next.setHours(0, 0, 0, 0);
  }
  return Math.max(0, next.getTime() - now.getTime());
}

export function useKioskColorModeSchedule(isDark: Ref<boolean>) {
  let intervalId: number | null = null;
  let boundaryTimerId: number | null = null;

  function syncFromPreference() {
    const effective = resolveKioskEffectiveColorMode();
    isDark.value = effective === 'dark';
    applyColorModeToDocument(effective);
  }

  function setPreference(pref: KioskColorModePreference) {
    localStorage.setItem(KIOSK_COLOR_MODE_KEY, pref);
    syncFromPreference();
  }

  function toggleManualPreference() {
    const effective = resolveKioskEffectiveColorMode();
    setPreference(effective === 'dark' ? 'light' : 'dark');
  }

  function scheduleNextBoundaryCheck() {
    if (boundaryTimerId) window.clearTimeout(boundaryTimerId);
    boundaryTimerId = window.setTimeout(() => {
      if (readKioskColorModePreference() === 'auto') syncFromPreference();
      scheduleNextBoundaryCheck();
    }, msUntilNextScheduleBoundary() + 250);
  }

  onMounted(() => {
    syncFromPreference();
    intervalId = window.setInterval(() => {
      if (readKioskColorModePreference() === 'auto') syncFromPreference();
    }, 60_000);
    scheduleNextBoundaryCheck();
  });

  onBeforeUnmount(() => {
    if (intervalId) window.clearInterval(intervalId);
    if (boundaryTimerId) window.clearTimeout(boundaryTimerId);
    applyMainColorModeFromStorage();
  });

  return {
    syncFromPreference,
    setPreference,
    toggleManualPreference,
  };
}
