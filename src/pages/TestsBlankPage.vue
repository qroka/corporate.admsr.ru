<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { TabsItem } from '@nuxt/ui';
import { currentRole } from '../stores/role';
import TestBuilder from './Tests/TestBuilder.vue';

type TabValue = 'list' | 'forme' | 'builder' | 'stats';

const tab = ref<TabValue>('list');
const isAdmin = computed(() => currentRole.value === 'admin');

const tabItems = computed<TabsItem[]>(() => [
  { label: 'Список', value: 'list', icon: 'i-lucide-list' },
  { label: 'Тесты для меня', value: 'forme', icon: 'i-lucide-clipboard-check' },
  ...(isAdmin.value
    ? ([
        { label: 'Конструктор', value: 'builder', icon: 'i-lucide-square-pen' },
        { label: 'Статистика', value: 'stats', icon: 'i-lucide-bar-chart-3' },
      ] as TabsItem[])
    : []),
]);

// Если админ-режим выключили на админской вкладке — вернуть на «Список».
watch(isAdmin, (admin) => {
  if (!admin && (tab.value === 'builder' || tab.value === 'stats')) {
    tab.value = 'list';
  }
});
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4">
    <UTabs v-model="tab" :items="tabItems" size="xl" class="w-full" />

    <section class="flex-1 min-h-0 flex flex-col">
      <div v-if="tab === 'list'" class="flex-1 min-h-0 overflow-y-auto">
        <!-- Список тестов -->
      </div>
      <div v-else-if="tab === 'forme'" class="flex-1 min-h-0 overflow-y-auto">
        <!-- Тесты для меня -->
      </div>
      <div v-else-if="tab === 'builder' && isAdmin" class="flex-1 min-h-0 flex">
        <TestBuilder />
      </div>
      <div v-else-if="tab === 'stats' && isAdmin" class="flex-1 min-h-0 overflow-y-auto">
        <!-- Статистика -->
      </div>
    </section>
  </UMain>
</template>
