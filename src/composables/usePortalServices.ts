import { computed, ref } from 'vue';
import { apiSessionFetch, type ApiResult } from './useAuthSession';

export type PortalServiceKind = 'internal' | 'external';

export type PortalService = {
  id: number;
  kind: PortalServiceKind;
  label: string;
  description: string;
  icon: string;
  internalKey?: string | null;
  path?: string | null;
  externalUrl?: string | null;
  sortOrder: number;
  isEnabled: boolean;
};

export type PortalServiceInput = {
  id?: number;
  kind: PortalServiceKind;
  label: string;
  description?: string;
  icon?: string;
  internalKey?: string | null;
  path?: string | null;
  externalUrl?: string | null;
  sortOrder?: number;
  isEnabled?: boolean;
};

/** Fallback, если API недоступен / таблица ещё не создана. */
export const FALLBACK_PORTAL_SERVICES: PortalService[] = [
  {
    id: -1,
    kind: 'internal',
    label: 'Журнал отсутствия',
    description: 'Отметки об отсутствии сотрудника',
    icon: 'i-lucide-calendar-off',
    internalKey: 'absence-journal',
    path: '/absence-journal',
    sortOrder: 10,
    isEnabled: true,
  },
  {
    id: -2,
    kind: 'internal',
    label: 'Формы',
    description: 'Опросы, анкеты и тесты',
    icon: 'i-lucide-clipboard-list',
    internalKey: 'tests',
    path: '/tests',
    sortOrder: 20,
    isEnabled: true,
  },
  {
    id: -3,
    kind: 'internal',
    label: 'Заявки',
    description: 'Подача и отслеживание сервисных заявок',
    icon: 'i-lucide-file-text',
    internalKey: 'applications',
    path: '/applications',
    sortOrder: 30,
    isEnabled: true,
  },
];

const services = ref<PortalService[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const loaded = ref(false);
let loadPromise: Promise<void> | null = null;

function unwrap<T>(res: ApiResult<T>, fallbackMsg: string): T {
  if (!res.success) throw new Error(res.message || fallbackMsg);
  return res.data as T;
}

function normalizeItem(raw: any): PortalService {
  return {
    id: Number(raw.id),
    kind: raw.kind === 'external' ? 'external' : 'internal',
    label: String(raw.label ?? ''),
    description: String(raw.description ?? ''),
    icon: String(raw.icon || 'i-lucide-layout-grid'),
    internalKey: raw.internalKey ?? null,
    path: raw.path ?? null,
    externalUrl: raw.externalUrl ?? null,
    sortOrder: Number(raw.sortOrder ?? 0),
    isEnabled: raw.isEnabled !== false,
  };
}

export function usePortalServices() {
  const enabledServices = computed(() => services.value.filter((s) => s.isEnabled));
  const internalEnabled = computed(() =>
    enabledServices.value.filter((s) => s.kind === 'internal' && s.path),
  );

  async function ensureLoaded(opts?: { all?: boolean }) {
    if (loaded.value && !opts?.all && !loadPromise) return;
    if (loadPromise && !opts?.all) return loadPromise;
    loadPromise = doLoad(opts?.all === true);
    return loadPromise;
  }

  async function doLoad(all: boolean) {
    loading.value = true;
    error.value = null;
    try {
      const url = all ? '/api/portal_services.php?all=1' : '/api/portal_services.php';
      const res = await apiSessionFetch<PortalService[]>(url, { method: 'GET' });
      const data = unwrap(res, 'Не удалось загрузить сервисы');
      const list = Array.isArray(data) ? data.map(normalizeItem) : [];
      services.value = list.length ? list : [...FALLBACK_PORTAL_SERVICES];
    } catch (e: any) {
      error.value = e?.message || 'Ошибка загрузки';
      if (!services.value.length) {
        services.value = [...FALLBACK_PORTAL_SERVICES];
      }
    } finally {
      loading.value = false;
      loaded.value = true;
      loadPromise = null;
    }
  }

  async function reload(all = false) {
    loaded.value = false;
    return ensureLoaded({ all });
  }

  async function createService(payload: PortalServiceInput) {
    const res = await apiSessionFetch<PortalService>('/api/portal_services.php', {
      method: 'POST',
      json: payload,
    });
    const created = normalizeItem(unwrap(res, 'Не удалось создать сервис'));
    await reload(true);
    return created;
  }

  async function updateService(payload: PortalServiceInput & { id: number }) {
    const res = await apiSessionFetch<PortalService>('/api/portal_services.php', {
      method: 'PUT',
      json: payload,
    });
    const updated = normalizeItem(unwrap(res, 'Не удалось сохранить сервис'));
    await reload(true);
    return updated;
  }

  async function deleteService(id: number) {
    const res = await apiSessionFetch('/api/portal_services.php', {
      method: 'DELETE',
      json: { id },
    });
    unwrap(res, 'Не удалось удалить сервис');
    await reload(true);
  }

  async function reorderServices(ids: number[]) {
    const res = await apiSessionFetch('/api/portal_services.php', {
      method: 'POST',
      json: { action: 'reorder', ids },
    });
    unwrap(res, 'Не удалось сохранить порядок');
    await reload(true);
  }

  return {
    services,
    enabledServices,
    internalEnabled,
    loading,
    error,
    loaded,
    ensureLoaded,
    reload,
    createService,
    updateService,
    deleteService,
    reorderServices,
  };
}
