<script setup lang="ts">
/**
 * Версия выпуска девблога под заголовком (ADR-052): основной цвет темы с
 * переливом. Цвет — переменная темы --ui-primary (следует за настройкой
 * «Основной»); перелив выключается при «уменьшить движение» в системе.
 */
defineProps<{ version: string }>();
</script>

<template>
  <p class="devblog-version text-base font-bold tracking-wide tabular-nums">
    Версия {{ version }}
  </p>
</template>

<style scoped>
.devblog-version {
  /* Перелив: полоса светлее основного цвета бежит по тексту. */
  background-image: linear-gradient(
    100deg,
    var(--ui-primary) 0%,
    var(--ui-primary) 35%,
    color-mix(in oklab, var(--ui-primary) 40%, white) 50%,
    var(--ui-primary) 65%,
    var(--ui-primary) 100%
  );
  background-size: 250% 100%;
  background-clip: text;
  -webkit-background-clip: text;
  color: transparent;
  animation: devblog-version-shimmer 3.2s linear infinite;
}

@keyframes devblog-version-shimmer {
  from {
    background-position: 100% 0;
  }
  to {
    background-position: -150% 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .devblog-version {
    animation: none;
    background-image: none;
    color: var(--ui-primary);
  }
}
</style>
