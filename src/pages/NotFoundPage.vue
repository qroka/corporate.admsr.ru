<script setup lang="ts">
import { useRoute } from 'vue-router';

/** Неизвестный адрес: раньше был пустой экран без выхода (IMP-43). */
const route = useRoute();
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
      <UEmpty
        variant="naked"
        icon="i-lucide-map-pin-off"
        title="Страница не найдена"
        :description="`Адреса ${route.path} на портале нет — возможно, ссылка устарела или в ней опечатка.`"
        :actions="[
          { label: 'На рабочий стол', icon: 'i-lucide-grip', color: 'primary', to: { name: 'home' } },
          { label: 'Назад', icon: 'i-lucide-arrow-left', color: 'neutral', variant: 'ghost', onClick: () => $router.back() },
        ]"
        class="py-24"
      />
    </div>
  </UMain>
</template>
