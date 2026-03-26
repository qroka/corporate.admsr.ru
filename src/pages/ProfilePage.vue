<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-0">
    <UContainer class="shrink-0 max-w-full w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 py-4">
      <h1 class="text-2xl sm:text-3xl font-semibold text-highlighted tracking-tight">
        Профиль
      </h1>
    </UContainer>

    <UContainer
      class="flex-1 min-h-0 overflow-y-auto max-w-full w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 scrollbar-hide mx-0 pb-8"
    >
      <!-- Обложка -->
      <div
        class="relative h-40 sm:h-48 rounded-xl overflow-hidden ring ring-default bg-linear-to-br from-primary-500 via-primary-600 to-violet-700 dark:from-primary-600 dark:via-violet-700 dark:to-violet-900"
      >
        <div
          class="absolute inset-0 opacity-30 dark:opacity-20"
          style="background-image: radial-gradient(circle at 20% 50%, white 0%, transparent 45%), radial-gradient(circle at 80% 30%, white 0%, transparent 40%), radial-gradient(circle at 50% 80%, white 0%, transparent 35%);"
        />
        <div class="absolute inset-0 bg-[url('data:image/svg+xml,%3Csvg width=\'60\' height=\'60\' viewBox=\'0 0 60 60\' xmlns=\'http://www.w3.org/2000/svg\'%3E%3Cg fill=\'none\' fill-rule=\'evenodd\'%3E%3Cg fill=\'%23ffffff\' fill-opacity=\'0.08\'%3E%3Cpath d=\'M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z\'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E')] opacity-40"
        />
        <UButton
          color="neutral"
          variant="solid"
          size="sm"
          icon="i-lucide-camera"
          class="absolute top-3 inset-e-3 shadow-md bg-default/95 hover:bg-default"
          @click="onChangeCover"
        >
          Сменить обложку
        </UButton>
      </div>

      <!-- Контент: сетка под обложкой, частичный заход карточки на баннер -->
      <div class="relative z-10 -mt-14 sm:-mt-16 px-0">
        <div class="grid grid-cols-1 xl:grid-cols-12 gap-4 xl:gap-6">
          <!-- Левая колонка: карточка профиля -->
          <div class="xl:col-span-4">
            <UCard class="shadow-lg ring-0 overflow-visible">
              <div class="flex flex-col items-center text-center pt-2 pb-2">
                <div class="relative -mt-20 sm:-mt-22 mb-4">
                  <UAvatar
                    :src="avatarSrc"
                    :alt="displayName"
                    size="3xl"
                    class="ring-4 ring-default shadow-lg size-28 sm:size-32"
                  />
                  <UButton
                    type="button"
                    color="primary"
                    variant="solid"
                    size="xs"
                    square
                    icon="i-lucide-camera"
                    class="absolute bottom-1 inset-e-1 rounded-full shadow-md"
                    aria-label="Сменить фото"
                    @click="onChangeAvatar"
                  />
                </div>
                <h2 class="text-lg sm:text-xl font-semibold text-highlighted">
                  {{ displayName }}
                </h2>
                <p class="text-sm text-muted mt-0.5">
                  {{ companyName }}
                </p>
              </div>

              <USeparator class="my-4" />

              <ul class="space-y-3 px-1">
                <li class="flex items-center justify-between gap-3 text-sm">
                  <span class="text-muted">Заявок подано</span>
                  <span class="font-semibold text-warning">32</span>
                </li>
                <li class="flex items-center justify-between gap-3 text-sm">
                  <span class="text-muted">Мероприятий посещено</span>
                  <span class="font-semibold text-success">26</span>
                </li>
                <li class="flex items-center justify-between gap-3 text-sm">
                  <span class="text-muted">Активных заявок</span>
                  <span class="font-semibold text-primary">6</span>
                </li>
              </ul>

              <div class="mt-6 flex flex-col gap-3">
                <UButton
                  color="neutral"
                  variant="outline"
                  size="lg"
                  block
                  @click="onViewPublicProfile"
                >
                  Просмотр публичного профиля
                </UButton>
                <UInput
                  :model-value="publicProfileUrl"
                  readonly
                  size="lg"
                  color="neutral"
                  variant="outline"
                  class="w-full"
                  aria-label="Ссылка на профиль"
                >
                  <template #trailing>
                    <UTooltip text="Копировать ссылку">
                      <UButton
                        color="neutral"
                        variant="ghost"
                        square
                        size="sm"
                        icon="i-lucide-copy"
                        :aria-label="copied ? 'Скопировано' : 'Копировать ссылку'"
                        @click="copyProfileUrl"
                      />
                    </UTooltip>
                  </template>
                </UInput>
                <p v-if="copied" class="text-xs text-success text-center">
                  Ссылка скопирована
                </p>
              </div>
            </UCard>
          </div>

          <!-- Правая колонка: вкладки и формы -->
          <div class="xl:col-span-8 min-w-0">
            <UCard class="shadow-lg ring-0 overflow-visible">
              <UTabs
                v-model="profileTab"
                :items="profileTabItems"
                size="md"
                variant="link"
                class="w-full"
                :ui="{
                  list: 'border-b border-default gap-6 -mb-px px-4 sm:px-6 pt-2 flex-wrap',
                }"
              />

              <div class="p-4 sm:p-6 pt-6 border-t border-default">
                <div v-if="profileTab === 'account'" class="space-y-6">
                  <UForm :state="accountForm" class="space-y-5" @submit.prevent="onUpdateAccount">
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-5">
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
                        <UInput v-model="accountForm.email" size="xl" class="w-full" type="email" autocomplete="email" />
                      </UFormField>
                      <UFormField label="Город" name="city">
                        <USelectMenu
                          v-model="accountForm.city"
                          :items="cityItems"
                          size="xl"
                          class="w-full"
                          placeholder="Выберите город"
                          :content="{ sideOffset: 8 }"
                        />
                      </UFormField>
                      <UFormField label="Регион / область" name="state">
                        <UInput v-model="accountForm.state" size="xl" class="w-full" />
                      </UFormField>
                      <UFormField label="Индекс" name="postcode">
                        <USelectMenu
                          v-model="accountForm.postcode"
                          :items="postcodeItems"
                          size="xl"
                          class="w-full"
                          placeholder="Индекс"
                          :content="{ sideOffset: 8 }"
                        />
                      </UFormField>
                      <UFormField label="Страна" name="country">
                        <USelectMenu
                          v-model="accountForm.country"
                          :items="countryItems"
                          size="xl"
                          class="w-full"
                          placeholder="Страна"
                          :content="{ sideOffset: 8 }"
                        />
                      </UFormField>
                    </div>

                    <USeparator />

                    <div>
                      <h3 class="text-sm font-semibold text-highlighted mb-2">
                        Роль в интерфейсе
                      </h3>
                      <p class="text-sm text-muted mb-3">
                        Демо: переключение влияет на доступ к админ‑разделам.
                      </p>
                      <URadioGroup
                        v-model="roleLocal"
                        :items="roleOptions"
                        orientation="horizontal"
                        color="primary"
                      />
                    </div>

                    <USeparator />

                    <div>
                      <h3 class="text-sm font-semibold text-highlighted mb-2">
                        Цвета интерфейса
                      </h3>
                      <p class="text-sm text-muted mb-4">
                        Выберите палитры. Настройка сохранится в браузере и применится сразу.
                      </p>

                      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-5">
                        <UFormField label="Основной цвет (primary)" name="uiPrimary">
                          <USelectMenu
                            v-model="themePrimary"
                            :items="primaryColorItems"
                            size="xl"
                            class="w-full"
                            placeholder="Выберите цвет"
                            :content="{ sideOffset: 8 }"
                          />
                        </UFormField>

                        <UFormField label="Нейтральный (neutral)" name="uiNeutral">
                          <USelectMenu
                            v-model="themeNeutral"
                            :items="neutralColorItems"
                            size="xl"
                            class="w-full"
                            placeholder="Выберите нейтральный"
                            :content="{ sideOffset: 8 }"
                          />
                        </UFormField>
                      </div>

                      <div class="mt-4 flex items-center gap-3">
                        <div class="h-8 w-8 rounded-full bg-primary-500 ring-2 ring-default" aria-hidden="true" />
                        <div class="h-8 w-8 rounded-full bg-neutral-800 ring-2 ring-default" aria-hidden="true" />
                        <span class="text-sm text-muted">
                          Превью: primary 500 и neutral 800
                        </span>
                      </div>
                    </div>

                    <div class="flex flex-wrap items-center gap-3 pt-2">
                      <UButton type="submit" color="primary" size="xl">
                        Сохранить
                      </UButton>
                    </div>
                  </UForm>
                </div>

                <div v-else-if="profileTab === 'company'" class="text-sm text-muted py-4">
                  Раздел «Компания» — здесь появятся реквизиты и контакты организации.
                </div>
                <div v-else-if="profileTab === 'documents'" class="text-sm text-muted py-4">
                  Раздел «Документы» — загрузка и список файлов.
                </div>
                <div v-else-if="profileTab === 'billing'" class="text-sm text-muted py-4">
                  Раздел «Оплата» — счета и способы оплаты.
                </div>
                <div v-else class="text-sm text-muted py-4">
                  Раздел «Уведомления» — настройки e‑mail и push.
                </div>
              </div>
            </UCard>
          </div>
        </div>
      </div>
    </UContainer>
  </UMain>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { useAppToast } from '../composables/useAppToast';
