<script setup lang="ts">
import { computed, h, reactive, ref, resolveComponent } from 'vue';
import type { TableColumn } from '@nuxt/ui';
import { useAppToast } from '../../composables/useAppToast';
import { currentRole } from '../../stores/role';

const { adminUserSaved } = useAppToast();

const UBadge = resolveComponent('UBadge');
const UButton = resolveComponent('UButton');
const UDropdownMenu = resolveComponent('UDropdownMenu');

const isAdmin = computed(() => currentRole.value === 'admin');

const tabItems = [
  { label: 'Пользователи', value: 'users' },
  { label: 'Группы пользователей', value: 'groups' },
  { label: 'ОФО', value: 'ofo' },
];
const tab = ref<(typeof tabItems)[number]['value']>('users');

type AdminUserRow = {
  id: number;
  status: 'Активен' | 'Заблокирован';
  fullName: string;
  login: string;
  password: string;
  ofo: string;
  group: string;
  lastAuthAt: string;
};

const users = ref<AdminUserRow[]>([
  {
    id: 1001,
    status: 'Активен',
    fullName: 'Петров Иван Сергеевич',
    login: 'i.petrov',
    password: '••••••••',
    ofo: 'ОФО №1',
    group: 'HR',
    lastAuthAt: '2026-03-25 09:14',
  },
  {
    id: 1002,
    status: 'Активен',
    fullName: 'Смирнова Анна Викторовна',
    login: 'a.smirnova',
    password: '••••••••',
    ofo: 'ОФО №2',
    group: 'Администраторы',
    lastAuthAt: '2026-03-26 11:02',
  },
  {
    id: 1003,
    status: 'Заблокирован',
    fullName: 'Кузнецов Дмитрий Олегович',
    login: 'd.kuznetsov',
    password: '••••••••',
    ofo: 'ОФО №3',
    group: 'Муниципальная служба',
    lastAuthAt: '2026-02-10 16:45',
  },
  {
    id: 1004,
    status: 'Активен',
    fullName: 'Волкова Елена Павловна',
    login: 'e.volkova',
    password: '••••••••',
    ofo: 'ОФО №1',
    group: 'Развитие и мотивация',
    lastAuthAt: '2026-03-24 08:30',
  },
]);

const userSearchQuery = ref('');
const filterStatus = ref('');
const filterOfo = ref('');
const filterGroup = ref('');

const statusFilterItems = [
  { value: '', label: 'Все статусы' },
  { value: 'Активен', label: 'Активен' },
  { value: 'Заблокирован', label: 'Заблокирован' },
];

const ofoFilterItems = computed(() => {
  const uniq = [...new Set(users.value.map((u) => u.ofo))].sort((a, b) => a.localeCompare(b, 'ru'));
  return [{ value: '', label: 'Все ОФО' }, ...uniq.map((o) => ({ value: o, label: o }))];
});

const groupFilterItems = computed(() => {
  const uniq = [...new Set(users.value.map((u) => u.group))].sort((a, b) => a.localeCompare(b, 'ru'));
  return [{ value: '', label: 'Все группы' }, ...uniq.map((g) => ({ value: g, label: g }))];
});

const filteredUsers = computed(() => {
  let list = users.value;

  if (filterStatus.value) {
    list = list.filter((u) => u.status === filterStatus.value);
  }
  if (filterOfo.value) {
    list = list.filter((u) => u.ofo === filterOfo.value);
  }
  if (filterGroup.value) {
    list = list.filter((u) => u.group === filterGroup.value);
  }

  const q = userSearchQuery.value.trim().toLowerCase();
  if (q) {
    list = list.filter((u) => {
      const hay = [
        String(u.id),
        u.status,
        u.fullName,
        u.login,
        u.ofo,
        u.group,
        u.lastAuthAt,
      ]
        .join(' ')
        .toLowerCase();
      return hay.includes(q);
    });
  }

  return list;
});

function resetUserFilters() {
  userSearchQuery.value = '';
  filterStatus.value = '';
  filterOfo.value = '';
  filterGroup.value = '';
}

function toggleUserStatus(user: AdminUserRow) {
  const next: AdminUserRow['status'] = user.status === 'Активен' ? 'Заблокирован' : 'Активен';
  users.value = users.value.map((u) => (u.id === user.id ? { ...u, status: next } : u));
}

const editOpen = ref(false);
const editForm = reactive<AdminUserRow>({
  id: 0,
  status: 'Активен',
  fullName: '',
  login: '',
  password: '••••••••',
  ofo: '',
  group: '',
  lastAuthAt: '',
});

function openEdit(user: AdminUserRow) {
  Object.assign(editForm, { ...user });
  editOpen.value = true;
}

function saveEdit() {
  users.value = users.value.map((u) =>
    u.id === editForm.id ? { ...editForm } : u,
  );
  adminUserSaved(editForm.fullName);
  editOpen.value = false;
}

