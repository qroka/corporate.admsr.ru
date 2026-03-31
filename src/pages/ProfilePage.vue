<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import type { DropdownMenuItem, TabsItem } from '@nuxt/ui';
import { useAppToast } from '../composables/useAppToast';
import { useAppConfig } from '../composables/useAppConfig';
import { useOfoData } from '../composables/useOfoData';
import { useProfileDisplay } from '../composables/useProfileDisplay';
import { currentRole, setRole } from '../stores/role';
import { NEUTRAL_COLORS, PRIMARY_COLORS } from '../composables/useUiTheme';

const { profileSaved } = useAppToast();

const { ofoStats, officeSeats, loadDirectory, loadSeats, loading: ofoLoading, error: ofoError } = useOfoData();
void loadDirectory();
void loadSeats();

const profileTabItems = [
  { label: 'Настройки аккаунта', value: 'account', slot: 'account' },
  { label: 'Компания', value: 'company', slot: 'company' },
  { label: 'Документы', value: 'documents', slot: 'documents' },
  { label: 'Оплата', value: 'billing', slot: 'billing' },
  { label: 'Уведомления', value: 'notifications', slot: 'notifications' },
] satisfies TabsItem[];

const profileTab = ref<(typeof profileTabItems)[number]['value']>('account');

const { displayName, subtitle, avatarSrc, setAvatarSrc, setDisplayName, setSubtitle } = useProfileDisplay();
const avatarPickerOpen = ref(false);
const avatarFilenames = [
  'Alien.png',
  'Clown Face.png',
  'Cold Face.png',
  'Face With Symbols On Mouth.png',
  'Face With Thermometer.png',
  'Nerd Face.png',
  'Pleading Face.png',
  'Rolling On The Floor Laughing.png',
  'Skull.png',
  'Slightly Smiling Face.png',
  'Smiling Face With Hearts.png',
  'Smiling Face With Horns.png',
  'Star Struck.png',
  'Winking Face With Tongue.png',
  'Winking Face.png',
] as const;

function avatarUrlFromFilename(filename: string): string {
  return `/img/FullPic/avatars/${encodeURIComponent(filename)}`;
}

const publicProfileUrl = `${typeof window !== 'undefined' ? window.location.origin : ''}/profile/public/demo`;

const copied = ref(false);
let copyTimer: ReturnType<typeof setTimeout> | null = null;

