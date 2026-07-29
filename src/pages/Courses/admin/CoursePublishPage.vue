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
const confirmChecked = ref(false);
const report = ref<{ ready: boolean; errors: string[]; warnings: string[] } | null>(null);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Публикация' },
]);

onMounted(async () => {
  try {
    await store.loadCourse(courseId.value);
    report.value = await store.readiness(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onPublish() {
  if (!report.value?.ready || !confirmChecked.value) return;
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
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Публикация курса</h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 3" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <template v-else>
      <UAlert
        v-if="report"
        :color="report.ready ? 'success' : 'error'"
        variant="subtle"
        :title="report.ready ? 'Готов к публикации' : 'Есть блокирующие ошибки'"
        :description="report.ready
          ? `Предупреждений: ${report.warnings.length}`
          : `Ошибок: ${report.errors.length}. Исправьте их на странице проверки.`"
      />

      <ul v-if="report?.errors.length" class="text-sm text-error list-disc pl-5">
        <li v-for="(e, i) in report.errors" :key="i">{{ e }}</li>
      </ul>

      <UCheckbox
        v-model="confirmChecked"
        :disabled="!report?.ready"
        label="Подтверждаю публикацию текущей версии. После публикации правки контента потребуют новой версии."
      />

      <div class="flex gap-2">
        <UButton
          color="primary"
          size="lg"
          icon="i-lucide-send"
          :loading="publishing"
          :disabled="!report?.ready || !confirmChecked"
          @click="onPublish"
        >
          Опубликовать
        </UButton>
        <UButton
          color="neutral"
          variant="ghost"
          size="lg"
          :to="{ name: 'admin-course-review', params: { courseId } }"
        >
          К проверке
        </UButton>
      </div>
    </template>
  </UMain>
</template>
