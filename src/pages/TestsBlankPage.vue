<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import type { TabsItem } from '@nuxt/ui';
import { currentRole } from '../stores/role';
import { useTestsStore } from '../composables/useTestsStore';
import { useUsersData } from '../composables/useUsersData';
import { useAppToast } from '../composables/useAppToast';
import { cloneForm, type TestForm } from './Tests/testForm';
import TestBuilder from './Tests/TestBuilder.vue';
import TestRunner from './Tests/TestRunner.vue';
import StatsDetail from './Tests/StatsDetail.vue';
import OfoMultiSelect from '../components/OfoMultiSelect.vue';

type TabValue = 'list' | 'forme' | 'builder' | 'stats';

const tab = ref<TabValue>('list');
const isAdmin = computed(() => currentRole.value === 'admin');
const store = useTestsStore();
const { toast } = useAppToast();
onMounted(() => { store.ensureLoaded(); });

const tabItems = computed<TabsItem[]>(() => [
  { label: 'Список', value: 'list', icon: 'i-lucide-list' },
  { label: 'Тесты для меня', value: 'forme', icon: 'i-lucide-clipboard-check' },
  ...(isAdmin.value
    ? ([
        { label: 'Конструктор', value: 'builder', icon: 'i-lucide-square-pen' },
        { label: 'Статистика', value: 'stats', icon: 'i-lucide-bar-chart-3' },
      ] as TabsItem[])
    : []),
]);

watch(isAdmin, (admin) => {
  if (!admin) {
    if (tab.value === 'builder' || tab.value === 'stats') tab.value = 'list';
    listSub.value = 'all'; // у обычного пользователя только «Все формы»
  }
});

// При переходе на вкладку со списком — подтягиваем свежие данные из БД
watch(tab, (t) => {
  if (t === 'list' || t === 'forme' || t === 'stats') store.refresh().catch(() => {});
});

// Подвкладки списка
const listSub = ref<'all' | 'mine'>('all');
const listSubItems = [
  { label: 'Все формы', value: 'all', icon: 'i-lucide-list' },
  { label: 'Мои формы', value: 'mine', icon: 'i-lucide-user' },
];
const mineForms = computed(() => store.mine.value);
// Подвкладки — только у админа; обычный пользователь всегда видит «Все формы»
const currentList = computed(() => (isAdmin.value && listSub.value === 'mine' ? mineForms.value : store.published.value));

const kindLabel = (k: string) =>
  ({ test: 'Тест', survey: 'Опрос', poll: 'Голосование' } as Record<string, string>)[k] ?? k;

// ── Прохождение ───────────────────────────────────────────────────────────────
const runOpen = ref(false);
const runForm = ref<TestForm | null>(null);
function openRun(f: TestForm) {
  runForm.value = cloneForm(f);
  runOpen.value = true;
}
function copyLink(f: TestForm) {
  if (!f.accessToken) return;
  const url = `${location.origin}/t/${f.accessToken}`;
  navigator.clipboard?.writeText(url).then(
    () => toast.add({ title: 'Ссылка скопирована', description: url, color: 'success', icon: 'i-lucide-link' }),
    () => toast.add({ title: 'Ссылка формы', description: url, color: 'info', icon: 'i-lucide-link' }),
  );
}
function attemptsAllowed(f: TestForm): number {
  if (f.kind === 'poll' && f.allowRevote) return Infinity;
  return f.limitAttempts ? f.attempts : 1;
}
function canTake(f: TestForm): boolean {
  return store.hasActiveSession(f) || (f.attemptsUsed ?? 0) < attemptsAllowed(f);
}

const completionOpen = ref(false);
const completionForm = ref<TestForm | null>(null);
const completionText = computed(() => {
  const f = completionForm.value;
  if (!f) return '';
  return f.completionMessage.trim() || (f.kind === 'poll' ? 'Спасибо! Ваш голос учтён.' : 'Спасибо за прохождение!');
});
function onRunFinish() {
  // запись прохождения делает сам TestRunner (record); здесь — благодарность + обновление списков
  completionForm.value = runForm.value;
  runOpen.value = false;
  completionOpen.value = true;
  store.refresh().catch(() => {});
}

