<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import type { ButtonProps } from '@nuxt/ui';
import { useCoursesStore, type EnrollmentSummary } from '../../../composables/useCoursesStore';
import CourseStatusBadge from './CourseStatusBadge.vue';

const router = useRouter();
const store = useCoursesStore();
const loading = ref(true);
const items = ref<EnrollmentSummary[]>([]);

const learningLinks = <ButtonProps[]>[
  {
    icon: 'i-lucide-arrow-up-right',
    to: '/courses',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
];

onMounted(async () => {
  loading.value = true;
  try {
    const groups = await store.loadMyCourses() as any;
    const activeStatuses = new Set(['not_started', 'in_progress', 'overdue']);
    if (groups?.overdue || groups?.active) {
      items.value = [...(groups.overdue || []), ...(groups.active || [])]
        .filter((e: EnrollmentSummary) => activeStatuses.has(e.status))
        .slice(0, 3);
    } else {
      items.value = store.myEnrollments.value
        .filter((e) => activeStatuses.has(e.status))
        .slice(0, 3);
    }
  } catch {
    items.value = [];
  } finally {
    loading.value = false;
  }
});

const visible = computed(() => !loading.value && items.value.length > 0);

function open(e: EnrollmentSummary) {
  router.push({ name: 'course-enrollment', params: { enrollmentId: String(e.id) } });
}
</script>

<template>
  <section
    v-if="visible"
    class="flex flex-col gap-3 w-full max-w-full min-w-0"
    aria-labelledby="learning-home-title"
  >
    <UPageHeader title="" :links="learningLinks" class="border-none p-0">
      <template #title>
        <h1 id="learning-home-title" class="text-2xl font-medium">Моё обучение</h1>
      </template>
    </UPageHeader>

    <div class="min-w-0 w-full max-w-full p-1">
      <ul class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
        <li
          v-for="e in items"
          :key="e.id"
          class="rounded-xl ring-1 ring-default bg-elevated/30 p-3 flex flex-col gap-2 cursor-pointer hover:bg-elevated/60 transition-colors min-w-0"
          @click="open(e)"
        >
          <div class="flex items-start justify-between gap-2 min-w-0">
            <p class="font-medium text-highlighted text-sm min-w-0 break-words">{{ e.courseTitle }}</p>
            <CourseStatusBadge :status="e.status" class="shrink-0" />
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
    </div>
  </section>
</template>
