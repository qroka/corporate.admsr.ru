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
  <header class="flex items-center h-fit justify-between w-full gap-3 min-w-0 p-0">
    <RouterLink
      :to="{ name: 'home' }"
      class="group shrink-0 w-[72px] sm:w-auto bg-elevated flex items-center justify-center sm:justify-start gap-1.5 p-3 sm:pr-5 rounded-full cursor-pointer transition-all duration-300 ease-out"
    >
      <div class="header-logo flex items-center gap-1.5 p-0">
        <div class="relative w-10 h-10 p-0">
          <svg
            :class="[
              'absolute w-10 h-10 duration-300 ease-out',
              isDark
                ? 'opacity-100 dark:fill-zinc-500 group-hover:opacity-0'
                : 'opacity-100 fill-zinc-400 group-hover:fill-brand'
            ]"
            viewBox="0 0 48 48"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M37.6895 8C38.963 8.00009 39.9969 9.0301 39.9971 10.2988V11.2412L40 24.0781C40 32.8696 32.8431 40 24.0186 40H8L35.2393 12.8623L30.3594 8H37.6895ZM24.0586 23.9971L8.00195 39.9971V34.5879H12.252V30.3545H8.00195V26.1172H12.252V21.8799H8.00195V17.6426H12.252V13.4092H8.00195V8L24.0586 23.9971ZM12.2578 26.1143V30.3506H16.5107V26.1143H12.2578ZM16.5176 21.8799V26.1172H20.7705V21.8799H16.5176ZM12.2578 17.6426V21.8799H16.5107V17.6426H12.2578Z" />
          </svg>
          <svg
            :class="[
              'overflow-visible absolute w-10 h-10 transition-opacity duration-300 ease-out',
              isDark ? 'opacity-0 group-hover:opacity-100' : 'opacity-0'
            ]"
            viewBox="16 16 48 48"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M53.8815 24.192C55.155 24.1921 56.1889 25.2221 56.1891 26.4908V27.4332L56.192 40.2701C56.192 49.0616 49.0351 56.192 40.2106 56.192H24.192L51.4313 29.0543L46.5514 24.192H53.8815ZM40.2506 40.1891L24.194 56.1891V50.7799H28.444V46.5465H24.194V42.3092H28.444V38.0719H24.194V33.8346H28.444V29.6012H24.194V24.192L40.2506 40.1891ZM28.4498 42.3062V46.5426H32.7028V42.3062H28.4498ZM32.7096 38.0719V42.3092H36.9625V38.0719H32.7096ZM28.4498 33.8346V38.0719H32.7028V33.8346H28.4498Z"
              fill="#50C891"
            />
          </svg>
        </div>
        <span class="hidden sm:inline text-xl font-display font-normal leading-10 text-zinc-700 dark:text-zinc-50">
          Корпоративный портал
        </span>
      </div>
    </RouterLink>

    <div class="header-nav hidden lg:flex bg-elevated rounded-full w-fit items-center gap-0 p-3 relative z-0">
      <UButton
        type="button"
        color="neutral"
        variant="subtle"
        class="rounded-full"
        size="xl"
        icon="i-lucide-calendar"
        @click="navigate('events')"
      >
        Мероприятия
      </UButton>
      <UButton
      type="button"
        color="neutral"
        variant="subtle"
        class="rounded-full"
        size="xl"  
        icon="i-lucide-images"
        @click="navigate('gallery')"
      >
        Фотогалерея
      </UButton>
      <UButton
        type="button"
        color="neutral"
        variant="subtle"
        class="rounded-full"
        size="xl"
        icon="i-lucide-sparkles"
        @click="navigate('newcomers')"
      >
        Новичкам
      </UButton>
      <UButton
        type="button"
        color="neutral"
        variant="subtle"
        class="rounded-full"
        size="xl"
        @click="navigate('culture')"
        icon="i-lucide-users"
      >
        Корпоративная культура
      </UButton>
    </div>

    <div class="header-right hidden xl:flex p-0">
      <RouterLink
        :to="{ name: 'profile' }"
        class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-fit cursor-pointer flex items-center p-3 hover:bg-zinc-200 dark:hover:bg-zinc-700 group transition-all duration-300 ease-out"
      >
        <UAvatar src="/src/img/sticker 1.png" size="xl" />
        <div class="flex flex-col px-3 p-0">
          <div class="leading-6 text-base text-zinc-700 dark:text-zinc-50">Константин Константинович</div>
          <div class="leading-4 text-sm text-zinc-400 dark:text-zinc-500">Инженер</div>
        </div>
      </RouterLink>
    </div>
  </header>

</template>
