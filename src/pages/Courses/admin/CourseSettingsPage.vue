<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
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
const saving = ref(false);

const form = reactive({
  title: '',
  category: '',
  shortDescription: '',
  fullDescription: '',
  sequentialProgress: true,
  requireFinalTest: true,
  defaultDeadlineDays: null as number | null,
});

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Настройки' },
]);

const isDraft = computed(() => store.version.value?.status === 'draft');

onMounted(async () => {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
    const c = store.current.value;
    const v = store.version.value;
    if (c) {
      form.title = c.title;
      form.category = c.category || '';
    }
    if (v) {
      form.shortDescription = v.shortDescription || '';
      form.fullDescription = v.fullDescription || '';
      form.sequentialProgress = v.sequentialProgress !== false;
      form.requireFinalTest = v.requireFinalTest !== false;
      form.defaultDeadlineDays = v.defaultDeadlineDays ?? null;
    }
  } catch (e: any) {
    toast.add({ title: 'Ошибка загрузки', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onSave() {
  if (!isDraft.value) {
    toast.add({ title: 'Только черновик можно редактировать', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  saving.value = true;
  try {
    await store.updateCourse({
      courseId: courseId.value,
      title: form.title.trim(),
      category: form.category.trim() || null,
      shortDescription: form.shortDescription,
      fullDescription: form.fullDescription,
      sequentialProgress: form.sequentialProgress,
      requireFinalTest: form.requireFinalTest,
      defaultDeadlineDays: form.defaultDeadlineDays,
    });
    toast.add({ title: 'Сохранено', color: 'success', icon: 'i-lucide-check' });
    await router.push({ name: 'admin-course-workspace', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось сохранить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Настройки курса</h1>

    <UAlert
      v-if="!loading && !isDraft"
      color="warning"
      variant="subtle"
      icon="i-lucide-lock"
      title="Опубликованная версия"
      description="Изменение настроек доступно только для черновика."
    />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 5" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-4">
      <UFormField label="Название" required>
        <UInput v-model="form.title" size="lg" class="w-full" :disabled="!isDraft" />
      </UFormField>
      <UFormField label="Категория">
        <UInput v-model="form.category" size="lg" class="w-full" :disabled="!isDraft" />
      </UFormField>
      <UFormField label="Краткое описание">
        <UTextarea v-model="form.shortDescription" :rows="3" class="w-full" :disabled="!isDraft" />
      </UFormField>
      <UFormField label="Полное описание">
        <UTextarea v-model="form.fullDescription" :rows="6" class="w-full" :disabled="!isDraft" />
      </UFormField>
      <UFormField label="Последовательное прохождение">
        <USwitch v-model="form.sequentialProgress" label="Темы открываются по порядку" :disabled="!isDraft" />
      </UFormField>
      <UFormField label="Итоговый тест">
        <USwitch v-model="form.requireFinalTest" label="Требовать итоговый тест" :disabled="!isDraft" />
      </UFormField>
      <UFormField label="Срок по умолчанию (дней)">
        <UInput v-model.number="form.defaultDeadlineDays" type="number" :min="1" size="lg" class="w-full" :disabled="!isDraft" placeholder="Например, 14" />
      </UFormField>

      <div class="flex gap-2 pt-2">
        <UButton color="primary" size="lg" :loading="saving" :disabled="!isDraft" icon="i-lucide-check" @click="onSave">
          Сохранить
        </UButton>
        <UButton color="neutral" variant="ghost" size="lg" :to="{ name: 'admin-course-workspace', params: { courseId } }">
          Назад
        </UButton>
      </div>
    </div>
  </UMain>
</template>
