<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useBreadcrumbCurrentLabel } from '../../../composables/usePortalNavigation';
import { openCourseCertificatePrint, downloadCourseCertificatePdf } from '../certificatePdf';

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
const completedAtLabel = computed(() => {
  const raw = completedAt.value;
  if (!raw) return '';
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return '';
  const date = d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${date}, ${time}`;
});
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

onMounted(async () => {
  loadError.value = null;
  try {
    payload.value = await store.loadResult(enrollmentId.value);
  } catch (e: any) {
    loadError.value = e?.message || 'Результат недоступен';
    toast.add({
      title: 'Результат недоступен',
      description: loadError.value,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  } finally {
    loading.value = false;
  }
});

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
  <UMain class="flex flex-1 flex-col w-full max-w-2xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <h1 class="text-2xl font-medium text-highlighted break-words">{{ title }}</h1>

    <div v-if="loading" class="flex flex-col gap-3 p-1">
      <USkeleton v-for="n in 3" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <template v-else-if="completion">
      <div class="min-w-0 p-1 flex flex-col gap-4">
        <div
          v-if="generateCertificate && passed"
          class="rounded-xl ring-1 ring-primary/30 bg-elevated/40 p-6 flex flex-col gap-4 min-w-0"
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

        <div v-else class="rounded-xl ring-1 ring-default p-5 flex flex-col gap-3 min-w-0">
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
      color="error"
      variant="subtle"
      icon="i-lucide-alert-circle"
      title="Результат недоступен"
      :description="loadError || 'Не удалось загрузить итог прохождения.'"
    >
      <template #actions>
        <UButton color="neutral" variant="outline" :to="{ name: 'courses' }">К моему обучению</UButton>
      </template>
    </UAlert>
  </UMain>
</template>
