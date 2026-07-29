<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const loading = ref(true);

onMounted(async () => {
  loading.value = true;
  try {
    await store.loadList();
  } catch (e: any) {
    toast.add({ title: 'Не удалось загрузить курсы', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

function openWorkspace(id: number) {
  router.push({ name: 'admin-course-workspace', params: { courseId: String(id) } });
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <h1 class="text-2xl font-medium text-highlighted">Управление курсами</h1>
      <UButton
        color="primary"
        icon="i-lucide-plus"
        size="lg"
        :to="{ name: 'admin-course-create' }"
      >
        Создать курс
      </UButton>
    </div>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-20 w-full rounded-xl" />
    </div>

    <UEmpty
      v-else-if="!store.courses.value.length"
      icon="i-lucide-graduation-cap"
      title="Курсов пока нет"
      description="Создайте первый курс и наполните его темами и материалами."
      class="py-12"
    >
      <template #actions>
        <UButton color="primary" icon="i-lucide-plus" :to="{ name: 'admin-course-create' }">
          Создать курс
        </UButton>
      </template>
    </UEmpty>

    <div v-else class="flex-1 min-h-0 overflow-y-auto flex flex-col gap-3 p-1">
      <button
        v-for="c in store.courses.value"
        :key="c.id"
        type="button"
        class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 bg-elevated/30 text-left hover:bg-elevated/60 transition-colors cursor-pointer"
        @click="openWorkspace(c.id)"
      >
        <div class="flex-1 min-w-0 flex flex-col gap-1">
          <div class="flex items-center gap-2 flex-wrap">
            <CourseStatusBadge :status="c.status" />
            <span v-if="c.category" class="text-xs text-dimmed">{{ c.category }}</span>
            <span class="font-medium text-highlighted truncate">{{ c.title }}</span>
          </div>
          <p class="text-xs text-dimmed">
            <template v-if="c.versionNumber">Версия {{ c.versionNumber }} · </template>
            Тем: {{ c.topicsCount ?? 0 }}
            <template v-if="c.updatedAt"> · обновлён {{ new Date(c.updatedAt).toLocaleDateString('ru-RU') }}</template>
          </p>
        </div>
        <UButton
          color="neutral"
          variant="soft"
          icon="i-lucide-arrow-right"
          aria-label="Открыть рабочее пространство курса"
          @click.stop="openWorkspace(c.id)"
        >
          Открыть
        </UButton>
      </button>
    </div>
  </UMain>
</template>
