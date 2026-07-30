<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore, type CourseTopic } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const deleteOpen = ref(false);
const deleteTarget = ref<CourseTopic | null>(null);
const ordering = ref(false);
const unpublishOpen = ref(false);
const unpublishing = ref(false);

const course = computed(() => store.current.value);
const version = computed(() => store.version.value);
const topics = computed(() => store.topics.value);
const isPublished = computed(() => version.value?.status === 'published');
const isEditable = computed(() => version.value?.status !== 'archived');

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: course.value?.title || 'Курс' },
]);

async function reload() {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Курс не загружен', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
}

onMounted(reload);

async function moveTopic(index: number, dir: -1 | 1) {
  const list = [...topics.value];
  const j = index + dir;
  if (j < 0 || j >= list.length || !version.value?.id) return;
  const tmp = list[index];
  list[index] = list[j];
  list[j] = tmp;
  ordering.value = true;
  try {
    await store.orderTopics(version.value.id, list.map((t) => t.id));
    await store.loadCourse(courseId.value);
    toast.add({ title: 'Порядок обновлён', color: 'success', icon: 'i-lucide-check' });
  } catch (e: any) {
    toast.add({ title: 'Не удалось изменить порядок', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    ordering.value = false;
  }
}

function askDelete(t: CourseTopic) {
  deleteTarget.value = t;
  deleteOpen.value = true;
}

async function confirmDelete() {
  if (!deleteTarget.value) return;
  try {
    await store.deleteTopic(deleteTarget.value.id);
    deleteOpen.value = false;
    deleteTarget.value = null;
    await store.loadCourse(courseId.value);
    toast.add({ title: 'Тема удалена', color: 'success', icon: 'i-lucide-check' });
  } catch (e: any) {
    toast.add({ title: 'Не удалось удалить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  }
}

function askUnpublish() {
  unpublishOpen.value = true;
}

async function confirmUnpublish() {
  unpublishing.value = true;
  try {
    await store.unpublishCourse(courseId.value);
    unpublishOpen.value = false;
    await store.loadCourse(courseId.value);
    toast.add({ title: 'Публикация снята', color: 'success', icon: 'i-lucide-check' });
  } catch (e: any) {
    toast.add({ title: 'Не удалось снять публикацию', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    unpublishing.value = false;
  }
}

function topicTest(t: CourseTopic) {
  return (t as any).testLink || (t as any).topicTest || null;
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-full min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <UBreadcrumb :items="crumbs" />

    <div v-if="loading" class="flex flex-col gap-3 p-1">
      <USkeleton class="h-10 w-2/3 rounded-lg" />
      <USkeleton class="h-24 w-full rounded-xl" />
      <USkeleton v-for="n in 3" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <template v-else-if="course">
      <div class="flex flex-col lg:flex-row lg:items-start justify-between gap-4 min-w-0 p-1">
        <div class="flex flex-col gap-2 min-w-0">
          <div class="flex items-center gap-2 flex-wrap">
            <CourseStatusBadge :status="version?.status" />
          </div>
          <h1 class="text-2xl font-medium text-highlighted break-words">{{ course.title }}</h1>
          <p v-if="version?.shortDescription" class="text-sm text-muted break-words whitespace-pre-wrap">
            {{ version.shortDescription }}
          </p>
        </div>

        <div class="flex flex-wrap gap-2 shrink-0">
          <UButton
            color="neutral"
            variant="soft"
            icon="i-lucide-settings"
            :to="{ name: 'admin-course-settings', params: { courseId } }"
          >
            Настройки
          </UButton>
          <UButton
            v-if="isPublished"
            color="primary"
            icon="i-lucide-user-plus"
            :to="{ name: 'admin-course-assign', params: { courseId } }"
          >
            Назначить
          </UButton>

          <UButton
            v-else-if="isEditable"
            color="primary"
            icon="i-lucide-send"
            :to="{ name: 'admin-course-publish', params: { courseId } }"
          >
            Опубликовать
          </UButton>

          <UButton
            v-if="isPublished"
            color="neutral"
            variant="soft"
            icon="i-lucide-x"
            @click="askUnpublish"
          >
            Снять публикацию
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            icon="i-lucide-bar-chart-3"
            :to="{ name: 'admin-course-results', params: { courseId } }"
          >
            Результаты
          </UButton>
        </div>
      </div>

      <section class="flex flex-col gap-3 min-w-0 p-1">
        <div class="flex items-center justify-between gap-2 flex-wrap">
          <h2 class="text-lg font-medium text-highlighted">Темы</h2>
          <UButton
            v-if="isEditable"
            color="primary"
            variant="soft"
            icon="i-lucide-plus"
            :to="{ name: 'admin-course-topic-create', params: { courseId } }"
          >
            Добавить тему
          </UButton>
        </div>

        <UEmpty
          v-if="!topics.length"
          icon="i-lucide-list"
          title="Тем пока нет"
          description="Добавьте первую тему и материалы."
          class="py-8"
        />

        <ul v-else class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
          <li
            v-for="(t, idx) in topics"
            :key="t.id"
            class="rounded-xl ring-1 ring-default bg-elevated/30 p-4 flex flex-col md:flex-row md:items-center gap-3 min-w-0"
          >
            <div class="flex-1 min-w-0 flex flex-col gap-1">
              <div class="flex items-start gap-2 flex-wrap">
                <span class="text-xs text-dimmed tabular-nums shrink-0 pt-0.5">{{ idx + 1 }}.</span>
                <span class="font-medium text-highlighted break-words min-w-0">{{ t.title }}</span>
                <UBadge v-if="t.ready === false" color="warning" variant="subtle">Не готова</UBadge>
              </div>
              <p class="text-xs text-dimmed">
                Материалов: {{ t.materialsCount ?? t.materials?.length ?? 0 }}
                · вопросов: {{ t.questionCount ?? topicTest(t)?.questionCount ?? 0 }}
              </p>
            </div>

            <div class="flex items-center gap-1 flex-wrap shrink-0">
              <UButton
                v-if="isEditable"
                color="neutral"
                variant="ghost"
                size="sm"
                icon="i-lucide-arrow-up"
                :disabled="idx === 0 || ordering"
                aria-label="Переместить тему выше"
                @click="moveTopic(idx, -1)"
              />
              <UButton
                v-if="isEditable"
                color="neutral"
                variant="ghost"
                size="sm"
                icon="i-lucide-arrow-down"
                :disabled="idx === topics.length - 1 || ordering"
                aria-label="Переместить тему ниже"
                @click="moveTopic(idx, 1)"
              />
              <UButton
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-pencil"
                :to="{ name: 'admin-course-topic-edit', params: { courseId, topicId: t.id } }"
              >
                Тема
              </UButton>
              <UButton
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-clipboard-list"
                :to="{ name: 'admin-course-topic-test', params: { courseId, topicId: t.id } }"
              >
                Тест
              </UButton>
              <UButton
                v-if="isEditable"
                color="error"
                variant="ghost"
                size="sm"
                icon="i-lucide-trash-2"
                aria-label="Удалить тему"
                @click="askDelete(t)"
              />
            </div>
          </li>
        </ul>
      </section>

      <section class="rounded-xl ring-1 ring-default bg-elevated/30 p-4 flex flex-col md:flex-row md:items-center gap-3 min-w-0 mx-1 mb-1">
        <div class="flex-1 min-w-0">
          <h2 class="text-lg font-medium text-highlighted">Итоговый тест</h2>
          <p class="text-sm text-muted">
            <template v-if="version?.finalTest">
              Вопросов: {{ version.finalTest.questionCount ?? 0 }}
              <template v-if="version.requireFinalTest"> · обязателен</template>
            </template>
            <template v-else>Ещё не создан</template>
          </p>
        </div>
        <UButton
          color="primary"
          variant="soft"
          icon="i-lucide-clipboard-check"
          :to="{ name: 'admin-course-final-test', params: { courseId } }"
        >
          {{ version?.finalTest ? 'Редактировать' : 'Создать' }}
        </UButton>
      </section>
    </template>

    <UModal v-model:open="deleteOpen" title="Удалить тему?">
      <template #body>
        <p class="text-sm text-muted">
          Тема «{{ deleteTarget?.title }}» будет удалена вместе с материалами. Это действие нельзя отменить.
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-trash-2" @click="confirmDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>

      <UModal v-model:open="unpublishOpen" title="Снять публикацию курса?">
        <template #body>
          <p class="text-sm text-muted">
            Курс станет «черновиком». Публикацию можно будет снова включить позже.
          </p>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton color="neutral" variant="ghost" @click="unpublishOpen = false">Отмена</UButton>
            <UButton color="primary" icon="i-lucide-x" :loading="unpublishing" @click="confirmUnpublish">
              Снять
            </UButton>
          </div>
        </template>
      </UModal>
  </UMain>
</template>
