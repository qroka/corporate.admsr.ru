<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import type { DropdownMenuItem } from '@nuxt/ui';
import { useAppToast } from '../composables/useAppToast';
import { useAppConfig } from '../composables/useAppConfig';
import { useOfoTree, type OfoPosition } from '../composables/useOfoTree';
import OfoSelect from '../components/OfoSelect.vue';
import { useProfileDisplay } from '../composables/useProfileDisplay';
import { NEUTRAL_COLORS, PRIMARY_COLORS } from '../composables/useUiTheme';
import { avatarUrlFromFilename, PROFILE_AVATAR_FILENAMES } from '../constants/profileAvatars';
import { useProfileWall, wallPostPlainText, type WallPost } from '../composables/useProfileWall';
import ProfileWallPost from '../components/profile/ProfileWallPost.vue';
import ProfileCreatePost from '../components/profile/ProfileCreatePost.vue';

const { profileSaved } = useAppToast();

const { ensureLoaded: ensureOfoLoaded, error: ofoError, unitNumberOf, fetchPositions } = useOfoTree();
ensureOfoLoaded();

const positionsList = ref<OfoPosition[]>([]);
const positionsLoading = ref(false);

type PageView = 'wall' | 'edit';
const pageView = ref<PageView>('wall');

const { displayName, subtitle, avatarSrc, setAvatarSrc, setDisplayName, setSubtitle } = useProfileDisplay();
const avatarPickerOpen = ref(false);
const avatarFilenames = PROFILE_AVATAR_FILENAMES;

const currentUserId = ref(0);

// --- Wall ---
const {
  sortedPosts,
  ensureLoaded: ensureWallLoaded,
  createPost,
  deletePost,
  updatePost,
} = useProfileWall();
ensureWallLoaded();

const postEditorOpen = ref(false);
const editingPost = ref<WallPost | null>(null);
const wallSearch = ref('');
const wallSearchOpen = ref(false);

const filteredPosts = computed(() => {
  const q = wallSearch.value.trim().toLowerCase();
  if (!q) return sortedPosts.value;
  return sortedPosts.value.filter((p) => {
    const plain = wallPostPlainText(p.content).toLowerCase();
    return plain.includes(q) || p.authorName.toLowerCase().includes(q);
  });
});

function toggleWallSearch() {
  wallSearchOpen.value = !wallSearchOpen.value;
  if (!wallSearchOpen.value) wallSearch.value = '';
}

function openCreatePost() {
  editingPost.value = null;
  postEditorOpen.value = true;
}

function openEditPost(post: WallPost) {
  editingPost.value = post;
  postEditorOpen.value = true;
}

function onPostSubmit(payload: { content: string; postId?: string }) {
  if (payload.postId) {
    updatePost(payload.postId, { content: payload.content });
    editingPost.value = null;
    return;
  }
  if (!currentUserId.value) return;
  createPost({
    userId: currentUserId.value,
    authorName: displayName.value || 'Сотрудник',
    authorAvatar: avatarSrc.value,
    content: payload.content,
  });
  editingPost.value = null;
}

// --- Theme dropdown ---
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

function selectAvatar(filename: string) {
  setAvatarSrc(avatarUrlFromFilename(filename));
  avatarPickerOpen.value = false;
}

const profileLoading = ref(false);
const profileSaving = ref(false);

const accountForm = reactive({
  firstName: '',
  lastName: '',
  patronymic: '',
  phone: '',
  email: '',
  ofoId: null as number | null,
  positionId: null as number | null,
  role: '',
});

async function loadProfile() {
  const raw = localStorage.getItem('auth-user');
  if (!raw) return;
  const user = JSON.parse(raw) as { id: number };
  if (!user?.id) return;
  currentUserId.value = user.id;

  profileLoading.value = true;
  try {
    const res = await fetch(`/api/profile.php?id=${user.id}`);
    const data = await res.json();
    if (!data.success) return;

    const p = data.data;
    accountForm.firstName  = p.firstname ?? '';
    accountForm.lastName   = p.surname   ?? '';
    accountForm.patronymic = p.lastname  ?? '';
    accountForm.phone      = p.phone     ?? '';
    accountForm.email      = p.email     ?? '';
    const ofoNum = Number(p.ofo);
    accountForm.ofoId      = (p.ofo != null && Number.isFinite(ofoNum) && ofoNum > 0) ? ofoNum : null;
    accountForm.role       = p.role      ?? '';

    if (p.avatar_url) setAvatarSrc(p.avatar_url);
    if (p.role)       setSubtitle(p.role);

    const full = [p.surname, p.firstname].filter(Boolean).join(' ');
    if (full) setDisplayName(full);
  } finally {
    profileLoading.value = false;
  }
}

