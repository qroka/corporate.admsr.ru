<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import type { TimelineItem } from '@nuxt/ui';
import { useOfoData } from '../composables/useOfoData';
import { useProfileDisplay } from '../composables/useProfileDisplay';
import { useAppToast } from '../composables/useAppToast';
import {
  avatarUrlFromFilename,
  PROFILE_AVATAR_FILENAMES,
} from '../constants/profileAvatars';
import {
  defaultAvatarUrl,
  fetchProfileSnapshot,
  isOfoUnset,
  isProfileIncomplete,
  markOnboardingComplete,
  saveOnboardingProfile,
  type ProfileSnapshot,
} from '../composables/useOnboarding';

const router = useRouter();
const { success, error } = useAppToast();
const { setAvatarSrc, setDisplayName, setSubtitle } = useProfileDisplay();

const { ofoStats, officeSeats, loadDirectory, loadSeats, loading: ofoLoading, error: ofoError } = useOfoData();
void loadDirectory();
void loadSeats();

const STEPS = [
  { id: 'welcome', label: 'Приветствие', icon: 'i-lucide-sparkles', date: 'Шаг 1' },
  { id: 'work', label: 'Место работы', icon: 'i-lucide-building-2', date: 'Шаг 2' },
  { id: 'avatar', label: 'Аватар', icon: 'i-lucide-smile', date: 'Шаг 3' },
  { id: 'done', label: 'Готово', icon: 'i-lucide-check-circle', date: 'Шаг 4' },
] as const;

type StepId = (typeof STEPS)[number]['id'];

const currentStep = ref<StepId>('welcome');
const stepIndex = computed(() => STEPS.findIndex((s) => s.id === currentStep.value));

const timelineItems = computed<TimelineItem[]>(() =>
  STEPS.map((step) => ({
    value: step.id,
    title: step.label,
    date: step.date,
    icon: step.icon,
  })),
);

function onTimelineSelect(_event: Event, item: TimelineItem) {
  const id = item.value as StepId | undefined;
  if (!id) return;
  const targetIndex = STEPS.findIndex((s) => s.id === id);
  if (targetIndex >= 0 && targetIndex <= stepIndex.value) {
    currentStep.value = id;
  }
}

const profile = ref<ProfileSnapshot | null>(null);
const profileLoading = ref(true);

const form = reactive({
  ofoId: '',
  positionId: '',
  role: '',
  avatarUrl: defaultAvatarUrl(),
});

const saving = ref(false);

const portalFeatures = [
  {
    icon: 'i-lucide-newspaper',
    title: 'Новости и мероприятия',
    description: 'Актуальные события компании, анонсы и регистрация на мероприятия.',
  },
  {
    icon: 'i-lucide-calendar-off',
    title: 'Журнал отсутствия',
    description: 'Отмечайте отпуск, командировку и больничный — коллеги всегда в курсе.',
  },
  {
    icon: 'i-lucide-file-text',
    title: 'Заявки и база знаний',
    description: 'Сервисные заявки, документы и ответы на частые вопросы в одном месте.',
  },
  {
    icon: 'i-lucide-cake',
    title: 'Дни рождения',
    description: 'Поздравляйте коллег и не пропускайте важные даты.',
  },
  {
    icon: 'i-lucide-user-circle',
    title: 'Профиль',
    description: 'Настройки аккаунта, аватар и данные для корпоративных сервисов.',
  },
];

const ofoItems = computed(() =>
  ofoStats.value.map((o) => ({
    value: o.id,
    label: o.title,
  })),
);

const positionItems = computed(() => {
  const ofoId = String(form.ofoId ?? '').trim();
  if (!ofoId) return [];

  const list = officeSeats.value
    .filter((s) => String(s.ofo ?? '').trim() === ofoId)
    .map((s) => ({ value: s.id, label: s.title }));

  const byLabel = new Map<string, { value: string; label: string }>();
  for (const it of list) {
    const key = String(it.label ?? '').trim();
    if (!key) continue;
    if (!byLabel.has(key)) byLabel.set(key, { value: String(it.value ?? ''), label: key });
  }
  return [...byLabel.values()].sort((a, b) => a.label.localeCompare(b.label, 'ru'));
});

