<script setup lang="ts">
/** /admin/courses — список программ обучения для администратора. */
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useCoursesStore, type CourseListItem } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const loading = ref(true);
const loadError = ref<string | null>(null);

const courses = computed(() => store.courses.value);

async function loadCourses() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadList();
  } catch (e: any) {
    loadError.value = e?.message || 'Ошибка загрузки';
    toast.add({
      title: 'Не удалось загрузить список обучения',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    loading.value = false;
  }
}

onMounted(loadCourses);

function goCreateCourse() {
  router.push({ name: 'admin-course-create' });
}

function openWorkspace(id: number) {
  router.push({ name: 'admin-course-workspace', params: { courseId: String(id) } });
}

const deleteOpen = ref(false);
const deleteTarget = ref<{ id: number; title: string } | null>(null);
const deleting = ref(false);

function askDelete(c: { id: number; title: string }) {
  deleteTarget.value = { id: c.id, title: c.title };
  deleteOpen.value = true;
}

function goAssign(id: number) {
  router.push({ name: 'admin-course-assign', params: { courseId: String(id) } });
}

function goResults(id: number) {
  router.push({ name: 'admin-course-results', params: { courseId: String(id) } });
}

/** Редкое и опасное — под ⋯, чтобы «Удалить» не стояло вровень с «Открыть». */
function rowMenu(c: CourseListItem) {
  return [[
    { label: 'Назначить', icon: 'i-lucide-users', onSelect: () => goAssign(c.id) },
    { label: 'Результаты', icon: 'i-lucide-bar-chart-3', onSelect: () => goResults(c.id) },
  ], [
    {
      label: 'Удалить',
      icon: 'i-lucide-trash-2',
      color: 'error' as const,
      onSelect: () => askDelete({ id: c.id, title: c.title }),
    },
  ]];
}

async function confirmDelete() {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await store.deleteCourse(deleteTarget.value.id);
    toast.add({ title: 'Обучение удалено', color: 'success', icon: 'i-lucide-check' });
    deleteOpen.value = false;
    deleteTarget.value = null;
    await loadCourses();
  } catch (e: any) {
    toast.add({
      title: 'Не удалось удалить',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    deleting.value = false;
  }
}

function formatDate(iso?: string) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        title="Управление обучением"
        description="Программы, темы, материалы, назначения и результаты"
      >
        <template #links>
          <UButton
            color="primary"
            icon="i-lucide-plus"
            label="Создать обучение"
            @click="goCreateCourse"
          />
        </template>
      </UPageHeader>

      <div v-if="loading" class="flex flex-col gap-3" aria-busy="true" aria-label="Загрузка списка обучения">
        <div v-for="n in 4" :key="n" class="rounded-panel bg-elevated p-4 flex flex-col gap-3">
          <div class="flex items-start gap-3">
            <div class="flex-1 min-w-0 flex flex-col gap-2">
              <USkeleton class="h-4 w-1/2 rounded" />
              <USkeleton class="h-3 w-3/4 rounded" />
              <div class="flex items-center gap-2 pt-0.5">
                <USkeleton class="h-5 w-24 rounded-full" />
                <USkeleton class="h-3 w-40 rounded" />
              </div>
            </div>
            <USkeleton class="h-8 w-32 rounded-md shrink-0" />
          </div>
          <div class="border-t border-accented pt-2">
            <USkeleton class="h-6 w-48 rounded" />
          </div>
        </div>
      </div>

      <UAlert
        v-else-if="loadError"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Не удалось загрузить список обучения"
        :description="loadError"
      >
        <template #actions>
          <UButton color="warning" variant="solid" icon="i-lucide-rotate-ccw" @click="loadCourses">
            Повторить
          </UButton>
        </template>
      </UAlert>

      <UEmpty
        v-else-if="!courses.length"
        variant="naked"
        icon="i-lucide-library-big"
        title="Обучения пока нет"
        description="Создайте первое обучение и наполните его темами и материалами."
        class="w-full py-12"
      >
        <template #actions>
          <UButton color="primary" icon="i-lucide-plus" @click="goCreateCourse">
            Создать обучение
          </UButton>
        </template>
      </UEmpty>

      <div v-else class="flex flex-col gap-3">
        <UCard
          v-for="c in courses"
          :key="c.id"
          variant="soft"
          class="w-full rounded-panel transition-shadow hover:shadow-md"
          :ui="{
            root: 'rounded-panel bg-elevated ring-0 border-0 divide-y-0',
            body: 'flex flex-col gap-3 p-4 sm:p-4',
          }"
        >
          <div class="flex items-start gap-3">
            <div class="flex-1 min-w-0 flex flex-col gap-1">
              <button
                type="button"
                class="font-medium text-highlighted text-left break-words hover:text-primary focus:outline-none focus-visible:underline"
                @click="openWorkspace(c.id)"
              >
                {{ c.title }}
              </button>
              <p v-if="c.shortDescription" class="text-sm text-toned line-clamp-2 break-words">
                {{ c.shortDescription }}
              </p>
              <div class="flex items-center gap-x-2 gap-y-1 flex-wrap text-xs text-muted pt-0.5">
                <CourseStatusBadge :status="c.status" />
                <span v-if="c.category">{{ c.category }}</span>
                <span v-if="c.category && c.versionNumber" aria-hidden="true">·</span>
                <span v-if="c.versionNumber">версия {{ c.versionNumber }}</span>
                <span v-if="c.updatedAt" aria-hidden="true">·</span>
                <span v-if="c.updatedAt">обновлён {{ formatDate(c.updatedAt) }}</span>
              </div>
            </div>
            <div class="flex items-center gap-1.5 shrink-0">
              <UButton color="neutral" variant="soft" icon="i-lucide-pencil" @click="openWorkspace(c.id)">
                Открыть
              </UButton>
              <UDropdownMenu :items="rowMenu(c)">
                <UButton
                  type="button"
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-ellipsis"
                  :aria-label="`Ещё действия: ${c.title}`"
                />
              </UDropdownMenu>
            </div>
          </div>

          <div class="flex items-center gap-1 flex-wrap border-t border-accented pt-2 -mb-1">
            <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-users" @click="goAssign(c.id)">
              Назначить
            </UButton>
            <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-bar-chart-3" @click="goResults(c.id)">
              Результаты
            </UButton>
          </div>
        </UCard>
      </div>
    </div>

    <UModal
      v-model:open="deleteOpen"
      title="Удалить обучение?"
      description="Обучение и его материалы будут удалены. Это действие нельзя отменить."
    >
      <template #body>
        <p class="text-sm text-muted">
          Обучение:
          <span class="text-highlighted font-medium">{{ deleteTarget?.title || '—' }}</span>
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-trash-2" :loading="deleting" @click="confirmDelete">
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