const positionItems = computed(() =>
  positionsList.value.map((p) => ({ value: p.id, label: p.name })),
);

watch(
  () => accountForm.ofoId,
  async (id) => {
    accountForm.positionId = null;
    positionsList.value = [];
    const un = unitNumberOf(id);
    if (un == null) { accountForm.role = ''; return; }
    positionsLoading.value = true;
    try {
      positionsList.value = await fetchPositions(un);
      const match = positionsList.value.find((p) => p.name === accountForm.role);
      if (match) accountForm.positionId = match.id;
      else accountForm.role = '';
    } catch {
      positionsList.value = [];
    } finally {
      positionsLoading.value = false;
    }
  },
);

watch(
  () => accountForm.positionId,
  (val) => {
    const pos = positionsList.value.find((p) => p.id === val);
    if (pos?.name) accountForm.role = pos.name;
  },
);

async function onUpdateAccount() {
  const raw = localStorage.getItem('auth-user');
  if (!raw) return;
  const user = JSON.parse(raw) as { id: number };
  if (!user?.id) return;

  profileSaving.value = true;
  try {
    const res = await fetch('/api/profile.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id:         user.id,
        firstname:  accountForm.firstName,
        surname:    accountForm.lastName,
        lastname:   accountForm.patronymic,
        phone:      accountForm.phone,
        email:      accountForm.email,
        ofo:        String(accountForm.ofoId ?? ''),
        role:       accountForm.role,
        avatar_url: avatarSrc.value,
      }),
    });
    const data = await res.json();
    if (!data.success) return;

    const full = [accountForm.lastName, accountForm.firstName]
      .filter(Boolean).join(' ');
    if (full) setDisplayName(full);
    if (accountForm.role) setSubtitle(accountForm.role);
    profileSaved();
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new Event('ui:user-profile-updated'));
    }
    pageView.value = 'wall';
  } finally {
    profileSaving.value = false;
  }
}

const profileInfoLines = computed(() => {
  const lines: { icon: string; text: string }[] = [];
  if (accountForm.role) lines.push({ icon: 'i-lucide-briefcase', text: accountForm.role });
  if (accountForm.phone) lines.push({ icon: 'i-lucide-phone', text: accountForm.phone });
  if (accountForm.email) lines.push({ icon: 'i-lucide-mail', text: accountForm.email });
  return lines;
});

onMounted(() => {
  void loadProfile();
});
</script>

