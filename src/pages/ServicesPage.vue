<script setup lang="ts">
import { computed } from 'vue';

const services = computed(() => [
  {
    label: 'Журнал отсутствия',
    description: 'Отметки об отсутствии сотрудника',
    icon: 'i-lucide-calendar-off',
    to: '/absence-journal',
  },
  {
    label: 'Формы',
    description: 'Опросы, анкеты и тесты',
    icon: 'i-lucide-clipboard-list',
    to: '/tests',
  },
  {
    label: 'Заявки',
    description: 'Подача и отслеживание сервисных заявок',
    icon: 'i-lucide-file-text',
    to: '/applications',
  },
]);
</script>

<template>
  <div class="flex flex-col gap-6 w-full max-w-[1600px] mx-auto">
    <UPageHeader
      headline="Сервисы"
      title="Все сервисы"
      description="Рабочие системы и информационные системы в одном месте"
    />

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <UPageCard
        v-for="service in services"
        :key="service.to"
        :title="service.label"
        :description="service.description"
        :icon="service.icon"
        :to="service.to"
        variant="soft"
        class="bg-elevated"
      />
    </div>
  </div>
</template>
