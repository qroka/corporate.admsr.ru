<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useTestsStore } from '../composables/useTestsStore';
import OfoSelect from '../components/OfoSelect.vue';
import TestRunner from './Tests/TestRunner.vue';
import { type TestForm } from './Tests/testForm';

const route = useRoute();
const router = useRouter();
const store = useTestsStore();

const token = String(route.params.token || '');
const form = ref<TestForm | null>(null);
const loading = ref(true);
const error = ref('');

const kindLabel = (k?: string) => ({ test: 'Тест', survey: 'Опрос', poll: 'Голосование' } as Record<string, string>)[k ?? ''] ?? '';

const isAuthed = computed(() => {
  try { const u = JSON.parse(localStorage.getItem('auth-user') || 'null'); return Number(u?.id ?? 0) > 0; } catch { return false; }
});
const linkAccess = computed(() => form.value?.linkAccess ?? 'any');

// ── Гость ─────────────────────────────────────────────────────────────────────
const guestOpen = ref(false);
const guestName = ref('');
const guestOfo = ref<number | null>(null);
const guestStarted = ref(false);

function respondentToken(): string {
  let t = localStorage.getItem('tests-respondent');
  if (!t) { t = (crypto as any)?.randomUUID?.() ?? `r_${Date.now()}_${Math.random().toString(36).slice(2)}`; localStorage.setItem('tests-respondent', t); }
  return t;
}

const showRunner = computed(() => !!form.value && (isAuthed.value || guestStarted.value));

const completionOpen = ref(false);
const completionText = computed(() => {
  const f = form.value;
  if (!f) return '';
  return f.completionMessage.trim() || (f.kind === 'poll' ? 'Спасибо! Ваш голос учтён.' : 'Спасибо за прохождение!');
});

async function load() {
  loading.value = true; error.value = '';
  try { form.value = await store.loadByToken(token); }
  catch (e) { error.value = String((e as Error).message || e); }
  finally { loading.value = false; }
}
onMounted(load);

function login() {
  localStorage.setItem('post-login-redirect', route.fullPath);
  router.push({ name: 'login' });
}
function startGuest() {
  guestName.value = ''; guestOfo.value = null; guestOpen.value = true;
}
function confirmGuest() {
  if (!guestName.value.trim()) return;
  guestOpen.value = false;
  guestStarted.value = true;
}

async function onSubmit(answers: Record<string, unknown>, durationSec: number) {
  const payload: any = { answers, durationSec };
  if (!isAuthed.value) {
    payload.guestName = guestName.value.trim();
    payload.guestOfoId = guestOfo.value ?? 0;
    payload.respondentToken = respondentToken();
  }
  await store.submitByToken(token, payload);
}
function onFinish() { completionOpen.value = true; }
</script>

<template>
  <div class="h-screen w-full bg-default flex flex-col">
    <div class="shrink-0 px-6 py-3 border-b border-default flex items-center gap-2">
      <UIcon name="i-lucide-clipboard-check" class="size-5 text-primary" />
      <span class="font-semibold text-highlighted">Корпоративный портал — прохождение</span>
    </div>

    <div class="flex-1 min-h-0 p-4 sm:p-6">
      <div v-if="loading" class="h-full grid place-items-center text-muted text-sm">Загрузка…</div>

      <div v-else-if="error" class="h-full grid place-items-center text-center">
        <div class="flex flex-col items-center gap-3">
          <UIcon name="i-lucide-unlink" class="size-10 text-error" />
          <p class="text-error font-medium">{{ error }}</p>
        </div>
      </div>

      <!-- Прохождение -->
      <TestRunner v-else-if="showRunner && form" :form="form" :submit="onSubmit" persist-session class="h-full" @finish="onFinish" />

      <!-- Лендинг для неавторизованного -->
      <div v-else-if="form" class="h-full grid place-items-center">
        <div class="w-full max-w-lg rounded-2xl ring-1 ring-default bg-elevated/30 p-8 flex flex-col items-center text-center gap-4">
          <UBadge color="primary" variant="subtle" size="lg">{{ kindLabel(form.kind) }}</UBadge>
          <h1 class="text-2xl font-semibold text-highlighted">{{ form.title || 'Без названия' }}</h1>
          <p v-if="form.description" class="text-muted line-clamp-4 whitespace-pre-line">{{ form.description }}</p>
          <p class="text-xs text-dimmed">Вопросов: {{ form.questions.length }}</p>

          <div class="flex flex-col sm:flex-row gap-3 mt-2 w-full sm:w-auto">
            <UButton v-if="linkAccess !== 'guest'" color="primary" size="xl" icon="i-lucide-log-in" @click="login">Войти</UButton>
            <UButton
              v-if="linkAccess !== 'authorized'"
              :color="linkAccess === 'guest' ? 'primary' : 'neutral'"
              :variant="linkAccess === 'guest' ? 'solid' : 'outline'"
              size="xl"
              icon="i-lucide-user"
              @click="startGuest"
            >
              Продолжить как гость
            </UButton>
          </div>
          <p v-if="linkAccess === 'authorized'" class="text-xs text-dimmed">Форма доступна только сотрудникам портала — войдите, чтобы пройти.</p>
        </div>
      </div>
    </div>

    <!-- Окно гостя: ФИО + ОФО -->
    <UModal v-model:open="guestOpen" title="Представьтесь" description="" :dismissible="false">
      <template #body>
        <div class="flex flex-col gap-4">
          <UFormField label="ФИО" required>
            <UInput v-model="guestName" size="lg" class="w-full" placeholder="Фамилия Имя Отчество" />
          </UFormField>
          <UFormField label="Подразделение (ОФО)">
            <OfoSelect v-model="guestOfo" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" @click="guestOpen = false">Отмена</UButton>
          <UButton color="primary" :disabled="!guestName.trim()" icon="i-lucide-arrow-right" @click="confirmGuest">Начать</UButton>
        </div>
      </template>
    </UModal>

    <!-- Благодарность -->
    <UModal v-model:open="completionOpen" :title="form?.kind === 'poll' ? 'Голос учтён' : 'Готово'" description="" :dismissible="false">
      <template #body>
        <div class="flex flex-col items-center text-center gap-3 py-2">
          <div class="size-12 rounded-full bg-success/10 flex items-center justify-center"><UIcon name="i-lucide-check" class="size-7 text-success" /></div>
          <p class="text-default whitespace-pre-line">{{ completionText }}</p>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end w-full">
          <UButton color="primary" @click="completionOpen = false">Закрыть</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
