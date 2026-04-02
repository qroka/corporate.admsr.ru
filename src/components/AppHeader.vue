<script setup>
import { ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useHeaderUser } from '../composables/useHeaderUser';
import { currentRole, setRole } from '../stores/role';

const { headerName, subtitle, avatarSrc } = useHeaderUser();

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

const isAdminRole = ref(currentRole.value === 'admin');

watch(isAdminRole, (v) => setRole(v ? 'admin' : 'user'));
watch(currentRole, (r) => {
  isAdminRole.value = r === 'admin';
});

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
    <UTooltip arrow :content="{ side: 'bottom' }" text="На главную">
      <RouterLink :to="{ name: 'home' }"
        class="group bg-elevated flex p-3 pr-5 rounded-full">
      <UContainer class="header-logo flex items-center gap-1.5">
        <UContainer class="relative w-10 h-10">
          <svg :class="[
            'absolute w-10 h-10 transition-all duration-300 ease-out fill-current text-dimmed',
            'opacity-100 group-hover:opacity-0'
          ]" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M37.6895 8C38.963 8.00009 39.9969 9.0301 39.9971 10.2988V11.2412L40 24.0781C40 32.8696 32.8431 40 24.0186 40H8L35.2393 12.8623L30.3594 8H37.6895ZM24.0586 23.9971L8.00195 39.9971V34.5879H12.252V30.3545H8.00195V26.1172H12.252V21.8799H8.00195V17.6426H12.252V13.4092H8.00195V8L24.0586 23.9971ZM12.2578 26.1143V30.3506H16.5107V26.1143H12.2578ZM16.5176 21.8799V26.1172H20.7705V21.8799H16.5176ZM12.2578 17.6426V21.8799H16.5107V17.6426H12.2578Z" />
          </svg>
          <svg :class="[
            'overflow-visible absolute w-10 h-10 transition-all duration-300 ease-out',
            isDark
              ? 'group-hover:filter-[drop-shadow(0_0_10px_currentColor)_drop-shadow(0_0_5px_currentColor)_drop-shadow(0_0_1px_currentColor)]'
              : '',
            'opacity-0 group-hover:opacity-100 fill-primary text-primary'
          ]" viewBox="16 16 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M53.8815 24.192C55.155 24.1921 56.1889 25.2221 56.1891 26.4908V27.4332L56.192 40.2701C56.192 49.0616 49.0351 56.192 40.2106 56.192H24.192L51.4313 29.0543L46.5514 24.192H53.8815ZM40.2506 40.1891L24.194 56.1891V50.7799H28.444V46.5465H24.194V42.3092H28.444V38.0719H24.194V33.8346H28.444V29.6012H24.194V24.192L40.2506 40.1891ZM28.4498 42.3062V46.5426H32.7028V42.3062H28.4498ZM32.7096 38.0719V42.3092H36.9625V38.0719H32.7096ZM28.4498 33.8346V38.0719H32.7028V33.8346H28.4498Z"
              fill="var(--ui-primary)" />
          </svg>
        </UContainer>
        <span class="text-xl font-unbounded font-normal leading-10 text-highlighted">
          Корпоративный портал
        </span>
      </UContainer>
    </RouterLink>
    </UTooltip>

    <UContainer class="header-nav bg-elevated relative rounded-full w-fit gap-0 sm:p-3 md:p-3 lg:p-3 xl:p-3 z-0 mx-0">
      <UTooltip arrow :content="{ side: 'bottom' }" text="Мероприятия">
        <UButton type="button" color="neutral" square  class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'events' ? 'z-10 bg-primary text-zinc-50 shadow-none dark:shadow-brand [&_svg]:text-zinc-50 hover:bg-primary hover:text-zinc-50 active:bg-primary active:text-zinc-50 active:[&_svg]:text-zinc-50' : ''" size="xl" icon="i-lucide-calendar"
          @click="navigate('events')">
          <span class="hidden min-[1600px]:inline">Мероприятия</span>
        </UButton>
      </UTooltip>
      <UTooltip arrow :content="{ side: 'bottom' }" text="Фотогалерея">
        <UButton type="button" color="neutral" square class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'gallery' ? 'z-10 bg-primary text-zinc-50 shadow-none dark:shadow-brand [&_svg]:text-zinc-50 hover:bg-primary hover:text-zinc-50 active:bg-primary active:text-zinc-50 active:[&_svg]:text-zinc-50' : ''" size="xl" icon="i-lucide-images"
          @click="navigate('gallery')">
          <span class="hidden min-[1600px]:inline">Фотогалерея</span>
        </UButton>
      </UTooltip>
      <UTooltip arrow :content="{ side: 'bottom' }" text="Новичкам">
        <UButton type="button" color="neutral" square  class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'newcomers' ? 'z-10 bg-primary text-zinc-50 shadow-none dark:shadow-brand [&_svg]:text-zinc-50 hover:bg-primary hover:text-zinc-50 active:bg-primary active:text-zinc-50 active:[&_svg]:text-zinc-50' : ''" size="xl" icon="i-lucide-sparkles"
          @click="navigate('newcomers')">
          <span class="hidden min-[1600px]:inline">Новичкам</span>
        </UButton>
      </UTooltip>
      <UTooltip arrow :content="{ side: 'bottom' }" text="Корпоративная культура">
        <UButton type="button" color="neutral" square  class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted" :class="activeNav === 'culture' ? 'z-10 bg-primary text-zinc-50 shadow-none dark:shadow-brand [&_svg]:text-zinc-50 hover:bg-primary hover:text-zinc-50 active:bg-primary active:text-zinc-50 active:[&_svg]:text-zinc-50' : ''" size="xl"
          @click="navigate('culture')" icon="i-lucide-users">
          <span class="hidden min-[1600px]:inline">Корпоративная культура</span>
        </UButton>
      </UTooltip>
    </UContainer>

    <div class="flex items-center gap-2 min-w-0">
      <UTooltip arrow :content="{ side: 'bottom' }" text="Роль: администратор">
        <div class="bg-elevated rounded-full p-3 flex items-center gap-2">
          <USwitch v-model="isAdminRole" size="xl" aria-label="Переключить роль администратора" />
        </div>
      </UTooltip>

      <UTooltip arrow :content="{ side: 'bottom' }" text="Профиль">
        <RouterLink :to="{ name: 'profile' }" class="bg-elevated rounded-full py-2.5 pl-3 pr-5">
          <UUser
            :name="headerName"
            size="xl"
            class="bg-elevated rounded-full"
            :description="subtitle"
            :avatar="{
              src: avatarSrc,
              loading: 'lazy',
              icon: 'i-lucide-image',
              ui: { root: '!bg-accented' },
            }"
          />
        </RouterLink>
      </UTooltip>
    </div>
  </header>

</template>
