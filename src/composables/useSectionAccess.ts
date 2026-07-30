import { computed, ref } from 'vue';
import { apiSessionFetch, getAuthUser, getSessionToken } from './useAuthSession';
import { currentRole } from '../stores/role';
import { PORTAL_SECTION_KEYS, type PortalSectionKey } from '../pages/Admin/portalSections';
import { COURSE_CATEGORY_ITEMS, type CourseCategory } from '../pages/Courses/courseCategories';

const ALL_COURSE_CATEGORIES: CourseCategory[] = COURSE_CATEGORY_ITEMS.map((c) => c.value);

const isSuperAdmin = ref(false);
const sections = ref<PortalSectionKey[]>([]);
const courseCategories = ref<CourseCategory[]>([]);
const loaded = ref(false);
let loadPromise: Promise<void> | null = null;

function normalizeSections(raw: unknown): PortalSectionKey[] {
  if (!Array.isArray(raw)) return [];
  const allowed = new Set<string>(PORTAL_SECTION_KEYS);
  return raw
    .map((x) => String(x))
    .filter((k): k is PortalSectionKey => allowed.has(k));
}

function normalizeCourseCategories(raw: unknown): CourseCategory[] {
  if (!Array.isArray(raw)) return [];
  const allowed = new Set<string>(ALL_COURSE_CATEGORIES);
  return raw
    .map((x) => String(x))
    .filter((k): k is CourseCategory => allowed.has(k));
}

function applyFromAuthUser() {
  const u = getAuthUser() as {
    user_group?: string;
    isAdmin?: boolean;
    sections?: unknown;
    courseCategories?: unknown;
  } | null;
  if (!u) return;
  const admin = Boolean(u.isAdmin) || String(u.user_group ?? '').toLowerCase() === 'admin';
  isSuperAdmin.value = admin;
  if (Array.isArray(u.sections)) {
    sections.value = admin ? [...PORTAL_SECTION_KEYS] : normalizeSections(u.sections);
  } else if (admin) {
    sections.value = [...PORTAL_SECTION_KEYS];
  }
  if (Array.isArray(u.courseCategories)) {
    courseCategories.value = admin ? [...ALL_COURSE_CATEGORIES] : normalizeCourseCategories(u.courseCategories);
  } else if (admin) {
    courseCategories.value = [...ALL_COURSE_CATEGORIES];
  }
}

async function doLoad(): Promise<void> {
  applyFromAuthUser();
  if (!getSessionToken()) {
    loaded.value = true;
    return;
  }
  try {
    const res = await apiSessionFetch<{
      isAdmin?: boolean;
      sections?: string[];
      courseCategories?: string[];
    }>('/api/portal_my_permissions.php', { method: 'GET' });
    if (res.success && res.data) {
      isSuperAdmin.value = Boolean(res.data.isAdmin);
      sections.value = isSuperAdmin.value
        ? [...PORTAL_SECTION_KEYS]
        : normalizeSections(res.data.sections);
      courseCategories.value = isSuperAdmin.value
        ? [...ALL_COURSE_CATEGORIES]
        : normalizeCourseCategories(res.data.courseCategories);
      const u = getAuthUser() as Record<string, unknown> | null;
      if (u) {
        try {
          localStorage.setItem(
            'auth-user',
            JSON.stringify({
              ...u,
              isAdmin: isSuperAdmin.value,
              sections: sections.value,
              courseCategories: courseCategories.value,
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

  function canEditCourseCategory(category: string | null | undefined): boolean {
    if (!canEditSection('courses')) return false;
    if (isSuperAdmin.value) return currentRole.value === 'admin';
    const cat = String(category ?? '').trim();
    if (!cat) return false;
    return courseCategories.value.includes(cat as CourseCategory);
  }

  const allowedCourseCategoryItems = computed(() => {
    if (isSuperAdmin.value) {
      return currentRole.value === 'admin' ? [...COURSE_CATEGORY_ITEMS] : [];
    }
    const allowed = new Set(courseCategories.value);
    return COURSE_CATEGORY_ITEMS.filter((c) => allowed.has(c.value));
  });

  const hasAnySectionEdit = computed(() => {
    if (isSuperAdmin.value) return currentRole.value === 'admin';
    return sections.value.length > 0;
  });

  return {
    isSuperAdmin,
    sections,
    courseCategories,
    loaded,
    ensureLoaded,
    reload,
    canEdit,
    canEditSection,
    canEditCourseCategory,
    allowedCourseCategoryItems,
    hasAnySectionEdit,
  };
}
