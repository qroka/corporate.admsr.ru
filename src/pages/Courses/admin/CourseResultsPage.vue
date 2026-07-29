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

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const summary = ref<Record<string, number | null>>({});
const rows = ref<any[]>([]);
const selectedId = ref<number | null>(
  route.query.enrollmentId ? Number(route.query.enrollmentId) : null,
);
const participant = ref<any | null>(null);
const detailLoading = ref(false);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'admin-courses' } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Результаты' },
]);

async function load() {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
    const data = (await store.loadResults({
      courseId: courseId.value,
      versionId: store.version.value?.id,
      limit: 100,
    })) as any;
    const agg = data?.aggregates || data?.summary || data?.stats || {};
    summary.value = {
      total: agg.total,
      completed: agg.completed,
      in_progress: agg.inProgress ?? agg.in_progress,
      overdue: agg.overdue,
      avg_score: agg.avgScore ?? agg.avg_score,
    };
    rows.value = (data?.items || data?.rows || data?.participants || []).map((r: any) => {
      const enr = r.enrollment || r;
      const user = r.user || {};
      return {
        ...r,
        id: enr.id ?? r.enrollmentId ?? r.id,
        fio: user.fio || r.fio || r.fullName || 'Сотрудник',
        status: enr.status ?? r.status,
        progressPercent: enr.progress?.percent ?? r.progressPercent ?? 0,
        finalScore: enr.finalScore ?? r.finalScore,
        deadlineAt: enr.deadlineAt ?? r.deadlineAt,
      };
    });
  } catch (e: any) {
    toast.add({ title: 'Не удалось загрузить отчёт', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await load();
  if (selectedId.value) await openDetail(selectedId.value);
});

async function openDetail(enrollmentId: number) {
  selectedId.value = enrollmentId;
  detailLoading.value = true;
  try {
    participant.value = await store.loadParticipant({ enrollmentId, courseId: courseId.value });
  } catch (e: any) {
    toast.add({ title: 'Не удалось открыть карточку', description: e?.message, color: 'error', icon: 'i-lucide-x' });
    participant.value = null;
  } finally {
    detailLoading.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Результаты курса</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton class="h-24 w-full rounded-xl" />
      <USkeleton class="h-64 w-full rounded-xl" />
    </div>

    <template v-else>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="rounded-xl ring-1 ring-default p-3">
          <p class="text-xs text-dimmed">Всего</p>
          <p class="text-xl font-medium">{{ summary.total ?? rows.length }}</p>
        </div>
        <div class="rounded-xl ring-1 ring-default p-3">
          <p class="text-xs text-dimmed">Завершили</p>
          <p class="text-xl font-medium">{{ summary.completed ?? 0 }}</p>
        </div>
        <div class="rounded-xl ring-1 ring-default p-3">
          <p class="text-xs text-dimmed">В процессе</p>
          <p class="text-xl font-medium">{{ summary.in_progress ?? summary.inProgress ?? 0 }}</p>
        </div>
        <div class="rounded-xl ring-1 ring-default p-3">
          <p class="text-xs text-dimmed">Средний балл</p>
          <p class="text-xl font-medium">
            {{ summary.avg_score != null ? Number(summary.avg_score).toFixed(0) : (summary.avgScore != null ? Number(summary.avgScore).toFixed(0) : '—') }}
          </p>
        </div>
      </div>

      <UEmpty
        v-if="!rows.length"
        icon="i-lucide-bar-chart-3"
        title="Пока нет участников"
        description="Назначьте курс сотрудникам, чтобы видеть прогресс."
        class="py-10"
      />

      <div v-else class="flex flex-col lg:flex-row gap-4 min-h-0 flex-1">
        <ul class="flex-1 min-w-0 overflow-auto flex flex-col gap-2 list-none p-0 m-0">
          <li
            v-for="row in rows"
            :key="row.id"
            class="rounded-xl ring-1 ring-default p-3 flex items-center gap-3 cursor-pointer hover:bg-elevated/50"
            :class="selectedId === row.id ? 'ring-primary' : ''"
            @click="openDetail(row.id)"
          >
            <div class="flex-1 min-w-0">
              <p class="font-medium truncate">{{ row.fio }}</p>
              <div class="flex items-center gap-2 mt-1 flex-wrap">
                <CourseStatusBadge :status="row.status" />
                <span class="text-xs text-dimmed">{{ row.progressPercent ?? 0 }}%</span>
              </div>
            </div>
            <span class="text-sm text-dimmed tabular-nums">
              {{ row.finalScore == null ? '—' : row.finalScore }}
            </span>
          </li>
        </ul>

        <aside v-if="selectedId" class="w-full lg:w-96 shrink-0 rounded-xl ring-1 ring-default p-4 flex flex-col gap-3">
          <div class="flex items-center justify-between gap-2">
            <h2 class="text-lg font-medium">Участник</h2>
            <UButton
              color="neutral"
              variant="ghost"
              size="sm"
              icon="i-lucide-x"
              aria-label="Закрыть карточку"
              @click="selectedId = null; participant = null"
            />
          </div>
          <USkeleton v-if="detailLoading" class="h-40 w-full rounded-lg" />
          <template v-else-if="participant">
            <p class="font-medium">
              {{ participant.fio || participant.user?.fio || participant.enrollment?.fio || 'Сотрудник' }}
            </p>
            <CourseStatusBadge :status="participant.status || participant.enrollment?.status" />
            <p class="text-sm text-muted">
              Прогресс:
              {{ participant.progress?.percent ?? participant.progressPercent ?? 0 }}%
            </p>
            <p v-if="participant.finalScore != null" class="text-sm">
              Итоговый балл: {{ participant.finalScore }}
            </p>
          </template>
        </aside>
      </div>
    </template>
  </UMain>
</template>