function impersonate(user: AdminUserRow) {
  window.alert(
    `Имитация входа под пользователем «${user.fullName}» (${user.login}). В продакшене здесь будет серверный токен/сессия.`,
  );
}

/** Цвет бейджа ОФО по номеру в строке (демо). */
function ofoBadgeColor(ofo: string): 'primary' | 'success' | 'warning' | 'info' | 'neutral' {
  if (ofo.includes('№1')) return 'primary';
  if (ofo.includes('№2')) return 'success';
  if (ofo.includes('№3')) return 'warning';
  return 'neutral';
}

/** Цвет бейджа группы по справочнику (демо). */
function groupBadgeColor(group: string): 'primary' | 'success' | 'warning' | 'error' | 'info' | 'neutral' {
  const map: Record<string, 'primary' | 'success' | 'warning' | 'error' | 'info' | 'neutral'> = {
    HR: 'primary',
    Администраторы: 'error',
    'Муниципальная служба': 'info',
    'Развитие и мотивация': 'success',
  };
  return map[group] ?? 'neutral';
}

function rowActionItems(user: AdminUserRow) {
  return [
    {
      label: user.status === 'Активен' ? 'Отключить пользователя' : 'Включить пользователя',
      icon: user.status === 'Активен' ? 'i-lucide-user-x' : 'i-lucide-user-check',
      onSelect() {
        toggleUserStatus(user);
      },
    },
    {
      label: 'Редактировать информацию',
      icon: 'i-lucide-pencil',
      onSelect() {
        openEdit(user);
      },
    },
    {
      label: 'Авторизоваться под ним',
      icon: 'i-lucide-log-in',
      onSelect() {
        impersonate(user);
      },
    },
  ];
}

const userColumns: TableColumn<AdminUserRow>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'status',
    header: 'Статус',
    cell: ({ row }) => {
      const s = row.getValue('status') as AdminUserRow['status'];
      const color = s === 'Активен' ? 'success' : 'error';
      return h(UBadge, { variant: 'subtle', color }, () => s);
    },
  },
  { accessorKey: 'fullName', header: 'ФИО' },
  { accessorKey: 'login', header: 'Логин' },
  {
    accessorKey: 'password',
    header: 'Пароль',
    cell: ({ row }) =>
      h('span', { class: 'font-mono text-muted tracking-widest select-none', title: 'Мок: в интерфейсе пароль не хранится открытым текстом' }, row.getValue('password') as string),
  },
  {
    accessorKey: 'ofo',
    header: 'ОФО',
    cell: ({ row }) => {
      const o = row.getValue('ofo') as string;
      return h(UBadge, {
        variant: 'subtle',
        color: ofoBadgeColor(o),
        leading: true,
        leadingIcon: 'i-lucide-building-2',
      }, () => o);
    },
  },
  {
    accessorKey: 'group',
    header: 'Группа',
    cell: ({ row }) => {
      const g = row.getValue('group') as string;
      return h(UBadge, {
        variant: 'subtle',
        color: groupBadgeColor(g),
        leading: true,
        leadingIcon: 'i-lucide-users-round',
      }, () => g);
    },
  },
  { accessorKey: 'lastAuthAt', header: 'Последняя авторизация' },
  {
    id: 'actions',
    header: () => h('span', { class: 'sr-only' }, 'Действия'),
    enableHiding: false,
    meta: {
      class: {
        th: 'w-14 text-right',
        td: 'text-right',
      },
    },
    cell: ({ row }) => {
      const user = row.original as AdminUserRow;
      return h(
        UDropdownMenu,
        {
          content: { align: 'end' },
          items: rowActionItems(user),
          'aria-label': 'Действия с пользователем',
        },
        () =>
          h(UButton, {
            icon: 'i-lucide-ellipsis-vertical',
            color: 'neutral',
            variant: 'ghost',
            square: true,
            size: 'sm',
            'aria-label': 'Действия',
          }),
      );
    },
  },
];

const groups = [
  { name: 'HR', members: 12 },
  { name: 'Муниципальная служба', members: 8 },
  { name: 'Развитие и мотивация', members: 6 },
];

