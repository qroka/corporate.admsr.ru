<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import draggable from 'vuedraggable';
import { useCoursesStore, type CourseTopic } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';
import { courseAuthorNextStep, courseReadiness } from '../courseReadiness';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';
import { plural } from '../courseDuration';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs({ asCurrent: true });

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const loadError = ref<string | null>(null);
const deleteOpen = ref(false);
const deleteTarget = ref<CourseTopic | null>(null);
const deleting = ref(false);
const ordering = ref(false);
const unpublishOpen = ref(false);
const unpublishing = ref(false);

const removeTestOpen = ref(false);
const removeTestTarget = ref<{ linkId: number; label: string } | null>(null);
const removingTest = ref(false);

const course = computed(() => store.current.value);
const version = computed(() => store.version.value);
const topics = computed(() => store.topics.value);
const isPublished = computed(() => version.value?.status === 'published');
const isEditable = computed(() => version.value?.status !== 'archived');
const readiness = computed(() => courseReadiness(version.value));
const authorNext = computed(() =>
  isPublished.value ? null : courseAuthorNextStep(courseId.value, version.value),
);
const guideMode = computed(() => String(route.query.guide || '') === '1');

/** Панель готовности нужна только пока курс дорабатывают. */
const showReadiness = computed(() => !isPublished.value && isEditable.value);
const checksDone = computed(() => readiness.value.checks.filter((c) => c.ok).length);
const checksTotal = computed(() => readiness.value.checks.length);

async function reload() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    loadError.value = e?.message || 'Не удалось загрузить курс';
  } finally {
    loading.value = false;
  }
}

onMounted(reload);

function topicTest(t: CourseTopic) {
  return t.topicTest || t.testLink || null;
}

function materialsCount(t: CourseTopic) {
  return Number(t.materialsCount ?? t.materials?.length ?? 0);
}

function questionsCount(t: CourseTopic) {
  return Number(t.questionCount ?? topicTest(t)?.questionCount ?? 0);
}

/** Строка меты вместо «Материалов: 0 · вопросов: 0» — читается как фраза. */
function topicMeta(t: CourseTopic) {
  const mats = materialsCount(t);
  const parts = [`${mats} ${plural(mats, ['материал', 'материала', 'материалов'])}`];
  const test = topicTest(t);
  if (!test) {
    parts.push('тест не подключён');
  } else {
    const q = questionsCount(t);
    parts.push(q ? `тест: ${q} ${plural(q, ['вопрос', 'вопроса', 'вопросов'])}` : 'тест без вопросов');
  }
  return parts.join(' · ');
}

/** Обязательная тема без материалов блокирует публикацию — помечаем прямо в строке. */
function topicWarning(t: CourseTopic) {
  if (t.isRequired === false) return null;
  if (!materialsCount(t)) return 'Нет материалов';
  if (topicTest(t) && !questionsCount(t)) return 'Тест без вопросов';
  return null;
}

/**
 * Локальная копия для перетаскивания: список переставляется сразу,
 * а на сервер уходит один запрос. Не получилось — возвращаем прежний порядок.
 */
const orderedTopics = ref<CourseTopic[]>([]);

watch(
  topics,
  (list) => {
    if (!ordering.value) orderedTopics.value = [...list];
  },
  { immediate: true },
);

async function applyOrder(next: CourseTopic[]) {
  const prev = [...orderedTopics.value];
  const versionId = version.value?.id;
  if (!versionId) return;
  if (next.length === prev.length && next.every((t, i) => t.id === prev[i]?.id)) return;

  orderedTopics.value = next;
  ordering.value = true;
  try {
    await store.orderTopics(versionId, next.map((t) => t.id));
  } catch (e: any) {
    orderedTopics.value = prev;
    ordering.value = false;
    toast.add({ title: 'Не удалось изменить порядок', description: e?.message, color: 'error', icon: 'i-lucide-x' });
    return;
  }
  ordering.value = false;
  toast.add({ title: 'Порядок тем обновлён', color: 'success', icon: 'i-lucide-check' });
  // Порядок уже сохранён — падение перезагрузки не должно откатывать список.
  try {
    await store.loadCourse(courseId.value);
  } catch {
    /* оставляем локальный порядок, он совпадает с сервером */
  }
}

/** Стрелки в меню — тот же путь, что и перетаскивание: нужны для клавиатуры. */
async function moveTopic(index: number, dir: -1 | 1) {
  const list = [...orderedTopics.value];
  const j = index + dir;
  if (j < 0 || j >= list.length) return;
  [list[index], list[j]] = [list[j], list[index]];
  await applyOrder(list);
}

function askDelete(t: CourseTopic) {
  deleteTarget.value = t;
  deleteOpen.value = true;
}