import { currentRole, setRole } from '../stores/role';
import { setSavedUiTheme, getSavedUiTheme } from '../composables/useUiTheme';

const { profileSaved } = useAppToast();

const profileTab = ref('account');
const profileTabItems = [
  { label: 'Настройки аккаунта', value: 'account' },
  { label: 'Компания', value: 'company' },
  { label: 'Документы', value: 'documents' },
  { label: 'Оплата', value: 'billing' },
  { label: 'Уведомления', value: 'notifications' },
];

const displayName = 'Иван Петров';
const companyName = 'Администрация муниципального округа';
const avatarSrc = '/src/img/Logo.svg';
const publicProfileUrl = `${typeof window !== 'undefined' ? window.location.origin : ''}/profile/public/demo`;

const copied = ref(false);
let copyTimer = null;

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

function onChangeCover() {
  window.alert('Здесь будет загрузка обложки профиля.');
}

function onChangeAvatar() {
  window.alert('Здесь будет загрузка фото профиля.');
}

function onViewPublicProfile() {
  window.alert('Публичный профиль откроется в новой вкладке, когда будет готов маршрут.');
}

const accountForm = reactive({
  firstName: 'Иван',
  lastName: 'Петров',
  phone: '+7 (900) 000-00-00',
  email: 'ivan.petrov@example.gov.ru',
  city: 'Москва',
  state: 'Москва',
  postcode: '101000',
  country: 'Россия',
});

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

const primaryColorItems = [
  { value: 'sky', label: 'Sky (голубой)' },
  { value: 'emerald', label: 'Emerald (зелёный)' },
  { value: 'violet', label: 'Violet (фиолетовый)' },
  { value: 'rose', label: 'Rose (розовый)' },
  { value: 'amber', label: 'Amber (оранжевый)' },
];

const neutralColorItems = [
  { value: 'slate', label: 'Slate (сланцевый)' },
  { value: 'zinc', label: 'Zinc (цинк)' },
  { value: 'gray', label: 'Gray (серый)' },
  { value: 'neutral', label: 'Neutral (нейтральный)' },
];

const savedTheme = getSavedUiTheme();
const themePrimary = ref(savedTheme.primary);
const themeNeutral = ref(savedTheme.neutral);

watch(
  [themePrimary, themeNeutral],
  ([primary, neutral]) => {
    setSavedUiTheme({ primary, neutral });
  },
  { immediate: true },
);
</script>
