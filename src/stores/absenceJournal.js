import { ref } from 'vue';

const STORAGE_KEY = 'absence-journal:has-active';

function readStored() {
  if (typeof localStorage === 'undefined') return false;
  return localStorage.getItem(STORAGE_KEY) === '1';
}

export const hasActiveAbsence = ref(readStored());

export function setHasActiveAbsence(value) {
  const next = Boolean(value);
  hasActiveAbsence.value = next;
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(STORAGE_KEY, next ? '1' : '0');
  }
}

let storageListenerAttached = false;
export function attachAbsenceStorageSync() {
  if (storageListenerAttached) return;
  if (typeof window === 'undefined') return;
  storageListenerAttached = true;

  window.addEventListener('storage', (e) => {
    if (e.key !== STORAGE_KEY) return;
    hasActiveAbsence.value = readStored();
  });
}

