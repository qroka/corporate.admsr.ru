<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import CourseStatusBadge from '../components/CourseStatusBadge.vue';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const enrollmentId = computed(() => Number(route.params.enrollmentId));
const topicId = computed(() => Number(route.params.topicId));
const loading = ref(true);
const data = ref<any>(null);
const activeMaterialId = ref<number | null>(null);
const lastActivityAt = ref(Date.now());
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Мои курсы', to: { name: 'courses' } },
  {
    label: 'Курс',
    to: { name: 'course-enrollment', params: { enrollmentId: enrollmentId.value } },
  },
  { label: data.value?.topic?.title || 'Тема' },
]);

const materials = computed(() => data.value?.topic?.materials || data.value?.materials || []);
const topicTest = computed(() => data.value?.topic?.topicTest || data.value?.topic?.testLink || data.value?.testLink);

function onActivity() {
  lastActivityAt.value = Date.now();
}

function isPageActive() {
  return document.visibilityState === 'visible' && document.hasFocus();
}

async function tickHeartbeat() {
  if (!activeMaterialId.value || !isPageActive()) return;
  // считаем активным, если было взаимодействие за последние 30с
  if (Date.now() - lastActivityAt.value > 30_000) return;
  try {
    await store.heartbeat({
      enrollmentId: enrollmentId.value,
      materialId: activeMaterialId.value,
      seconds: 15,
    });
  } catch {
    /* ignore transient */
  }
}

onMounted(async () => {
  loading.value = true;
  try {
    data.value = await store.getTopic(enrollmentId.value, topicId.value);
  } catch (e: any) {
    toast.add({ title: 'Тема недоступна', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
  window.addEventListener('mousemove', onActivity);
  window.addEventListener('keydown', onActivity);
  window.addEventListener('scroll', onActivity, true);
  window.addEventListener('click', onActivity);
  heartbeatTimer = setInterval(() => void tickHeartbeat(), 15_000);
});

onUnmounted(() => {
  if (heartbeatTimer) clearInterval(heartbeatTimer);
  window.removeEventListener('mousemove', onActivity);
  window.removeEventListener('keydown', onActivity);
  window.removeEventListener('scroll', onActivity, true);
  window.removeEventListener('click', onActivity);
});

async function openMaterial(m: any) {
  try {
    await store.openMaterial(enrollmentId.value, m.id);
    activeMaterialId.value = m.id;
    lastActivityAt.value = Date.now();
    if (m.type === 'link' && m.externalUrl) {
      window.open(m.externalUrl, '_blank', 'noopener');
    } else if (m.fileUrl) {
      window.open(m.fileUrl, '_blank', 'noopener');
    }
  } catch (e: any) {
    toast.add({ title: 'Не удалось открыть', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  }
}

async function completeMaterial(m: any) {
  try {
    await store.completeMaterial(enrollmentId.value, m.id);
    toast.add({ title: 'Материал отмечен', color: 'success', icon: 'i-lucide-check' });
    data.value = await store.getTopic(enrollmentId.value, topicId.value);
  } catch (e: any) {
    toast.add({ title: 'Не удалось завершить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  }
}

function matStatus(m: any) {
  return m.progress?.status || 'not_started';
}

function goTest() {
  if (!topicTest.value?.id) return;
  router.push({
    name: 'course-test',
    params: {
      enrollmentId: enrollmentId.value,
      courseTestLinkId: topicTest.value.id,
    },
  });
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-xl" />
    </div>

    <template v-else-if="data">
      <h1 class="text-2xl font-medium text-highlighted">
        {{ data.topic?.title || 'Тема' }}
      </h1>
      <p v-if="data.topic?.description" class="text-sm text-muted">
        {{ data.topic.description }}
      </p>

      <section class="flex flex-col gap-2">
        <h2 class="text-lg font-medium">Материалы</h2>
        <UEmpty
          v-if="!materials.length"
          icon="i-lucide-file"
          title="Нет материалов"
          class="py-8"
        />
        <ul v-else class="flex flex-col gap-2 list-none p-0 m-0">
          <li
            v-for="m in materials"
            :key="m.id"
            class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-3"
            :class="activeMaterialId === m.id ? 'ring-primary' : ''"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="font-medium">{{ m.title }}</p>
                <CourseStatusBadge :status="matStatus(m)" class="mt-1" />
              </div>
            </div>

            <div
              v-if="m.type === 'rich_text' && m.contentHtml && activeMaterialId === m.id"
              class="prose prose-sm dark:prose-invert max-w-none text-sm rounded-lg bg-elevated/40 p-3"
              v-html="m.contentHtml"
            />

            <div class="flex flex-wrap gap-2">
              <UButton
                color="primary"
                variant="soft"
                size="sm"
                icon="i-lucide-book-open"
                @click="openMaterial(m)"
              >
                Открыть
              </UButton>
              <UButton
                v-if="matStatus(m) !== 'completed'"
                color="neutral"
                variant="outline"
                size="sm"
                icon="i-lucide-check"
                @click="completeMaterial(m)"
              >
                Отметить изученным
              </UButton>
            </div>
          </li>
        </ul>
      </section>

      <UButton
        v-if="topicTest"
        color="primary"
        size="lg"
        class="w-fit"
        icon="i-lucide-clipboard-check"
        @click="goTest"
      >
        Пройти тест темы
      </UButton>
    </template>
  </UMain>
</template>
