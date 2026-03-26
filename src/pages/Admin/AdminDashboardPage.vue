<script setup lang="ts">
import { computed, ref } from 'vue';
import { currentRole } from '../../stores/role';

const isAdmin = computed(() => currentRole.value === 'admin');

const tabItems = [
  { label: 'Пользователи', value: 'users' },
  { label: 'Группы пользователей', value: 'groups' },
  { label: 'ОФО', value: 'ofo' },
];
const tab = ref<(typeof tabItems)[number]['value']>('users');

const users = [
  { name: 'Иван Петров', email: 'ivan.petrov@company.ru', role: 'user' },
  { name: 'Анна Смирнова', email: 'anna.smirnova@company.ru', role: 'admin' },
  { name: 'Дмитрий Кузнецов', email: 'd.kuznetsov@company.ru', role: 'user' },
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

        <UCard v-if="tab === 'users'" class="w-full">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-lg font-semibold text-highlighted truncate">Пользователи</h2>
                <p class="text-sm text-muted">Список пользователей (пока мок‑данные).</p>
              </div>
              <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-user-plus">
                Добавить
              </UButton>
            </div>
          </template>

          <div class="grid grid-cols-1 gap-3">
            <UCard v-for="u in users" :key="u.email" class="w-full">
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="font-medium text-highlighted truncate">{{ u.name }}</div>
                  <div class="text-sm text-muted truncate">{{ u.email }}</div>
                </div>
                <UBadge color="neutral" variant="subtle">{{ u.role }}</UBadge>
              </div>
            </UCard>
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
  </UMain>
</template>
