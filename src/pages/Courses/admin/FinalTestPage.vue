<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { createEmptyForm, type TestForm } from '../../Tests/testForm';
import type { TestSettingsModel } from '../../Tests/components/TestSettingsForm.vue';
import CourseTestEditor from '../components/CourseTestEditor.vue';

const route = useRoute();
const store = useCoursesStore();
const { toast } = useAppToast();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const saving = ref(false);
const linkId = ref<number | null>(null);
const form = reactive<TestForm>(createEmptyForm());
form.kind = 'test';
form.visibility = 'private';
form.title = 'Итоговый тест';

const settings = reactive<TestSettingsModel>({
  usePassingScore: false,
  passingScore: 70,
  limitAttempts: false,
  attempts: 1,
  useTimeLimit: false,
  timeLimit: '',
  showCorrectAnswers: false,
});

function syncSettingsFromForm() {
  settings.usePassingScore = form.usePassingScore;
  settings.passingScore = form.passingScore;
  settings.limitAttempts = form.limitAttempts;
  settings.attempts = form.attempts;
  settings.useTimeLimit = form.useTimeLimit;
  settings.timeLimit = form.timeLimit;
  settings.showCorrectAnswers = form.showCorrectAnswers;
}

function applySettingsToForm() {
  Object.assign(form, settings);
}

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Итоговый тест' },
]);

function applyForm(src: any) {
  Object.assign(form, createEmptyForm(), src, {
    kind: 'test',
    visibility: 'private',
    questions: Array.isArray(src?.questions) ? src.questions : [],
  });
  syncSettingsFromForm();
}

onMounted(async () => {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
    const versionId = store.version.value?.id;
    try {
      const data = await store.getCourseTest({ versionId, type: 'final', courseId: courseId.value }) as any;
      if (data?.link?.id) linkId.value = Number(data.link.id);
      if (data?.form) applyForm(data.form);
    } catch {
      const created = await store.createCourseTest({
        versionId,
        courseId: courseId.value,
        type: 'final',
      }) as any;
      if (created?.link?.id) linkId.value = Number(created.link.id);
      const data = await store.getCourseTest({
        courseTestLinkId: linkId.value,
        versionId,
        type: 'final',
      }) as any;
      if (data?.form) applyForm(data.form);
    }
  } catch (e: any) {
    toast.add({ title: 'Не удалось загрузить тест', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onSave() {
  saving.value = true;
  try {
    applySettingsToForm();
    await store.updateCourseTest({
      courseTestLinkId: linkId.value,
      testFormId: form.id,
      form: { ...form },
      isRequired: true,
    });
    toast.add({ title: 'Итоговый тест сохранён', color: 'success', icon: 'i-lucide-check' });
  } catch (e: any) {
    toast.add({ title: 'Не удалось сохранить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-full min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <UBreadcrumb :items="crumbs" />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <CourseTestEditor
      v-else
      :form="form"
      :settings="settings"
      :saving="saving"
      headline="Итоговый тест"
      title-placeholder="Итоговый тест"
      @save="onSave"
    />
  </UMain>
</template>