<template>
  <UMain class="profile-page flex flex-col w-full h-full min-h-0 mx-0 max-w-none">
    <!-- Обложка -->
    <div class="profile-cover">
      <div class="profile-cover__gradient" />
      <div class="profile-cover__pattern" />
    </div>

    <!-- Шапка профиля -->
    <header class="profile-header">
      <div class="profile-header__main">
        <div class="profile-header__avatar-wrap">
          <UAvatar
            :src="avatarSrc"
            :alt="displayName"
            size="3xl"
            class="profile-header__avatar"
            :ui="{ root: '!bg-elevated ring-4 ring-(--ui-bg)' }"
          />
          <UPopover v-model:open="avatarPickerOpen">
            <UButton
              type="button"
              color="primary"
              variant="solid"
              size="xs"
              icon="i-lucide-plus"
              class="profile-header__avatar-btn rounded-full"
              aria-label="Изменить аватар"
            />
            <template #content>
              <div class="p-3 grid grid-cols-3 gap-2 w-56">
                <UButton
                  v-for="name in avatarFilenames"
                  :key="name"
                  size="sm"
                  variant="subtle"
                  color="neutral"
                  @click="selectAvatar(name)"
                >
                  <img :src="avatarUrlFromFilename(name)" alt="" class="w-10 h-10 object-contain" />
                </UButton>
              </div>
            </template>
          </UPopover>
        </div>

        <div class="profile-header__info min-w-0">
          <h1 class="profile-header__name">{{ displayName || 'Профиль' }}</h1>
          <p v-if="subtitle" class="profile-header__role">{{ subtitle }}</p>
        </div>
      </div>

      <div class="profile-header__actions">
        <UButton
          type="button"
          :color="pageView === 'edit' ? 'primary' : 'neutral'"
          :variant="pageView === 'edit' ? 'solid' : 'outline'"
          size="lg"
          icon="i-lucide-pencil"
          label="Редактировать профиль"
          class="rounded-xl"
          @click="pageView = 'edit'"
        />
        <UButton
          v-if="pageView === 'edit'"
          type="button"
          color="neutral"
          variant="ghost"
          size="lg"
          icon="i-lucide-arrow-left"
          label="К стене"
          class="rounded-xl"
          @click="pageView = 'wall'"
        />
      </div>
    </header>

    <!-- Контент -->
    <div class="profile-body">
      <!-- Стена -->
      <section v-if="pageView === 'wall'" class="profile-wall">
        <div class="profile-wall__composer" role="button" tabindex="0" @click="openCreatePost" @keydown.enter="openCreatePost">
          <UIcon name="i-lucide-plus" class="size-5 text-dimmed shrink-0" />
          <span class="text-muted text-sm sm:text-base">Создать пост</span>
          <div class="profile-wall__composer-tools" @click.stop>
            <UButton type="button" color="neutral" variant="ghost" size="sm" icon="i-lucide-image" class="rounded-full" aria-label="Добавить фото" @click="openCreatePost" />
            <UButton type="button" color="neutral" variant="ghost" size="sm" icon="i-lucide-smile" class="rounded-full" aria-label="Эмодзи" @click="openCreatePost" />
          </div>
        </div>

        <div class="profile-wall__toolbar">
          <span class="text-sm font-medium text-highlighted">Все посты</span>
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            size="sm"
            :icon="wallSearchOpen ? 'i-lucide-x' : 'i-lucide-search'"
            class="rounded-full shrink-0"
            :aria-label="wallSearchOpen ? 'Закрыть поиск' : 'Поиск по постам'"
            @click="toggleWallSearch"
          />
        </div>

        <UInput
          v-if="wallSearchOpen"
          v-model="wallSearch"
          placeholder="Поиск по записям..."
          size="lg"
          icon="i-lucide-search"
          class="w-full"
          autofocus
        />

        <div v-if="filteredPosts.length" class="profile-wall__feed">
          <ProfileWallPost
            v-for="post in filteredPosts"
            :key="post.id"
            :post="post"
            :is-owner="post.userId === currentUserId"
            @delete="deletePost"
            @edit="openEditPost"
          />
        </div>

        <div v-else class="profile-wall__empty">
          <UIcon name="i-lucide-message-square-plus" class="size-10 text-dimmed mb-3" />
          <p class="text-sm font-medium text-highlighted">Пока нет записей</p>
          <p class="text-xs text-muted mt-1 max-w-xs text-center">
            Создайте первый пост — поделитесь новостью или фотографией с коллегами
          </p>
          <UButton
            type="button"
            color="primary"
            variant="soft"
            size="md"
            label="Создать пост"
            class="mt-4 rounded-full"
            @click="openCreatePost"
          />
        </div>
      </section>

      <!-- Редактирование -->
      <section v-else class="profile-edit">
        <UCard class="profile-edit__card">
          <template #header>
            <div>
              <h2 class="text-lg font-semibold text-highlighted">Личные данные</h2>
              <p class="text-sm text-muted mt-0.5">Информация отображается в профиле и на стене</p>
            </div>
          </template>

          <UForm :state="accountForm" class="space-y-6" @submit.prevent="onUpdateAccount">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-5">
              <UFormField label="Фамилия" name="lastName">
                <UInput v-model="accountForm.lastName" size="lg" class="w-full" autocomplete="family-name" :disabled="profileLoading" />
              </UFormField>
              <UFormField label="Имя" name="firstName">
                <UInput v-model="accountForm.firstName" size="lg" class="w-full" autocomplete="given-name" :disabled="profileLoading" />
              </UFormField>
              <UFormField label="Отчество" name="patronymic">
                <UInput v-model="accountForm.patronymic" size="lg" class="w-full" autocomplete="additional-name" :disabled="profileLoading" />
              </UFormField>
              <UFormField label="Телефон" name="phone">
                <UInput v-model="accountForm.phone" size="lg" class="w-full" type="tel" autocomplete="tel" :disabled="profileLoading" />
              </UFormField>
              <UFormField label="Электронная почта" name="email">
                <UInput v-model="accountForm.email" size="lg" class="w-full" type="email" autocomplete="email" :disabled="profileLoading" />
              </UFormField>
              <UFormField label="ОФО" name="ofoId" :help="ofoError ? String(ofoError) : undefined">
                <OfoSelect v-model="accountForm.ofoId" />
              </UFormField>
              <UFormField label="Должность" name="positionId">
                <USelectMenu
                  v-model="accountForm.positionId"
                  :items="positionItems"
                  value-key="value"
                  label-key="label"
                  :placeholder="accountForm.ofoId == null ? 'Сначала выберите ОФО' : 'Выберите должность'"
                  size="lg"
                  color="neutral"
                  class="w-full"
                  :disabled="accountForm.ofoId == null || positionsLoading"
                  :loading="positionsLoading"
                  :content="{ align: 'start', sideOffset: 8 }"
                />
              </UFormField>
            </div>

            <USeparator />

            <div>
              <h3 class="text-sm font-semibold text-highlighted mb-1">Аватар</h3>
              <p class="text-sm text-muted mb-3">Выберите изображение для профиля</p>
              <div class="flex flex-wrap gap-2">
                <UButton
                  v-for="name in avatarFilenames"
                  :key="name"
                  size="md"
                  :variant="avatarSrc.includes(name) ? 'solid' : 'outline'"
                  :color="avatarSrc.includes(name) ? 'primary' : 'neutral'"
                  @click="selectAvatar(name)"
                >
                  <img :src="avatarUrlFromFilename(name)" alt="" class="w-8 h-8 object-contain" />
                </UButton>
              </div>
            </div>

            <USeparator />

            <div>
              <h3 class="text-sm font-semibold text-highlighted mb-1">Цвета интерфейса</h3>
              <p class="text-sm text-muted mb-3">Настройка сохранится в браузере</p>
              <UDropdownMenu :items="themeMenuItems">
                <UButton type="button" icon="i-lucide-palette" color="neutral" variant="outline" size="lg" trailing-icon="i-lucide-chevrons-up-down">
                  Кастомизация портала
                </UButton>
                <template #chip-leading="{ item }">
                  <div class="inline-flex items-center justify-center shrink-0 size-5">
                    <span
                      class="rounded-full ring ring-bg bg-(--chip-light) dark:bg-(--chip-dark) size-2"
                      :style="{
                        '--chip-light': `var(--color-${(item as any).chip}-500)`,
                        '--chip-dark': `var(--color-${(item as any).chip}-400)`,
                      }"
                    />
                  </div>
                </template>
              </UDropdownMenu>
            </div>

            <div class="flex flex-wrap items-center gap-3 pt-2">
              <UButton type="submit" color="primary" size="lg" :loading="profileSaving" :disabled="profileLoading">
                Сохранить изменения
              </UButton>
              <UButton type="button" color="neutral" variant="ghost" size="lg" @click="pageView = 'wall'">
                Отмена
              </UButton>
            </div>
          </UForm>
        </UCard>
      </section>

      <!-- Сайдбар -->
      <aside class="profile-sidebar">
        <UCard v-if="profileInfoLines.length" class="profile-sidebar__card">
          <template #header>
            <h3 class="text-sm font-semibold text-highlighted">Информация</h3>
          </template>
          <ul class="space-y-2.5">
            <li v-for="(line, i) in profileInfoLines" :key="i" class="flex items-start gap-2.5 text-sm">
              <UIcon :name="line.icon" class="size-4 text-dimmed shrink-0 mt-0.5" />
              <span class="text-muted break-words">{{ line.text }}</span>
            </li>
          </ul>
        </UCard>

        <UCard class="profile-sidebar__card">
          <template #header>
            <h3 class="text-sm font-semibold text-highlighted">О стене</h3>
          </template>
          <p class="text-sm text-muted leading-relaxed">
            Публикуйте новости, фото и заметки для коллег.
          </p>
        </UCard>
      </aside>
    </div>

    <ProfileCreatePost
      v-model:open="postEditorOpen"
      :post-id="editingPost?.id ?? null"
      :initial-content="editingPost?.content ?? ''"
      @submit="onPostSubmit"
    />
  </UMain>
