<script setup lang="ts">
import { computed } from 'vue';

const services = computed(() => [
  {
    label: 'Журнал отсутствия',
    description: 'Отметки отсутствия и график коллег',
    icon: 'i-lucide-calendar-off',
    to: '/absence-journal',
  },
  {
    label: 'Заявки',
    description: 'Подача и отслеживание заявок',
    icon: 'i-lucide-file-text',
    to: '/applications',
  },
  {
    label: 'Справочник',
    description: 'База знаний и памятки',
    icon: 'i-lucide-book-open',
    to: '/knowledge-base',
  },
  {
    label: 'Кадровый резерв',
    description: 'Резерв и кадровые программы',
    icon: 'i-lucide-users',
    to: '/personnel-reserve',
  },
]);
</script>

<template>
  <div class="flex flex-col gap-6 w-full max-w-5xl mx-auto">
    <UPageHeader
      title="Сервисы"
      description="Корпоративные сервисы портала"
    />

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <UPageCard
        v-for="service in services"
        :key="service.to"
        :title="service.label"
        :description="service.description"
        :icon="service.icon"
        :to="service.to"
        variant="outline"
        class="bg-elevated"
      />
    </div>
  </div>
</template>
