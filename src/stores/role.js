import { ref } from 'vue';

const STORAGE_KEY = 'ui-role';

const saved = typeof localStorage !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null;

export const currentRole = ref(saved === 'admin' ? 'admin' : 'user');

export function setRole(role) {
  currentRole.value = role;
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(STORAGE_KEY, role);
  }
}

