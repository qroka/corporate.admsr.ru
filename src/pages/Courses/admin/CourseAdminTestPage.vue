<script setup lang="ts">
/**
 * Единый хост конструктора теста курса: тема или итоговый.
 * Режимы по route.name / наличию topicId.
 */
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { createEmptyForm, type TestForm } from '../../Tests/testForm';
import type { TestSettingsModel } from '../../Tests/components/TestSettingsForm.vue';
import CourseTestEditor from '../components/CourseTestEditor.vue';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs();

const courseId = computed(() => Number(route.params.courseId));
const topicId = computed(() => {
  const raw = route.params.topicId;
  const n = Number(Array.isArray(raw) ? raw[0] : raw);
  return Number.isFinite(n) && n > 0 ? n : null;
});
const isFinal = computed(() => !topicId.value || String(route.name || '').includes('final-test'));

const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const removing = ref(false);
const removeOpen = ref(false);
const linkId = ref<number | null>(null);
const form = reactive<TestForm>(createEmptyForm());
form.kind = 'test';
form.visibility = 'private';

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

const headline = computed(() => (isFinal.value ? 'Итоговый тест' : 'Тест темы'));
const titlePlaceholder = computed(() =>
  isFinal.value ? 'Итоговый тест' : 'Например: Проверка по теме',
);
const guideMode = computed(() => String(route.query.guide || '') === '1');

function applyForm(src: any) {
  Object.assign(form, createEmptyForm(), src, {
    kind: 'test',
    visibility: 'private',
    questions: Array.isArray(src?.questions) ? src.questions : [],
  });
  if (isFinal.value && !form.title) form.title = 'Итоговый тест';
  syncSettingsFromForm();
}

async function load() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadCourse(courseId.value);
    const versionId = store.version.value?.id;

    if (isFinal.value) {
      try {
        const data = (await store.getCourseTest({
          versionId,
          type: 'final',
          courseId: courseId.value,
        })) as any;
        if (data?.link?.id) linkId.value = Number(data.link.id);
        if (data?.form) applyForm(data.form);
      } catch {
        const created = (await store.createCourseTest({
          versionId,
          courseId: courseId.value,
          type: 'final',
        })) as any;
        if (created?.link?.id) linkId.value = Number(created.link.id);
        const data = (await store.getCourseTest({
          courseTestLinkId: linkId.value,
          versionId,
          type: 'final',
        })) as any;
        if (data?.form) applyForm(data.form);
      }
    } else {
      try {
        const data = (await store.getCourseTest({ topicId: topicId.value! })) as any;
        if (data?.link?.id) linkId.value = Number(data.link.id);
        if (data?.form) applyForm(data.form);
      } catch {
        const created = (await store.createCourseTest({ topicId: topicId.value! })) as any;
        if (created?.link?.id) linkId.value = Number(created.link.id);
        if (created?.link?.testFormId || created?.form) {
          const data = (await store.getCourseTest({
            courseTestLinkId: linkId.value,
            topicId: topicId.value!,
          })) as any;
          if (data?.form) applyForm(data.form);
        }
      }
    }
  } catch (e: any) {
    loadError.value = e?.message || 'Не удалось загрузить тест';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

async function onSave() {
  if (!isFinal.value && !linkId.value && !form.id) {
    toast.add({ title: 'Тест ещё не создан', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  saving.value = true;
  try {
    applySettingsToForm();
    await store.updateCourseTest({
      courseTestLinkId: linkId.value,
      testFormId: form.id,
      form: { ...form },
      isRequired: true,
    });
    toast.add({
      title: isFinal.value ? 'Итоговый тест сохранён' : 'Тест сохранён',
      description: guideMode.value ? 'Можно вернуться в курс и проверить готовность' : undefined,
      color: 'success',
      icon: 'i-lucide-check',
    });
    if (guideMode.value) {
      await router.push({
        name: 'admin-course-workspace',
        params: { courseId: courseId.value },
        query: { guide: '1' },
      });
    }
  } catch (e: any) {
    toast.add({
      title: 'Не удалось сохранить',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    saving.value = false;
  }
}

async function confirmRemove() {
  if (!linkId.value) return;
  removing.value = true;
  try {
    await store.deleteCourseTest({ courseTestLinkId: linkId.value });
    toast.add({
      title: isFinal.value ? 'Итоговый тест убран' : 'Тест темы убран',
      color: 'success',
      icon: 'i-lucide-check',
    });
    removeOpen.value = false;
    if (isFinal.value) {
      await router.push({
        name: 'admin-course-workspace',
        params: { courseId: courseId.value },
      });
    } else {
      await router.push({
        name: 'admin-course-topic-edit',
        params: { courseId: courseId.value, topicId: topicId.value! },
      });
    }
  } catch (e: any) {
    toast.add({
      title: 'Не удалось убрать тест',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    removing.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <UAlert
      v-else-if="loadError"
      color="warning"
      variant="subtle"
      icon="i-lucide-server"
      :title="isFinal ? 'Не удалось загрузить итоговый тест' : 'Не удалось загрузить тест темы'"
      :description="loadError"
    >
      <template #actions>
        <UButton color="warning" icon="i-lucide-rotate-ccw" @click="load">Повторить</UButton>
        <UButton
          color="neutral"
          variant="ghost"
          :to="{ name: 'admin-course-workspace', params: { courseId } }"
        >
          Вернуться к курсу
        </UButton>
      </template>
    </UAlert>

    <template v-else>
      <UAlert
        v-if="guideMode && !isFinal"
        color="primary"
        variant="subtle"
        icon="i-lucide-list-ordered"
        title="Шаг 4 из 4 — тест темы"
        description="Добавьте вопросы и сохраните. После этого откроется хаб курса с чеклистом готовности."
      />
      <CourseTestEditor
        :form="form"
        :settings="settings"
        :saving="saving"
        :removing="removing"
        :can-remove="Boolean(linkId)"
        :headline="headline"
        :title-placeholder="titlePlaceholder"
        @save="onSave"
        @remove="removeOpen = true"
      />
    </template>

    <UModal
      v-model:open="removeOpen"
      :title="isFinal ? 'Убрать итоговый тест?' : 'Убрать тест темы?'"
      description="Связь с темой будет удалена. Черновик формы без попыток тоже удалится."
    >
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="removeOpen = false">Отмена</UButton>
          <UButton
            color="error"
            icon="i-lucide-trash-2"
            :loading="removing"
            @click="confirmRemove"
          >
            Убрать
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
