<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore, type EnrollmentSummary } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useSectionAccess } from '../../../composables/useSectionAccess';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const loadingMine = ref(true);
const loadingManage = ref(false);
const loadErrorMine = ref<string | null>(null);
const loadErrorManage = ref<string | null>(null);

const overdue = ref<EnrollmentSummary[]>([]);
const inProgress = ref<EnrollmentSummary[]>([]);
const fresh = ref<EnrollmentSummary[]>([]);
const completed = ref<EnrollmentSummary[]>([]);

const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const isCourseAdmin = computed(() => canEditSection('courses'));

const tabItems = computed(() => {
  const items = [{ label: 'Назначенные мне', value: 'mine' as const }];
  if (isCourseAdmin.value) {
    items.push({ label: 'Управление курсами', value: 'manage' as const });
  }
  return items;
});

const tab = ref<'mine' | 'manage'>('mine');

watch(
  () => route.query.tab,
  (q) => {
    if (q === 'manage' && isCourseAdmin.value) tab.value = 'manage';
    else if (q === 'mine') tab.value = 'mine';
  },
  { immediate: true },
);

watch(isCourseAdmin, (ok) => {
  if (!ok && tab.value === 'manage') {
    tab.value = 'mine';
    if (route.query.tab === 'manage') {
      void router.replace({ query: {} });
    }
  }
});

watch(tab, async (t) => {
  if (!isCourseAdmin.value && t === 'manage') {
    tab.value = 'mine';
    return;
  }
  const q = t === 'manage' ? 'manage' : undefined;
  if ((route.query.tab || undefined) !== q) {
    await router.replace({ query: q ? { tab: q } : {} });
  }
  if (t === 'manage' && isCourseAdmin.value) {
    await ensureManageLoaded();
  }
});

async function loadMine() {
  loadingMine.value = true;
  loadErrorMine.value = null;
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
    loadErrorMine.value = e?.message || 'Ошибка загрузки';
    if (!isCourseAdmin.value) {
      toast.add({
        title: 'Не удалось загрузить курсы',
        description: e?.message,
        color: 'error',
        icon: 'i-lucide-alert-circle',
      });
    }
  } finally {
    loadingMine.value = false;
  }
}

