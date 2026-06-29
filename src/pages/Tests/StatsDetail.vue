<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useUsersData } from '../../composables/useUsersData';
import { useOfoTree } from '../../composables/useOfoTree';
import { useTestsStore } from '../../composables/useTestsStore';
import { fmtDuration, type FormStats } from '../../composables/useTestStats';
import { questionTypeLabel } from './questionTypes';
import { type TestForm } from './testForm';
import StatChart, { type ChartItem } from './StatChart.vue';

const props = defineProps<{ form: TestForm }>();

const { users, ensureLoaded: ensureUsers } = useUsersData();
const { unitById, ensureLoaded: ensureOfo } = useOfoTree();
const store = useTestsStore();
ensureUsers();
ensureOfo();

const chartType = ref<'bar' | 'pie'>('bar');
const stats = ref<FormStats | null>(null);
const loading = ref(false);
const error = ref('');

async function load() {
  if (props.form.id == null) return;
  loading.value = true;
  error.value = '';
  try {
    stats.value = await store.loadStats(props.form.id);
  } catch (e) {
    error.value = String((e as Error).message || e);
  } finally {
    loading.value = false;
  }
}
watch(() => props.form.id, load, { immediate: true });

const kindLabel = computed(() =>
  ({ test: 'Тест', survey: 'Опрос', poll: 'Голосование' } as Record<string, string>)[props.form.kind] ?? props.form.kind,
);
const uniq = (a: number[]) => [...new Set(a)];
const directedOfoNames = computed(() =>
  uniq([...(props.form.directedOfo ?? []), ...(props.form.ofoIds ?? [])])
    .map((id) => unitById.value.get(id)?.name)
    .filter(Boolean) as string[],
);
const directedUserNames = computed(() =>
  uniq([...(props.form.directedUsers ?? []), ...(props.form.recipients ?? [])])
    .map((id) => users.value.find((u) => u.id === id)?.fullName)
    .filter(Boolean) as string[],
);

const byOfoItems = computed<ChartItem[]>(() => (stats.value?.byOfo ?? []).map((o) => ({ label: o.name, count: o.count, percent: o.percent })));
function optionItems(opts: { label: string; count: number; percent: number; correct?: boolean }[]): ChartItem[] {
  return opts.map((o) => ({ label: (o.correct ? '✓ ' : '') + o.label, count: o.count, percent: o.percent }));
}

// Ответы конкретного участника (для создателя; только не-анонимные формы)
const participantOpen = ref(false);
const participantName = ref('');
const participantData = ref<any>(null);
const loadingP = ref(false);
const errorP = ref('');
async function openParticipant(p: { id: number; name: string }) {
  if (props.form.id == null) return;
  participantName.value = p.name;
  participantData.value = null;
  errorP.value = '';
  participantOpen.value = true;
  loadingP.value = true;
  try { participantData.value = await store.loadParticipant(props.form.id, p.id); }
  catch (e) { errorP.value = String((e as Error).message || e); }
  finally { loadingP.value = false; }
}
</script>

