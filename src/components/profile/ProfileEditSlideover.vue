<script setup lang="ts">
/**
 * «Редактировать страницу» — аватар, «О себе» и «Интересы». ФИО, телефон, почта,
 * подразделение и должность на портале не редактируются (ADR-041): сервер их
 * игнорирует, здесь они не показываются полями.
 */
import { reactive, ref, watch } from 'vue';
import { avatarUrlFromFilename, PROFILE_AVATAR_FILENAMES } from '../../constants/profileAvatars';
import { apiSessionFetch, apiSessionUpload } from '../../composables/useAuthSession';
import { isUploadedAvatar } from '../../utils/userName';
import { useAppToast } from '../../composables/useAppToast';

export type ProfileEditState = {
  avatarUrl: string;
  about: string;
  interests: string;
};

const props = defineProps<{ userId: number; initial: ProfileEditState }>();
const open = defineModel<boolean>('open', { default: false });
const emit = defineEmits<{
  (e: 'saved', state: ProfileEditState): void;
  /** Фото загружено и уже сохранено сервером как аватар. */
  (e: 'avatar-uploaded', url: string): void;
}>();

const { profileSaved, success, error } = useAppToast();

const AVATAR_MAX_BYTES = 5 * 1024 * 1024;
const AVATAR_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const fileInput = ref<HTMLInputElement | null>(null);
const uploading = ref(false);

async function onFilePicked(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = ''; // тот же файл можно выбрать повторно
  if (!file) return;
  if (!AVATAR_TYPES.includes(file.type)) {
    error('Это не подходит', 'Допустимы только JPEG, PNG или WebP.');
    return;
  }
  if (file.size > AVATAR_MAX_BYTES) {
    error('Файл слишком большой', 'Выберите фото до 5 МБ.');
    return;
  }
  uploading.value = true;
  try {
    const fd = new FormData();
    fd.append('avatar', file);
    const res = await apiSessionUpload<{ avatar_url: string }>('/api/profile_avatar.php', fd);
    const url = res?.data?.avatar_url;
    if (!res?.success || !url) {
      error('Фото не загружено', res?.message);
      return;
    }
    form.avatarUrl = url;
    emit('avatar-uploaded', url);
    success('Фото загружено', 'Оно обрезано по центру до квадрата.');
  } catch {
    error('Фото не загружено', 'Проверьте подключение к сети и попробуйте ещё раз.');
  } finally {
    uploading.value = false;
  }
}

const form = reactive<ProfileEditState>({ ...props.initial });
const saving = ref(false);

watch(open, (v) => {
  if (v) Object.assign(form, props.initial);
});

async function save() {
  if (saving.value) return;
  saving.value = true;
  try {
    const res = await apiSessionFetch<any>('/api/profile.php', {
      method: 'POST',
      json: { id: props.userId, avatar_url: form.avatarUrl, about: form.about, interests: form.interests },
    });
    if (!res?.success) {
      error('Не удалось сохранить профиль', res?.message);
      return;
    }
    profileSaved();
    emit('saved', { ...form });
    open.value = false;
  } catch {
    error('Не удалось сохранить профиль', 'Проверьте подключение к сети и попробуйте ещё раз.');
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <USlideover v-model:open="open" side="right" title="Редактировать страницу" description="Это увидят коллеги в вашем профиле">
    <template #body>
      <UForm id="profile-edit-form" :state="form" class="flex flex-col gap-6" @submit="save">
        <UAlert
          color="neutral"
          variant="subtle"
          icon="i-lucide-lock"
          title="ФИО, телефон, почта, подразделение и должность здесь не меняются"
          description="Если в них ошибка, обратитесь к администратору портала."
        />

        <UFormField label="О себе" name="about" hint="до 2000 символов">
          <UTextarea v-model="form.about" :rows="3" autoresize :maxrows="10" :maxlength="2000" class="w-full" placeholder="Чем занимаетесь, в чём можете помочь коллегам" />
        </UFormField>
        <UFormField label="Интересы" name="interests" hint="до 1000 символов">
          <UTextarea v-model="form.interests" :rows="2" autoresize :maxrows="6" :maxlength="1000" class="w-full" placeholder="Например: бег, настольные игры, история города" />
        </UFormField>

        <USeparator />

        <UFormField label="Аватар" name="avatar">
          <div class="flex items-center gap-4 pb-4">
            <div class="size-20 shrink-0 rounded-panel bg-elevated grid place-items-center overflow-hidden">
              <img :src="form.avatarUrl" alt="Текущий аватар" :class="isUploadedAvatar(form.avatarUrl) ? 'size-full object-cover' : 'size-3/4 object-contain'" />
            </div>
            <div class="flex flex-col items-start gap-1.5 min-w-0">
              <UButton type="button" color="neutral" variant="outline" icon="i-lucide-upload" :loading="uploading" @click="fileInput?.click()">
                Загрузить своё фото
              </UButton>
              <p class="text-xs text-muted">JPG, PNG или WebP до 5 МБ. Фото обрежется по центру до квадрата.</p>
              <input ref="fileInput" type="file" accept="image/jpeg,image/png,image/webp" class="sr-only" tabindex="-1" aria-hidden="true" @change="onFilePicked" />
            </div>
          </div>
          <p class="text-xs text-muted pb-2">Или выберите готовый:</p>
          <div class="flex flex-wrap gap-2" role="radiogroup" aria-label="Готовые аватары">
            <UButton
              v-for="name in PROFILE_AVATAR_FILENAMES"
              :key="name"
              type="button"
              size="md"
              square
              role="radio"
              :aria-checked="form.avatarUrl === avatarUrlFromFilename(name)"
              :aria-label="name.replace('.png', '')"
              :variant="form.avatarUrl === avatarUrlFromFilename(name) ? 'soft' : 'ghost'"
              :color="form.avatarUrl === avatarUrlFromFilename(name) ? 'primary' : 'neutral'"
              :class="form.avatarUrl === avatarUrlFromFilename(name) ? 'ring-2 ring-primary' : ''"
              @click="form.avatarUrl = avatarUrlFromFilename(name)"
            >
              <img :src="avatarUrlFromFilename(name)" alt="" class="size-8 object-contain" />
            </UButton>
          </div>
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" :disabled="saving" @click="open = false">Отмена</UButton>
        <UButton type="submit" form="profile-edit-form" color="primary" :loading="saving">Сохранить</UButton>
      </div>
    </template>
  </USlideover>
</template>