async function ensureManageLoaded() {
  if (!isCourseAdmin.value) return;
  loadingManage.value = true;
  loadErrorManage.value = null;
  try {
    await store.loadList();
  } catch (e: any) {
    loadErrorManage.value = e?.message || 'Ошибка загрузки';
    toast.add({
      title: 'Не удалось загрузить список курсов',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    loadingManage.value = false;
  }
}

onMounted(async () => {
  await loadMine();
  if (tab.value === 'manage' && isCourseAdmin.value) {
    await ensureManageLoaded();
  }
});

const emptyMine = computed(
  () => !overdue.value.length && !inProgress.value.length && !fresh.value.length && !completed.value.length,
);

async function goCreateCourse() {
  if (!isCourseAdmin.value) return;
  await router.push({ name: 'admin-course-create' });
}

function openWorkspace(id: number) {
  if (!isCourseAdmin.value) return;
  router.push({ name: 'admin-course-workspace', params: { courseId: String(id) } });
}

function open(e: EnrollmentSummary) {
  router.push({ name: 'course-enrollment', params: { enrollmentId: String(e.id) } });
}

function ctaLabel(e: EnrollmentSummary) {
  if (e.status === 'not_started') return 'Начать';
  if (e.status === 'completed') return 'Смотреть';
  if (e.status === 'failed') return 'Смотреть';
  if (e.status === 'overdue') return 'Открыть';
  return e.nextAction?.label || 'Продолжить';
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        title="Мои курсы"
        description="Назначенные курсы, прогресс прохождения и управление материалами"
      >
        <template #links>
          <UButton
            v-if="isCourseAdmin"
            color="primary"
            icon="i-lucide-plus"
            label="Создать курс"
            @click="goCreateCourse"
          />
        </template>
      </UPageHeader>

      <UTabs
        v-if="isCourseAdmin"
        v-model="tab"
        :items="tabItems"
        variant="link"
        color="primary"
        size="md"
        :content="false"
        class="w-full border-b border-default"
        :ui="{
          list: 'w-full gap-1',
          trigger: 'grow-0',
        }"
      />

      <!-- Назначенные мне -->
      <template v-if="tab === 'mine'">
        <div v-if="loadingMine" class="flex flex-col gap-3">
          <USkeleton v-for="n in 4" :key="n" class="h-20 w-full rounded-panel" />
        </div>

        <UAlert
          v-else-if="loadErrorMine"
          color="warning"
          variant="subtle"
          icon="i-lucide-server"
          title="Не удалось загрузить назначения"
          :description="loadErrorMine"
        />

        <UEmpty
          v-else-if="emptyMine"
          variant="naked"
          icon="i-lucide-graduation-cap"
          title="Вам пока ничего не назначено"
          description="Когда HR направит курс, он появится здесь."
          class="w-full py-12"
        >
          <template v-if="isCourseAdmin" #actions>
            <UButton color="primary" variant="soft" @click="tab = 'manage'">
              Перейти к управлению курсами
            </UButton>
          </template>
        </UEmpty>

        <div v-else class="flex flex-col gap-6">
          <section v-if="overdue.length" class="flex flex-col gap-3">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-error">
              Просроченные
            </h2>
            <UCard
              v-for="e in overdue"
              :key="e.id"
              variant="soft"
              class="w-full rounded-panel"
              :ui="{
                root: 'rounded-panel bg-error/5 ring-1 ring-inset ring-error/25 border-0 divide-y-0',
                body: 'flex flex-col md:flex-row md:items-center gap-3 p-4 sm:p-4',
              }"
            >
              <div class="flex-1 min-w-0 flex flex-col gap-1.5">
                <div class="flex items-center gap-2 flex-wrap">
                  <CourseStatusBadge :status="e.status" />
                  <span class="font-medium text-highlighted break-words">{{ e.courseTitle }}</span>
                </div>
                <p class="text-xs text-muted">
                  Завершено {{ e.topicsCompleted ?? 0 }} из {{ e.topicsTotal ?? 0 }} тем · {{ e.progressPercent ?? 0 }}%
                </p>
              </div>
              <UButton color="primary" class="shrink-0" @click="open(e)">
                {{ ctaLabel(e) }}
              </UButton>
            </UCard>
          </section>

          <section v-if="inProgress.length" class="flex flex-col gap-3">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">
              В процессе
            </h2>
            <UCard
              v-for="e in inProgress"
              :key="e.id"
              variant="soft"
              class="w-full rounded-panel"
              :ui="{
                root: 'rounded-panel bg-elevated ring-0 border-0 divide-y-0',
                body: 'flex flex-col md:flex-row md:items-center gap-3 p-4 sm:p-4',
              }"
            >
              <div class="flex-1 min-w-0 flex flex-col gap-2">
                <div class="flex items-center gap-2 flex-wrap">
                  <CourseStatusBadge :status="e.status" />
                  <span class="font-medium text-highlighted break-words">{{ e.courseTitle }}</span>
                </div>
                <UProgress
                  :model-value="e.progressPercent ?? 0"
                  size="sm"
                  color="primary"
                  :aria-label="`Завершено ${e.topicsCompleted ?? 0} из ${e.topicsTotal ?? 0} тем, ${e.progressPercent ?? 0} процентов`"
                />
                <p class="text-xs text-muted">
                  Завершено {{ e.topicsCompleted ?? 0 }} из {{ e.topicsTotal ?? 0 }} тем · {{ e.progressPercent ?? 0 }}%
                </p>
              </div>
              <UButton color="primary" icon="i-lucide-play" class="shrink-0" @click="open(e)">
                Продолжить
              </UButton>
            </UCard>
          </section>

          <section v-if="fresh.length" class="flex flex-col gap-3">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">
              Новые
            </h2>
            <UCard
              v-for="e in fresh"
              :key="e.id"
              variant="soft"
              class="w-full rounded-panel"
              :ui="{
                root: 'rounded-panel bg-elevated ring-0 border-0 divide-y-0',
                body: 'flex flex-col md:flex-row md:items-center gap-3 p-4 sm:p-4',
              }"
            >
              <div class="flex-1 min-w-0">
                <span class="font-medium text-highlighted break-words block">{{ e.courseTitle }}</span>
              </div>
              <UButton color="primary" icon="i-lucide-play" class="shrink-0" @click="open(e)">
                Начать
              </UButton>
            </UCard>
          </section>

          <section v-if="completed.length" class="flex flex-col gap-3">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">
              Завершённые
            </h2>
            <UCard
              v-for="e in completed"
              :key="e.id"
              variant="soft"
              class="w-full rounded-panel"
              :ui="{
                root: 'rounded-panel bg-elevated/60 ring-0 border-0 divide-y-0',
                body: 'flex flex-col md:flex-row md:items-center gap-3 p-4 sm:p-4',
              }"
            >
              <div class="flex-1 min-w-0 flex items-center gap-2 flex-wrap">
                <CourseStatusBadge status="completed" />
                <span class="font-medium text-highlighted break-words">{{ e.courseTitle }}</span>
              </div>
              <UButton color="neutral" variant="soft" class="shrink-0" @click="open(e)">
                Смотреть
              </UButton>
            </UCard>
          </section>
        </div>
      </template>

      <!-- Управление курсами -->
      <template v-else-if="tab === 'manage' && isCourseAdmin">
        <div v-if="loadingManage" class="flex flex-col gap-3">
          <USkeleton v-for="n in 4" :key="n" class="h-20 w-full rounded-panel" />
        </div>

        <UAlert
          v-else-if="loadErrorManage"
          color="warning"
          variant="subtle"
          icon="i-lucide-server"
          title="API курсов недоступен"
          :description="`${loadErrorManage}. Нужны файлы api/courses_*.php и миграция V4 на сервере.`"
        />

        <UEmpty
          v-else-if="!store.courses.value.length"
          variant="naked"
          icon="i-lucide-library-big"
          title="Курсов пока нет"
          description="Создайте первый курс и наполните его темами и материалами."
          class="w-full py-12"
        >
          <template #actions>
            <UButton color="primary" icon="i-lucide-plus" @click="goCreateCourse">
              Создать курс
            </UButton>
          </template>
        </UEmpty>

        <div v-else class="flex flex-col gap-3">
          <UCard
            v-for="c in store.courses.value"
            :key="c.id"
            variant="soft"
            class="w-full rounded-panel cursor-pointer transition-colors hover:bg-elevated/80"
            :ui="{
              root: 'rounded-panel bg-elevated ring-0 border-0 divide-y-0',
              body: 'flex flex-col md:flex-row md:items-center gap-3 p-4 sm:p-4',
            }"
            @click="openWorkspace(c.id)"
          >
            <div class="flex-1 min-w-0 flex flex-col gap-1">
              <div class="flex items-center gap-2 flex-wrap">
                <CourseStatusBadge :status="c.status" />
                <UBadge v-if="c.category" color="neutral" variant="subtle" size="sm">
                  {{ c.category }}
                </UBadge>
                <span class="font-medium text-highlighted break-words">{{ c.title }}</span>
              </div>
              <p class="text-xs text-muted">
                Тем: {{ c.topicsCount ?? 0 }}
                <template v-if="c.updatedAt">
                  · обновлён {{ new Date(c.updatedAt).toLocaleDateString('ru-RU') }}
                </template>
              </p>
            </div>
            <UButton
              color="neutral"
              variant="soft"
              icon="i-lucide-arrow-right"
              class="shrink-0"
              aria-label="Открыть курс"
              @click.stop="openWorkspace(c.id)"
            >
              Открыть
            </UButton>
          </UCard>
        </div>
      </template>
    </div>
  </UMain>
</template>