<template>
  <div class="flex flex-col gap-5">
    <div v-if="loading" class="py-10 text-center text-muted text-sm">Загрузка статистики…</div>
    <div v-else-if="error" class="py-10 text-center text-error text-sm">{{ error }}</div>

    <template v-else-if="stats">
      <!-- Переключатель графиков -->
      <div class="flex items-center justify-end">
        <div class="flex items-center gap-1 rounded-lg ring-1 ring-default p-0.5">
          <UButton :color="chartType === 'bar' ? 'primary' : 'neutral'" :variant="chartType === 'bar' ? 'solid' : 'ghost'" size="xs" icon="i-lucide-bar-chart-3" @click="chartType = 'bar'">Столбцы</UButton>
          <UButton :color="chartType === 'pie' ? 'primary' : 'neutral'" :variant="chartType === 'pie' ? 'solid' : 'ghost'" size="xs" icon="i-lucide-pie-chart" @click="chartType = 'pie'">Кольцо</UButton>
        </div>
      </div>

      <!-- Метрики -->
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
        <div class="rounded-xl ring-1 ring-default p-3 bg-elevated/30">
          <p class="text-xs text-muted">Прохождений</p>
          <p class="text-2xl font-semibold text-highlighted tabular-nums">{{ stats.completions }}</p>
        </div>
        <div class="rounded-xl ring-1 ring-default p-3 bg-elevated/30">
          <p class="text-xs text-muted">Завершаемость</p>
          <p class="text-2xl font-semibold text-highlighted tabular-nums">{{ stats.completionRate }}%</p>
          <p class="text-xs text-dimmed">из {{ stats.started }} начавших</p>
        </div>
        <div class="rounded-xl ring-1 ring-default p-3 bg-elevated/30">
          <p class="text-xs text-muted">Ср. время</p>
          <p class="text-2xl font-semibold text-highlighted">{{ fmtDuration(stats.avgTimeSec) }}</p>
        </div>
        <div v-if="stats.avgScore != null" class="rounded-xl ring-1 ring-default p-3 bg-elevated/30">
          <p class="text-xs text-muted">Ср. балл</p>
          <p class="text-2xl font-semibold text-highlighted tabular-nums">{{ stats.avgScore }}%</p>
        </div>
        <div v-if="stats.passRate != null" class="rounded-xl ring-1 ring-default p-3 bg-elevated/30">
          <p class="text-xs text-muted">Проходимость</p>
          <p class="text-2xl font-semibold text-highlighted tabular-nums">{{ stats.passRate }}%</p>
        </div>
      </div>

      <!-- Доступ / кому направлялась -->
      <div class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-2">
        <p class="text-sm font-medium text-highlighted">Кому направлялась</p>
        <p v-if="form.visibility === 'public'" class="text-sm text-muted">Публичная форма — доступна всем сотрудникам.</p>
        <template v-else>
          <div v-if="directedOfoNames.length" class="flex items-start gap-2 flex-wrap">
            <span class="text-xs text-dimmed mt-1">ОФО:</span>
            <UBadge v-for="n in directedOfoNames" :key="n" color="neutral" variant="subtle">{{ n }}</UBadge>
          </div>
          <div v-if="directedUserNames.length" class="flex items-start gap-2 flex-wrap">
            <span class="text-xs text-dimmed mt-1">Лично:</span>
            <UBadge v-for="n in directedUserNames" :key="n" color="neutral" variant="subtle">{{ n }}</UBadge>
          </div>
          <p v-if="!directedOfoNames.length && !directedUserNames.length" class="text-sm text-muted">Приватная форма — пока никому не направлена.</p>
        </template>
      </div>

      <!-- По ОФО -->
      <div v-if="byOfoItems.length" class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-3">
        <p class="text-sm font-medium text-highlighted">Прохождения по ОФО</p>
        <StatChart :type="chartType" :items="byOfoItems" />
      </div>

      <!-- Участники -->
      <div class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-2">
        <p class="text-sm font-medium text-highlighted">Участники</p>
        <p v-if="stats.participants === null" class="text-sm text-muted">Анонимная форма — участники не отображаются.</p>
        <template v-else>
          <p class="text-xs text-dimmed">Прошли: {{ stats.participants.length }} · нажмите на участника, чтобы посмотреть ответы</p>
          <div v-if="stats.participants.length" class="flex flex-wrap gap-1.5 max-h-40 overflow-y-auto p-0.5">
            <template v-for="p in stats.participants" :key="p.id">
              <UButton v-if="!p.guest" color="neutral" variant="outline" size="xs" trailing-icon="i-lucide-eye" @click="openParticipant(p)">{{ p.name }}</UButton>
              <UBadge v-else color="neutral" variant="subtle">{{ p.name }}</UBadge>
            </template>
          </div>
          <p v-else class="text-sm text-muted">Пока никто не прошёл.</p>
          <p v-if="stats.guestCompletions" class="text-xs text-dimmed">По ссылке (гости): {{ stats.guestCompletions }}</p>
        </template>
      </div>

      <!-- Самый сложный вопрос -->
      <div v-if="stats.hardest" class="rounded-xl ring-1 ring-warning/40 bg-warning/5 p-3 flex items-center gap-2 text-sm">
        <UIcon name="i-lucide-trending-down" class="size-4 text-warning shrink-0" />
        <span class="text-default">Сложнее всего: «{{ stats.hardest.title }}» — верно ответили {{ stats.hardest.correctRate }}%</span>
      </div>

      <!-- Вопросы -->
      <div class="flex flex-col gap-3">
        <p class="text-sm font-medium text-highlighted">По вопросам</p>
        <div v-for="(q, i) in stats.questions" :key="q.id" class="rounded-xl ring-1 ring-default p-4 flex flex-col gap-3">
          <div class="flex items-start justify-between gap-3 flex-wrap">
            <div class="flex items-baseline gap-2 min-w-0">
              <span class="text-muted text-sm tabular-nums">{{ i + 1 }}.</span>
              <span class="text-base text-highlighted">{{ q.title || 'Без названия' }}</span>
            </div>
            <div class="flex items-center gap-1.5 shrink-0">
              <UBadge color="neutral" variant="subtle">{{ questionTypeLabel(q.type) }}</UBadge>
              <UBadge v-if="q.correctRate != null" :color="q.correctRate >= 60 ? 'success' : 'error'" variant="subtle">верно {{ q.correctRate }}%</UBadge>
            </div>
          </div>

          <p class="text-xs text-dimmed">Ответили: {{ q.answered }} · пропустили: {{ q.skipped }}</p>

          <StatChart v-if="q.options" :type="chartType" :items="optionItems(q.options)" />
          <p v-else class="text-sm text-muted">
            Свободные ответы.
            <span v-if="q.avgNumber != null"> Среднее значение: {{ q.avgNumber }}.</span>
          </p>
        </div>
      </div>
    </template>

    <!-- Ответы участника -->
    <UModal v-model:open="participantOpen" :title="`Ответы — ${participantName}`" description="" :ui="{ content: 'max-w-2xl' }">
      <template #body>
        <div class="max-h-[70vh] overflow-y-auto px-1 py-1 flex flex-col gap-3">
          <div v-if="loadingP" class="py-8 text-center text-muted text-sm">Загрузка…</div>
          <div v-else-if="errorP" class="py-8 text-center text-error text-sm">{{ errorP }}</div>
          <template v-else-if="participantData">
            <div class="flex items-center gap-2 flex-wrap">
              <UBadge v-if="participantData.attempt.score != null" :color="participantData.attempt.passed === false ? 'error' : 'success'" variant="subtle" class="tabular-nums">{{ Math.round(participantData.attempt.score) }}%</UBadge>
              <UBadge v-if="participantData.attempt.passed != null" :color="participantData.attempt.passed ? 'success' : 'error'" variant="subtle">{{ participantData.attempt.passed ? 'Тест пройден' : 'Не пройден' }}</UBadge>
              <span v-if="participantData.attempt.durationSec != null" class="text-xs text-dimmed">время: {{ fmtDuration(participantData.attempt.durationSec) }}</span>
            </div>
            <div
              v-for="(a, i) in participantData.answers"
              :key="i"
              class="rounded-xl ring-1 p-3 flex flex-col gap-1"
              :class="a.isCorrect === false ? 'ring-red-500/40 bg-red-500/5' : (a.isCorrect === true ? 'ring-green-500/40 bg-green-500/5' : 'ring-default')"
            >
              <div class="flex items-center gap-2">
                <UIcon v-if="a.isCorrect === true" name="i-lucide-check-circle-2" class="size-4 text-success shrink-0" />
                <UIcon v-else-if="a.isCorrect === false" name="i-lucide-x-circle" class="size-4 text-error shrink-0" />
                <span class="text-sm text-highlighted">{{ i + 1 }}. {{ a.title || 'Без названия' }}</span>
              </div>
              <p class="text-sm text-muted pl-6">Ответ: {{ a.userAnswer }}</p>
              <p v-if="a.isCorrect === false && a.correctAnswer" class="text-sm text-success pl-6">Правильно: {{ a.correctAnswer }}</p>
            </div>
          </template>
        </div>
      </template>
    </UModal>
  </div>
</template>
