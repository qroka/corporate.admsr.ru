<script setup lang="ts">
type SurroundItem = {
  title: string;
  to: string;
};

defineProps<{
  prev?: SurroundItem | null;
  next?: SurroundItem | null;
}>();
</script>

<template>
  <div class="p-px grid grid-cols-1 sm:grid-cols-2 gap-3 select-none">
    <UPageCard
      v-if="prev"
      :to="prev.to"
      variant="outline"
      class="rounded-2xl"
      :ui="{ container: 'p-4 sm:p-5' }"
      :title="prev.title"
    >
      <template #header>
        <div class="flex items-center gap-2 text-sm text-muted">
          <UIcon name="i-lucide-arrow-left" class="size-4" />
          <span>Предыдущее</span>
        </div>
      </template>
    </UPageCard>
    <div v-else class="hidden sm:block" />

    <UPageCard
      v-if="next"
      :to="next.to"
      variant="outline"
      class="rounded-2xl"
      :ui="{
        container: 'p-4 sm:p-5',
        wrapper: 'flex flex-col flex-1 items-end justify-start select-none',
        header: 'mb-2',
        body: 'flex-none',
        title: 'text-end',
      }"
      :title="next.title"
    >
      <template #header>
        <div class="flex items-center justify-end gap-2 text-sm text-muted">
          <span>Следующее</span>
          <UIcon name="i-lucide-arrow-right" class="size-4" />
        </div>
      </template>
    </UPageCard>
  </div>
</template>