async function copyProfileUrl() {
  try {
    await navigator.clipboard.writeText(publicProfileUrl);
    copied.value = true;
    if (copyTimer) clearTimeout(copyTimer);
    copyTimer = setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch {
    copied.value = false;
  }
}

// --- Theme dropdown (primary / neutral) ---
const colors = PRIMARY_COLORS;
const neutrals = NEUTRAL_COLORS;
const appConfig = useAppConfig();

const themeMenuItems = computed<DropdownMenuItem[][]>(() => [
  [
    {
      label: 'Primary',
      slot: 'chip',
      chip: appConfig.ui.colors.primary,
      content: { align: 'center', collisionPadding: 16 },
      children: colors.map((c) => ({
        label: String(c),
        chip: c,
        slot: 'chip',
        checked: appConfig.ui.colors.primary === c,
        type: 'checkbox',
        onSelect: (e: Event) => {
          e.preventDefault();
          appConfig.ui.colors.primary = c;
        },
      })),
    },
    {
      label: 'Neutral',
      slot: 'chip',
      chip: appConfig.ui.colors.neutral,
      content: { align: 'end', collisionPadding: 16 },
      children: neutrals.map((c) => ({
        label: String(c),
        chip: c,
        slot: 'chip',
        type: 'checkbox',
        checked: appConfig.ui.colors.neutral === c,
        onSelect: (e: Event) => {
          e.preventDefault();
          appConfig.ui.colors.neutral = c;
        },
      })),
    },
  ],
]);

function chipColorHex(chip: string | undefined, shade: '500' | '400'): string {
  if (!chip) return '#999';
  if (typeof window === 'undefined') return '#999';
  const v = getComputedStyle(document.documentElement).getPropertyValue(`--color-${chip}-${shade}`).trim();
  return v || '#999';
}

function dropdownChip(item: any): string | undefined {
  const label = String(item?.label ?? '').trim();
  if (label === 'Primary') return appConfig.ui.colors.primary;
  if (label === 'Neutral') return appConfig.ui.colors.neutral;
  if ((colors as readonly string[]).includes(label)) return label;
  if ((neutrals as readonly string[]).includes(label)) return label;
  return undefined;
}

function onChangeCover() {
  window.alert('Здесь будет загрузка обложки профиля.');
}

function onChangeAvatar() {
  avatarPickerOpen.value = true;
}

function selectAvatar(filename: string) {
  setAvatarSrc(avatarUrlFromFilename(filename));
  avatarPickerOpen.value = false;
}

function onViewPublicProfile() {
  window.alert('Публичный профиль откроется в новой вкладке, когда будет готов маршрут.');
}

const accountForm = reactive({
  firstName: 'Иван',
  lastName: 'Петров',
  phone: '+7 (900) 000-00-00',
  email: 'ivan.petrov@example.gov.ru',
  ofoId: '',
  positionId: '',
  city: 'Москва',
  state: 'Москва',
  postcode: '101000',
  country: 'Россия',
});

const ofoItems = computed(() =>
  ofoStats.value.map((o) => ({
    value: o.id,
    label: o.title,
  })),
);

const positionItems = computed(() => {
  const ofoId = String(accountForm.ofoId ?? '').trim();
  if (!ofoId) return [];

  const list = officeSeats.value
    .filter((s) => String((s as any).ofo ?? '').trim() === ofoId)
    .map((s) => ({ value: (s as any).id, label: (s as any).title }));

  const byLabel = new Map<string, { value: string; label: string }>();
  for (const it of list) {
    const key = String(it.label ?? '').trim();
    if (!key) continue;
    if (!byLabel.has(key)) byLabel.set(key, { value: String(it.value ?? ''), label: key });
  }
  return [...byLabel.values()].sort((a, b) => a.label.localeCompare(b.label, 'ru'));
});

watch(
  () => accountForm.ofoId,
  () => {
    accountForm.positionId = '';
  },
);

const cityItems = [
  { value: 'Москва', label: 'Москва' },
  { value: 'Санкт-Петербург', label: 'Санкт-Петербург' },
  { value: 'Казань', label: 'Казань' },
];

const postcodeItems = [
  { value: '101000', label: '101000' },
  { value: '190000', label: '190000' },
  { value: '420000', label: '420000' },
];

const countryItems = [
  { value: 'Россия', label: 'Россия' },
  { value: 'Казахстан', label: 'Казахстан' },
  { value: 'Беларусь', label: 'Беларусь' },
];

function onUpdateAccount() {
  const full = `${accountForm.firstName} ${accountForm.lastName}`.trim();
  if (full) setDisplayName(full);
  const pos = positionItems.value.find((p) => p.value === accountForm.positionId);
  if (pos?.label) setSubtitle(pos.label);
  profileSaved();
}

const roleOptions = [
  { value: 'user', label: 'Пользователь' },
  { value: 'admin', label: 'Администратор' },
];

const roleLocal = computed({
  get: () => currentRole.value,
  set: (val) => setRole(val),
});

watch(currentRole, (val) => {
  if (!roleOptions.some((r) => r.value === val)) {
    setRole('user');
  }
});

onMounted(() => {
  const parts = displayName.value.trim().split(/\s+/);
  if (parts.length >= 1) accountForm.firstName = parts[0];
  if (parts.length >= 2) accountForm.lastName = parts.slice(1).join(' ');
});

</script>
<template>
  <UMain class="flex flex-col  w-full  h-full min-h-0 gap-6 mx-0 max-w-none">
    <UPageHeader class="border-none p-0 w-full max-w-none">
      <template #title>
        <h1 class="text-4xl font-normal font-unbounded">Профиль</h1>
      </template>
    </UPageHeader>

    <UContainer class="flex-1 min-h-0 overflow-y-auto max-w-none w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
      <!-- Обложка -->
      <UContainer
        class="max-w-none sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 relative h-40 sm:h-48 rounded-xl overflow-hidden ring ring-default bg-linear-to-br from-primary-500 via-primary-600 to-violet-700 dark:from-primary-600 dark:via-violet-700 dark:to-violet-900">
        <UContainer class="max-w-none absolute inset-0 opacity-30 dark:opacity-20 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0"
          style="background-image: radial-gradient(circle at 20% 50%, white 0%, transparent 45%), radial-gradient(circle at 80% 30%, white 0%, transparent 40%), radial-gradient(circle at 50% 80%, white 0%, transparent 35%);" />
        <UContainer
          class="max-w-none absolute inset-0 bg-[url('data:image/svg+xml,%3Csvg width=\\'60\\' height=\\'60\\' viewBox=\\'0 0 60 60\\' xmlns=\\'http://www.w3.org/2000/svg\\'%3E%3Cg fill=\\'none\\' fill-rule=\\'evenodd\\'%3E%3Cg fill=\\'%23ffffff\\' fill-opacity=\\'0.08\\'%3E%3Cpath d=\\'M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z\\'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E')] opacity-40" />
      </UContainer>

      <!-- Контент -->
      <UContainer class="sm:p-0 md:p-0 lg:p-0 xl:p-0 relative z-10 -mt-14 sm:-mt-16">
        <UContainer class="grid grid-cols-1 xl:grid-cols-12 gap-4 xl:gap-6 sm:p-0 md:p-0 lg:p-0 xl:p-0">
          <!-- Левая колонка: карточка профиля -->
          <UContainer class="xl:col-span-4 p-px ">
            <UCard class="overflow-visible">
              <UContainer class="flex flex-col items-center text-center pt-2 pb-2">
                <UContainer class="relative -mt-20 sm:-mt-22 mb-4">
                  <UAvatar
                    :src="avatarSrc"
                    :alt="displayName"
                    size="3xl"
                    class="size-28 sm:size-32"
                    :ui="{ root: '!bg-elevated' }"
                  />
                </UContainer>
                <h2 class="text-lg sm:text-xl font-semibold text-highlighted">
                  {{ displayName }}
                </h2>
                <p class="text-sm text-muted mt-0.5">
                  {{ subtitle }}
                </p>
              </UContainer>
              <UPopover v-model:open="avatarPickerOpen"  >
                    <UButton
                      type="button"
                      color="neutral"
                      label="Изменить аватар"
                      variant="outline"
                      size="xl"
                      class="w-full items-center justify-center"
                      icon="i-lucide-smile"
                      aria-label="Изменить аватар"
                    />

                    <template #content>
                      <UContainer class="sm:p-3 md:p-3 lg:p-3 xl:p-3 mx-0">
                        <UContainer class="grid grid-cols-3 gap-2 w-full sm:p-0 md:p-0 lg:p-0 xl:p-0">
                          <UButton
                            v-for="name in avatarFilenames"
                            :key="name"
                            size="xl"
                            variant="subtle"
                            color="neutral"
                            @click="selectAvatar(name)">
                              <img
                                :src="avatarUrlFromFilename(name)"
                                alt=""
                                class="w-12 h-auto p-0"
                              />
                          </UButton>
                        </UContainer>
                      </UContainer>
                    </template>
                  </UPopover>
            </UCard>
          </UContainer>

          <!-- Правая колонка -->
          <UContainer class="xl:col-span-8 min-w-0 p-px">
            <UCard class="overflow-visible">

              <UTabs v-model="profileTab" :items="profileTabItems" size="xl" variant="link" class="w-full">
                <template #account>
                  <UContainer class="p-4 sm:p-6 pt-6 space-y-6">
                    <UForm :state="accountForm" class="space-y-5" @submit.prevent="onUpdateAccount">
                      <UContainer
                        class="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-5 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
                        <UFormField label="Имя" name="firstName">
                          <UInput v-model="accountForm.firstName" size="xl" class="w-full" autocomplete="given-name" />
                        </UFormField>
                        <UFormField label="Фамилия" name="lastName">
                          <UInput v-model="accountForm.lastName" size="xl" class="w-full" autocomplete="family-name" />
                        </UFormField>
                        <UFormField label="Телефон" name="phone">
                          <UInput v-model="accountForm.phone" size="xl" class="w-full" type="tel" autocomplete="tel" />
                        </UFormField>
                        <UFormField label="Электронная почта" name="email">
                          <UInput v-model="accountForm.email" size="xl" class="w-full" type="email"
                            autocomplete="email" />
                        </UFormField>
                        <UFormField label="ОФО" name="ofoId" :help="ofoError ? String(ofoError) : undefined">
                          <USelectMenu
                            v-model="accountForm.ofoId"
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
                        <UFormField label="Должность" name="positionId">
                          <USelectMenu
                            v-model="accountForm.positionId"
                            :items="positionItems"
                            value-key="value"
                            label-key="label"
                            placeholder="Выберите должность"
                            size="xl"
                            color="neutral"
                            class="w-full"
                            :disabled="!accountForm.ofoId || ofoLoading"
                            :content="{ align: 'start', sideOffset: 8 }"
                          />
                        </UFormField>
                      </UContainer>

                      <USeparator />

                      <UContainer class="sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
                        <h3 class="text-sm font-semibold text-highlighted mb-2">Цвета интерфейса</h3>
                        <p class="text-sm text-muted mb-4">Выберите палитры. Настройка сохранится в браузере и
                          применится сразу.</p>

                        <UDropdownMenu :items="themeMenuItems">
                          <UButton type="button" icon="i-lucide-palette" color="neutral" variant="outline" size="xl"
                           trailing-icon="i-lucide-chevrons-up-down">
                            Кастомизация портала
                          </UButton>

                          <template #chip-leading="{ item }">
                            <div class="inline-flex items-center justify-center shrink-0 size-5">
                              <span class="rounded-full ring ring-bg bg-(--chip-light) dark:bg-(--chip-dark) size-2"
                                :style="{
                                  '--chip-light': `var(--color-${(item as any).chip}-500)`,
                                  '--chip-dark': `var(--color-${(item as any).chip}-400)`
                                }" />
                            </div>
                          </template>
                        </UDropdownMenu>
                      </UContainer>

                      <USeparator />

                      <UContainer class="flex flex-wrap items-center gap-3 pt-2 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0">
                        <UButton type="submit" color="primary" size="xl">Сохранить</UButton>
                      </UContainer>
                    </UForm>
                  </UContainer>
                </template>

                <template #company>
                  <UContainer class="p-4 sm:p-6 pt-6 text-sm text-muted">
                    Раздел «Компания» — здесь появятся реквизиты и контакты организации.
                  </UContainer>
                </template>

                <template #documents>
                  <UContainer class="p-4 sm:p-6 pt-6 text-sm text-muted">
                    Раздел «Документы» — загрузка и список файлов.
                  </UContainer>
                </template>

                <template #billing>
                  <UContainer class="p-4 sm:p-6 pt-6 text-sm text-muted">
                    Раздел «Оплата» — счета и способы оплаты.
                  </UContainer>
                </template>

                <template #notifications>
                  <UContainer class="p-4 sm:p-6 pt-6 text-sm text-muted">
                    Раздел «Уведомления» — настройки e‑mail и push.
                  </UContainer>
                </template>
              </UTabs>
            </UCard>
          </UContainer>
        </UContainer>
      </UContainer>
    </UContainer>

  </UMain>
</template>
