import { ref } from 'vue';
import { defaultAvatarUrl } from './useOnboarding';

/**
 * Имя, подпись и аватар для экрана профиля — только в памяти, источник —
 * /api/profile.php. Раньше копия лежала в localStorage без привязки к
 * пользователю: после смены учётной записи в том же браузере профиль
 * показывал чужой аватар, а сохранение записывало его новому пользователю.
 */
const LEGACY_KEYS = ['ui-profile-display-name:v1', 'ui-profile-subtitle:v1', 'ui-profile-avatar-src:v1'];
if (typeof window !== 'undefined') {
  try {
    for (const key of LEGACY_KEYS) window.localStorage.removeItem(key);
  } catch {
    /* хранилище недоступно — чистить нечего */
  }
}

const displayName = ref('');
const subtitle = ref('');
const avatarSrc = ref(defaultAvatarUrl());

export function useProfileDisplay() {
  function setAvatarSrc(src: string) {
    avatarSrc.value = src.trim() || defaultAvatarUrl();
  }

  function setDisplayName(name: string) {
    displayName.value = name.trim();
  }

  function setSubtitle(text: string) {
    subtitle.value = text.trim();
  }

  return {
    displayName,
    subtitle,
    avatarSrc,
    setAvatarSrc,
    setDisplayName,
    setSubtitle,
  };
}
