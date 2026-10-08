<script setup lang="ts">
/**
 * Обложка девблога между версией и текстом (ADR-052) — так же, как фото на
 * странице новости: картинка целиком (object-contain) на размытой подложке из
 * неё же, без обрезки своей загруженной обложки.
 */
withDefaults(defineProps<{ src: string; alt?: string }>(), { alt: 'Обложка девблога' });
</script>

<template>
  <div class="relative h-48 sm:h-64 w-full shrink-0 overflow-hidden rounded-panel bg-muted">
    <img :src="src" alt="" aria-hidden="true" class="absolute inset-0 size-full scale-110 object-cover blur-2xl" />
    <img :src="src" :alt="alt" class="relative z-10 block size-full object-contain" />
  </div>
</template>