</template>

<style scoped>
.profile-page {
  --profile-cover-h: 11rem;
  gap: 0;
}

@media (min-width: 640px) {
  .profile-page { --profile-cover-h: 13rem; }
}

.profile-cover {
  position: relative;
  height: var(--profile-cover-h);
  border-radius: 1rem;
  overflow: hidden;
  flex-shrink: 0;
}

.profile-cover__gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--ui-primary) 70%, #1a1a2e) 0%,
    color-mix(in srgb, var(--ui-primary) 40%, #16213e) 45%,
    color-mix(in srgb, var(--ui-primary) 25%, #0f0f14) 100%
  );
}

.profile-cover__pattern {
  position: absolute;
  inset: 0;
  opacity: 0.35;
  background-image:
    radial-gradient(circle at 18% 42%, rgb(255 255 255 / 0.12) 0%, transparent 42%),
    radial-gradient(circle at 82% 28%, rgb(255 255 255 / 0.08) 0%, transparent 38%);
}

.profile-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem 1.5rem;
  padding: 0 0.25rem;
  margin-top: -3.25rem;
  position: relative;
  z-index: 2;
}

@media (min-width: 640px) {
  .profile-header { margin-top: -3.75rem; }
}

.profile-header__main {
  display: flex;
  align-items: flex-end;
  gap: 1rem 1.25rem;
  min-width: 0;
}

