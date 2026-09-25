<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { z } from 'zod';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useSectionAccess } from '../../../composables/useSectionAccess';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const { allowedCourseCategoryItems, ensureLoaded } = useSectionAccess();
ensureLoaded();
useAdminCoursePortalBreadcrumbs();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);

/** Те же правила, что и при создании — иначе одно поле можно обойти через другой экран. */
const schema = z.object({
  title: z.string().trim().min(1, 'Укажите название обучения'),
  category: z.string().min(1, 'Выберите категорию'),
  shortDescription: z.string().trim().max(300, 'Не длиннее 300 символов'),
});

const form = reactive({
  title: '',
  category: '',
  shortDescription: '',
  sequentialProgress: true,
  requireFinalTest: true,
  generateCertificate: false,
});

const isEditable = computed(() => store.version.value?.status !== 'archived');

async function load() {
  loading.value = true;
  loadError.value = null;
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
      form.generateCertificate = v.generateCertificate === true;
    }
  } catch (e: any) {
    loadError.value = e?.message || 'Не удалось загрузить курс';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

async function onSubmit() {
  if (!isEditable.value) {
    toast.add({
      title: 'Архивированную версию нельзя редактировать',
      color: 'warning',
      icon: 'i-lucide-alert-triangle',
    });
    return;
  }
  saving.value = true;
  try {
    await store.updateCourse({
      courseId: courseId.value,
      title: form.title.trim(),
      category: form.category,
      shortDescription: form.shortDescription.trim(),
      sequentialProgress: form.sequentialProgress,
      requireFinalTest: form.requireFinalTest,
      generateCertificate: form.generateCertificate,
    });
    toast.add({ title: 'Настройки сохранены', color: 'success', icon: 'i-lucide-check' });
    await router.push({ name: 'admin-course-workspace', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось сохранить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        title="Настройки курса"
        description="Название, категория и правила прохождения"
      />

      <div v-if="loading" class="flex flex-col gap-4">
        <USkeleton class="h-16 w-full rounded-lg" />
        <USkeleton class="h-16 w-full rounded-lg" />
        <USkeleton class="h-24 w-full rounded-lg" />
        <USkeleton class="h-12 w-2/3 rounded-lg" />
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
          <UButton color="neutral" variant="ghost" :to="{ name: 'admin-courses' }">К списку</UButton>
        </template>
      </UAlert>

      <template v-else>
        <UAlert
          v-if="!isEditable"
          color="warning"
          variant="subtle"
          icon="i-lucide-lock"
          title="Версия в архиве"
          description="Настройки можно менять только у черновика или опубликованной версии."
        />

        <UForm :schema="schema" :state="form" class="flex flex-col gap-4" @submit="onSubmit">
          <UFormField label="Название" name="title" required>
            <UInput v-model="form.title" size="lg" class="w-full" :disabled="!isEditable" />
          </UFormField>

          <UFormField label="Категория" name="category" required>
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

          <UFormField
            label="Краткое описание"
            name="shortDescription"
            hint="Необязательно"
            description="Одна-две строки — их видно в списке обучения."
          >
            <UTextarea
              v-model="form.shortDescription"
              :rows="3"
              :maxlength="300"
              class="w-full"
              :disabled="!isEditable"
              placeholder="О чём этот курс и кому он нужен"
            />
          </UFormField>

          <UFormField
            label="Последовательное прохождение"
            name="sequentialProgress"
            description="Выключите, если темы можно изучать в любом порядке."
          >
            <USwitch
              v-model="form.sequentialProgress"
              label="Темы открываются по порядку"
              :disabled="!isEditable"
            />
          </UFormField>

          <UFormField
            label="Итоговый тест"
            name="requireFinalTest"
            description="Без сданного итогового теста курс не будет засчитан."
          >
            <USwitch v-model="form.requireFinalTest" label="Требовать итоговый тест" :disabled="!isEditable" />
          </UFormField>

          <UFormField
            label="Сертификат"
            name="generateCertificate"
            description="Сотрудник сможет скачать его на экране итогов."
          >
            <USwitch
              v-model="form.generateCertificate"
              label="Формировать сертификат после прохождения"
              :disabled="!isEditable"
            />
          </UFormField>

          <div class="flex gap-2 pt-1">
            <UButton
              type="submit"
              color="primary"
              size="lg"
              :loading="saving"
              :disabled="!isEditable"
              icon="i-lucide-check"
            >
              Сохранить
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
        </UForm>
      </template>
    </div>
  </UMain>
</template>
