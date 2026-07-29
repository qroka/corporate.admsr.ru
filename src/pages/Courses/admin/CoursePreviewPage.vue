<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';

const route = useRoute();
const store = useCoursesStore();
const { toast } = useAppToast();
const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'admin-courses' } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Превью' },
]);

onMounted(async () => {
  try {
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

function matCount(t: any) {
  return t.materialsCount ?? t.materials?.length ?? 0;
}

function hasTopicTest(t: any) {
  return !!(t.testLink || t.topicTest);
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Превью для сотрудника</h1>
    <p class="text-sm text-muted">Только просмотр структуры — так курс увидит участник.</p>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <template v-else-if="store.current.value">
      <div class="rounded-xl ring-1 ring-default bg-elevated/30 p-5 flex flex-col gap-2">
        <h2 class="text-xl font-medium">{{ store.current.value.title }}</h2>
        <p v-if="store.version.value?.shortDescription" class="text-sm text-muted">
          {{ store.version.value.shortDescription }}
        </p>
        <div v-if="store.version.value?.fullDescription" class="prose prose-sm dark:prose-invert max-w-none text-sm" v-html="store.version.value.fullDescription" />
      </div>

      <ol class="flex flex-col gap-3 list-decimal pl-5">
        <li v-for="t in store.topics.value" :key="t.id" class="pl-1">
          <p class="font-medium text-highlighted">{{ t.title }}</p>
          <p class="text-xs text-dimmed">Материалов: {{ matCount(t) }}</p>
          <ul v-if="t.materials?.length" class="mt-1 text-sm text-muted list-disc pl-5">
            <li v-for="m in t.materials" :key="m.id">{{ m.title }}</li>
          </ul>
          <p v-if="hasTopicTest(t)" class="text-xs text-primary mt-1">+ промежуточный тест</p>
        </li>
      </ol>

      <div v-if="store.version.value?.finalTest || store.version.value?.requireFinalTest" class="rounded-xl ring-1 ring-default p-4">
        <p class="font-medium">Итоговый тест</p>
        <p class="text-sm text-muted">Завершает курс после всех тем</p>
      </div>
    </template>
  </UMain>
</template>
