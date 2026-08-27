<template>
  <div class="app-root h-screen overflow-hidden">
    <UApp :locale="ru">
      <RouterView v-if="isKiosk || isAuth || isPublic" />

      <!-- Основной лейаут приложения -->
      <div
        v-else
        class="h-screen w-full max-w-[1600px] mx-auto overflow-hidden flex flex-col justify-start p-6 gap-6 transition-colors"
        :class="containerClass"
      >
        <AppHeader
          :is-dark="isDark"
          :active-nav="activeNav"
          @toggle-theme="startThemeTransition"
        />

        <main class="flex flex-1 w-full h-full gap-6 min-h-0">
          <AppAside
            :is-dark="isDark"
            @toggle-theme="startThemeTransition"
          />

          <section class="flex-1 min-w-0 min-h-0 overflow-y-auto max-h-full">
            <RouterView />
          </section>
        </main>
      </div>
    </UApp>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { ru } from '@nuxt/ui/locale';
import { useRoute, useRouter } from 'vue-router';
import AppHeader from './components/AppHeader.vue';
import AppAside from './components/AppAside.vue';
import { startSessionActivity } from './composables/useSessionActivity';
import { useColorMode } from './composables/useColorMode';

// Текущий маршрут (используется для подсветки активных пунктов навигации)
const route = useRoute();
const router = useRouter();

// Авто-логаут по 24ч бездействия (heartbeat + проверка статуса)
onMounted(() => startSessionActivity(router));

// Имя активного пункта верхнего меню (events / gallery / newcomers / culture)
const activeNav = computed(() => (route.name ?? 'events'));

const isKiosk = computed(() => route.matched.some((r) => r.meta?.kiosk));
const isAuth = computed(() => route.meta?.layout === 'auth');
const isPublic = computed(() => route.meta?.public === true);


// --- Поддержка светлой / тёмной темы для Nuxt UI ---
const isDark = ref(false);
const { syncFromStorage, toggleColorMode } = useColorMode(isDark, {
  enabled: computed(() => !isKiosk.value),
});

watch(isKiosk, (kiosk) => {
  if (!kiosk) syncFromStorage();
});

// Анимированная смена темы с помощью View Transitions API
const startThemeTransition = (event) => {
  const anyDoc = document;

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
};

// Класс фона/текста для корневого контейнера (визуальное переключение темы)
const containerClass = computed(() => 'bg-(--ui-bg) text-(--ui-text-highlighted)');
</script>

<style>
::view-transition-old(root),
::view-transition-new(root) {
  animation: none;
  mix-blend-mode: normal;
}

::view-transition-new(root) {
  z-index: 9999;
}

::view-transition-old(root) {
  z-index: 1;
}
</style>
