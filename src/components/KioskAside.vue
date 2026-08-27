<script setup>
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

const route = useRoute();
const router = useRouter();

const activeNav = computed(() => {
  const name = String(route.name ?? '');
  // kiosk-* routes should highlight the corresponding section
  if (name.includes('events')) return 'events';
  // Только список альбомов — внутри альбома кнопка не активна
  if (name === 'kiosk-gallery' || name === 'gallery') return 'gallery';
  return '';
});

function navigate(name) {
  router.push({ name });
}
</script>

<template>
   <UContainer class="header-nav relative grid grid-cols-2 gap-3 rounded-3xl h-[160px] w-full z-0 mx-0">
      <UButton type="button" color="neutral"  class="relative z-0 shadow-none p-6 text-3xl font-unbounded transition-all text-start cursor-pointer duration-300 ease-out rounded-3xl bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:size-12 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'events' ? 'z-10 bg-primary text-zinc-50 shadow-none dark:shadow-brand [&_svg]:text-zinc-50 hover:bg-primary hover:text-zinc-50 active:bg-primary active:text-zinc-50 active:[&_svg]:text-zinc-50' : ''" size="xl" icon="i-lucide-calendar"
        @click="navigate('events')">
        Мероприятия
      </UButton>
      <UButton type="button" color="neutral"  class="relative z-0 shadow-none p-6 text-3xl font-unbounded transition-all text-start cursor-pointer duration-300 ease-out rounded-3xl bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:size-12 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'gallery' ? 'z-10 bg-primary text-zinc-50 shadow-none dark:shadow-brand [&_svg]:text-zinc-50 hover:bg-primary hover:text-zinc-50 active:bg-primary active:text-zinc-50 active:[&_svg]:text-zinc-50' : ''" size="xl" icon="i-lucide-images"
        @click="navigate('gallery')">
        Фотогалерея
      </UButton>
    </UContainer>
</template>
