<template>
  <div class="min-h-screen grid place-items-center bg-default">
    <div class="w-[min(1080px,100vw)] h-[min(1920px,100vh)] aspect-9/16 flex flex-col gap-6 p-6">
      <KioskHeader
        :hero-title="heroTitle"
        :hero-subtitle="heroSubtitle"
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
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import KioskHeader from './components/KioskHeader.vue';
import KioskAside from './components/KioskAside.vue';

type KioskTile = { title: string; icon: string; routeName?: string; to?: string; disabled?: boolean };

const router = useRouter();
const allServicesOpen = ref(false);

const heroTitle = 'Корпоративный портал';
const heroSubtitle = 'Киоск-режим';

function go(tile: KioskTile) {
  if (tile.disabled) return;
  if (tile.routeName) router.push({ name: tile.routeName });
  else if (tile.to) router.push(tile.to);
}
</script>