const ofo = [
  { name: 'ОФО №1', manager: 'А. Смирнова' },
  { name: 'ОФО №2', manager: 'И. Петров' },
  { name: 'ОФО №3', manager: 'Д. Кузнецов' },
];
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">
    <UContainer class="flex flex-col max-w-full w-full gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
      <UPageHeader
        title=""
        class="border-none p-0 w-full"
        :description="isAdmin ? 'Админ‑панель управления пользователями и структурами.' : 'Доступ только для администратора.'"
      >
        <template #title>
          <h1 class="text-4xl font-normal font-unbounded">Дэшборд администратора</h1>
        </template>
      </UPageHeader>
    </UContainer>

    <UContainer class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UAlert
        v-if="!isAdmin"
        color="red"
        variant="subtle"
        icon="i-lucide-shield-alert"
        title="Недостаточно прав"
        description="Эта страница доступна только администраторам."
      />

      <div v-else class="flex flex-col gap-4">
        <UTabs v-model="tab" :items="tabItems" size="xl" />

        <UCard v-if="tab === 'users'" class="w-full overflow-visible">
          <template #header>
            <div class="min-w-0">
              <h2 class="text-lg font-semibold text-highlighted truncate">Пользователи</h2>
              <p class="text-sm text-muted">Список пользователей (пока мок‑данные).</p>
            </div>
          </template>

          <div class="flex flex-col gap-4 p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 overflow-visible">
            <UContainer class="flex flex-col w-full gap-3 sm:flex-row sm:flex-wrap sm:items-end p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 max-w-full mx-0 overflow-visible">
              <UInput
                v-model="userSearchQuery"
                icon="i-lucide-search"
                size="xl"
                color="neutral"
                variant="outline"
                placeholder="Поиск по ID, ФИО, логину, ОФО, группе, дате…"
                class="w-full sm:flex-1 sm:min-w-[240px]"
              />
              <USelectMenu
                v-model="filterStatus"
                :items="statusFilterItems"
                size="xl"
                color="neutral"
                placeholder="Статус"
                class="w-full sm:w-52"
                :content="{ align: 'start', sideOffset: 8 }"
              />
              <USelectMenu
                v-model="filterOfo"
                :items="ofoFilterItems"
                size="xl"
                color="neutral"
                placeholder="ОФО"
                class="w-full sm:w-52"
                :content="{ align: 'start', sideOffset: 8 }"
              />
              <USelectMenu
                v-model="filterGroup"
                :items="groupFilterItems"
                size="xl"
                color="neutral"
                placeholder="Группа"
                class="w-full sm:w-56"
                :content="{ align: 'start', sideOffset: 8 }"
              />
              <UButton
                color="neutral"
                variant="outline"
                size="xl"
                icon="i-lucide-rotate-ccw"
                class="w-full sm:w-auto justify-center"
                @click="resetUserFilters"
              >
                Сбросить
              </UButton>
            </UContainer>

            <p
              v-if="filteredUsers.length !== users.length || userSearchQuery.trim() || filterStatus || filterOfo || filterGroup"
              class="text-sm text-muted -mt-2"
            >
              Найдено: {{ filteredUsers.length }} из {{ users.length }}
            </p>

            <div class="overflow-x-auto -mx-1 px-1">
              <UTable
                :columns="userColumns"
                :data="filteredUsers"
                sticky
                class="min-w-[960px]"
              />
            </div>
          </div>
        </UCard>

        <UCard v-else-if="tab === 'groups'" class="w-full">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-lg font-semibold text-highlighted truncate">Группы пользователей</h2>
                <p class="text-sm text-muted">Справочники групп (пока мок‑данные).</p>
              </div>
              <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-users">
                Создать группу
              </UButton>
            </div>
          </template>

          <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            <UCard v-for="g in groups" :key="g.name" class="w-full">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="font-medium text-highlighted truncate">{{ g.name }}</div>
                  <div class="text-sm text-muted">{{ g.members }} участников</div>
                </div>
                <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-chevron-right" square />
              </div>
            </UCard>
          </div>
        </UCard>

        <UCard v-else class="w-full">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-lg font-semibold text-highlighted truncate">ОФО</h2>
                <p class="text-sm text-muted">Структура ОФО (пока мок‑данные).</p>
              </div>
              <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-plus">
                Добавить ОФО
              </UButton>
            </div>
          </template>

          <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            <UCard v-for="o in ofo" :key="o.name" class="w-full">
              <div class="min-w-0">
                <div class="font-medium text-highlighted truncate">{{ o.name }}</div>
                <div class="text-sm text-muted truncate">Руководитель: {{ o.manager }}</div>
              </div>
            </UCard>
          </div>
        </UCard>
      </div>
    </UContainer>

    <USlideover v-model:open="editOpen" side="right" title="Редактирование пользователя" description="">
      <template #body>
        <UForm :state="editForm" class="space-y-4" @submit.prevent="saveEdit">
          <UFormField label="ФИО" name="fullName">
            <UInput v-model="editForm.fullName" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Логин" name="login">
            <UInput v-model="editForm.login" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="ОФО" name="ofo">
            <UInput v-model="editForm.ofo" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Группа" name="group">
            <UInput v-model="editForm.group" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Статус" name="status">
            <USelect
              v-model="editForm.status"
              :items="[
                { value: 'Активен', label: 'Активен' },
                { value: 'Заблокирован', label: 'Заблокирован' },
              ]"
              size="xl"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Последняя авторизация" name="lastAuthAt">
            <UInput v-model="editForm.lastAuthAt" size="xl" class="w-full" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" size="xl" @click="editOpen = false">
            Отмена
          </UButton>
          <UButton size="xl" @click="saveEdit">
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>
  </UMain>
</template>
