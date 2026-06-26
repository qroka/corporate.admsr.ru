import { computed, onMounted, onUnmounted, ref, watch } from 'vue';

type DbProfile = {
  id: number;
  firstname?: string;
  surname?: string;
  lastname?: string;
  ofo?: string;
  user_group?: string;
  phone?: string;
  email?: string;
  role?: string;
  avatar_url?: string;
};

const DEFAULT_NAME = '';
const DEFAULT_SUBTITLE = 'Инженер';
const DEFAULT_AVATAR = `/img/FullPic/avatars/${encodeURIComponent('Alien.png')}`;

function safeParseAuthUser(): { id?: number } | null {
  if (typeof window === 'undefined') return null;
  try {
    return JSON.parse(window.localStorage.getItem('auth-user') || 'null');
  } catch {
    return null;
  }
}

function normalizeAvatarSrc(raw: unknown): string | undefined {
  const s = String(raw ?? '').trim();
  if (!s) return undefined;
  if (s.startsWith('/')) return s;
  if (/^https?:\/\//i.test(s)) return s;
  return s;
}

export function useHeaderUser() {
  const loading = ref(false);
  const error = ref<unknown>(null);
  const profile = ref<DbProfile | null>(null);

  const userId = computed(() => {
    const user = safeParseAuthUser();
    const id = Number(user?.id ?? 0);
    return Number.isFinite(id) && id > 0 ? id : null;
  });

  const headerName = computed(() => {
    const p = profile.value;
    const surname = String(p?.surname ?? '').trim();
    const firstname = String(p?.firstname ?? '').trim();
    const base = [surname, firstname].filter(Boolean).join(' ').trim();
    return base || DEFAULT_NAME;
  });

  const subtitle = computed(() => {
    const p = profile.value;
    const s =
      String(p?.role ?? '').trim() ||
      String(p?.user_group ?? '').trim() ||
      DEFAULT_SUBTITLE;
    return s;
  });

  const avatarSrc = computed(() => {
    const p = profile.value;
    return normalizeAvatarSrc(p?.avatar_url) || DEFAULT_AVATAR;
  });

  const canToggleAdminRole = computed(() => {
    const fromProfile = String(profile.value?.user_group ?? '').trim().toLowerCase();
    if (fromProfile) return fromProfile === 'admin';

    const authUser = safeParseAuthUser() as { user_group?: string } | null;
    return String(authUser?.user_group ?? '').trim().toLowerCase() === 'admin';
  });

  async function reload() {
    error.value = null;
    profile.value = null;

    const id = userId.value;
    if (!id) return;

    loading.value = true;
    try {
      const res = await fetch(`/api/profile.php?id=${encodeURIComponent(String(id))}`);
      const json = await res.json();
      if (!json?.success) throw new Error(json?.message || 'Не удалось загрузить профиль');
      profile.value = json.data as DbProfile;
    } catch (e) {
      error.value = e;
    } finally {
      loading.value = false;
    }
  }

  function onProfileUpdated() {
    void reload();
  }

  function onStorage(e: StorageEvent) {
    if (e.key === 'auth-user') {
      void reload();
    }
  }

  onMounted(() => {
    void reload();
    if (typeof window !== 'undefined') {
      window.addEventListener('ui:user-profile-updated', onProfileUpdated as EventListener);
      window.addEventListener('storage', onStorage);
    }
  });

  onUnmounted(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('ui:user-profile-updated', onProfileUpdated as EventListener);
      window.removeEventListener('storage', onStorage);
    }
  });

  watch(userId, () => {
    void reload();
  });

  return {
    loading,
    error,
    userId,
    profile,
    headerName,
    subtitle,
    avatarSrc,
    canToggleAdminRole,
    reload,
  };
}

