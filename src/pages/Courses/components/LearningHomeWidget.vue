<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useCoursesStore, type EnrollmentSummary } from '../../../composables/useCoursesStore';

const emit = defineEmits<{ visible: [value: boolean] }>();

const router = useRouter();
const store = useCoursesStore();
const loading = ref(true);
const items = ref<EnrollmentSummary[]>([]);
const loaded = ref(false);

onMounted(async () => {
  loading.value = true;
  try {
    const groups = await store.loadMyCourses() as any;
    const activeStatuses = new Set(['not_started', 'in_progress', 'overdue']);
    if (groups?.overdue || groups?.active) {
      items.value = [...(groups.overdue || []), ...(groups.active || [])]
        .filter((e: EnrollmentSummary) => activeStatuses.has(e.status))
        .slice(0, 3);
    } else {
      items.value = store.myEnrollments.value
        .filter((e) => activeStatuses.has(e.status))
        .slice(0, 3);
    }
  } catch {
    items.value = [];
  } finally {
    loading.value = false;
    loaded.value = true;
  }
});

const primary = computed(() => items.value[0] ?? null);
const showWidget = computed(() => loaded.value && !loading.value && items.value.length > 0);

watch(showWidget, (v) => emit('visible', v), { immediate: true });

function open(e: EnrollmentSummary) {
  router.push({ name: 'course-enrollment', params: { enrollmentId: String(e.id) } });
}
</script>

<template>
  <UCard
    v-if="showWidget"
    variant="soft"
    class="w-full rounded-panel"
    :ui="{
      root: 'rounded-panel bg-elevated ring-0 border-0 divide-y-0',
      header: 'px-4 py-4 sm:px-4',
      body: 'flex flex-col gap-2 px-4 pb-4 pt-0 sm:px-4 sm:pb-4 sm:pt-0',
    }"
  >
    <template #header>
      <div class="flex items-center justify-between gap-1">
        <div class="flex items-center gap-1 min-w-0">
          <h2 id="learning-home-title" class="text-lg font-bold leading-7 text-highlighted truncate">
            Моё обучение
          </h2>
          <UTooltip text="Активные и просроченные назначения">
            <UButton
              type="button"
              color="neutral"
              variant="ghost"
              size="xs"
              icon="i-lucide-info"
              square
              aria-label="Об обучении"
            />
          </UTooltip>
        </div>
        <UButton
          to="/courses"
          color="neutral"
          variant="ghost"
          size="xs"
          icon="i-lucide-arrow-up-right"
          square
          aria-label="Открыть обучение"
        />
      </div>
    </template>

    <template v-if="primary">
      <button
        type="button"
        class="flex w-full flex-col gap-2 text-left rounded-lg p-2 -mx-2 hover:bg-elevated/60 transition-colors"
        @click="open(primary)"
      >
        <div class="flex flex-col gap-1 min-w-0">
          <p class="font-medium text-sm leading-5 text-highlighted min-w-0 break-words">
            {{ primary.courseTitle }}
          </p>
          <p class="text-xs font-medium leading-4 text-muted">
            Курс • {{ primary.progressPercent ?? 0 }}% завершено
          </p>
        </div>
        <UProgress
          :model-value="primary.progressPercent ?? 0"
          size="md"
          color="primary"
          :ui="{ base: 'bg-accented' }"
          :aria-label="`Завершено ${primary.topicsCompleted ?? 0} из ${primary.topicsTotal ?? 0} тем, ${primary.progressPercent ?? 0} процентов`"
        />
      </button>

      <ul v-if="items.length > 1" class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
        <li
          v-for="e in items.slice(1)"
          :key="e.id"
          class="rounded-lg bg-elevated/30 p-2.5 flex flex-col gap-1.5 cursor-pointer hover:bg-elevated/60 transition-colors min-w-0"
          @click="open(e)"
        >
          <p class="font-medium text-highlighted text-xs min-w-0 break-words">{{ e.courseTitle }}</p>
          <UProgress
            :model-value="e.progressPercent ?? 0"
            size="sm"
            color="primary"
          />
        </li>
      </ul>
    </template>
  </UCard>
</template>
