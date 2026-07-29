<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const route = useRoute();
const store = useCoursesStore();
const { toast } = useAppToast();
const enrollmentId = computed(() => Number(route.params.enrollmentId));
const loading = ref(true);
const result = ref<any>(null);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Мои курсы', to: { name: 'courses' } },
  { label: 'Результат' },
]);

const title = computed(
  () => result.value?.course?.title || result.value?.enrollment?.course?.title || 'Результат курса',
);

onMounted(async () => {
  try {
    result.value = await store.loadResult(enrollmentId.value);
  } catch (e: any) {
    toast.add({ title: 'Результат недоступен', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-2xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">{{ title }}</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 3" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <template v-else-if="result">
      <CourseStatusBadge :status="result.enrollment?.status || result.status || 'completed'" />

      <div class="rounded-xl ring-1 ring-default p-5 flex flex-col gap-3">
        <p v-if="result.finalScore != null || result.enrollment?.finalScore != null" class="text-3xl font-medium text-highlighted">
          {{ result.finalScore ?? result.enrollment?.finalScore }}%
        </p>
        <p class="text-sm text-muted">
          <template v-if="result.passed === true || result.enrollment?.status === 'completed'">
            Курс успешно завершён.
          </template>
          <template v-else-if="result.passed === false || result.enrollment?.status === 'failed'">
            Курс не сдан. Обратитесь к администратору при необходимости переназначения.
          </template>
          <template v-else>
            Итоги по вашему прохождению.
          </template>
        </p>
        <p v-if="result.completedAt || result.enrollment?.completedAt" class="text-xs text-dimmed">
          Дата:
          {{ new Date(result.completedAt || result.enrollment.completedAt).toLocaleString('ru-RU') }}
        </p>
      </div>

      <div class="flex gap-2">
        <UButton color="primary" :to="{ name: 'courses' }">К моим курсам</UButton>
        <UButton color="neutral" variant="ghost" :to="{ name: 'course-history' }">История</UButton>
      </div>
    </template>
  </UMain>
</template>
