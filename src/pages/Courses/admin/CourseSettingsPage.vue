<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useSectionAccess } from '../../../composables/useSectionAccess';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const { allowedCourseCategoryItems, ensureLoaded } = useSectionAccess();
ensureLoaded();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const saving = ref(false);

const form = reactive({
  title: '',
  category: '' as string,
  shortDescription: '',
  sequentialProgress: true,
  requireFinalTest: true,
});

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Настройки' },
]);

const isEditable = computed(() => store.version.value?.status !== 'archived');

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
      form.sequentialProgress = v.sequentialProgress !== false;
      form.requireFinalTest = v.requireFinalTest !== false;
    }
  } catch (e: any) {
    toast.add({ title: 'Ошибка загрузки', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onSave() {
  if (!isEditable.value) {
    toast.add({ title: 'Архивированную версию нельзя редактировать', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  if (!form.category) {
    toast.add({ title: 'Выберите категорию', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  saving.value = true;
  try {
    await store.updateCourse({
      courseId: courseId.value,
      title: form.title.trim(),
      category: form.category,
      shortDescription: form.shortDescription,
      sequentialProgress: form.sequentialProgress,
      requireFinalTest: form.requireFinalTest,
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
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Настройки курса</h1>

    <UAlert
      v-if="!loading && !isEditable"
      color="warning"
      variant="subtle"
      icon="i-lucide-lock"
      title="В архиве"
      description="Изменение настроек доступно только для черновика/публикации."
    />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 5" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-4">
      <UFormField label="Название" required>
        <UInput v-model="form.title" size="lg" class="w-full" :disabled="!isEditable" />
      </UFormField>
      <UFormField label="Категория" required>
        <USelectMenu
          v-model="form.category"
          :items="allowedCourseCategoryItems"
          value-key="value"
          label-key="label"
          placeholder="Выберите категорию"
          size="lg"
          color="neutral"
          :search-input="false"
          class="w-full"
          :disabled="!isEditable"
          :content="{ align: 'start', sideOffset: 8 }"
        />
      </UFormField>
      <UFormField label="Краткое описание">
        <UTextarea v-model="form.shortDescription" :rows="3" class="w-full" :disabled="!isEditable" />
      </UFormField>
      <UFormField label="Последовательное прохождение">
        <USwitch v-model="form.sequentialProgress" label="Темы открываются по порядку" :disabled="!isEditable" />
      </UFormField>
      <UFormField label="Итоговый тест">
        <USwitch v-model="form.requireFinalTest" label="Требовать итоговый тест" :disabled="!isEditable" />
      </UFormField>

      <div class="flex gap-2 pt-2">
        <UButton color="primary" size="lg" :loading="saving" :disabled="!isEditable" icon="i-lucide-check" @click="onSave">
          Сохранить
        </UButton>
        <UButton color="neutral" variant="ghost" size="lg" :to="{ name: 'admin-course-workspace', params: { courseId } }">
          Назад
        </UButton>
      </div>
    </div>
  </UMain>
</template>
