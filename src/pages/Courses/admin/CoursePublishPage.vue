<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { courseReadiness } from '../courseReadiness';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs();
const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const publishing = ref(false);
const loadError = ref<string | null>(null);

const title = computed(() => store.current.value?.title || 'Обучение');
const status = computed(() => String(store.version.value?.status || ''));
const readiness = computed(() => courseReadiness(store.version.value));
/** Сервер отвечает 409 на повторную публикацию и на архивную версию — не предлагаем кнопку зря. */
const isPublished = computed(() => status.value === 'published');
const isArchived = computed(() => status.value === 'archived');
const checksDone = computed(() => readiness.value.checks.filter((c) => c.ok).length);

async function load() {
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

onMounted(load);

async function onPublish() {
  if (!readiness.value.ready || isPublished.value || isArchived.value) return;
  publishing.value = true;
  try {
    await store.publishCourse(courseId.value);
    toast.add({
      title: 'Курс опубликован',
      description: 'Теперь его можно назначать сотрудникам.',
      color: 'success',
      icon: 'i-lucide-check',
    });
    await router.push({ name: 'admin-course-assign', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось опубликовать', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    publishing.value = false;
  }
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        title="Публикация"
        :description="loading || loadError ? undefined : title"
      />

      <div v-if="loading" class="flex flex-col gap-4" aria-busy="true" aria-label="Загрузка курса">
        <USkeleton class="h-16 w-full rounded-panel" />
        <USkeleton class="h-40 w-full rounded-panel" />
        <USkeleton class="h-10 w-48 rounded-lg" />
      </div>

      <UAlert
        v-else-if="loadError"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Не удалось загрузить курс"
        :description="loadError"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="load">Повторить</UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'admin-courses' }">К списку обучения</UButton>
        </template>
      </UAlert>

      <!-- Уже опубликован: публиковать повторно нечего, следующий шаг — назначение -->
      <UAlert
        v-else-if="isPublished"
        color="success"
        variant="subtle"
        icon="i-lucide-circle-check"
        title="Курс уже опубликован"
        description="Его можно назначать сотрудникам. Правки тем и материалов опубликованного курса сразу видят те, кто его проходит."
      >
        <template #actions>
          <UButton color="primary" icon="i-lucide-user-plus" :to="{ name: 'admin-course-assign', params: { courseId } }">
            Назначить
          </UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'admin-course-workspace', params: { courseId } }">
            К курсу
          </UButton>
        </template>
      </UAlert>

      <UAlert
        v-else-if="isArchived"
        color="neutral"
        variant="subtle"
        icon="i-lucide-archive"
        title="Версия в архиве"
        description="Архивную версию опубликовать нельзя."
      >
        <template #actions>
          <UButton color="neutral" variant="soft" :to="{ name: 'admin-course-workspace', params: { courseId } }">
            К курсу
          </UButton>
        </template>
      </UAlert>

      <template v-else>
        <section class="rounded-panel bg-elevated p-4 sm:p-5 flex flex-col gap-4" aria-labelledby="publish-check-title">
          <div class="flex items-center justify-between gap-2 flex-wrap">
            <h2 id="publish-check-title" class="text-lg font-bold leading-7 text-highlighted">Проверка перед публикацией</h2>
            <UBadge :color="readiness.ready ? 'success' : 'warning'" variant="subtle">
              {{ checksDone }} из {{ readiness.checks.length }}
            </UBadge>
          </div>

          <ul class="flex flex-col gap-2 list-none p-0 m-0">
            <li v-for="check in readiness.checks" :key="check.id" class="flex items-start gap-2 text-sm">
              <UIcon
                :name="check.ok ? 'i-lucide-circle-check' : 'i-lucide-circle'"
                class="size-4 mt-0.5 shrink-0"
                :class="check.ok ? 'text-success' : 'text-dimmed'"
                aria-hidden="true"
              />
              <span :class="check.ok ? 'text-muted' : 'text-highlighted'">{{ check.label }}</span>
            </li>
          </ul>

          <ul v-if="readiness.errors.length" class="flex flex-col gap-1.5 list-none p-0 m-0">
            <li
              v-for="item in readiness.errors"
              :key="item"
              class="rounded-lg ring-1 ring-inset ring-error/30 bg-error/5 px-3 py-2 text-sm text-default flex items-start gap-2"
            >
              <UIcon name="i-lucide-circle-x" class="size-4 mt-0.5 shrink-0 text-error" aria-hidden="true" />
              <span class="break-words">{{ item }}</span>
            </li>
          </ul>

          <ul v-if="readiness.warnings.length" class="flex flex-col gap-1.5 list-none p-0 m-0">
            <li
              v-for="item in readiness.warnings"
              :key="item"
              class="rounded-lg ring-1 ring-inset ring-warning/30 bg-warning/5 px-3 py-2 text-sm text-muted flex items-start gap-2"
            >
              <UIcon name="i-lucide-triangle-alert" class="size-4 mt-0.5 shrink-0 text-warning" aria-hidden="true" />
              <span class="break-words">{{ item }}</span>
            </li>
          </ul>
        </section>

        <p class="text-sm text-muted">
          <template v-if="readiness.ready">
            После публикации курс можно назначать сотрудникам. Дальнейшие правки тем и материалов
            сразу увидят те, кто его уже проходит.
          </template>
          <template v-else>
            Публикация станет доступна, когда все пункты выше будут выполнены.
          </template>
        </p>

        <div class="flex gap-2 flex-wrap">
          <UButton
            color="primary"
            size="lg"
            icon="i-lucide-send"
            :loading="publishing"
            :disabled="!readiness.ready"
            @click="onPublish"
          >
            Опубликовать
          </UButton>
          <UButton
            color="neutral"
            variant="ghost"
            size="lg"
            :to="{ name: 'admin-course-workspace', params: { courseId } }"
          >
            {{ readiness.ready ? 'Назад к курсу' : 'Исправить в курсе' }}
          </UButton>
        </div>
      </template>
    </div>
  </UMain>
</template>
