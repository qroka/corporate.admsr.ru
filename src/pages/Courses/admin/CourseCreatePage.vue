<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useSectionAccess } from '../../../composables/useSectionAccess';

const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const { allowedCourseCategoryItems, ensureLoaded } = useSectionAccess();
ensureLoaded();
const saving = ref(false);

const form = reactive({
  title: '',
  category: '' as string,
  shortDescription: '',
  fullDescription: '',
  sequentialProgress: true,
});

const crumbs: BreadcrumbItem[] = [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: 'Новый курс' },
];

async function onSave() {
  if (!form.title.trim()) {
    toast.add({ title: 'Укажите название', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  if (!form.category) {
    toast.add({ title: 'Выберите категорию', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  saving.value = true;
  try {
    const created = await store.createCourse({
      title: form.title.trim(),
      category: form.category,
      shortDescription: form.shortDescription,
      fullDescription: form.fullDescription,
      sequentialProgress: form.sequentialProgress,
    }) as any;
    const id = created?.course?.id ?? created?.id;
    if (!id) throw new Error('Сервер не вернул id курса');
    toast.add({ title: 'Курс создан', color: 'success', icon: 'i-lucide-check' });
    await router.push({ name: 'admin-course-workspace', params: { courseId: String(id) } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось создать', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Новый курс</h1>

    <div class="flex flex-col gap-4">
      <UFormField label="Название" required>
        <UInput v-model="form.title" size="lg" class="w-full" placeholder="Например: Онбординг новых сотрудников" />
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
          :content="{ align: 'start', sideOffset: 8 }"
        />
      </UFormField>

      <UFormField label="Краткое описание">
        <UTextarea v-model="form.shortDescription" :rows="3" class="w-full" placeholder="Отображается в списках" />
      </UFormField>

      <UFormField label="Полное описание">
        <UTextarea v-model="form.fullDescription" :rows="6" class="w-full" placeholder="Подробности для участников" />
      </UFormField>

      <UFormField label="Последовательное прохождение">
        <USwitch v-model="form.sequentialProgress" label="Темы открываются по порядку" />
      </UFormField>

      <div class="flex items-center gap-2 pt-2">
        <UButton color="primary" size="lg" :loading="saving" icon="i-lucide-check" @click="onSave">
          Создать и открыть
        </UButton>
        <UButton color="neutral" variant="ghost" size="lg" :to="{ name: 'courses', query: { tab: 'manage' } }">
          Отмена
        </UButton>
      </div>
    </div>
  </UMain>
</template>
