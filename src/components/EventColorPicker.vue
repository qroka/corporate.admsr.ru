<script setup lang="ts">
/**
 * Цвет личного события: «По умолчанию» (цвет по типу события), несколько
 * базовых цветов и палитра (UColorPicker) для любого другого. v-model — '#rrggbb' или null.
 *
 * Цвета здесь — данные пользователя, а не оформление портала, поэтому заданы
 * hex-значениями, а не токенами темы (ADR-051). Текст события остаётся в цветах темы.
 */
import { computed, ref, watch } from 'vue';

const model = defineModel<string | null>({ default: null });

/** Базовые цвета — с хорошим контрастом и в светлой, и в тёмной схеме. */
const EVENT_BASE_COLORS = [
  { value: '#ef4444', label: 'Красный' },
  { value: '#f97316', label: 'Оранжевый' },
  { value: '#eab308', label: 'Жёлтый' },
  { value: '#22c55e', label: 'Зелёный' },
  { value: '#14b8a6', label: 'Бирюзовый' },
  { value: '#3b82f6', label: 'Синий' },
  { value: '#8b5cf6', label: 'Фиолетовый' },
  { value: '#ec4899', label: 'Розовый' },
] as const;

const pickerOpen = ref(false);
const custom = ref<string>(model.value ?? '#3b82f6');

const isBase = computed(() => EVENT_BASE_COLORS.some((c) => c.value === model.value));
const isCustom = computed(() => !!model.value && !isBase.value);

watch(model, (v) => {
  if (v) custom.value = v;
});

function pick(value: string | null) {
  model.value = value;
}

function onCustom(v: string | undefined) {
  if (!v || !/^#[0-9a-fA-F]{6}$/.test(v)) return;
  custom.value = v.toLowerCase();
  model.value = custom.value;
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2" role="radiogroup" aria-label="Цвет события">
    <UTooltip text="По умолчанию">
      <button
        type="button"
        role="radio"
        :aria-checked="!model"
        aria-label="По умолчанию — цвет по типу события"
        class="grid size-7 place-items-center rounded-full border border-dashed border-accented text-muted transition focus-visible:outline-2 focus-visible:outline-primary"
        :class="!model ? 'ring-2 ring-primary ring-offset-2 ring-offset-default' : ''"
        @click="pick(null)"
      >
        <UIcon name="i-lucide-ban" class="size-3.5" aria-hidden="true" />
      </button>
    </UTooltip>

    <UTooltip v-for="c in EVENT_BASE_COLORS" :key="c.value" :text="c.label">
      <button
        type="button"
        role="radio"
        :aria-checked="model === c.value"
        :aria-label="c.label"
        class="size-7 rounded-full transition focus-visible:outline-2 focus-visible:outline-primary"
        :class="model === c.value ? 'ring-2 ring-primary ring-offset-2 ring-offset-default' : ''"
        :style="{ backgroundColor: c.value }"
        @click="pick(c.value)"
      />
    </UTooltip>

    <UPopover v-model:open="pickerOpen" :content="{ side: 'bottom', align: 'start' }">
      <UTooltip text="Другой цвет">
        <button
          type="button"
          role="radio"
          :aria-checked="isCustom"
          aria-label="Другой цвет — открыть палитру"
          class="grid size-7 place-items-center rounded-full border border-accented transition focus-visible:outline-2 focus-visible:outline-primary"
          :class="isCustom ? 'ring-2 ring-primary ring-offset-2 ring-offset-default' : ''"
          :style="isCustom ? { backgroundColor: model ?? undefined } : undefined"
        >
          <UIcon v-if="!isCustom" name="i-lucide-palette" class="size-3.5 text-muted" aria-hidden="true" />
        </button>
      </UTooltip>

      <template #content>
        <div class="flex flex-col gap-2 p-3">
          <UColorPicker :model-value="custom" format="hex" size="sm" @update:model-value="onCustom" />
          <p class="text-xs text-muted tabular-nums">{{ custom }}</p>
        </div>
      </template>
    </UPopover>
  </div>
</template>
