<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { TabsItem } from '@nuxt/ui';
import { currentRole } from '../stores/role';
import { useTestsStore } from '../composables/useTestsStore';
import { cloneForm, type TestForm } from './Tests/testForm';
import TestBuilder from './Tests/TestBuilder.vue';
import TestRunner from './Tests/TestRunner.vue';

type TabValue = 'list' | 'forme' | 'builder' | 'stats';

const tab = ref<TabValue>('list');
const isAdmin = computed(() => currentRole.value === 'admin');
const store = useTestsStore();

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

const kindLabel = (k: string) =>
  ({ test: 'Тест', survey: 'Опрос', poll: 'Голосование' } as Record<string, string>)[k] ?? k;

// Прохождение опубликованной формы
const runOpen = ref(false);
const runForm = ref<TestForm | null>(null);
function openRun(f: TestForm) {
  runForm.value = cloneForm(f);
  runOpen.value = true;
}

// Благодарность показывается уже на странице (после закрытия окна прохождения)
const completionOpen = ref(false);
const completionForm = ref<TestForm | null>(null);
const completionText = computed(() => {
  const f = completionForm.value;
  if (!f) return '';
  return f.completionMessage.trim() || (f.kind === 'poll' ? 'Спасибо! Ваш голос учтён.' : 'Спасибо за прохождение!');
});
function onRunFinish() {
  completionForm.value = runForm.value;
  runOpen.value = false;
  completionOpen.value = true;
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4">
    <UTabs v-model="tab" :items="tabItems" size="xl" class="w-full" />

    <section class="flex-1 min-h-0 flex flex-col">
      <!-- Список опубликованных -->
      <div v-if="tab === 'list'" class="flex-1 min-h-0 overflow-y-auto">
        <UEmpty
          v-if="!store.published.value.length"
          icon="i-lucide-list"
          title="Здесь пока пусто"
          description="Опубликованные формы появятся здесь и станут доступны для прохождения."
          class="py-12"
        />
        <div v-else class="flex flex-col gap-3">
          <div
            v-for="f in store.published.value"
            :key="f.id ?? 0"
            class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 bg-elevated/30"
          >
            <div class="flex-1 min-w-0 flex flex-col gap-1">
              <div class="flex items-center gap-2 flex-wrap">
                <UBadge color="neutral" variant="subtle" class="tabular-nums">#{{ f.id }}</UBadge>
                <UBadge color="primary" variant="subtle">{{ kindLabel(f.kind) }}</UBadge>
                <span class="font-medium text-highlighted truncate">{{ f.title || 'Без названия' }}</span>
              </div>
              <p v-if="f.description" class="text-sm text-muted line-clamp-1">{{ f.description }}</p>
              <p class="text-xs text-dimmed">Вопросов: {{ f.questions.length }}</p>
            </div>
            <div class="shrink-0">
              <UButton
                color="primary"
                :icon="store.hasActiveSession(f) ? 'i-lucide-rotate-ccw' : 'i-lucide-play'"
                @click="openRun(f)"
              >
                {{ store.hasActiveSession(f) ? 'Продолжить' : 'Пройти' }}
              </UButton>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="tab === 'forme'" class="flex-1 min-h-0 overflow-y-auto">
        <!-- Тесты для меня -->
      </div>

      <div v-else-if="tab === 'builder' && isAdmin" class="flex-1 min-h-0 flex">
        <TestBuilder @published="tab = 'list'" />
      </div>

      <div v-else-if="tab === 'stats' && isAdmin" class="flex-1 min-h-0 overflow-y-auto">
        <!-- Статистика -->
      </div>
    </section>

    <!-- Окно прохождения -->
    <UModal
      v-model:open="runOpen"
      fullscreen
      :title="runForm?.title || 'Прохождение'"
      :ui="{ content: 'flex flex-col', body: 'flex-1 min-h-0 p-4 sm:p-6' }"
    >
      <template #body>
        <TestRunner v-if="runOpen && runForm" :form="runForm" persist-session class="h-full" @finish="onRunFinish" />
      </template>
    </UModal>

    <!-- Сообщение после завершения (на странице до прохождения) -->
    <UModal v-model:open="completionOpen" :title="completionForm?.kind === 'poll' ? 'Голос учтён' : 'Готово'" description="" :dismissible="false">
      <template #body>
        <div class="flex flex-col items-center text-center gap-3 py-2">
          <div class="size-12 rounded-full bg-success/10 flex items-center justify-center">
            <UIcon name="i-lucide-check" class="size-7 text-success" />
          </div>
          <p class="text-default whitespace-pre-line">{{ completionText }}</p>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end w-full">
          <UButton color="primary" @click="completionOpen = false">Закрыть</UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
