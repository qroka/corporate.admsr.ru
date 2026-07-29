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
const report = ref<{ ready: boolean; errors: string[]; warnings: string[] } | null>(null);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'admin-courses' } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Проверка' },
]);

onMounted(async () => {
  try {
    await store.loadCourse(courseId.value);
    report.value = await store.readiness(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Ошибка проверки', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Проверка готовности</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 3" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <template v-else-if="report">
      <UAlert
        :color="report.ready ? 'success' : 'error'"
        variant="subtle"
        :icon="report.ready ? 'i-lucide-check-circle' : 'i-lucide-x-circle'"
        :title="report.ready ? 'Курс готов к публикации' : 'Исправьте ошибки перед публикацией'"
      />

      <section v-if="report.errors.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium text-error">Ошибки</h2>
        <ul class="flex flex-col gap-2 list-none p-0 m-0">
          <li
            v-for="(err, i) in report.errors"
            :key="i"
            class="rounded-lg ring-1 ring-error/30 bg-error/5 px-3 py-2 text-sm"
          >
            {{ err }}
          </li>
        </ul>
      </section>

      <section v-if="report.warnings.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium text-warning">Предупреждения</h2>
        <ul class="flex flex-col gap-2 list-none p-0 m-0">
          <li
            v-for="(w, i) in report.warnings"
            :key="i"
            class="rounded-lg ring-1 ring-warning/30 bg-warning/5 px-3 py-2 text-sm"
          >
            {{ w }}
          </li>
        </ul>
      </section>

      <UEmpty
        v-if="!report.errors.length && !report.warnings.length"
        icon="i-lucide-check"
        title="Замечаний нет"
        description="Можно переходить к публикации."
        class="py-8"
      />

      <div class="flex gap-2">
        <UButton
          color="primary"
          size="lg"
          icon="i-lucide-send"
          :disabled="!report.ready"
          :to="{ name: 'admin-course-publish', params: { courseId } }"
        >
          К публикации
        </UButton>
        <UButton color="neutral" variant="ghost" size="lg" :to="{ name: 'admin-course-workspace', params: { courseId } }">
          Назад
        </UButton>
      </div>
    </template>
  </UMain>
</template>
