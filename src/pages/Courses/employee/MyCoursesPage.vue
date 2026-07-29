<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useCoursesStore, type EnrollmentSummary } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useHeaderUser } from '../../../composables/useHeaderUser';
import { currentRole, setRole } from '../../../stores/role';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const { canToggleAdminRole } = useHeaderUser();
const loading = ref(true);
const loadError = ref<string | null>(null);

const overdue = ref<EnrollmentSummary[]>([]);
const inProgress = ref<EnrollmentSummary[]>([]);
const fresh = ref<EnrollmentSummary[]>([]);
const completed = ref<EnrollmentSummary[]>([]);

const isCourseAdmin = computed(
  () => canToggleAdminRole.value || currentRole.value === 'admin',
);

onMounted(async () => {
  loading.value = true;
  loadError.value = null;
  try {
    const groups = (await store.loadMyCourses()) as any;
    if (groups?.overdue || groups?.active) {
      overdue.value = groups.overdue || [];
      const active: EnrollmentSummary[] = groups.active || [];
      inProgress.value = active.filter((e) => e.status === 'in_progress');
      fresh.value = active.filter((e) => e.status === 'not_started');
      completed.value = groups.completed || [];
    } else {
      const all = store.myEnrollments.value;
      overdue.value = all.filter((e) => e.status === 'overdue');
      inProgress.value = all.filter((e) => e.status === 'in_progress');
      fresh.value = all.filter((e) => e.status === 'not_started');
      completed.value = all.filter((e) => e.status === 'completed');
    }
  } catch (e: any) {
    loadError.value = e?.message || 'Ошибка загрузки';
    toast.add({
      title: 'Не удалось загрузить курсы',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    loading.value = false;
  }
});

const emptyAll = computed(
  () => !overdue.value.length && !inProgress.value.length && !fresh.value.length && !completed.value.length,
);

async function goManageCourses() {
  if (canToggleAdminRole.value) setRole('admin');
  await nextTick();
  await router.push({ name: 'admin-courses' });
}

async function goCreateCourse() {
  if (canToggleAdminRole.value) setRole('admin');
  await nextTick();
  await router.push({ name: 'admin-course-create' });
}

function open(e: EnrollmentSummary) {
  if (e.status === 'completed') {
    router.push({ name: 'course-result', params: { enrollmentId: String(e.id) } });
  } else {
    router.push({ name: 'course-enrollment', params: { enrollmentId: String(e.id) } });
  }
}

function ctaLabel(e: EnrollmentSummary) {
  if (e.status === 'not_started') return 'Начать';
  if (e.status === 'completed') return 'Результат';
  if (e.status === 'overdue') return 'Открыть';
  return e.nextAction?.label || 'Продолжить';
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <h1 class="text-2xl font-medium text-highlighted">Мои курсы</h1>
      <div class="flex items-center gap-2 flex-wrap">
        <UButton
          v-if="isCourseAdmin"
          color="primary"
          icon="i-lucide-plus"
          @click="goCreateCourse"
        >
          Создать курс
        </UButton>
        <UButton
          v-if="isCourseAdmin"
          color="neutral"
          variant="soft"
          icon="i-lucide-library-big"
          @click="goManageCourses"
        >
          Управление
        </UButton>
        <UButton color="neutral" variant="ghost" icon="i-lucide-history" :to="{ name: 'course-history' }">
          История
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="isCourseAdmin"
      color="primary"
      variant="subtle"
      icon="i-lucide-shield-check"
      title="Режим администратора"
      description="Вы можете создавать курсы, темы, материалы и назначать их сотрудникам. Включите переключатель «администратор» в шапке или нажмите «Создать курс»."
      class="shrink-0"
    />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-20 w-full rounded-xl" />
    </div>

    <UEmpty
      v-else-if="emptyAll"
      icon="i-lucide-graduation-cap"
      :title="isCourseAdmin ? 'Назначенных вам курсов нет' : 'Вам пока ничего не назначено'"
      :description="isCourseAdmin
        ? 'Как администратор вы можете создать курс и назначить его сотрудникам.'
        : 'Когда HR направит курс, он появится здесь.'"
      class="py-12"
    >
      <template v-if="isCourseAdmin" #actions>
        <div class="flex flex-wrap gap-2 justify-center">
          <UButton color="primary" icon="i-lucide-plus" @click="goCreateCourse">
            Создать курс
          </UButton>
          <UButton color="neutral" variant="soft" icon="i-lucide-library-big" @click="goManageCourses">
            Все курсы
          </UButton>
        </div>
      </template>
    </UEmpty>

    <UAlert
      v-if="!loading && loadError && isCourseAdmin"
      color="warning"
      variant="subtle"
      icon="i-lucide-server"
      title="API курсов недоступен"
      :description="`Сервер ответил: ${loadError}. Убедитесь, что на сервер загружены файлы api/courses_*.php и применена миграция V4.`"
      class="shrink-0"
    />

    <template v-else-if="!emptyAll">
      <section v-if="overdue.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium text-error">Просроченные</h2>
        <div
          v-for="e in overdue"
          :key="e.id"
          class="rounded-xl ring-1 ring-error/30 bg-error/5 p-4 flex flex-col md:flex-row md:items-center gap-3"
        >
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <CourseStatusBadge :status="e.status" />
              <span class="font-medium truncate">{{ e.courseTitle }}</span>
            </div>
            <p class="text-xs text-dimmed mt-1">
              Завершено {{ e.topicsCompleted ?? 0 }} из {{ e.topicsTotal ?? 0 }} тем, {{ e.progressPercent ?? 0 }}%
            </p>
          </div>
          <UButton color="primary" @click="open(e)">{{ ctaLabel(e) }}</UButton>
        </div>
      </section>

      <section v-if="inProgress.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium">В процессе</h2>
        <div
          v-for="e in inProgress"
          :key="e.id"
          class="rounded-xl ring-1 ring-default bg-elevated/30 p-4 flex flex-col md:flex-row md:items-center gap-3"
        >
          <div class="flex-1 min-w-0 flex flex-col gap-2">
            <div class="flex items-center gap-2 flex-wrap">
              <CourseStatusBadge :status="e.status" />
              <span class="font-medium truncate">{{ e.courseTitle }}</span>
            </div>
            <UProgress
              :model-value="e.progressPercent ?? 0"
              size="sm"
              color="primary"
              :aria-label="`Завершено ${e.topicsCompleted ?? 0} из ${e.topicsTotal ?? 0} тем, ${e.progressPercent ?? 0} процентов`"
            />
            <p class="text-xs text-dimmed">
              Завершено {{ e.topicsCompleted ?? 0 }} из {{ e.topicsTotal ?? 0 }} тем, {{ e.progressPercent ?? 0 }}%
            </p>
          </div>
          <UButton color="primary" icon="i-lucide-play" @click="open(e)">Продолжить</UButton>
        </div>
      </section>

      <section v-if="fresh.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium">Новые</h2>
        <div
          v-for="e in fresh"
          :key="e.id"
          class="rounded-xl ring-1 ring-default bg-elevated/30 p-4 flex flex-col md:flex-row md:items-center gap-3"
        >
          <div class="flex-1 min-w-0">
            <span class="font-medium">{{ e.courseTitle }}</span>
            <p v-if="e.deadlineAt" class="text-xs text-dimmed mt-1">
              Срок: {{ new Date(e.deadlineAt).toLocaleDateString('ru-RU') }}
            </p>
          </div>
          <UButton color="primary" icon="i-lucide-play" @click="open(e)">Начать</UButton>
        </div>
      </section>

      <section v-if="completed.length" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium">Завершённые</h2>
        <div
          v-for="e in completed"
          :key="e.id"
          class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 opacity-90"
        >
          <div class="flex-1 min-w-0 flex items-center gap-2 flex-wrap">
            <CourseStatusBadge status="completed" />
            <span class="font-medium truncate">{{ e.courseTitle }}</span>
          </div>
          <UButton color="neutral" variant="soft" @click="open(e)">Результат</UButton>
        </div>
      </section>
    </template>
  </UMain>
</template>
