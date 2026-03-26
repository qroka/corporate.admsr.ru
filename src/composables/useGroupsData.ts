import { onMounted, ref } from 'vue';

export type GroupRow = { name: string; members: number };

export function useGroupsData() {
  const loading = ref(true);
  const error = ref<string | null>(null);
  const groups = ref<GroupRow[]>([]);

  async function load() {
    loading.value = true;
    error.value = null;
    try {
      const res = await fetch('/data/groups.json');
      if (!res.ok) throw new Error(`Не удалось загрузить groups.json: ${res.status}`);
      const data = await res.json();
      if (!Array.isArray(data)) throw new Error('groups.json должен быть массивом');
      groups.value = data
        .map((g) => ({
          name: String((g as any).name ?? '').trim(),
          members: Number((g as any).members ?? 0),
        }))
        .filter((g) => g.name);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка загрузки групп';
      groups.value = [];
    } finally {
      loading.value = false;
    }
  }

  onMounted(load);

  return { loading, error, groups, load };
}