.profile-header__avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.profile-header__avatar {
  width: 6.5rem;
  height: 6.5rem;
}

@media (min-width: 640px) {
  .profile-header__avatar {
    width: 7.5rem;
    height: 7.5rem;
  }
}

.profile-header__avatar-btn {
  position: absolute;
  right: 0.25rem;
  bottom: 0.25rem;
  box-shadow: 0 2px 8px rgb(0 0 0 / 0.25);
}

.profile-header__name {
  font-size: 1.375rem;
  font-weight: 600;
  line-height: 1.25;
  color: var(--ui-text-highlighted);
  letter-spacing: -0.02em;
}

@media (min-width: 640px) {
  .profile-header__name { font-size: 1.625rem; }
}

.profile-header__role {
  font-size: 0.875rem;
  color: var(--ui-text-muted);
  margin-top: 0.2rem;
}

.profile-header__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding-bottom: 0.25rem;
}

.profile-body {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
  margin-top: 1.25rem;
  padding-bottom: 1rem;
}

@media (min-width: 1024px) {
  .profile-body {
    grid-template-columns: minmax(0, 1fr) 17.5rem;
    gap: 1.25rem;
    align-items: start;
  }

  .profile-edit {
    grid-column: 1 / -1;
    max-width: 48rem;
  }
}

@media (min-width: 1280px) {
  .profile-body {
    grid-template-columns: minmax(0, 1fr) 20rem;
  }
}

.profile-wall {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-width: 0;
}

.profile-wall__composer {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  background: var(--ui-bg-elevated);
  border: 1px solid color-mix(in srgb, var(--ui-border) 55%, transparent);
  border-radius: 1rem;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.profile-wall__composer:hover {
  border-color: color-mix(in srgb, var(--ui-primary) 35%, var(--ui-border));
  background: color-mix(in srgb, var(--ui-bg-elevated) 92%, var(--ui-primary));
}

.profile-wall__composer-tools {
  margin-left: auto;
  display: flex;
  gap: 0.125rem;
}

.profile-wall__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.25rem 0;
}

.profile-wall__feed {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.profile-wall__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1.5rem;
  background: var(--ui-bg-elevated);
  border: 1px dashed color-mix(in srgb, var(--ui-border) 70%, transparent);
  border-radius: 1rem;
  text-align: center;
}

.profile-sidebar {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.profile-sidebar__card :deep([data-slot="header"]) {
  padding-bottom: 0.5rem;
}

.profile-edit__card {
  border-radius: 1rem;
}
</style>