const selectedOfoLabel = computed(() =>
  ofoItems.value.find((o) => o.value === form.ofoId)?.label ?? '',
);

const canProceedFromWork = computed(
  () => Boolean(String(form.ofoId ?? '').trim()) && Boolean(String(form.positionId ?? '').trim()),
);

const canProceedFromAvatar = computed(() => Boolean(String(form.avatarUrl ?? '').trim()));

const greetingName = computed(() => {
  const p = profile.value;
  if (!p) return 'коллега';
  const namePatronymic = [p.firstname, p.lastname].filter(Boolean).join(' ');
  return namePatronymic || 'коллега';
});

watch(
  () => form.ofoId,
  () => {
    form.positionId = '';
    form.role = '';
  },
);

watch(
  () => form.positionId,
  (val) => {
    const pos = positionItems.value.find((p) => p.value === val);
    form.role = pos?.label ?? '';
  },
);

watch(
  () => [form.role, form.ofoId, officeSeats.value.length] as const,
  () => {
    if (form.positionId) return;
    const roleLabel = String(form.role ?? '').trim();
    const ofoId = String(form.ofoId ?? '').trim();
    if (!roleLabel || !ofoId) return;
    const match = positionItems.value.find((p) => p.label === roleLabel);
    if (match) form.positionId = match.value;
  },
  { immediate: true },
);

function goNext() {
  const idx = stepIndex.value;
  if (idx < 0 || idx >= STEPS.length - 1) return;
  currentStep.value = STEPS[idx + 1].id;
}

function goBack() {
  const idx = stepIndex.value;
  if (idx <= 0) return;
  currentStep.value = STEPS[idx - 1].id;
}

function onWorkNext() {
  if (!canProceedFromWork.value) {
    error('Заполните данные', 'Выберите ОФО и должность, чтобы продолжить.');
    return;
  }
  goNext();
}

function onAvatarNext() {
  if (!canProceedFromAvatar.value) {
    error('Выберите аватар', 'Нажмите на изображение, которое будет отображаться в профиле.');
    return;
  }
  goNext();
}

function selectAvatar(filename: string) {
  form.avatarUrl = avatarUrlFromFilename(filename);
}

async function finishOnboarding() {
  const p = profile.value;
  if (!p?.id) return;

  saving.value = true;
  try {
    const ok = await saveOnboardingProfile({
      id: p.id,
      ofo: form.ofoId,
      role: form.role,
      avatar_url: form.avatarUrl,
      firstname: p.firstname,
      surname: p.surname,
      lastname: p.lastname,
      phone: p.phone,
      email: p.email,
    });

    if (!ok) {
      error('Не удалось сохранить', 'Проверьте подключение и попробуйте снова.');
      return;
    }

    setAvatarSrc(form.avatarUrl);
    if (form.role) setSubtitle(form.role);
    const full = [p.surname, p.firstname].filter(Boolean).join(' ');
    if (full) setDisplayName(full);

    const raw = localStorage.getItem('auth-user');
    if (raw) {
      try {
        const user = JSON.parse(raw) as Record<string, unknown>;
        user.ofo = form.ofoId;
        user.role = form.role;
        localStorage.setItem('auth-user', JSON.stringify(user));
      } catch {
        /* ignore */
      }
    }

    markOnboardingComplete(p.id);
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new Event('ui:user-profile-updated'));
    }

    success('Добро пожаловать!', 'Профиль настроен — можно пользоваться порталом.');
    await router.replace({ name: 'home' });
  } finally {
    saving.value = false;
  }
}

