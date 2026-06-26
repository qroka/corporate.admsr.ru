<script setup lang="ts">
import { computed } from 'vue';

export type ChartItem = { label: string; count: number; percent: number };

const props = withDefaults(defineProps<{ type?: 'bar' | 'pie'; items: ChartItem[] }>(), {
  type: 'bar',
});

const PALETTE = ['#10b981', '#3b82f6', '#f59e0b', '#8b5cf6', '#ef4444', '#14b8a6', '#ec4899', '#84cc16', '#f97316', '#06b6d4'];
const color = (i: number) => PALETTE[i % PALETTE.length];

// Сегменты для кольцевой диаграммы (circ ≈ 100 при r = 15.915)
const segments = computed(() => {
  let cum = 0;
  return props.items.map((it, i) => {
    const seg = { percent: Math.max(0, it.percent), offset: 25 - cum, color: color(i) };
    cum += Math.max(0, it.percent);
    return seg;
  });
});
</script>

<template>
  <!-- Столбцовая -->
  <div v-if="type === 'bar'" class="flex flex-col gap-2">
    <div v-for="(it, i) in items" :key="i" class="flex items-center gap-3">
      <span class="w-36 shrink-0 truncate text-sm text-default" :title="it.label">{{ it.label }}</span>
      <div class="flex-1 h-4 rounded bg-elevated overflow-hidden">
        <div class="h-full rounded transition-all" :style="{ width: `${it.percent}%`, background: color(i) }" />
      </div>
      <span class="w-20 shrink-0 text-right text-xs text-muted tabular-nums">{{ it.percent }}% · {{ it.count }}</span>
    </div>
  </div>

  <!-- Кольцевая -->
  <div v-else class="flex items-center gap-5 flex-wrap">
    <svg viewBox="0 0 42 42" class="size-32 shrink-0">
      <circle cx="21" cy="21" r="15.915" fill="none" class="text-default opacity-10" stroke="currentColor" stroke-width="5" />
      <circle
        v-for="(seg, i) in segments"
        :key="i"
        cx="21" cy="21" r="15.915"
        fill="none"
        :stroke="seg.color"
        stroke-width="5"
        :stroke-dasharray="`${seg.percent} ${100 - seg.percent}`"
        :stroke-dashoffset="seg.offset"
      />
    </svg>
    <div class="flex flex-col gap-1.5 text-sm min-w-0">
      <div v-for="(it, i) in items" :key="i" class="flex items-center gap-2">
        <span class="size-3 rounded-sm shrink-0" :style="{ background: color(i) }" />
        <span class="truncate text-default" :title="it.label">{{ it.label }}</span>
        <span class="text-muted tabular-nums shrink-0">— {{ it.percent }}% ({{ it.count }})</span>
      </div>
    </div>
  </div>
</template>
