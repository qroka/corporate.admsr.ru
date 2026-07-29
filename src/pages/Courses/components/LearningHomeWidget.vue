<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useCoursesStore, type EnrollmentSummary } from '../../../composables/useCoursesStore';
import CourseStatusBadge from './CourseStatusBadge.vue';

const router = useRouter();
const store = useCoursesStore();
const loading = ref(true);
const items = ref<EnrollmentSummary[]>([]);

onMounted(async () => {
  loading.value = true;
  try {
    const groups = await store.loadMyCourses() as any;
    if (groups?.overdue || groups?.active) {
      items.value = [...(groups.overdue || []), ...(groups.active || [])].slice(0, 3);
    } else {
      items.value = store.myEnrollments.value
        .filter((e) => !['completed', 'cancelled'].includes(e.status))
        .slice(0, 3);
    }
  } catch {
    items.value = [];
  } finally {
    loading.value = false;
  }
});

const hasItems = computed(() => items.value.length > 0);

function open(e: EnrollmentSummary) {
  router.push({ name: 'course-enrollment', params: { enrollmentId: String(e.id) } });
}
</script>

<template>
  <section class="flex flex-col gap-3 w-full" aria-labelledby="learning-home-title">
    <div class="flex items-center justify-between gap-2">
      <h2 id="learning-home-title" class="text-xl font-medium text-highlighted">Моё обучение</h2>
      <UButton
        color="neutral"
        variant="ghost"
        size="sm"
        trailing-icon="i-lucide-arrow-right"
        :to="{ name: 'courses' }"
      >
        Все курсы
      </UButton>
    </div>

    <div v-if="loading" class="flex flex-col gap-2">
      <USkeleton class="h-16 w-full rounded-xl" />
      <USkeleton class="h-16 w-full rounded-xl" />
    </div>

    <UEmpty
      v-else-if="!hasItems"
      icon="i-lucide-graduation-cap"
      title="Нет активных курсов"
      description="Когда вам назначат обучение, оно появится здесь."
      class="py-6"
    />

    <ul v-else class="flex flex-col gap-2 list-none p-0 m-0">
      <li
        v-for="e in items"
        :key="e.id"
        class="rounded-xl ring-1 ring-default bg-elevated/30 p-3 flex flex-col gap-2 cursor-pointer hover:bg-elevated/60 transition-colors"
        @click="open(e)"
      >
        <div class="flex items-start justify-between gap-2">
          <p class="font-medium text-highlighted line-clamp-2 text-sm">{{ e.courseTitle }}</p>
          <CourseStatusBadge :status="e.status" />
        </div>
        <UProgress
          :model-value="e.progressPercent ?? 0"
          size="sm"
          color="primary"
          :aria-label="`Завершено ${e.topicsCompleted ?? 0} из ${e.topicsTotal ?? 0} тем, ${e.progressPercent ?? 0} процентов`"
        />
        <p class="text-xs text-dimmed">
          Завершено {{ e.topicsCompleted ?? 0 }} из {{ e.topicsTotal ?? 0 }} тем · {{ e.progressPercent ?? 0 }}%
        </p>
      </li>
    </ul>
  </section>
</template>
