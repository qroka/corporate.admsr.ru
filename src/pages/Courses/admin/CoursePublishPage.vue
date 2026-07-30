<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const publishing = ref(false);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Публикация' },
]);

const title = computed(() => store.current.value?.title || 'Курс');

onMounted(async () => {
  try {
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onPublish() {
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
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Публикация курса</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 3" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <template v-else>
      <p class="text-sm text-muted">
        Опубликовать «{{ title }}»? После публикации курс можно назначать сотрудникам.
      </p>

      <div class="flex gap-2">
        <UButton
          color="primary"
          size="lg"
          icon="i-lucide-send"
          :loading="publishing"
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
          Назад
        </UButton>
      </div>
    </template>
  </UMain>
</template>
