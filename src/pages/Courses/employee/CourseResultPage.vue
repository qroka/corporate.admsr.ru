<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useBreadcrumbCurrentLabel } from '../../../composables/usePortalNavigation';
import { openCourseCertificatePrint, downloadCourseCertificatePdf } from '../certificatePdf';
import { formatDateTime } from '../courseDeadline';

const route = useRoute();
const store = useCoursesStore();
const { toast } = useAppToast();
const breadcrumbLabel = useBreadcrumbCurrentLabel();
const enrollmentId = computed(() => Number(route.params.enrollmentId));
const loading = ref(true);
const loadError = ref<string | null>(null);
const payload = ref<any>(null);
const downloading = ref(false);
const printing = ref(false);

const completion = computed(() => payload.value?.completion || null);
const snapshot = computed(() => completion.value?.resultSnapshot || {});

const courseTitle = computed(
  () =>
    String(
      payload.value?.course?.title ||
        completion.value?.courseTitle ||
        snapshot.value?.courseTitle ||
        '',
    ).trim() || 'Обучение',
);

const userFio = computed(
  () =>
    String(completion.value?.userFio || snapshot.value?.userFio || '').trim() || 'Сотрудник',
);

const ofoName = computed(() => {
  const v = completion.value?.ofoName ?? snapshot.value?.ofoName;
  return v ? String(v) : '';
});

const passed = computed(() => completion.value?.passed !== false);
const finalScore = computed(() => {
  const raw = completion.value?.finalScore ?? snapshot.value?.finalScore;
  if (raw == null || raw === '') return null;
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
});
const completedAt = computed(
  () => completion.value?.completedAt || snapshot.value?.completedAt || null,
);
const completedAtLabel = computed(() => formatDateTime(completedAt.value));
const generateCertificate = computed(() => payload.value?.generateCertificate === true);
const requireFinalTest = computed(() => payload.value?.requireFinalTest === true);

const title = computed(() => 'Итоги');

watch(
  title,
  (t) => {
    breadcrumbLabel.value = t.trim() || null;
  },
  { immediate: true },
);

onUnmounted(() => {
  breadcrumbLabel.value = null;
});

async function load() {
  loading.value = true;
  loadError.value = null;
  try {
    payload.value = await store.loadResult(enrollmentId.value);
  } catch (e: any) {
    // Ошибка остаётся на экране в UAlert — тост поверх дублировал бы её.
    payload.value = null;
    loadError.value = e?.message || 'Результат недоступен';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function certificatePayload() {
  return {
    courseTitle: courseTitle.value,
    userFio: userFio.value,
    completedAt: completedAt.value,
    ofoName: ofoName.value || null,
    finalScore: requireFinalTest.value ? finalScore.value : null,
    passed: passed.value,
  };
}

async function onDownloadCertificate() {
  downloading.value = true;
  try {
    await downloadCourseCertificatePdf(certificatePayload());
    toast.add({
      title: 'Сертификат скачан',
      color: 'success',
      icon: 'i-lucide-check',
    });
  } catch (e: any) {
    toast.add({
      title: 'Не удалось скачать PDF',
      description: e?.message || 'Попробуйте ещё раз',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    downloading.value = false;
  }
}

async function onPrintCertificate() {
  printing.value = true;
  try {
    await openCourseCertificatePrint(certificatePayload());
  } catch (e: any) {
    toast.add({
      title: 'Не удалось открыть печать',
      description: e?.message || 'Попробуйте ещё раз',
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    printing.value = false;
  }
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-2xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
    <UPageHeader headline="Обучение" :title="title" :description="completion ? courseTitle : undefined" />

    <div v-if="loading" class="flex flex-col gap-4" aria-busy="true" aria-label="Загрузка итогов">
      <USkeleton class="h-56 w-full rounded-panel" />
      <USkeleton class="h-10 w-72 rounded-lg" />
    </div>

    <template v-else-if="completion">
      <div class="min-w-0 flex flex-col gap-4">
        <div
          v-if="generateCertificate && passed"
          class="rounded-panel ring-1 ring-inset ring-primary/30 bg-elevated p-6 flex flex-col gap-4 min-w-0"
        >
          <div class="flex flex-col gap-1">
            <p class="text-xs uppercase tracking-[0.2em] text-muted">Сертификат о прохождении</p>
            <p class="text-sm text-muted">Настоящим подтверждается, что</p>
            <p class="text-xl font-medium text-highlighted break-words">{{ userFio }}</p>
            <p class="text-sm text-muted">успешно прошёл(а) обучение</p>
            <p class="text-lg font-medium text-primary break-words">«{{ courseTitle }}»</p>
          </div>
          <div class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-muted">
            <p v-if="completedAtLabel">
              Дата: {{ completedAtLabel }}
            </p>
            <p v-if="ofoName">Подразделение: {{ ofoName }}</p>
            <p v-if="requireFinalTest && finalScore != null">Результат: {{ Math.round(finalScore) }}%</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <UButton
              color="primary"
              icon="i-lucide-file-down"
              :loading="downloading"
              :disabled="printing"
              @click="onDownloadCertificate"
            >
              Скачать PDF
            </UButton>
            <UButton
              color="neutral"
              variant="outline"
              icon="i-lucide-printer"
              :loading="printing"
              :disabled="downloading"
              @click="onPrintCertificate"
            >
              Печать
            </UButton>
          </div>
        </div>

        <div v-else class="rounded-panel bg-elevated p-5 flex flex-col gap-3 min-w-0">
          <p v-if="requireFinalTest && finalScore != null" class="text-3xl font-medium text-highlighted">
            {{ Math.round(finalScore) }}%
          </p>
          <p class="text-sm text-muted">
            <template v-if="passed">
              Вы успешно прошли обучение «{{ courseTitle }}».
            </template>
            <template v-else>
              Обучение не сдано. Обратитесь к администратору при необходимости переназначения.
            </template>
          </p>
          <p v-if="userFio" class="text-sm text-highlighted">{{ userFio }}</p>
          <p v-if="completedAtLabel" class="text-xs text-dimmed">
            Дата: {{ completedAtLabel }}
          </p>
        </div>

        <div class="flex gap-2 flex-wrap">
          <UButton
            color="primary"
            :variant="generateCertificate && passed ? 'outline' : 'solid'"
            icon="i-lucide-book-open"
            :to="{ name: 'course-enrollment', params: { enrollmentId } }"
          >
            Смотреть материалы
          </UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'courses' }">К моему обучению</UButton>
        </div>
      </div>
    </template>

    <UAlert
      v-else
      color="warning"
      variant="subtle"
      icon="i-lucide-server"
      title="Итоги пока недоступны"
      :description="`${loadError || 'Не удалось загрузить итог прохождения.'} Если курс ещё не пройден до конца — вернитесь к нему.`"
    >
      <template #actions>
        <UButton color="warning" icon="i-lucide-rotate-ccw" @click="load">Повторить</UButton>
        <UButton color="neutral" variant="ghost" :to="{ name: 'course-enrollment', params: { enrollmentId } }">
          К курсу
        </UButton>
      </template>
    </UAlert>
    </div>
  </UMain>
</template>