onMounted(async () => {
  let userId = 0;
  try {
    const raw = localStorage.getItem('auth-user');
    const user = raw ? (JSON.parse(raw) as { id?: number }) : null;
    userId = Number(user?.id ?? 0);
  } catch {
    userId = 0;
  }

  if (!userId) {
    await router.replace({ name: 'login' });
    return;
  }

  profileLoading.value = true;
  try {
    const snap = await fetchProfileSnapshot(userId);
    if (!snap) {
      await router.replace({ name: 'login' });
      return;
    }
    if (!isProfileIncomplete(snap)) {
      markOnboardingComplete(userId);
      await router.replace({ name: 'home' });
      return;
    }

    profile.value = snap;
    if (!isOfoUnset(snap.ofo)) form.ofoId = String(snap.ofo);
    if (snap.role) form.role = snap.role;
    if (snap.avatar_url) form.avatarUrl = snap.avatar_url;
  } finally {
    profileLoading.value = false;
  }
});
</script>

<template>
  <div class="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-(--ui-bg) px-4 py-8 sm:py-11">
    <div
      class="pointer-events-none absolute inset-0 opacity-40 dark:opacity-25"
      aria-hidden="true"
      style="background-image: radial-gradient(circle at 15% 20%, color-mix(in oklab, var(--ui-color-primary-500) 35%, transparent) 0%, transparent 45%), radial-gradient(circle at 85% 75%, color-mix(in oklab, var(--ui-color-violet-500) 25%, transparent) 0%, transparent 40%);"
    />

    <div class="relative z-10 flex w-full max-w-4xl flex-col gap-6">
      <div class="flex flex-col items-center gap-2 text-center">
        <p
          class="text-[clamp(18px,3vw,28px)] leading-tight text-(--ui-text-highlighted)"
          style="font-family: 'Unbounded', sans-serif;"
        >
          <span class="font-bold">ADMSR</span>
          <span class="font-light"> | КОРПОРАТИВНЫЙ ПОРТАЛ</span>
        </p>
      </div>

      <UCard class="w-full shadow-lg ring-1 ring-default">
        <div class="flex flex-col gap-6 p-1 sm:p-2">
          <nav
            aria-label="Шаги настройки профиля"
            class="w-full px-2 sm:px-4 pt-2 pb-1"
          >
            <UTimeline
              v-model="currentStep"
              orientation="horizontal"
              :items="timelineItems"
              color="primary"
              size="2xl"
              class="w-full"
              :ui="{
                root: 'w-full flex flex-row items-start justify-between gap-0',
                item: 'flex-1 min-w-0 basis-0 flex flex-col items-center gap-3',
                container: 'w-full flex flex-row items-center justify-center',
                indicator: 'shrink-0 size-14 sm:size-16 [&_[data-slot=icon]]:size-7 sm:[&_[data-slot=icon]]:size-8',
                separator: 'flex-1 min-w-3 max-w-none mx-1.5 sm:mx-2 self-center',
                wrapper: 'w-full flex flex-col items-center text-center gap-1 px-1',
                date: 'w-full text-xs sm:text-sm text-muted tabular-nums',
                title: 'w-full text-sm sm:text-base font-medium leading-snug text-balance',
              }"
              @select="onTimelineSelect"
            />
          </nav>

          <USkeleton v-if="profileLoading" class="h-64 w-full" />

          <template v-else>
            <!-- Шаг 1: приветствие -->
            <section
              v-show="currentStep === 'welcome'"
              class="flex flex-col gap-5"
              aria-labelledby="onboarding-welcome-title"
            >
              <div class="text-center space-y-2">
                <h1
                  id="onboarding-welcome-title"
                  class="text-2xl sm:text-3xl font-semibold text-highlighted font-unbounded"
                >
                  Здравствуйте, {{ greetingName }}!
                </h1>
                <p class="text-base text-muted max-w-xl mx-auto">
                  Мы обновили корпоративный портал ADMSR. За пару шагов настроим профиль,
                  а затем покажем, чем пользоваться каждый день.
                </p>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <UCard
                  v-for="feature in portalFeatures"
                  :key="feature.title"
                  variant="subtle"
                  class="ring-1 ring-default"
                >
                  <div class="flex gap-3 p-1">
                    <div
                      class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary"
                    >
                      <UIcon :name="feature.icon" class="size-5" />
                    </div>
                    <div class="min-w-0 text-left">
                      <p class="font-medium text-highlighted text-sm">
                        {{ feature.title }}
                      </p>
                      <p class="text-xs text-muted mt-0.5 leading-relaxed">
                        {{ feature.description }}
                      </p>
                    </div>
                  </div>
                </UCard>
              </div>

              <div class="flex justify-end pt-2">
                <UButton
                  size="xl"
                  trailing-icon="i-lucide-arrow-right"
                  @click="goNext"
                >
                  Начать настройку
                </UButton>
              </div>
            </section>

            <!-- Шаг 2: ОФО и должность -->
            <section
              v-show="currentStep === 'work'"
              class="flex flex-col gap-5"
              aria-labelledby="onboarding-work-title"
            >
              <div class="text-center space-y-2">
                <h2
                  id="onboarding-work-title"
                  class="text-xl sm:text-2xl font-semibold text-highlighted font-unbounded"
                >
                  Место работы
                </h2>
                <p class="text-sm text-muted max-w-lg mx-auto">
                  Укажите ОФО и должность — они нужны для журнала отсутствия,
                  заявок и отображения вас в корпоративных сервисах.
                </p>
              </div>

              <UForm class="space-y-4 max-w-md mx-auto w-full">
                <UFormField
                  label="ОФО"
                  name="ofoId"
                  required
                  :help="ofoError ? String(ofoError) : undefined"
                >
                  <USelectMenu
                    v-model="form.ofoId"
                    :items="ofoItems"
                    value-key="value"
                    label-key="label"
                    placeholder="Выберите ОФО"
                    size="xl"
                    color="neutral"
                    class="w-full"
                    :disabled="ofoLoading"
                    :content="{ align: 'start', sideOffset: 8 }"
                  />
                </UFormField>

                <UFormField label="Должность" name="positionId" required>
                  <USelectMenu
                    v-model="form.positionId"
                    :items="positionItems"
                    value-key="value"
                    label-key="label"
                    placeholder="Выберите должность"
                    size="xl"
                    color="neutral"
                    class="w-full"
                    :disabled="!form.ofoId || ofoLoading"
                    :content="{ align: 'start', sideOffset: 8 }"
                  />
                </UFormField>
              </UForm>

              <div class="flex flex-wrap justify-between gap-3 pt-2">
                <UButton
                  size="xl"
                  color="neutral"
                  variant="outline"
                  leading-icon="i-lucide-arrow-left"
                  @click="goBack"
                >
                  Назад
                </UButton>
                <UButton
                  size="xl"
                  trailing-icon="i-lucide-arrow-right"
                  :disabled="!canProceedFromWork"
                  @click="onWorkNext"
                >
                  Далее
                </UButton>
              </div>
            </section>

            <!-- Шаг 3: аватар -->
            <section
              v-show="currentStep === 'avatar'"
              class="flex flex-col gap-5"
              aria-labelledby="onboarding-avatar-title"
            >
              <div class="text-center space-y-2">
                <h2
                  id="onboarding-avatar-title"
                  class="text-xl sm:text-2xl font-semibold text-highlighted font-unbounded"
                >
                  Выберите аватар
                </h2>
                <p class="text-sm text-muted">
                  Аватар отображается в шапке портала и в профиле.
                </p>
              </div>

              <div class="flex flex-col items-center gap-4">
                <UAvatar
                  :src="form.avatarUrl"
                  :alt="greetingName"
                  size="3xl"
                  class="size-24 sm:size-28 ring-4 ring-primary/20"
                  :ui="{ root: '!bg-elevated' }"
                />

                <div
                  class="grid grid-cols-3 sm:grid-cols-5 gap-2 w-full max-w-md"
                  role="listbox"
                  aria-label="Список аватаров"
                >
                  <UButton
                    v-for="name in PROFILE_AVATAR_FILENAMES"
                    :key="name"
                    type="button"
                    size="lg"
                    :variant="form.avatarUrl === avatarUrlFromFilename(name) ? 'solid' : 'subtle'"
                    :color="form.avatarUrl === avatarUrlFromFilename(name) ? 'primary' : 'neutral'"
                    class="aspect-square p-1"
                    :aria-selected="form.avatarUrl === avatarUrlFromFilename(name)"
                    role="option"
                    @click="selectAvatar(name)"
                  >
                    <img
                      :src="avatarUrlFromFilename(name)"
                      alt=""
                      class="w-10 h-auto mx-auto"
                    />
                  </UButton>
                </div>
              </div>

              <div class="flex flex-wrap justify-between gap-3 pt-2">
                <UButton
                  size="xl"
                  color="neutral"
                  variant="outline"
                  leading-icon="i-lucide-arrow-left"
                  @click="goBack"
                >
                  Назад
                </UButton>
                <UButton
                  size="xl"
                  trailing-icon="i-lucide-arrow-right"
                  :disabled="!canProceedFromAvatar"
                  @click="onAvatarNext"
                >
                  Далее
                </UButton>
              </div>
            </section>

            <!-- Шаг 4: завершение -->
            <section
              v-show="currentStep === 'done'"
              class="flex flex-col gap-5"
              aria-labelledby="onboarding-done-title"
            >
              <div class="text-center space-y-3">
                <div
                  class="mx-auto flex size-16 items-center justify-center rounded-full bg-primary/15 text-primary"
                >
                  <UIcon name="i-lucide-party-popper" class="size-8" />
                </div>
                <h2
                  id="onboarding-done-title"
                  class="text-xl sm:text-2xl font-semibold text-highlighted font-unbounded"
                >
                  Всё готово!
                </h2>
                <p class="text-sm text-muted max-w-md mx-auto">
                  Проверьте данные перед входом на главную страницу портала.
                </p>
              </div>

              <UCard variant="subtle" class="max-w-md mx-auto w-full ring-1 ring-default">
                <dl class="space-y-3 text-sm">
                  <div class="flex justify-between gap-4">
                    <dt class="text-muted">ОФО</dt>
                    <dd class="font-medium text-highlighted text-right">
                      {{ selectedOfoLabel || '—' }}
                    </dd>
                  </div>
                  <USeparator />
                  <div class="flex justify-between gap-4">
                    <dt class="text-muted">Должность</dt>
                    <dd class="font-medium text-highlighted text-right">
                      {{ form.role || '—' }}
                    </dd>
                  </div>
                  <USeparator />
                  <div class="flex items-center justify-between gap-4">
                    <dt class="text-muted">Аватар</dt>
                    <dd>
                      <UAvatar
                        :src="form.avatarUrl"
                        size="md"
                        :ui="{ root: '!bg-elevated' }"
                      />
                    </dd>
                  </div>
                </dl>
              </UCard>

              <div class="flex flex-wrap justify-between gap-3 pt-2">
                <UButton
                  size="xl"
                  color="neutral"
                  variant="outline"
                  leading-icon="i-lucide-arrow-left"
                  :disabled="saving"
                  @click="goBack"
                >
                  Назад
                </UButton>
                <UButton
                  size="xl"
                  icon="i-lucide-rocket"
                  :loading="saving"
                  @click="finishOnboarding"
                >
                  Перейти на главную
                </UButton>
              </div>
            </section>
          </template>
        </div>
      </UCard>
    </div>
  </div>
</template>
