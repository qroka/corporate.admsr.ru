<script setup>
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

const route = useRoute();
const router = useRouter();

const activeNav = computed(() => {
  const name = String(route.name ?? '');
  // kiosk-* routes should highlight the corresponding section
  if (name.includes('events')) return 'events';
  if (name.includes('gallery')) return 'gallery';
  if (name.includes('newcomers')) return 'newcomers';
  if (name.includes('culture')) return 'culture';
  return '';
});

function navigate(name) {
  router.push({ name });
}
</script>

<template>
   <UContainer class="header-nav relative grid grid-cols-2 gap-3 rounded-3xl h-[350px] w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 z-0 mx-0">
      <UButton type="button" color="neutral"  class="relative z-0 shadow-none p-6 text-3xl font-unbounded transition-all text-start cursor-pointer duration-300 ease-out rounded-3xl bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:size-12 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'events' ? 'z-10 bg-primary text-inverted shadow-none dark:shadow-brand [&_svg]:text-inverted hover:bg-primary hover:text-inverted active:bg-primary active:text-inverted active:[&_svg]:text-inverted' : ''" size="xl" icon="i-lucide-calendar"
        @click="navigate('events')">
        Мероприятия
      </UButton>
      <UButton type="button" color="neutral"  class="relative z-0 shadow-none p-6 text-3xl font-unbounded transition-all text-start cursor-pointer duration-300 ease-out rounded-3xl bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:size-12 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'gallery' ? 'z-10 bg-primary text-inverted shadow-none dark:shadow-brand [&_svg]:text-inverted hover:bg-primary hover:text-inverted active:bg-primary active:text-inverted active:[&_svg]:text-inverted' : ''" size="xl" icon="i-lucide-images"
        @click="navigate('gallery')">
        Фотогалерея
      </UButton>
      <UButton type="button" color="neutral"  class="relative z-0 shadow-none p-6 text-3xl font-unbounded transition-all text-start cursor-pointer duration-300 ease-out rounded-3xl bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:size-12 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'newcomers' ? 'z-10 bg-primary text-inverted shadow-none dark:shadow-brand [&_svg]:text-inverted hover:bg-primary hover:text-inverted active:bg-primary active:text-inverted active:[&_svg]:text-inverted' : ''" size="xl" icon="i-lucide-sparkles"
        @click="navigate('newcomers')">
        Новичкам
      </UButton>
      <UButton type="button" color="neutral"  class="relative z-0 shadow-none p-6 text-3xl font-unbounded transition-all text-start cursor-pointer duration-300 ease-out rounded-3xl bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:size-12 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'culture' ? 'z-10 bg-primary text-inverted shadow-none dark:shadow-brand [&_svg]:text-inverted hover:bg-primary hover:text-inverted active:bg-primary active:text-inverted active:[&_svg]:text-inverted' : ''" size="xl" icon="i-lucide-users"
        @click="navigate('culture')">
        Корпоративная культура
      </UButton>
    </UContainer>
</template>
