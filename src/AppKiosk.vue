<template>
  <div class="min-h-screen grid place-items-center bg-default">
    <div class="w-[min(1080px,100vw)] h-[min(1920px,100vh)] aspect-9/16 flex flex-col gap-6 p-6">
      <KioskHeader
        :hero-title="heroTitle"
        :hero-subtitle="heroSubtitle"
        :is-dark="isDark"
        @toggle-theme="startThemeTransition"
        @open-all-services="allServicesOpen = true"
      />

      <main class="flex-1 min-h-0 flex flex-col">
        <section class="flex-1 min-h-0">
          <RouterView />
        </section>
      </main>

      <aside class="flex-none">
        <KioskAside @open-all-services="allServicesOpen = true" />
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import KioskHeader from './components/KioskHeader.vue';
import KioskAside from './components/KioskAside.vue';

type KioskTile = { title: string; icon: string; routeName?: string; to?: string; disabled?: boolean };

const route = useRoute();
const router = useRouter();
const allServicesOpen = ref(false);

const heroTitle = 'Корпоративный портал';
const heroSubtitle = 'Киоск-режим';

function go(tile: KioskTile) {
  if (tile.disabled) return;
  if (tile.routeName) router.push({ name: tile.routeName });
  else if (tile.to) router.push(tile.to);
}

// --- Поддержка светлой / тёмной темы для Nuxt UI (kiosk) ---
const COLOR_MODE_KEY = 'ui-color-mode';
const isDark = ref(false);

function applyColorMode(mode: 'light' | 'dark') {
  const root = document.documentElement;
  if (!root) return;

  if (mode === 'dark') root.classList.add('dark');
  else root.classList.remove('dark');
}

function toggleColorMode() {
  isDark.value = !isDark.value;
}

// Анимированная смена темы с помощью View Transitions API
function startThemeTransition(event: MouseEvent) {
  const anyDoc = document as any;
  if (!anyDoc.startViewTransition) {
    toggleColorMode();
    return;
  }

  const x = event.clientX;
  const y = event.clientY;
  const endRadius =
    Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y),
    ) + 8;

  const transition = anyDoc.startViewTransition(() => {
    toggleColorMode();
  });

  transition.ready.then(() => {
    const duration = 600;
    anyDoc.documentElement.animate(
      {
        clipPath: [
          `circle(0px at ${x}px ${y}px)`,
          `circle(${endRadius}px at ${x}px ${y}px)`,
        ],
      },
      {
        duration,
        easing: 'cubic-bezier(.76,.32,.29,.99)',
        pseudoElement: '::view-transition-new(root)',
      },
    );
  });
}

// ─── Idle timeout: return to /kiosk ───────────────────────────────────────────
const IDLE_MS = 60_000;
let idleTimer: number | null = null;

function resetIdleTimer() {
  if (idleTimer) window.clearTimeout(idleTimer);
  idleTimer = window.setTimeout(() => {
    // only act if we are still inside kiosk routes
    if (!route.matched?.some((r) => r.meta?.kiosk)) return;
    if (route.name !== 'kiosk') void router.push({ name: 'kiosk' });
    else window.dispatchEvent(new Event('kiosk-idle'));
  }, IDLE_MS);
}

const idleEvents: Array<keyof WindowEventMap> = [
  'pointerdown',
  'pointermove',
  'keydown',
  'wheel',
  'touchstart',
  'scroll',
];

onMounted(() => {
  const saved = localStorage.getItem(COLOR_MODE_KEY);
  let mode: 'light' | 'dark';

  if (saved === 'light' || saved === 'dark') {
    mode = saved;
  } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    mode = 'dark';
  } else {
    mode = 'light';
  }

  isDark.value = mode === 'dark';
  applyColorMode(mode);

  resetIdleTimer();
  for (const ev of idleEvents) window.addEventListener(ev, resetIdleTimer, { passive: true, capture: true });
});

watch(isDark, (value) => {
  const mode: 'light' | 'dark' = value ? 'dark' : 'light';
  localStorage.setItem(COLOR_MODE_KEY, mode);
  applyColorMode(mode);
});

onUnmounted(() => {
  if (idleTimer) window.clearTimeout(idleTimer);
  idleTimer = null;
  for (const ev of idleEvents) window.removeEventListener(ev, resetIdleTimer, { capture: true } as any);
});
</script>
