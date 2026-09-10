<script setup>
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useSidebarNavItems } from '../composables/usePortalNavigation';
import { attachAbsenceStorageSync } from '../stores/absenceJournal';

defineProps({
  isDark: {
    type: Boolean,
    default: false,
  },
});

defineEmits(['toggle-theme']);

const router = useRouter();
const { mainItems, footerItems } = useSidebarNavItems();
const onboardingProgress = ref(60);

onMounted(() => {
  attachAbsenceStorageSync();
});

function goNextStep() {
  router.push({ name: 'onboarding' });
}
</script>

<template>
  <UDashboardSidebar
    id="portal-sidebar"
    collapsible
    resizable
    :default-size="268"
    :min-size="220"
    :max-size="400"
    :collapsed-size="0"
    :ui="{
      root: 'border-0 border-e-0 bg-elevated/50 rounded-2xl my-4 ms-4 min-h-0 h-[calc(100dvh-2rem)]',
      header: 'h-[60px] shrink-0 px-4',
      body: 'px-4 py-2 flex flex-col gap-4',
      footer: 'px-4 pb-4 pt-2 gap-2',
    }"
  >
    <template #header="{ collapsed }">
      <RouterLink
        :to="{ name: 'home' }"
        class="flex items-center gap-1 min-w-0 flex-1"
        :class="collapsed ? 'justify-center' : ''"
      >
        <span class="relative size-7 shrink-0 text-primary">
          <svg
            class="size-7 fill-current"
            viewBox="0 0 48 48"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            aria-hidden="true"
          >
            <path
              d="M37.6895 8C38.963 8.00009 39.9969 9.0301 39.9971 10.2988V11.2412L40 24.0781C40 32.8696 32.8431 40 24.0186 40H8L35.2393 12.8623L30.3594 8H37.6895ZM24.0586 23.9971L8.00195 39.9971V34.5879H12.252V30.3545H8.00195V26.1172H12.252V21.8799H8.00195V17.6426H12.252V13.4092H8.00195V8L24.0586 23.9971ZM12.2578 26.1143V30.3506H16.5107V26.1143H12.2578ZM16.5176 21.8799V26.1172H20.7705V21.8799H16.5176ZM12.2578 17.6426V21.8799H16.5107V17.6426H12.2578Z"
            />
          </svg>
        </span>
        <span
          v-if="!collapsed"
          class="text-sm font-semibold text-highlighted truncate"
        >
          Корпоративный портал
        </span>
      </RouterLink>

      <UButton
        v-if="!collapsed"
        color="neutral"
        variant="ghost"
        size="xs"
        square
        icon="i-lucide-chevrons-up-down"
        class="shrink-0"
        aria-label="Переключить раздел"
      />
    </template>

    <template #default="{ collapsed }">
      <UNavigationMenu
        :collapsed="collapsed"
        :items="mainItems"
        orientation="vertical"
        class="w-full"
      />

      <div
        v-if="!collapsed"
        class="mt-auto flex flex-col gap-2 w-full"
      >
        <div
          class="flex flex-col gap-2 rounded-lg px-4 py-2.5 bg-primary/20"
        >
          <p class="text-sm font-medium text-highlighted">
            Прогресс изучения портала
          </p>
          <UProgress
            v-model="onboardingProgress"
            color="primary"
            size="md"
            :ui="{ base: 'bg-accented' }"
          />
          <p class="text-xs font-medium text-muted">
            Курс • {{ onboardingProgress }}% завершено
          </p>
          <UButton
            color="neutral"
            variant="solid"
            size="xs"
            block
            trailing-icon="i-lucide-move-right"
            class="bg-inverted text-inverted hover:bg-inverted/90"
            @click="goNextStep"
          >
            Следующий шаг
          </UButton>
        </div>

        <UNavigationMenu
          :items="footerItems"
          orientation="vertical"
          class="w-full"
        />
      </div>

      <UNavigationMenu
        v-else
        :collapsed="collapsed"
        :items="footerItems"
        orientation="vertical"
        class="mt-auto w-full"
      />
    </template>

    <template #resize-handle="{ onMouseDown, onTouchStart, onDoubleClick }">
      <UDashboardResizeHandle
        class="after:absolute after:inset-y-0 after:right-0 after:w-px hover:after:bg-(--ui-border-accented) after:transition-colors"
        @mousedown="onMouseDown"
        @touchstart="onTouchStart"
        @dblclick="onDoubleClick"
      />
    </template>
  </UDashboardSidebar>
</template>
