import { computed, ref } from 'vue';
import { apiSessionFetch, getAuthUser, getSessionToken } from './useAuthSession';
import { currentRole } from '../stores/role';
import { PORTAL_SECTION_KEYS, type PortalSectionKey } from '../pages/Admin/portalSections';

const isSuperAdmin = ref(false);
const sections = ref<PortalSectionKey[]>([]);
const loaded = ref(false);
let loadPromise: Promise<void> | null = null;

function normalizeSections(raw: unknown): PortalSectionKey[] {
  if (!Array.isArray(raw)) return [];
  const allowed = new Set<string>(PORTAL_SECTION_KEYS);
  return raw
    .map((x) => String(x))
    .filter((k): k is PortalSectionKey => allowed.has(k));
}

function applyFromAuthUser() {
  const u = getAuthUser() as { user_group?: string; isAdmin?: boolean; sections?: unknown } | null;
  if (!u) return;
  const admin = Boolean(u.isAdmin) || String(u.user_group ?? '').toLowerCase() === 'admin';
  isSuperAdmin.value = admin;
  if (Array.isArray(u.sections)) {
    sections.value = admin ? [...PORTAL_SECTION_KEYS] : normalizeSections(u.sections);
  } else if (admin) {
    sections.value = [...PORTAL_SECTION_KEYS];
  }
}

async function doLoad(): Promise<void> {
  applyFromAuthUser();
  if (!getSessionToken()) {
    loaded.value = true;
    return;
  }
  try {
    const res = await apiSessionFetch<{ isAdmin?: boolean; sections?: string[] }>(
      '/api/portal_my_permissions.php',
      { method: 'GET' },
    );
    if (res.success && res.data) {
      isSuperAdmin.value = Boolean(res.data.isAdmin);
      sections.value = isSuperAdmin.value
        ? [...PORTAL_SECTION_KEYS]
        : normalizeSections(res.data.sections);
      // sync into auth-user cache
      const u = getAuthUser() as Record<string, unknown> | null;
      if (u) {
        try {
          localStorage.setItem(
            'auth-user',
            JSON.stringify({
              ...u,
              isAdmin: isSuperAdmin.value,
              sections: sections.value,
            }),
          );
        } catch {
          /* ignore */
        }
      }
    }
  } catch {
    /* keep cached */
  } finally {
    loaded.value = true;
    loadPromise = null;
  }
}

export function useSectionAccess() {
  function ensureLoaded() {
    applyFromAuthUser();
    if (loaded.value && !loadPromise) return;
    if (!loadPromise) loadPromise = doLoad();
    return loadPromise;
  }

  async function reload() {
    loaded.value = false;
    loadPromise = doLoad();
    return loadPromise;
  }

  const canEdit = computed(() => {
    return (section: PortalSectionKey | string): boolean => {
      if (!PORTAL_SECTION_KEYS.includes(section as PortalSectionKey)) return false;
      if (isSuperAdmin.value) {
        return currentRole.value === 'admin';
      }
      return sections.value.includes(section as PortalSectionKey);
    };
  });

  function canEditSection(section: PortalSectionKey | string): boolean {
    return canEdit.value(section);
  }

  const hasAnySectionEdit = computed(() => {
    if (isSuperAdmin.value) return currentRole.value === 'admin';
    return sections.value.length > 0;
  });

  return {
    isSuperAdmin,
    sections,
    loaded,
    ensureLoaded,
    reload,
    canEdit,
    canEditSection,
    hasAnySectionEdit,
  };
}
