<script setup lang="ts">
/**
 * Плавающая кнопка «Наверх» для страниц, которые скроллят свой контейнер
 * (корень приложения — h-dvh overflow-hidden, документ не скроллится).
 * Родитель должен быть `relative` — обычно это `UMain`.
 */
import { computed } from 'vue';
import { useMediaQuery, useScroll } from '@vueuse/core';

const props = withDefaults(
  defineProps<{
    target: HTMLElement | null;
    /** Сколько пикселей прокрутить, прежде чем показать кнопку. */
    threshold?: number;
  }>(),
  { threshold: 600 },
);

const { y } = useScroll(() => props.target);
const reduceMotion = useMediaQuery('(prefers-reduced-motion: reduce)');
const visible = computed(() => y.value > props.threshold);

function scrollToTop() {
  props.target?.scrollTo({ top: 0, behavior: reduceMotion.value ? 'auto' : 'smooth' });
}
</script>

<template>
  <Transition
    enter-active-class="transition duration-200 motion-reduce:transition-none"
    enter-from-class="opacity-0 translate-y-2"
    leave-active-class="transition duration-150 motion-reduce:transition-none"
    leave-to-class="opacity-0 translate-y-2"
  >
    <div v-if="visible" class="absolute bottom-6 right-6 z-10">
      <UTooltip text="Наверх">
        <UButton
          icon="i-lucide-arrow-up"
          color="neutral"
          variant="outline"
          size="xl"
          class="rounded-full bg-default shadow-lg"
          aria-label="Наверх"
          @click="scrollToTop"
        />
      </UTooltip>
    </div>
  </Transition>
</template>
