<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { hasActiveAbsence, setHasActiveAbsence } from '../../stores/absenceJournal';

import { parseLocalDateTime } from '../../utils/date';

const router = useRouter();

const loading = ref(false);
const startAt = ref<Date | null>(null);
const reason = ref('');
const nowMs = ref(Date.now());

let tickTimer: ReturnType<typeof setInterval> | null = null;
let fetchSeq = 0;

function authUserId(): string | null {
  try {
    const user = JSON.parse(localStorage.getItem('auth-user') ?? 'null');
    const id = user?.id;
    if (id == null || id === '') return null;
    return String(id);
  } catch {
    return null;
  }
}

function parseApiDate(raw: unknown): Date | null {
  return parseLocalDateTime(raw);
}

function pad2(n: number) {
  return n < 10 ? `0${n}` : String(n);
}

const elapsedParts = computed(() => {
  const start = startAt.value;
  if (!start) {
    return { days: 0, hours: '00', minutes: '00', seconds: '00', ready: false };
  }
  const ms = Math.max(0, nowMs.value - start.getTime());
  const totalSec = Math.floor(ms / 1000);
  const days = Math.floor(totalSec / 86400);
  const hours = Math.floor((totalSec % 86400) / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;
  return {
    days,
    hours: pad2(hours),
    minutes: pad2(minutes),
    seconds: pad2(seconds),
    ready: true,
  };
});

function startTick() {
  if (tickTimer) return;
  nowMs.value = Date.now();
  tickTimer = setInterval(() => {
    nowMs.value = Date.now();
  }, 1000);
}

function stopTick() {
  if (!tickTimer) return;
  clearInterval(tickTimer);
  tickTimer = null;
}

async function loadActiveAbsence() {
  if (!hasActiveAbsence.value) {
    startAt.value = null;
    reason.value = '';
    stopTick();
    return;
  }

  const uid = authUserId();
  if (!uid) return;

  const seq = ++fetchSeq;
  loading.value = true;
  try {
    const params = new URLSearchParams({
      user_id: uid,
      status: 'active',
      limit: '1',
      offset: '0',
    });
    const res = await fetch(`/api/absence_journal.php?${params.toString()}`, {
      cache: 'no-store',
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const json = await res.json();
    if (seq !== fetchSeq) return;
    if (!json.success) throw new Error(json.error || 'Ошибка загрузки');

    const row = Array.isArray(json.data) ? json.data[0] : null;
    if (!row) {
      setHasActiveAbsence(false);
      startAt.value = null;
      reason.value = '';
      stopTick();
      return;
    }

    startAt.value = parseApiDate(row.start_datetime);
    reason.value = String(row.reason ?? '').trim();
    startTick();
  } catch {
    if (seq !== fetchSeq) return;
    // Флаг остаётся — блок всё равно показываем, таймер появится после успешной загрузки.
  } finally {
    if (seq === fetchSeq) loading.value = false;
  }
}

watch(hasActiveAbsence, (active) => {
  if (active) void loadActiveAbsence();
  else {
    startAt.value = null;
    reason.value = '';
    stopTick();
  }
});

onMounted(() => {
  if (hasActiveAbsence.value) void loadActiveAbsence();
});

onUnmounted(() => {
  stopTick();
  fetchSeq += 1;
});
</script>

<template>
  <UCard
    v-if="hasActiveAbsence"
    variant="soft"
    class="w-full rounded-panel"
    :ui="{
      root: 'rounded-panel bg-warning/10 ring-1 ring-inset ring-warning/25 border-0 divide-y-0',
      body: 'flex flex-col gap-3 p-4 sm:p-4',
    }"
  >
    <div class="flex items-center justify-between gap-2">
      <p class="text-xs font-medium uppercase tracking-wide text-warning">
        Идёт отсутствие
      </p>
      <UIcon name="i-lucide-timer" class="size-4 text-warning" />
    </div>

    <div v-if="loading && !elapsedParts.ready" class="flex flex-col gap-2">
      <USkeleton class="h-10 w-40 rounded-lg" />
      <USkeleton class="h-4 w-full rounded" />
    </div>

    <template v-else>
      <div
        class="flex items-baseline gap-1.5 font-semibold tabular-nums tracking-tight text-highlighted"
        aria-live="polite"
        aria-atomic="true"
      >
        <template v-if="elapsedParts.days > 0">
          <span class="text-3xl leading-none">{{ elapsedParts.days }}</span>
          <span class="text-sm font-medium text-muted mr-1">д</span>
        </template>
        <span class="text-3xl leading-none">{{ elapsedParts.hours }}</span>
        <span class="text-xl text-muted leading-none">:</span>
        <span class="text-3xl leading-none">{{ elapsedParts.minutes }}</span>
        <span class="text-xl text-muted leading-none">:</span>
        <span class="text-3xl leading-none">{{ elapsedParts.seconds }}</span>
      </div>

      <p v-if="reason" class="text-sm text-toned line-clamp-3">
        {{ reason }}
      </p>
      <p v-else class="text-sm text-muted">
        Причина не указана
      </p>
    </template>

    <UButton
      label="Завершить"
      icon="i-lucide-check"
      color="warning"
      variant="solid"
      size="sm"
      block
      @click="void router.push({ name: 'absence-journal' })"
    />
  </UCard>
</template>
