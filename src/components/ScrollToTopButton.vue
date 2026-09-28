<script setup lang="ts">
/**
 * Плавающая кнопка «Наверх» для страниц, которые скроллят свой контейнер
 * (корень приложения — h-dvh overflow-hidden, документ не скроллится).
 * Закреплена в углу окна, а не внутри UMain: UMain ограничен max-w-[1600px]
 * и на широком экране кнопка оказывалась поверх карточек.
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
    <UButton
      v-if="visible"
      icon="i-lucide-arrow-up"
      label="Наверх"
      color="primary"
      size="lg"
      class="fixed bottom-4 right-4 z-50 rounded-full shadow-lg"
      @click="scrollToTop"
    />
  </Transition>
</template>
