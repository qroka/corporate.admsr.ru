import { ref, watch } from 'vue';

const KEY_NAME = 'ui-profile-display-name:v1';
const KEY_SUBTITLE = 'ui-profile-subtitle:v1';
const KEY_AVATAR = 'ui-profile-avatar-src:v1';

const DEFAULT_NAME = 'Константин Константинов';
const DEFAULT_SUBTITLE = 'Инженер';
const DEFAULT_AVATAR = `/img/FullPic/avatars/${encodeURIComponent('Alien.png')}`;

function readStored(key: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback;
  return window.localStorage.getItem(key) || fallback;
}

const displayName = ref(readStored(KEY_NAME, DEFAULT_NAME));
const subtitle = ref(readStored(KEY_SUBTITLE, DEFAULT_SUBTITLE));
const avatarSrc = ref(readStored(KEY_AVATAR, DEFAULT_AVATAR));

watch(displayName, (v) => {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(KEY_NAME, v);
});

watch(subtitle, (v) => {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(KEY_SUBTITLE, v);
});

watch(avatarSrc, (v) => {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(KEY_AVATAR, v);
});

export function useProfileDisplay() {
  function setAvatarSrc(src: string) {
    avatarSrc.value = src;
  }

  function setDisplayName(name: string) {
    displayName.value = name.trim() || DEFAULT_NAME;
  }

  function setSubtitle(text: string) {
    subtitle.value = text.trim() || DEFAULT_SUBTITLE;
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
