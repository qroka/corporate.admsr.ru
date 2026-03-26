<script setup>
import { ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

defineProps({
  isDark: {
    type: Boolean,
    default: false,
  },
  activeNav: {
    type: String,
    default: 'events',
  },
});

const emit = defineEmits(['toggle-theme']);

const route = useRoute();
const router = useRouter();
const menuOpen = ref(false);

function navigate(name) {
  router.push({ name });
}

function navigateAndClose(name) {
  menuOpen.value = false;
  navigate(name);
}

watch(
  () => route.fullPath,
  () => {
    menuOpen.value = false;
  },
);
</script>

<template>
  <header class="flex items-center h-fit w-full max-w-none min-w-0 p-0">
    <div class="bg-elevated flex items-center justify-between gap-4 p-6 rounded-3xl w-full min-w-0">
      <RouterLink :to="{ name: 'home' }" class="group min-w-0">
        <UContainer class="header-logo w-fit justify-start flex items-center gap-1.5 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
          <UContainer class="relative w-20 h-20 sm:p-0 md:p-0 lg:p-0 xl:p-0">
            <svg
              :class="[
                'absolute w-20 h-20 transition-all duration-300 ease-out fill-current text-dimmed',
                'opacity-100 group-hover:opacity-0',
              ]"
              viewBox="0 0 48 48"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M37.6895 8C38.963 8.00009 39.9969 9.0301 39.9971 10.2988V11.2412L40 24.0781C40 32.8696 32.8431 40 24.0186 40H8L35.2393 12.8623L30.3594 8H37.6895ZM24.0586 23.9971L8.00195 39.9971V34.5879H12.252V30.3545H8.00195V26.1172H12.252V21.8799H8.00195V17.6426H12.252V13.4092H8.00195V8L24.0586 23.9971ZM12.2578 26.1143V30.3506H16.5107V26.1143H12.2578ZM16.5176 21.8799V26.1172H20.7705V21.8799H16.5176ZM12.2578 17.6426V21.8799H16.5107V17.6426H12.2578Z"
              />
            </svg>
            <svg
              :class="[
                'overflow-visible absolute w-20 h-20 transition-all duration-300 ease-out',
                'group-hover:filter-[drop-shadow(0_0_12px_currentColor)_drop-shadow(0_0_6px_currentColor)_drop-shadow(0_0_1px_currentColor)]',
                'opacity-0 group-hover:opacity-100 fill-primary text-primary',
              ]"
              viewBox="16 16 48 48"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M53.8815 24.192C55.155 24.1921 56.1889 25.2221 56.1891 26.4908V27.4332L56.192 40.2701C56.192 49.0616 49.0351 56.192 40.2106 56.192H24.192L51.4313 29.0543L46.5514 24.192H53.8815ZM40.2506 40.1891L24.194 56.1891V50.7799H28.444V46.5465H24.194V42.3092H28.444V38.0719H24.194V33.8346H28.444V29.6012H24.194V24.192L40.2506 40.1891ZM28.4498 42.3062V46.5426H32.7028V42.3062H28.4498ZM32.7096 38.0719V42.3092H36.9625V38.0719H32.7096ZM28.4498 33.8346V38.0719H32.7028V33.8346H28.4498Z"
                fill="var(--ui-primary)"
              />
            </svg>
          </UContainer>
          <span class="text-4xl font-unbounded font-medium text-highlighted truncate">
            Корпоративный портал
          </span>
        </UContainer>
      </RouterLink>

      <div class="flex items-center gap-2 shrink-0">
        <UTooltip :text="isDark ? 'Светлая тема' : 'Тёмная тема'">
          <UButton
            type="button"
            color="neutral"
            variant="subtle"
            size="xl"
            class="rounded-3xl w-20 h-20 p-0 inline-flex items-center justify-center [&_svg]:size-10"
            :icon="isDark ? 'i-lucide-sun' : 'i-lucide-moon'"
            aria-label="Сменить тему"
            @click="(e) => emit('toggle-theme', e)"
          />
        </UTooltip>
      </div>
    </div>
  </header>

</template>
