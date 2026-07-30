import { ref } from 'vue';
import { apiSessionFetch } from './useAuthSession';
import type { PortalSectionKey } from '../pages/Admin/portalSections';

export type PortalGroupMember = {
  id: number;
  fio: string;
  login: string;
};

export type PortalGroup = {
  id: number;
  name: string;
  description: string;
  permissions: PortalSectionKey[];
  memberIds?: number[];
  members?: PortalGroupMember[];
  memberCount: number;
  createdAt?: string | null;
  updatedAt?: string | null;
};

export type PortalGroupPayload = {
  name: string;
  description?: string;
  permissions: string[];
  memberIds: number[];
};

const groups = ref<PortalGroup[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

function unwrap<T>(res: { success?: boolean; data?: T; message?: string }, fallback: string): T {
  if (!res?.success) throw new Error(res?.message || fallback);
  return res.data as T;
}

export function useGroupsData() {
  async function load() {
    loading.value = true;
    error.value = null;
    try {
      const res = await apiSessionFetch<PortalGroup[]>('/api/portal_groups.php', { method: 'GET' });
      groups.value = unwrap(res, 'Не удалось загрузить группы') || [];
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка загрузки групп';
      groups.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function getGroup(id: number): Promise<PortalGroup> {
    const res = await apiSessionFetch<PortalGroup>(`/api/portal_groups.php?id=${id}`, { method: 'GET' });
    return unwrap(res, 'Не удалось загрузить группу');
  }

  async function createGroup(payload: PortalGroupPayload): Promise<PortalGroup> {
    const res = await apiSessionFetch<PortalGroup>('/api/portal_groups.php', {
      method: 'POST',
      json: payload,
    });
    const g = unwrap(res, 'Не удалось создать группу');
    await load();
    return g;
  }

  async function updateGroup(id: number, payload: PortalGroupPayload): Promise<PortalGroup> {
    const res = await apiSessionFetch<PortalGroup>(`/api/portal_groups.php?id=${id}`, {
      method: 'PUT',
      json: payload,
    });
    const g = unwrap(res, 'Не удалось сохранить группу');
    await load();
    return g;
  }

  async function deleteGroup(id: number): Promise<void> {
    const res = await apiSessionFetch(`/api/portal_groups.php?id=${id}`, { method: 'DELETE' });
    unwrap(res, 'Не удалось удалить группу');
    await load();
  }

  return {
    groups,
    loading,
    error,
    load,
    getGroup,
    createGroup,
    updateGroup,
    deleteGroup,
  };
}
