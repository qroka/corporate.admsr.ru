<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { createEmptyForm, type TestForm } from '../../Tests/testForm';
import TestRunner from '../../Tests/TestRunner.vue';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const courseTestLinkId = computed(() => Number(route.params.courseTestLinkId));

const loading = ref(true);
const form = reactive<TestForm>(createEmptyForm());
const attemptId = ref<number | null>(null);
const ready = ref(false);
const savedAnswers = ref<Record<string, unknown>>({});

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Мои курсы', to: { name: 'courses' } },
  {
    label: 'Курс',
    to: { name: 'course-enrollment', params: { enrollmentId: enrollmentId.value } },
  },
  { label: form.title || 'Тест' },
]);

onMounted(async () => {
  loading.value = true;
  try {
    const started = (await store.attemptStart({
      enrollmentId: enrollmentId.value,
      courseTestLinkId: courseTestLinkId.value,
    })) as any;

    attemptId.value = Number(started?.attemptId ?? started?.attempt?.id);
    const src = started?.form;
    if (src) {
      Object.assign(form, createEmptyForm(), src, {
        kind: 'test',
        questions: Array.isArray(src.questions) ? src.questions : [],
      });
    } else {
      const got = (await store.getCourseTest({ courseTestLinkId: courseTestLinkId.value })) as any;
      if (got?.form) {
        Object.assign(form, createEmptyForm(), got.form, {
          kind: 'test',
          questions: Array.isArray(got.form.questions) ? got.form.questions : [],
        });
      }
    }

    if (attemptId.value) {
      try {
        const existing = (await store.attemptGet({ attemptId: attemptId.value })) as any;
        if (existing?.answers && typeof existing.answers === 'object') {
          savedAnswers.value = existing.answers;
        }
      } catch {
        /* no saved answers */
      }
    }
    ready.value = true;
  } catch (e: any) {
    toast.add({ title: 'Не удалось начать тест', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onSubmit(answers: Record<string, unknown>, durationSec: number) {
  if (!attemptId.value) throw new Error('Нет попытки');
  await store.attemptSave({
    attemptId: attemptId.value,
    answers,
    enrollmentId: enrollmentId.value,
    courseTestLinkId: courseTestLinkId.value,
  });
  const result = await store.attemptFinish({
    attemptId: attemptId.value,
    answers,
    durationSec,
    enrollmentId: enrollmentId.value,
    courseTestLinkId: courseTestLinkId.value,
  });
  return result;
}

function onFinish() {
  toast.add({ title: 'Тест завершён', color: 'success', icon: 'i-lucide-check' });
  router.push({ name: 'course-enrollment', params: { enrollmentId: enrollmentId.value } });
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-3">
    <UBreadcrumb :items="crumbs" />
    <h1 class="sr-only">{{ form.title || 'Прохождение теста' }}</h1>

    <div v-if="loading" class="flex flex-col gap-3 flex-1">
      <USkeleton class="h-full min-h-64 w-full rounded-xl" />
    </div>

    <UAlert
      v-else-if="!ready"
      color="error"
      variant="subtle"
      title="Тест недоступен"
      description="Вернитесь к курсу и попробуйте снова."
    />

    <TestRunner
      v-else
      :form="form"
      :initial-answers="Object.keys(savedAnswers).length ? savedAnswers : null"
      :submit="onSubmit"
      class="flex-1 min-h-0"
      @finish="onFinish"
    />
  </UMain>
</template>
