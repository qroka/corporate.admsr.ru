<template>
  <div class="app-root h-dvh overflow-hidden">
    <UApp :locale="ru">
      <RouterView v-if="isKiosk || isAuth || isPublic" />

      <UDashboardGroup
        v-else
        unit="px"
        storage-key="portal-dashboard"
        class="h-dvh w-full"
        :ui="{ base: 'bg-default text-highlighted' }"
      >
        <AppAside
          :is-dark="isDark"
          @toggle-theme="startThemeTransition"
        />

        <UDashboardSearch
          placeholder="Искать сотрудника, памятку, документ..."
          :groups="searchGroups"
          :color-mode="false"
        />

        <UDashboardPanel
          id="portal-main"
          class="bg-default"
          :ui="{
            root: 'bg-default',
            body: 'px-4 pt-4 pb-0 sm:px-4 sm:pt-4 sm:pb-0 bg-default overflow-hidden',
          }"
        >
          <template #header>
            <AppHeader
              :is-dark="isDark"
              :active-nav="activeNav"
              @toggle-theme="startThemeTransition"
            />
          </template>

          <template #body>
            <RouterView />
          </template>
        </UDashboardPanel>
      </UDashboardGroup>
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
import { usePortalSearchGroups } from './composables/usePortalNavigation';

const route = useRoute();
const router = useRouter();
const searchGroups = usePortalSearchGroups();

onMounted(() => startSessionActivity(router));

const activeNav = computed(() => (route.name ?? 'events'));

const isKiosk = computed(() => route.matched.some((r) => r.meta?.kiosk));
const isAuth = computed(() => route.meta?.layout === 'auth');
const isPublic = computed(() => route.meta?.public === true);

const isDark = ref(false);
const { syncFromStorage, toggleColorMode } = useColorMode(isDark, {
  enabled: computed(() => !isKiosk.value),
});

watch(isKiosk, (kiosk) => {
  if (!kiosk) syncFromStorage();
});

const startThemeTransition = (event) => {
  const anyDoc = document;

  if (!anyDoc.startViewTransition) {
    toggleColorMode();
    return;
  }

  const x = event?.clientX ?? window.innerWidth / 2;
  const y = event?.clientY ?? window.innerHeight / 2;
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
