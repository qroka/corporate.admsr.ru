<script setup>
import { computed, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useHeaderUser } from '../composables/useHeaderUser';
import { usePortalBreadcrumbs } from '../composables/usePortalNavigation';
import { currentRole, setRole } from '../stores/role';
import { clearAuthStorage } from '../composables/useAuthSession';

const { headerName, avatarSrc, loading, canToggleAdminRole } = useHeaderUser();
const breadcrumbItems = usePortalBreadcrumbs();

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

const router = useRouter();
const isAdminRole = ref(currentRole.value === 'admin');

watch(isAdminRole, (v) => {
  if (canToggleAdminRole.value) setRole(v ? 'admin' : 'user');
});
watch(currentRole, (r) => {
  isAdminRole.value = r === 'admin';
});
watch(canToggleAdminRole, (can) => {
  if (!can) setRole('user');
}, { immediate: true });

async function logout() {
  try {
    const user = JSON.parse(localStorage.getItem('auth-user') || 'null');
    if (user?.id) {
      await fetch('/api/logout.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: user.id }),
      });
    }
  } finally {
    clearAuthStorage();
    router.push({ name: 'login' });
  }
}

const userMenuItems = computed(() => {
  const groups = [
    [
      {
        label: headerName.value || 'Профиль',
        avatar: {
          src: avatarSrc.value,
          loading: 'lazy',
        },
        type: 'label',
      },
    ],
    [
      {
        label: 'Профиль',
        icon: 'i-lucide-user',
        to: '/profile',
      },
    ],
  ];

  if (canToggleAdminRole.value) {
    groups.push([
      {
        label: 'Роль администратора',
        icon: 'i-lucide-shield',
        type: 'checkbox',
        checked: isAdminRole.value,
        onUpdateChecked(checked) {
          isAdminRole.value = checked;
        },
        onSelect(e) {
          e.preventDefault();
        },
      },
      {
        label: 'Дэшборд администратора',
        icon: 'i-lucide-layout-dashboard',
        to: '/admin',
      },
    ]);
  }

  groups.push([
    {
      label: 'Сменить тему',
      icon: 'i-lucide-sun-moon',
      onSelect(e) {
        e.preventDefault();
        emit('toggle-theme', e);
      },
    },
    {
      label: 'Выйти',
      icon: 'i-lucide-log-out',
      color: 'error',
      onSelect() {
        void logout();
      },
    },
  ]);

  return groups;
});
</script>

<template>
  <UDashboardNavbar
    :ui="{
      root: 'h-[60px] shrink-0 border-0 mx-4 mt-4 rounded-2xl bg-elevated/75 px-4',
      left: 'min-w-0',
      right: 'gap-4',
      title: 'min-w-0',
    }"
  >
    <template #title>
      <UBreadcrumb
        :items="breadcrumbItems"
        class="min-w-0"
        :ui="{
          root: 'min-w-0',
          list: 'min-w-0 flex-nowrap overflow-hidden',
          item: 'min-w-0',
          linkLeadingIcon: 'size-3.5 text-muted',
          linkLabel: 'text-xs font-medium truncate',
          separatorIcon: 'size-3.5 text-muted',
        }"
      />
    </template>

    <template #right>
      <div class="flex items-center gap-2">
        <UDashboardSearchButton
          label="Искать сотрудника, памятку, документ..."
          color="neutral"
          variant="outline"
          size="sm"
          class="hidden md:inline-flex w-[340px] max-w-[340px] justify-start"
          :kbds="['meta', 'K']"
        />

        <UButton
          color="neutral"
          variant="outline"
          size="sm"
          square
          icon="i-lucide-bell"
          aria-label="Уведомления"
        />
      </div>

      <USeparator orientation="vertical" class="h-6" />

      <UDropdownMenu
        :items="userMenuItems"
        :content="{ align: 'end', side: 'bottom', sideOffset: 8 }"
        :ui="{ content: 'w-64' }"
      >
        <button
          type="button"
          class="flex items-center gap-2 rounded-md px-1 py-0.5 hover:bg-elevated/50 transition-colors min-w-0"
        >
          <template v-if="loading">
            <USkeleton class="size-7 rounded-full" />
            <USkeleton class="h-3 w-28 hidden sm:block" />
          </template>
          <template v-else>
            <UAvatar
              :src="avatarSrc"
              :alt="headerName"
              size="sm"
              icon="i-lucide-user"
            />
            <span class="hidden sm:inline text-xs font-medium text-highlighted truncate max-w-[140px]">
              {{ headerName }}
            </span>
            <UIcon name="i-lucide-chevron-down" class="size-4 text-muted shrink-0" />
          </template>
        </button>
      </UDropdownMenu>
    </template>
  </UDashboardNavbar>
</template>
