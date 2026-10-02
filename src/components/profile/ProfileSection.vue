<script setup lang="ts">
/** Блок страницы профиля с плашкой-заголовком, как «Информация» / «Стена» у старого ВКонтакте. */
defineProps<{ title: string; titleId?: string }>();
</script>

<template>
  <section class="flex flex-col gap-3" :aria-labelledby="titleId">
    <div class="flex items-center justify-between gap-3 rounded-md bg-elevated px-3 py-2">
      <h2 :id="titleId" class="text-sm font-semibold text-highlighted">{{ title }}</h2>
      <div v-if="$slots.aside" class="text-xs text-muted">
        <slot name="aside" />
      </div>
    </div>
    <div class="px-1">
      <slot />
    </div>
  </section>
</template>
