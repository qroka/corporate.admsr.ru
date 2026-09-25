<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore, type EnrollmentSummary } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useSectionAccess } from '../../../composables/useSectionAccess';
import MyCourseCard from '../components/MyCourseCard.vue';
import { compareByAttention } from '../courseDeadline';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const isCourseAdmin = computed(() => canEditSection('courses'));

const loading = ref(true);
const loadError = ref<string | null>(null);
const items = ref<EnrollmentSummary[]>([]);
const showCompleted = ref(false);

const active = computed(() =>
  items.value
    .filter((e) => e.status !== 'completed')
    .slice()
    .sort(compareByAttention),
);

const completed = computed(() =>
  items.value
    .filter((e) => e.status === 'completed')
    .slice()
    .sort((a, b) => new Date(b.completedAt || 0).getTime() - new Date(a.completedAt || 0).getTime()),
);

const overdueCount = computed(() => active.value.filter((e) => e.status === 'overdue').length);
const isEmpty = computed(() => !items.value.length);

async function loadMine() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadMyCourses();
    // Стор уже сводит все группы (active / overdue / failed / completed) в один список —
    // берём его целиком, чтобы ни один статус не потерялся.
    items.value = store.myEnrollments.value.slice();
  } catch (e: any) {
    loadError.value = e?.message || 'Ошибка загрузки';
    toast.add({
      title: 'Не удалось загрузить обучение',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  // Совместимость со старой ссылкой /courses?tab=manage
  if (route.query.tab === 'manage' && isCourseAdmin.value) {
    await router.replace({ name: 'admin-courses' });
    return;
  }
  if (route.query.tab) {
    await router.replace({ query: {} });
  }
  await loadMine();
});

function open(e: EnrollmentSummary) {
  router.push({ name: 'course-enrollment', params: { enrollmentId: String(e.id) } });
}

function goManage() {
  router.push({ name: 'admin-courses' });
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        title="Моё обучение"
        description="Назначенные вам программы и прогресс прохождения"
      >
        <template #links>
          <UButton
            v-if="isCourseAdmin"
            color="neutral"
            variant="soft"
            icon="i-lucide-settings-2"
            label="Управление обучением"
            @click="goManage"
          />
        </template>
      </UPageHeader>

      <!-- Загрузка: скелет повторяет форму реальной карточки -->
      <div v-if="loading" class="flex flex-col gap-3" aria-busy="true" aria-label="Загрузка назначений">
        <div
          v-for="n in 3"
          :key="n"
          class="rounded-panel bg-elevated p-4 flex flex-col md:flex-row md:items-center gap-3"
        >
          <div class="flex-1 min-w-0 flex flex-col gap-2">
            <USkeleton class="h-4 w-1/2 rounded" />
            <div class="flex items-center gap-2">
              <USkeleton class="h-5 w-20 rounded-full" />
              <USkeleton class="h-3 w-40 rounded" />
            </div>
            <USkeleton class="h-2 w-full rounded-full" />
          </div>
          <USkeleton class="h-8 w-28 rounded-md shrink-0" />
        </div>
      </div>

      <!-- Ошибка: с возможностью повторить, а не тупик -->
      <UAlert
        v-else-if="loadError"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Не удалось загрузить назначения"
        :description="loadError"
      >
        <template #actions>
          <UButton color="warning" variant="solid" icon="i-lucide-rotate-ccw" @click="loadMine">
            Повторить
          </UButton>
        </template>
      </UAlert>

      <!-- Назначений нет совсем -->
      <UEmpty
        v-else-if="isEmpty"
        variant="naked"
        icon="i-lucide-graduation-cap"
        title="Вам пока ничего не назначено"
        description="Когда HR направит курс, он появится здесь."
        class="w-full py-12"
      >
        <template v-if="isCourseAdmin" #actions>
          <UButton color="primary" variant="soft" icon="i-lucide-settings-2" @click="goManage">
            Перейти к управлению обучением
          </UButton>
        </template>
      </UEmpty>

      <div v-else class="flex flex-col gap-6">
        <!-- Нужно пройти -->
        <section v-if="active.length" class="flex flex-col gap-3">
          <div class="flex items-center gap-2 flex-wrap">
            <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">Нужно пройти</h2>
            <UBadge color="neutral" variant="subtle" size="sm">{{ active.length }}</UBadge>
            <UBadge v-if="overdueCount" color="error" variant="subtle" size="sm">
              просрочено: {{ overdueCount }}
            </UBadge>
          </div>
          <MyCourseCard v-for="e in active" :key="e.id" :enrollment="e" @open="open" />
        </section>

        <!-- Активного нет, но что-то пройдено -->
        <UEmpty
          v-else
          variant="naked"
          icon="i-lucide-check-circle-2"
          title="Всё пройдено"
          description="Новых назначений нет. Завершённые программы — ниже."
          class="w-full py-10"
        />

        <!-- Завершённые: по умолчанию свёрнуты, чтобы не выдавливать активное -->
        <section v-if="completed.length" class="flex flex-col gap-3">
          <UButton
            color="neutral"
            variant="ghost"
            class="self-start -ml-2"
            :icon="showCompleted ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
            :aria-expanded="showCompleted"
            @click="showCompleted = !showCompleted"
          >
            {{ showCompleted ? 'Скрыть завершённые' : `Показать завершённые (${completed.length})` }}
          </UButton>
          <template v-if="showCompleted">
            <MyCourseCard v-for="e in completed" :key="e.id" :enrollment="e" @open="open" />
          </template>
        </section>
      </div>
    </div>
  </UMain>
</template>
