<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { currentRole } from '../../../stores/role';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const loading = ref(true);
const items = ref<any[]>([]);

const crumbs: BreadcrumbItem[] = [
  { label: 'Мои курсы', to: { name: 'courses' } },
  { label: 'История' },
];

watch(
  currentRole,
  (role) => {
    if (role !== 'admin') router.replace({ name: 'courses' });
  },
  { immediate: true },
);

onMounted(async () => {
  if (currentRole.value !== 'admin') return;
  try {
    const data = (await store.loadHistory()) as any;
    items.value = Array.isArray(data) ? data : data?.items || data?.history || data?.completed || [];
  } catch (e: any) {
    toast.add({ title: 'Не удалось загрузить историю', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

function titleOf(row: any) {
  return row.courseTitle || row.course?.title || row.title || 'Курс';
}

function enrollmentId(row: any) {
  return row.enrollmentId || row.enrollment?.id || row.id;
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">История обучения</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <UEmpty
      v-else-if="!items.length"
      icon="i-lucide-history"
      title="История пуста"
      description="Завершённые курсы появятся здесь."
      class="py-12"
    />

    <ul v-else class="flex flex-col gap-2 list-none p-0 m-0">
      <li
        v-for="(row, i) in items"
        :key="enrollmentId(row) ?? i"
        class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3"
      >
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <CourseStatusBadge :status="row.status || row.enrollment?.status || 'completed'" />
            <span class="font-medium truncate">{{ titleOf(row) }}</span>
          </div>
          <p class="text-xs text-dimmed mt-1">
            <template v-if="row.completedAt || row.enrollment?.completedAt">
              {{ new Date(row.completedAt || row.enrollment.completedAt).toLocaleDateString('ru-RU') }}
            </template>
            <template v-if="row.finalScore != null || row.enrollment?.finalScore != null">
              · балл {{ row.finalScore ?? row.enrollment.finalScore }}
            </template>
          </p>
        </div>
        <UButton
          v-if="enrollmentId(row)"
          color="neutral"
          variant="soft"
          :to="{ name: 'course-result', params: { enrollmentId: enrollmentId(row) } }"
        >
          Открыть
        </UButton>
      </li>
    </ul>
  </UMain>
</template>
