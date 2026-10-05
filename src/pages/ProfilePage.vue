<script setup lang="ts">
/**
 * Профиль сотрудника в компоновке старого ВКонтакте (цвета и шрифт — из темы портала):
 * слева аватар, действия и коллеги по подразделению; справа имя, анкета, «Информация»
 * и стена. /profile — своя страница, /profile/:id — страница коллеги. Писать на стене
 * может любой вошедший сотрудник (ADR-040).
 */
import { computed, nextTick, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { apiSessionFetch, getAuthUser } from '../composables/useAuthSession';
import { useAppToast } from '../composables/useAppToast';
import { useProfileDisplay } from '../composables/useProfileDisplay';
import { useSectionAccess } from '../composables/useSectionAccess';
import { currentRole } from '../stores/role';
import { useProfileWall, WALL_POST_MAX_LENGTH, type WallPost } from '../composables/useProfileWall';
import { isUploadedAvatar, userAvatarSrc, userFullName } from '../utils/userName';
import { plural } from './Courses/courseDuration';
import ProfileSection from '../components/profile/ProfileSection.vue';
import ProfileWallPost from '../components/profile/ProfileWallPost.vue';
import ProfileEditSlideover, { type ProfileEditState } from '../components/profile/ProfileEditSlideover.vue';
import ProfileWishes, { type Wish } from '../components/profile/ProfileWishes.vue';
import ProfileAwards, { type Award } from '../components/profile/ProfileAwards.vue';

type Colleague = { id: number; firstname: string; surname: string; avatar_url: string };
type CompletedCourse = { courseId: number; title: string; completedAt: string | null; score: number | null; enrollmentId?: number };
type ProfileData = {
  id: number;
  firstname: string;
  surname: string;
  lastname: string;
  phone: string;
  email: string;
  ofo: string;
  ofoName: string;
  role: string;
  avatar_url: string;
  birthday: { month: number; day: number } | null;
  about: string;
  interests: string;
  colleagues: { total: number; items: Colleague[] };
  courses: { total: number; items: CompletedCourse[] };
  wishes: Wish[];
  awards: Award[];
};

const route = useRoute();
const { success, error: errorToast } = useAppToast();
const { setAvatarSrc } = useProfileDisplay();
const { isSuperAdmin, ensureLoaded: ensureAccessLoaded } = useSectionAccess();
void ensureAccessLoaded();
/** Награды выдаёт администратор портала в режиме «Администратор» (сервер проверяет группу сам). */
const canManageAwards = computed(() => isSuperAdmin.value && currentRole.value === 'admin');

const myId = computed(() => Number((getAuthUser() as { id?: number } | null)?.id) || 0);
const profileId = computed(() => Number(route.params.id) || myId.value);
const isOwn = computed(() => profileId.value > 0 && profileId.value === myId.value);

// ── Анкета ──────────────────────────────────────────────────────────────────
const profile = ref<ProfileData | null>(null);
const loading = ref(false);
const loadError = ref('');
const notFound = ref(false);
let loadSeq = 0;

async function loadProfile() {
  const id = profileId.value;
  const my = ++loadSeq;
  loading.value = true;
  loadError.value = '';
  notFound.value = false;
  try {
    const res = await apiSessionFetch<any>(`/api/profile.php?id=${id}&view=page`);
    if (my !== loadSeq) return;
    if (!res?.success) {
      if (/не найден/i.test(res?.message ?? '')) notFound.value = true;
      else loadError.value = res?.message || 'Не удалось загрузить профиль';
      profile.value = null;
      return;
    }
    const p = res.data as any;
    profile.value = {
      id: Number(p.id),
      firstname: p.firstname ?? '',
      surname: p.surname ?? '',
      lastname: p.lastname ?? '',
      phone: p.phone ?? '',
      email: p.email ?? '',
      ofo: String(p.ofo ?? ''),
      ofoName: p.ofoName ?? '',
      role: p.role ?? '',
      avatar_url: p.avatar_url ?? '',
      birthday: p.birthday?.month && p.birthday?.day ? { month: Number(p.birthday.month), day: Number(p.birthday.day) } : null,
      about: p.about ?? '',
      interests: p.interests ?? '',
      colleagues: {
        total: Number(p.colleagues?.total) || 0,
        items: Array.isArray(p.colleagues?.items) ? p.colleagues.items : [],
      },
      courses: {
        total: Number(p.courses?.total) || 0,
        items: Array.isArray(p.courses?.items) ? p.courses.items : [],
      },
      wishes: Array.isArray(p.wishes) ? p.wishes : [],
      awards: Array.isArray(p.awards) ? p.awards : [],
    };
  } catch {
    if (my === loadSeq) loadError.value = 'Проверьте подключение к сети и попробуйте ещё раз.';
  } finally {
    if (my === loadSeq) loading.value = false;
  }
}

const fullName = computed(() => userFullName(profile.value) || 'Сотрудник');

const BIRTH_MONTHS = ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря'];

const infoRows = computed(() => {
  const p = profile.value;
  if (!p) return [];
  const rows: { label: string; value: string; href?: string }[] = [];
  if (p.role) rows.push({ label: 'Должность', value: p.role });
  if (p.ofoName) rows.push({ label: 'Подразделение', value: p.ofoName });
  if (p.birthday) rows.push({ label: 'День рождения', value: `${p.birthday.day} ${BIRTH_MONTHS[p.birthday.month - 1] ?? ''}` });
  if (p.phone) rows.push({ label: 'Телефон', value: p.phone, href: `tel:${p.phone.replace(/[^\d+]/g, '')}` });
  if (p.email) rows.push({ label: 'Эл. почта', value: p.email, href: `mailto:${p.email}` });
  return rows;
});

const hasAbout = computed(() => !!(profile.value?.about || profile.value?.interests));

// ── Редактирование своей страницы ────────────────────────────────────────────
// Правятся только аватар, «О себе» и «Интересы»: остальное на портале не меняется (ADR-041).
const editOpen = ref(false);
const editInitial = computed<ProfileEditState>(() => ({
  avatarUrl: userAvatarSrc(profile.value),
  about: profile.value?.about ?? '',
  interests: profile.value?.interests ?? '',
}));

function onProfileSaved(s: ProfileEditState) {
  if (profile.value) profile.value = { ...profile.value, avatar_url: s.avatarUrl, about: s.about.trim(), interests: s.interests.trim() };
  setAvatarSrc(s.avatarUrl);
  window.dispatchEvent(new Event('ui:user-profile-updated'));
}

/** Фото уже сохранено сервером — обновляем страницу и шапку, не дожидаясь «Сохранить». */
function onAvatarUploaded(url: string) {
  if (profile.value) profile.value = { ...profile.value, avatar_url: url };
  setAvatarSrc(url);
  window.dispatchEvent(new Event('ui:user-profile-updated'));
}

// ── Блоки: желания, награды, курсы ───────────────────────────────
function patchProfile(patch: Partial<ProfileData>) {
  if (profile.value) profile.value = { ...profile.value, ...patch };
}

const shortDate = (iso: string | null) =>
  iso ? new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' }) : '';

// ── Стена ───────────────────────────────────────────────────────────────────
const wall = useProfileWall();
const { posts, total, loading: wallLoading, loadingMore, error: wallError } = wall;

const draft = ref('');
const posting = ref(false);
const composer = ref<{ textareaRef?: HTMLTextAreaElement } | null>(null);
const canPost = computed(() => {
  const text = draft.value.trim();
  return !!text && text.length <= WALL_POST_MAX_LENGTH && !posting.value;
});

async function submitPost() {
  if (!canPost.value) return;
  posting.value = true;
  try {
    await wall.create(draft.value.trim());
    draft.value = '';
  } catch (e) {
    errorToast('Запись не опубликована', e instanceof Error ? e.message : undefined);
  } finally {
    posting.value = false;
  }
}

async function savePost(id: number, content: string) {
  try {
    await wall.update(id, content);
  } catch (e) {
    errorToast('Запись не сохранена', e instanceof Error ? e.message : undefined);
    throw e;
  }
}

async function loadMorePosts() {
  try {
    await wall.loadMore();
  } catch (e) {
    errorToast('Не удалось загрузить записи', e instanceof Error ? e.message : undefined);
  }
}

async function focusComposer() {
  await nextTick();
  const el = composer.value?.textareaRef;
  el?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  el?.focus({ preventScroll: true });
}

const deleting = reactive<{ post: WallPost | null; busy: boolean }>({ post: null, busy: false });
const deleteOpen = computed({
  get: () => !!deleting.post,
  set: (v: boolean) => {
    if (!v && !deleting.busy) deleting.post = null;
  },
});

async function confirmDelete() {
  const post = deleting.post;
  if (!post) return;
  deleting.busy = true;
  try {
    await wall.remove(post.id);
    success('Запись удалена');
    deleting.post = null;
  } catch (e) {
    errorToast('Запись не удалена', e instanceof Error ? e.message : undefined);
  } finally {
    deleting.busy = false;
  }
}

watch(
  profileId,
  (id) => {
    draft.value = '';
    editOpen.value = false;
    if (!id) return;
    void loadProfile();
    void wall.load(id);
  },
  { immediate: true },
);
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1200px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
      <!-- Нет такого сотрудника -->
      <UEmpty
        v-if="notFound"
        variant="naked"
        icon="i-lucide-user-x"
        title="Сотрудник не найден"
        description="Возможно, ссылка устарела. Найти коллегу можно через поиск портала (Ctrl+K)."
        :actions="[{ label: 'На мою страницу', to: '/profile', color: 'neutral', variant: 'outline' }]"
        class="py-16"
      />

      <UAlert
        v-else-if="loadError && !profile"
        color="error"
        variant="subtle"
        icon="i-lucide-alert-triangle"
        title="Не удалось загрузить профиль"
        :description="loadError"
      >
        <template #actions>
          <UButton color="error" variant="outline" icon="i-lucide-rotate-ccw" @click="loadProfile">Повторить</UButton>
        </template>
      </UAlert>

      <div v-else class="grid grid-cols-1 md:grid-cols-[220px_minmax(0,1fr)] lg:grid-cols-[260px_minmax(0,1fr)] gap-6 items-start">
        <!-- Левая колонка: аватар, действия, коллеги -->
        <aside class="flex flex-col gap-4 min-w-0" aria-label="Карточка сотрудника">
          <USkeleton v-if="loading && !profile" class="w-full max-w-60 md:max-w-none aspect-square rounded-panel" />
          <div v-else class="w-full max-w-60 md:max-w-none aspect-square rounded-panel bg-elevated grid place-items-center overflow-hidden">
            <img :src="userAvatarSrc(profile)" :alt="fullName" :class="isUploadedAvatar(profile?.avatar_url) ? 'size-full object-cover' : 'size-3/4 object-contain'" />
          </div>

          <nav v-if="profile" class="flex flex-col" aria-label="Действия">
            <template v-if="isOwn">
              <UButton color="neutral" variant="ghost" icon="i-lucide-pencil" class="justify-start" @click="editOpen = true">
                Редактировать страницу
              </UButton>
            </template>
            <template v-else>
              <UButton color="neutral" variant="ghost" icon="i-lucide-message-square-plus" class="justify-start" @click="focusComposer">
                Написать на стене
              </UButton>
              <UButton v-if="profile.email" color="neutral" variant="ghost" icon="i-lucide-mail" class="justify-start" :href="`mailto:${profile.email}`">
                Написать письмо
              </UButton>
              <UButton color="neutral" variant="ghost" icon="i-lucide-user" class="justify-start" to="/profile">
                Моя страница
              </UButton>
            </template>
          </nav>

          <ProfileWishes
            v-if="profile"
            :wishes="profile.wishes"
            :is-own="isOwn"
            @update="patchProfile({ wishes: $event })"
          />

          <ProfileSection v-if="profile && profile.colleagues.total" title="Коллеги" title-id="profile-colleagues">
            <template #aside>{{ profile.colleagues.total }}</template>
            <ul class="grid grid-cols-3 gap-x-2 gap-y-3">
              <li v-for="c in profile.colleagues.items" :key="c.id" class="min-w-0">
                <RouterLink :to="`/profile/${c.id}`" class="group flex flex-col items-center gap-1 text-center rounded-md focus-visible:outline-2 focus-visible:outline-primary">
                  <span class="w-full aspect-square rounded-md bg-elevated grid place-items-center overflow-hidden">
                    <img :src="userAvatarSrc(c)" alt="" :class="isUploadedAvatar(c.avatar_url) ? 'size-full object-cover' : 'size-3/4 object-contain'" />
                  </span>
                  <span class="w-full text-xs leading-4 text-default group-hover:text-primary line-clamp-2 break-words">
                    {{ c.firstname }} {{ c.surname }}
                  </span>
                </RouterLink>
              </li>
            </ul>
          </ProfileSection>
        </aside>

        <!-- Правая колонка: анкета, информация, стена -->
        <div class="flex flex-col gap-6 min-w-0">
          <div v-if="loading && !profile" class="flex flex-col gap-3">
            <USkeleton class="h-8 w-2/3 rounded-lg" />
            <USkeleton class="h-4 w-1/2 rounded" />
            <USkeleton class="h-4 w-1/3 rounded" />
            <USkeleton class="h-4 w-2/5 rounded" />
          </div>

          <template v-else-if="profile">
            <header class="flex flex-col gap-4">
              <h1 class="text-2xl font-semibold leading-tight text-highlighted">{{ fullName }}</h1>
              <dl v-if="infoRows.length" class="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)] gap-x-4 gap-y-1.5 text-sm">
                <template v-for="row in infoRows" :key="row.label">
                  <dt class="text-muted">{{ row.label }}:</dt>
                  <dd class="min-w-0 break-words">
                    <a v-if="row.href" :href="row.href" class="text-primary hover:underline">{{ row.value }}</a>
                    <span v-else class="text-default">{{ row.value }}</span>
                  </dd>
                </template>
              </dl>
            </header>

            <ProfileSection v-if="hasAbout || isOwn" title="Информация" title-id="profile-info">
              <dl v-if="hasAbout" class="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)] gap-x-4 gap-y-3 text-sm">
                <template v-if="profile.about">
                  <dt class="text-muted">О себе:</dt>
                  <dd class="min-w-0 text-default whitespace-pre-line break-words">{{ profile.about }}</dd>
                </template>
                <template v-if="profile.interests">
                  <dt class="text-muted">Интересы:</dt>
                  <dd class="min-w-0 text-default whitespace-pre-line break-words">{{ profile.interests }}</dd>
                </template>
              </dl>
              <p v-else class="text-sm text-muted">
                Расскажите коллегам о себе и своих интересах.
                <UButton color="primary" variant="link" class="p-0 align-baseline" @click="editOpen = true">Заполнить</UButton>
              </p>
            </ProfileSection>
          </template>

          <template v-if="profile">
            <ProfileAwards
              :user-id="profile.id"
              :awards="profile.awards"
              :can-manage="canManageAwards"
              @update="patchProfile({ awards: $event })"
            />

            <ProfileSection v-if="profile.courses.total" title="Пройденные курсы" title-id="profile-courses">
              <template #aside>{{ profile.courses.total }}</template>
              <ul class="flex flex-col divide-y divide-default">
                <li v-for="c in profile.courses.items" :key="`${c.courseId}-${c.completedAt}`" class="flex items-start gap-3 py-2.5 first:pt-0">
                  <UIcon name="i-lucide-graduation-cap" class="size-5 text-success shrink-0 mt-0.5" aria-hidden="true" />
                  <div class="flex-1 min-w-0">
                    <RouterLink v-if="c.enrollmentId" :to="`/courses/${c.enrollmentId}/result`" class="text-sm font-medium text-primary hover:underline break-words">{{ c.title }}</RouterLink>
                    <p v-else class="text-sm font-medium text-highlighted break-words">{{ c.title }}</p>
                    <p class="text-xs text-muted">
                      <template v-if="c.completedAt">Пройден {{ shortDate(c.completedAt) }}</template>
                      <template v-if="c.score != null"><template v-if="c.completedAt"> · </template>результат {{ Math.round(c.score) }}%</template>
                    </p>
                  </div>
                </li>
              </ul>
              <p v-if="profile.courses.total > profile.courses.items.length" class="pt-2 text-xs text-muted">
                Показаны последние {{ profile.courses.items.length }} из {{ profile.courses.total }}
              </p>
            </ProfileSection>
          </template>

          <!-- Стена -->
          <ProfileSection v-if="profile || loading" title="Стена" title-id="profile-wall">
            <template v-if="total" #aside>{{ total }} {{ plural(total, ['запись', 'записи', 'записей']) }}</template>

            <form class="flex flex-col gap-2 pb-2" @submit.prevent="submitPost">
              <UTextarea
                ref="composer"
                v-model="draft"
                :rows="2"
                autoresize
                :maxrows="10"
                :maxlength="WALL_POST_MAX_LENGTH"
                :placeholder="isOwn ? 'Что у вас нового?' : 'Написать на стене…'"
                aria-label="Текст записи на стене"
                class="w-full"
                @keydown.ctrl.enter.prevent="submitPost"
                @keydown.meta.enter.prevent="submitPost"
              />
              <div class="flex items-center justify-between gap-3">
                <span class="text-xs text-dimmed">Ctrl+Enter — отправить</span>
                <UButton type="submit" color="primary" size="sm" :loading="posting" :disabled="!canPost">Отправить</UButton>
              </div>
            </form>

            <UAlert
              v-if="wallError"
              color="error"
              variant="subtle"
              icon="i-lucide-alert-triangle"
              title="Не удалось загрузить стену"
              :description="wallError"
            >
              <template #actions>
                <UButton color="error" variant="outline" icon="i-lucide-rotate-ccw" @click="wall.load(profileId)">Повторить</UButton>
              </template>
            </UAlert>

            <div v-else-if="wallLoading" class="flex flex-col divide-y divide-default" aria-busy="true" aria-label="Загрузка записей">
              <div v-for="n in 3" :key="n" class="flex gap-3 py-4">
                <USkeleton class="size-12 rounded-full shrink-0" />
                <div class="flex-1 flex flex-col gap-2">
                  <USkeleton class="h-4 w-40 rounded" />
                  <USkeleton class="h-4 w-full rounded" />
                  <USkeleton class="h-4 w-3/4 rounded" />
                </div>
              </div>
            </div>

            <UEmpty
              v-else-if="!posts.length"
              variant="naked"
              icon="i-lucide-message-square"
              title="На стене пока нет записей"
              :description="isOwn ? 'Напишите первую — коллеги увидят её на вашей странице.' : 'Будьте первым, кто напишет на этой стене.'"
              class="py-8"
            />

            <div v-else class="flex flex-col divide-y divide-default border-t border-default">
              <ProfileWallPost
                v-for="post in posts"
                :key="post.id"
                :post="post"
                :save="savePost"
                @delete="deleting.post = $event"
              />
            </div>

            <div v-if="posts.length < total && !wallLoading" class="flex justify-center pt-2">
              <UButton color="neutral" variant="outline" :loading="loadingMore" @click="loadMorePosts">
                Показать ещё
              </UButton>
            </div>
          </ProfileSection>
        </div>
      </div>
    </div>

    <ProfileEditSlideover
      v-if="isOwn && profile"
      v-model:open="editOpen"
      :user-id="myId"
      :initial="editInitial"
      @saved="onProfileSaved"
      @avatar-uploaded="onAvatarUploaded"
    />

    <UModal
      v-model:open="deleteOpen"
      title="Удалить запись?"
      description="Запись исчезнет со стены вместе с реакциями. Отменить это нельзя."
    >
      <template #body>
        <p class="text-sm text-default whitespace-pre-line line-clamp-4 break-words">{{ deleting.post?.content }}</p>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="deleting.busy" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" :loading="deleting.busy" @click="confirmDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
