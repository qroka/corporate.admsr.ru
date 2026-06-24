import { computed, ref } from 'vue';

export type OfoCategory = { id: number; name: string; sort_order: number };
export type OfoUnit = {
  id: number;
  name: string;
  category_id: number;
  parent_id: number | null;
  level: number;
  unit_number: number;
  family_number: number | null;
  sort_order: number;
  position_count?: number;
  user_count?: number;
};
export type OfoPosition = { id: number; name: string; is_head: boolean };

// ── Shared state ──────────────────────────────────────────────────────────────
const categories = ref<OfoCategory[]>([]);
const units = ref<OfoUnit[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
let loaded = false;
let loadPromise: Promise<void> | null = null;

async function doLoad(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const res = await fetch('/api/ofo_tree.php', { cache: 'no-store' });
    if (!res.ok) throw new Error(`Не удалось загрузить ОФО (${res.status})`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки ОФО');
    categories.value = json.data?.categories ?? [];
    units.value = json.data?.units ?? [];
    loaded = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки ОФО';
  } finally {
    loading.value = false;
    loadPromise = null;
  }
}

export function useOfoTree() {
  function ensureLoaded() {
    if (loaded) return;
    if (!loadPromise) loadPromise = doLoad();
  }
  async function reload() {
    loaded = false;
    if (!loadPromise) loadPromise = doLoad();
    return loadPromise;
  }

  const unitById = computed(() => {
    const m = new Map<number, OfoUnit>();
    for (const u of units.value) m.set(u.id, u);
    return m;
  });

  const childrenByParent = computed(() => {
    const m = new Map<number | null, OfoUnit[]>();
    for (const u of units.value) {
      const k = u.parent_id;
      if (!m.has(k)) m.set(k, []);
      m.get(k)!.push(u);
    }
    for (const arr of m.values()) arr.sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name, 'ru'));
    return m;
  });

  /** Корневые подразделения (level 2) категории. */
  function rootUnitsOf(categoryId: number): OfoUnit[] {
    return units.value
      .filter((u) => u.category_id === categoryId && u.parent_id === null)
      .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name, 'ru'));
  }
  function childrenOf(unitId: number): OfoUnit[] {
    return childrenByParent.value.get(unitId) ?? [];
  }
  function hasChildren(unitId: number): boolean {
    return childrenOf(unitId).length > 0;
  }

  /** Хлебные крошки: «Категория / … / Подразделение». */
  function pathLabel(unitId: number | null | undefined): string {
    if (unitId == null) return '';
    const u = unitById.value.get(Number(unitId));
    if (!u) return '';
    const parts: string[] = [u.name];
    let cur = u;
    while (cur.parent_id != null) {
      const p = unitById.value.get(cur.parent_id);
      if (!p) break;
      parts.unshift(p.name);
      cur = p;
    }
    const cat = categories.value.find((c) => c.id === u.category_id);
    if (cat) parts.unshift(cat.name);
    return parts.join(' / ');
  }

  /** unit_number по id (для запроса должностей). */
  function unitNumberOf(unitId: number | null | undefined): number | null {
    if (unitId == null) return null;
    return unitById.value.get(Number(unitId))?.unit_number ?? null;
  }

  /** Должности для подразделения (по unit_number). */
  async function fetchPositions(unitNumber: number): Promise<OfoPosition[]> {
    const res = await fetch(`/api/ofo_positions.php?unit_number=${unitNumber}`, { cache: 'no-store' });
    if (!res.ok) throw new Error(`Не удалось загрузить должности (${res.status})`);
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка загрузки должностей');
    return (json.data ?? []) as OfoPosition[];
  }

  return {
    categories,
    units,
    loading,
    error,
    ensureLoaded,
    reload,
    unitById,
    rootUnitsOf,
    childrenOf,
    hasChildren,
    pathLabel,
    unitNumberOf,
    fetchPositions,
  };
}