// ── Снятие с публикации ───────────────────────────────────────────────────────
async function unpublish(f: TestForm) {
  if (f.id == null) return;
  try {
    await store.unpublish(f.id);
    toast.add({ title: 'Форма убрана из списка', description: 'Перенесена в черновики (id сохранён).', color: 'info', icon: 'i-lucide-archive' });
  } catch (e) {
    toast.add({ title: 'Не удалось убрать', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' });
  }
}

// ── Направление формы ─────────────────────────────────────────────────────────
const { users, ensureLoaded: ensureUsersLoaded } = useUsersData();
ensureUsersLoaded();
const userItems = computed(() =>
  users.value.map((u) => ({ label: u.fullName, value: u.id })).sort((a, b) => a.label.localeCompare(b.label, 'ru')),
);

const directOpen = ref(false);
const directForm = ref<TestForm | null>(null);
const directMode = ref<'ofo' | 'users'>('ofo');
const directOfo = ref<number[]>([]);
const directUsers = ref<number[]>([]);

function openDirect(f: TestForm) {
  directForm.value = f;
  directMode.value = 'ofo';
  directOfo.value = [];
  directUsers.value = [];
  directOpen.value = true;
}
const directSelection = computed(() => (directMode.value === 'ofo' ? directOfo.value : directUsers.value));

const warnOpen = ref(false);
const warnText = ref('');
const pending = ref<{ mode: 'ofo' | 'users'; ids: number[] } | null>(null);

function buildWarn(mode: 'ofo' | 'users', n: number): string {
  if (mode === 'ofo') return `${n === 1 ? 'В этот ОФО' : 'В эти ОФО'} уже направлена данная форма. Направить её повторно?`;
  return `${n === 1 ? 'Этому пользователю' : 'Этим пользователям'} уже направлена данная форма. Направить её повторно?`;
}
async function submitDirect() {
  if (directForm.value?.id == null) return;
  const mode = directMode.value;
  const ids = mode === 'ofo' ? directOfo.value : directUsers.value;
  if (!ids.length) return;
  try {
    const r = await store.addDirections(directForm.value.id, mode, ids, false);
    if (r?.needConfirm) {
      pending.value = { mode, ids };
      warnText.value = buildWarn(mode, (r.already ?? []).length);
      directOpen.value = false;
      warnOpen.value = true;
    } else {
      directOpen.value = false;
      toast.add({ title: 'Форма направлена', color: 'success', icon: 'i-lucide-send' });
    }
  } catch (e) {
    toast.add({ title: 'Не удалось направить', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' });
  }
}
async function confirmWarn() {
  if (!pending.value || directForm.value?.id == null) return;
  try {
    await store.addDirections(directForm.value.id, pending.value.mode, pending.value.ids, true);
    warnOpen.value = false;
    pending.value = null;
    toast.add({ title: 'Форма направлена', color: 'success', icon: 'i-lucide-send' });
  } catch (e) {
    toast.add({ title: 'Не удалось направить', description: String((e as Error).message || e), color: 'error', icon: 'i-lucide-x' });
  }
}

// ── Статистика ────────────────────────────────────────────────────────────────
const statsOpen = ref(false);
const statsForm = ref<TestForm | null>(null);
function openStats(f: TestForm) {
  statsForm.value = f;
  statsOpen.value = true;
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4">
    <UTabs v-model="tab" :items="tabItems" size="xl" class="w-full" />

    <section class="flex-1 min-h-0 flex flex-col">
      <!-- Список опубликованных -->
      <div v-if="tab === 'list'" class="flex-1 min-h-0 flex flex-col gap-3">
        <UTabs v-if="isAdmin" v-model="listSub" :items="listSubItems" size="sm" class="w-fit" />

        <div class="flex-1 min-h-0 overflow-y-auto p-1">
          <UEmpty
            v-if="!currentList.length"
            :icon="listSub === 'mine' ? 'i-lucide-user' : 'i-lucide-list'"
            :title="listSub === 'mine' ? 'Вы пока ничего не публиковали' : 'Здесь пока пусто'"
            :description="listSub === 'mine' ? 'Опубликованные вами формы появятся здесь.' : 'Опубликованные формы появятся здесь и станут доступны для прохождения.'"
            class="py-12"
          />
          <div v-else class="flex flex-col gap-3">
            <div
              v-for="f in currentList"
              :key="f.id ?? 0"
              class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 bg-elevated/30"
            >
              <div class="flex-1 min-w-0 flex flex-col gap-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <UBadge color="neutral" variant="subtle" class="tabular-nums">#{{ f.listId }}</UBadge>
                  <UBadge color="primary" variant="subtle">{{ kindLabel(f.kind) }}</UBadge>
                  <UBadge v-if="f.visibility === 'private'" color="warning" variant="subtle">Приватный</UBadge>
                  <UBadge v-if="f.kind === 'survey'" :color="f.anonymous ? 'neutral' : 'warning'" variant="subtle">{{ f.anonymous ? 'Анонимный' : 'Не анонимный' }}</UBadge>
                  <span class="font-medium text-highlighted truncate">{{ f.title || 'Без названия' }}</span>
                </div>
                <p v-if="f.description" class="text-sm text-muted line-clamp-1">{{ f.description }}</p>
                <p class="text-xs text-dimmed">
                  Вопросов: {{ f.questions.length }}
                  <span v-if="listSub === 'mine' && (f.directedOfo.length || f.directedUsers.length)">
                    · направлена: ОФО {{ f.directedOfo.length }}, лично {{ f.directedUsers.length }}
                  </span>
                </p>
              </div>

              <div class="shrink-0 flex items-center gap-2 flex-wrap">
                <!-- свою форму из «Мои формы» проходить нельзя — только через «Тесты для меня» -->
                <UButton
                  v-if="listSub === 'all'"
                  color="primary"
                  :disabled="!canTake(f)"
                  :icon="store.hasActiveSession(f) ? 'i-lucide-rotate-ccw' : (canTake(f) ? 'i-lucide-play' : 'i-lucide-check')"
                  @click="openRun(f)"
                >
                  {{ store.hasActiveSession(f) ? 'Продолжить' : (canTake(f) ? 'Пройти' : 'Пройдено') }}
                </UButton>

                <template v-if="listSub === 'mine'">
                  <UButton v-if="f.accessByLink && f.accessToken" color="neutral" variant="soft" icon="i-lucide-link" @click="copyLink(f)">Ссылка</UButton>
                  <UButton color="neutral" variant="soft" icon="i-lucide-share-2" @click="openDirect(f)">Направить</UButton>
                  <UButton color="warning" variant="outline" icon="i-lucide-archive" @click="unpublish(f)">Убрать</UButton>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="tab === 'forme'" class="flex-1 min-h-0 overflow-y-auto p-1">
        <UEmpty
          v-if="!store.forMe.value.length"
          icon="i-lucide-clipboard-check"
          title="Вам пока ничего не направлено"
          description="Здесь появятся формы, которые направили лично вам или в ваш ОФО."
          class="py-12"
        />
        <div v-else class="flex flex-col gap-3">
          <div
            v-for="f in store.forMe.value"
            :key="f.id ?? 0"
            class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 bg-elevated/30"
          >
            <div class="flex-1 min-w-0 flex flex-col gap-1">
              <div class="flex items-center gap-2 flex-wrap">
                <UBadge color="neutral" variant="subtle" class="tabular-nums">#{{ f.listId }}</UBadge>
                <UBadge color="primary" variant="subtle">{{ kindLabel(f.kind) }}</UBadge>
                <UBadge v-if="f.kind === 'survey'" :color="f.anonymous ? 'neutral' : 'warning'" variant="subtle">{{ f.anonymous ? 'Анонимный' : 'Не анонимный' }}</UBadge>
                <span class="font-medium text-highlighted truncate">{{ f.title || 'Без названия' }}</span>
              </div>
              <p v-if="f.description" class="text-sm text-muted line-clamp-1">{{ f.description }}</p>
              <p class="text-xs text-dimmed">Вопросов: {{ f.questions.length }}</p>
            </div>
            <div class="shrink-0">
              <UButton
                color="primary"
                :disabled="!canTake(f)"
                :icon="store.hasActiveSession(f) ? 'i-lucide-rotate-ccw' : (canTake(f) ? 'i-lucide-play' : 'i-lucide-check')"
                @click="openRun(f)"
              >
                {{ store.hasActiveSession(f) ? 'Продолжить' : (canTake(f) ? 'Пройти' : 'Пройдено') }}
              </UButton>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="tab === 'builder' && isAdmin" class="flex-1 min-h-0 flex">
        <TestBuilder @published="tab = 'list'" />
      </div>

      <div v-else-if="tab === 'stats' && isAdmin" class="flex-1 min-h-0 overflow-y-auto p-1">
        <UEmpty
          v-if="!mineForms.length"
          icon="i-lucide-bar-chart-3"
          title="Пока нет опубликованных форм"
          description="Статистика появляется по опубликованным вами формам."
          class="py-12"
        />
        <div v-else class="flex flex-col gap-3">
          <div
            v-for="f in mineForms"
            :key="f.id ?? 0"
            class="rounded-xl ring-1 ring-default p-4 flex flex-col md:flex-row md:items-center gap-3 bg-elevated/30"
          >
            <div class="flex-1 min-w-0 flex flex-col gap-1">
              <div class="flex items-center gap-2 flex-wrap">
                <UBadge color="neutral" variant="subtle" class="tabular-nums">#{{ f.listId }}</UBadge>
                <UBadge color="primary" variant="subtle">{{ kindLabel(f.kind) }}</UBadge>
                <UBadge v-if="f.anonymous" color="neutral" variant="subtle">Анонимно</UBadge>
                <span class="font-medium text-highlighted truncate">{{ f.title || 'Без названия' }}</span>
              </div>
              <p class="text-xs text-dimmed">Вопросов: {{ f.questions.length }}</p>
            </div>
            <div class="shrink-0">
              <UButton color="neutral" variant="soft" icon="i-lucide-bar-chart-3" @click="openStats(f)">Показать подробно</UButton>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Окно прохождения -->
    <UModal
      v-model:open="runOpen"
      fullscreen
      :title="runForm?.title || 'Прохождение'"
      :ui="{ content: 'flex flex-col', body: 'flex-1 min-h-0 p-4 sm:p-6' }"
    >
      <template #body>
        <TestRunner v-if="runOpen && runForm" :form="runForm" persist-session record class="h-full" @finish="onRunFinish" />
      </template>
    </UModal>

    <!-- Сообщение после завершения -->
    <UModal v-model:open="completionOpen" :title="completionForm?.kind === 'poll' ? 'Голос учтён' : 'Готово'" description="" :dismissible="false">
      <template #body>
        <div class="flex flex-col items-center text-center gap-3 py-2">
          <div class="size-12 rounded-full bg-success/10 flex items-center justify-center">
            <UIcon name="i-lucide-check" class="size-7 text-success" />
          </div>
          <p class="text-default whitespace-pre-line">{{ completionText }}</p>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end w-full">
          <UButton color="primary" @click="completionOpen = false">Закрыть</UButton>
        </div>
      </template>
    </UModal>

    <!-- Направить для прохождения -->
    <UModal v-model:open="directOpen" title="Направить для прохождения" description="">
      <template #body>
        <div class="flex flex-col gap-4">
          <div class="flex gap-2">
            <UButton :color="directMode === 'ofo' ? 'primary' : 'neutral'" :variant="directMode === 'ofo' ? 'solid' : 'outline'" icon="i-lucide-building-2" @click="directMode = 'ofo'">Направить в ОФО</UButton>
            <UButton :color="directMode === 'users' ? 'primary' : 'neutral'" :variant="directMode === 'users' ? 'solid' : 'outline'" icon="i-lucide-user" @click="directMode = 'users'">Направить лично</UButton>
          </div>

          <OfoMultiSelect v-if="directMode === 'ofo'" v-model="directOfo" />
          <USelectMenu
            v-else
            v-model="directUsers"
            :items="userItems"
            value-key="value"
            label-key="label"
            multiple
            searchable
            size="lg"
            class="w-full"
            placeholder="Выбрать сотрудников"
            :content="{ align: 'start', sideOffset: 8 }"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" @click="directOpen = false">Отмена</UButton>
          <UButton color="primary" icon="i-lucide-send" :disabled="!directSelection.length" @click="submitDirect">Направить</UButton>
        </div>
      </template>
    </UModal>

    <!-- Повторное направление -->
    <UModal v-model:open="warnOpen" title="Повторное направление" description="">
      <template #body>
        <p class="text-default">{{ warnText }}</p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" @click="warnOpen = false">Отменить</UButton>
          <UButton color="primary" icon="i-lucide-send" @click="confirmWarn">Направить</UButton>
        </div>
      </template>
    </UModal>

    <!-- Подробная статистика -->
    <UModal
      v-model:open="statsOpen"
      :title="`Статистика — ${statsForm?.title || 'форма'}`"
      description=""
      :ui="{ content: 'max-w-3xl' }"
    >
      <template #body>
        <div class="max-h-[72vh] overflow-y-auto p-1">
          <StatsDetail v-if="statsOpen && statsForm" :form="statsForm" />
        </div>
      </template>
    </UModal>
  </UMain>
</template>
