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
const readiness = computed(() => courseReadiness(store.version.value));

onMounted(async () => {
  loadError.value = null;
  try {
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    loadError.value = e?.message || 'Не удалось загрузить курс';
    toast.add({ title: 'Ошибка', description: loadError.value, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onPublish() {
  if (!readiness.value.ready) return;
  publishing.value = true;
  try {
    await store.publishCourse(courseId.value);
    toast.add({ title: 'Курс опубликован', color: 'success', icon: 'i-lucide-check' });
    await router.push({ name: 'admin-course-assign', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось опубликовать', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    publishing.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <h1 class="text-2xl font-medium text-highlighted">Публикация курса</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 3" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <UAlert
      v-else-if="loadError"
      color="error"
      variant="subtle"
      icon="i-lucide-alert-circle"
      title="Курс не загрузился"
      :description="loadError"
    />

    <template v-else>
      <p class="text-sm text-muted">
        Перед публикацией «{{ title }}» проверьте, что структура курса собрана. После публикации курс можно назначать сотрудникам.
      </p>

      <UAlert
        v-if="readiness.ready"
        color="success"
        variant="subtle"
        icon="i-lucide-circle-check"
        title="Курс готов к публикации"
        description="Обязательные темы, материалы и тесты на месте."
      />
      <UAlert
        v-else
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="Публикация заблокирована"
        description="Исправьте пункты ниже и вернитесь сюда."
      />

      <ul v-if="readiness.errors.length" class="flex flex-col gap-2 list-none p-0 m-0">
        <li
          v-for="item in readiness.errors"
          :key="item"
          class="rounded-lg ring-1 ring-error/30 bg-error/5 px-3 py-2 text-sm text-default"
        >
          {{ item }}
        </li>
      </ul>

      <ul v-if="readiness.warnings.length" class="flex flex-col gap-2 list-none p-0 m-0">
        <li
          v-for="item in readiness.warnings"
          :key="item"
          class="rounded-lg ring-1 ring-warning/30 bg-warning/5 px-3 py-2 text-sm text-muted"
        >
          {{ item }}
        </li>
      </ul>

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
          Назад к курсу
        </UButton>
      </div>
    </template>
  </UMain>
</template>