async function confirmDelete() {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await store.deleteTopic(deleteTarget.value.id);
    deleteOpen.value = false;
    deleteTarget.value = null;
    await store.loadCourse(courseId.value);
    toast.add({ title: 'Тема удалена', color: 'success', icon: 'i-lucide-check' });
  } catch (e: any) {
    toast.add({ title: 'Не удалось удалить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    deleting.value = false;
  }
}

function askRemoveTopicTest(t: CourseTopic) {
  const link = topicTest(t);
  if (!link?.id) return;
  removeTestTarget.value = { linkId: Number(link.id), label: `тест темы «${t.title}»` };
  removeTestOpen.value = true;
}

function askRemoveFinalTest() {
  const link = version.value?.finalTest;
  if (!link?.id) return;
  removeTestTarget.value = { linkId: Number(link.id), label: 'итоговый тест' };
  removeTestOpen.value = true;
}

async function confirmRemoveTest() {
  if (!removeTestTarget.value) return;
  removingTest.value = true;
  try {
    await store.deleteCourseTest({ courseTestLinkId: removeTestTarget.value.linkId });
    toast.add({ title: 'Тест убран', color: 'success', icon: 'i-lucide-check' });
    removeTestOpen.value = false;
    removeTestTarget.value = null;
    await reload();
  } catch (e: any) {
    toast.add({ title: 'Не удалось убрать тест', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    removingTest.value = false;
  }
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

/** Порядок и удаление — под ⋯, иначе в строке темы шесть контролов. */
function topicMenu(t: CourseTopic, idx: number) {
  const move = [
    {
      label: 'Переместить выше',
      icon: 'i-lucide-arrow-up',
      disabled: idx === 0 || ordering.value,
      onSelect: () => moveTopic(idx, -1),
    },
    {
      label: 'Переместить ниже',
      icon: 'i-lucide-arrow-down',
      disabled: idx === orderedTopics.value.length - 1 || ordering.value,
      onSelect: () => moveTopic(idx, 1),
    },
  ];
  const tests = [
    {
      label: topicTest(t) ? 'Редактировать тест' : 'Добавить тест',
      icon: 'i-lucide-clipboard-list',
      onSelect: () =>
        router.push({ name: 'admin-course-topic-test', params: { courseId: courseId.value, topicId: t.id } }),
    },
  ];
  const destructive: any[] = [
    { label: 'Удалить тему', icon: 'i-lucide-trash-2', color: 'error' as const, onSelect: () => askDelete(t) },
  ];
  if (topicTest(t)) {
    destructive.unshift({
      label: 'Убрать тест темы',
      icon: 'i-lucide-unlink',
      color: 'error' as const,
      onSelect: () => askRemoveTopicTest(t),
    });
  }
  return [move, tests, destructive];
}

const headerMenu = computed(() => {
  if (!isPublished.value) return [];
  return [[
    {
      label: 'Снять публикацию',
      icon: 'i-lucide-eye-off',
      onSelect: () => {
        unpublishOpen.value = true;
      },
    },
  ]];
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <div v-if="loading" class="flex flex-col gap-6">
        <USkeleton class="h-16 w-2/3 rounded-lg" />
        <div class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_320px] gap-4 items-start">
          <div class="flex flex-col gap-2">
            <USkeleton v-for="n in 3" :key="n" class="h-16 w-full rounded-panel" />
          </div>
          <USkeleton class="h-64 w-full rounded-panel" />
        </div>
      </div>

      <UAlert
        v-else-if="loadError || !course"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Не удалось загрузить курс"
        :description="loadError || 'Курс не найден — возможно, его удалили.'"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="reload">Повторить</UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'admin-courses' }">К списку обучения</UButton>
        </template>
      </UAlert>

      <template v-else>
        <UPageHeader headline="Обучение" :title="course.title" :description="version?.shortDescription || undefined">
          <template #title>
            <div class="flex items-center gap-2 flex-wrap min-w-0">
              <span class="break-words">{{ course.title }}</span>
              <CourseStatusBadge :status="version?.status" />
            </div>
          </template>
          <template #links>
            <UButton
              color="neutral"
              variant="soft"
              icon="i-lucide-settings"
              :to="{ name: 'admin-course-settings', params: { courseId } }"
            >
              Настройки
            </UButton>
            <UButton
              color="neutral"
              variant="outline"
              icon="i-lucide-bar-chart-3"
              :to="{ name: 'admin-course-results', params: { courseId } }"
            >
              Результаты
            </UButton>
            <UButton
              v-if="isPublished"
              color="primary"
              icon="i-lucide-user-plus"
              :to="{ name: 'admin-course-assign', params: { courseId } }"
            >
              Назначить
            </UButton>
            <UDropdownMenu v-if="headerMenu.length" :items="headerMenu">
              <UButton
                type="button"
                color="neutral"
                variant="ghost"
                icon="i-lucide-ellipsis"
                aria-label="Ещё действия с курсом"
              />
            </UDropdownMenu>
          </template>
        </UPageHeader>

        <div
          class="grid grid-cols-1 gap-4 items-start"
          :class="showReadiness ? 'xl:grid-cols-[minmax(0,1fr)_320px]' : ''"
        >
          <!-- Левая колонка: структура курса -->
          <div class="flex flex-col gap-5 min-w-0">
            <section class="flex flex-col gap-3 min-w-0" aria-labelledby="hub-topics-title">
              <div class="flex items-center justify-between gap-2 flex-wrap">
                <div class="flex items-center gap-2">
                  <h2 id="hub-topics-title" class="text-lg font-bold leading-7 text-highlighted">Темы</h2>
                  <UBadge v-if="topics.length" color="neutral" variant="subtle" size="sm">
                    {{ topics.length }}
                  </UBadge>
                </div>
                <UButton
                  v-if="isEditable"
                  color="primary"
                  variant="soft"
                  size="sm"
                  icon="i-lucide-plus"
                  :to="{ name: 'admin-course-topic-create', params: { courseId }, query: guideMode ? { guide: '1' } : {} }"
                >
                  Добавить тему
                </UButton>
              </div>

              <UEmpty
                v-if="!topics.length"
                variant="naked"
                icon="i-lucide-list"
                title="Тем пока нет"
                description="Тема — это раздел курса. Добавьте первую, чтобы начать наполнять обучение."
                class="py-10"
              >
                <template #actions>
                  <UButton
                    v-if="isEditable"
                    color="primary"
                    icon="i-lucide-plus"
                    :to="{ name: 'admin-course-topic-create', params: { courseId }, query: { guide: '1' } }"
                  >
                    Добавить первую тему
                  </UButton>
                </template>
              </UEmpty>

              <draggable
                v-else
                :model-value="orderedTopics"
                item-key="id"
                handle=".topic-drag-handle"
                tag="ul"
                ghost-class="opacity-40"
                :animation="150"
                :disabled="!isEditable || ordering"
                class="flex flex-col gap-2 list-none p-0 m-0 min-w-0"
                @update:model-value="applyOrder"
              >
                <template #item="{ element: t, index: idx }">
                <li
                  class="rounded-panel bg-elevated p-3 sm:px-4 flex items-center gap-2 sm:gap-3 min-w-0 transition-shadow hover:shadow-md"
                >
                  <UButton
                    v-if="isEditable"
                    type="button"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    square
                    icon="i-lucide-grip-vertical"
                    class="topic-drag-handle shrink-0 cursor-grab active:cursor-grabbing"
                    :disabled="ordering"
                    title="Перетащите, чтобы изменить порядок"
                    :aria-label="`Перетащить тему ${t.title}`"
                  />
                  <span class="text-xs text-dimmed tabular-nums shrink-0 w-5 text-right">{{ idx + 1 }}</span>

                  <div class="flex-1 min-w-0 flex flex-col gap-0.5">
                    <div class="flex items-center gap-2 flex-wrap min-w-0">
                      <RouterLink
                        :to="{ name: 'admin-course-topic-edit', params: { courseId, topicId: t.id } }"
                        class="font-medium text-highlighted break-words hover:text-primary focus:outline-none focus-visible:underline"
                      >
                        {{ t.title }}
                      </RouterLink>
                      <UBadge v-if="topicWarning(t)" color="warning" variant="subtle" size="sm">
                        {{ topicWarning(t) }}
                      </UBadge>
                    </div>
                    <p class="text-xs text-muted">{{ topicMeta(t) }}</p>
                  </div>

                  <div class="flex items-center gap-1 shrink-0">
                    <UButton
                      color="neutral"
                      variant="soft"
                      size="sm"
                      :to="{ name: 'admin-course-topic-edit', params: { courseId, topicId: t.id } }"
                    >
                      Открыть
                    </UButton>
                    <UDropdownMenu v-if="isEditable" :items="topicMenu(t, idx)">
                      <UButton
                        type="button"
                        color="neutral"
                        variant="ghost"
                        size="sm"
                        icon="i-lucide-ellipsis"
                        :aria-label="`Ещё действия с темой ${t.title}`"
                      />
                    </UDropdownMenu>
                  </div>
                </li>
                </template>
              </draggable>
            </section>

            <!-- Итоговый тест — такая же строка, как тема -->
            <section class="flex flex-col gap-3 min-w-0" aria-labelledby="hub-final-title">
              <h2 id="hub-final-title" class="text-lg font-bold leading-7 text-highlighted">Итоговый тест</h2>
              <div class="rounded-panel bg-elevated p-3 sm:px-4 flex items-center gap-3 min-w-0">
                <div class="flex-1 min-w-0 flex flex-col gap-0.5">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-medium text-highlighted">
                      {{ version?.finalTest ? 'Настроен' : 'Не создан' }}
                    </span>
                    <UBadge v-if="version?.requireFinalTest" color="neutral" variant="subtle" size="sm">
                      Обязателен
                    </UBadge>
                  </div>
                  <p class="text-xs text-muted">
                    <template v-if="version?.finalTest">
                      {{ version.finalTest.questionCount ?? 0 }}
                      {{ plural(version.finalTest.questionCount ?? 0, ['вопрос', 'вопроса', 'вопросов']) }}
                    </template>
                    <template v-else>
                      Нужен, только если включено требование итогового теста в настройках.
                    </template>
                  </p>
                </div>
                <div class="flex items-center gap-1 shrink-0">
                  <UButton
                    color="neutral"
                    variant="soft"
                    size="sm"
                    :to="{ name: 'admin-course-final-test', params: { courseId } }"
                  >
                    {{ version?.finalTest ? 'Открыть' : 'Создать' }}
                  </UButton>
                  <UButton
                    v-if="isEditable && version?.finalTest"
                    color="error"
                    variant="ghost"
                    size="sm"
                    icon="i-lucide-unlink"
                    aria-label="Убрать итоговый тест"
                    @click="askRemoveFinalTest"
                  />
                </div>
              </div>
            </section>
          </div>

          <!-- Правая колонка: готовность -->
          <aside
            v-if="showReadiness"
            class="rounded-panel bg-elevated p-4 flex flex-col gap-3 min-w-0 xl:sticky xl:top-0"
            aria-labelledby="hub-readiness-title"
          >
            <div class="flex items-center justify-between gap-2">
              <h2 id="hub-readiness-title" class="text-sm font-semibold uppercase tracking-wide text-muted">
                Готовность
              </h2>
              <UBadge :color="readiness.ready ? 'success' : 'warning'" variant="subtle" size="sm">
                {{ checksDone }} из {{ checksTotal }}
              </UBadge>
            </div>

            <ul class="flex flex-col gap-2 list-none p-0 m-0">
              <li v-for="check in readiness.checks" :key="check.id" class="flex items-start gap-2 text-sm min-w-0">
                <UIcon
                  :name="check.ok ? 'i-lucide-circle-check' : 'i-lucide-circle'"
                  class="size-4 mt-0.5 shrink-0"
                  :class="check.ok ? 'text-success' : 'text-dimmed'"
                  aria-hidden="true"
                />
                <span :class="check.ok ? 'text-muted' : 'text-highlighted'">{{ check.label }}</span>
              </li>
            </ul>

            <ul
              v-if="readiness.errors.length"
              class="flex flex-col gap-1 list-none p-0 m-0 rounded-lg bg-warning/5 ring-1 ring-warning/20 px-3 py-2"
            >
              <li v-for="err in readiness.errors" :key="err" class="text-xs text-muted flex items-start gap-1.5">
                <UIcon name="i-lucide-alert-triangle" class="size-3.5 mt-0.5 text-warning shrink-0" aria-hidden="true" />
                <span class="break-words">{{ err }}</span>
              </li>
            </ul>

            <div v-if="authorNext" class="border-t border-accented pt-3 flex flex-col gap-2">
              <p class="text-xs text-muted">Следующий шаг</p>
              <UButton
                color="primary"
                block
                :icon="readiness.ready ? 'i-lucide-send' : 'i-lucide-arrow-right'"
                :to="authorNext.to"
              >
                {{ authorNext.label }}
              </UButton>
            </div>

            <UButton
              color="primary"
              variant="outline"
              block
              icon="i-lucide-send"
              :disabled="!readiness.ready"
              :title="readiness.ready ? undefined : 'Сначала закройте пункты выше'"
              :to="readiness.ready ? { name: 'admin-course-publish', params: { courseId } } : undefined"
            >
              Опубликовать
            </UButton>
          </aside>
        </div>
      </template>
    </div>

    <UModal v-model:open="deleteOpen" title="Удалить тему?">
      <template #body>
        <p class="text-sm text-muted">
          Тема «{{ deleteTarget?.title }}» будет удалена вместе с материалами. Это действие нельзя отменить.
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-trash-2" :loading="deleting" @click="confirmDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="removeTestOpen"
      title="Убрать тест?"
      description="Тест будет отвязан от обучения. Черновик без попыток удалится."
    >
      <template #body>
        <p class="text-sm text-muted">
          Убрать <span class="text-highlighted font-medium">{{ removeTestTarget?.label || 'тест' }}</span>?
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="removeTestOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-unlink" :loading="removingTest" @click="confirmRemoveTest">
            Убрать
          </UButton>
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
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="unpublishOpen = false">Отмена</UButton>
          <UButton color="primary" icon="i-lucide-eye-off" :loading="unpublishing" @click="confirmUnpublish">
            Снять
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
