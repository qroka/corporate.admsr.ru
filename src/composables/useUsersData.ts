import { onMounted, ref } from 'vue';

function extractTableData(exportJson: unknown, tableName: string) {
  if (!Array.isArray(exportJson)) return [];
  for (const item of exportJson) {
    if (item && typeof item === 'object') {
      const maybe = item as { type?: string; name?: string; data?: unknown[] };
      if (maybe.type === 'table' && maybe.name === tableName && Array.isArray(maybe.data)) {
        return maybe.data;
      }
    }
  }
  return [];
}

export type AdminUserRow = {
  id: number;
  status: 'Активен' | 'Заблокирован';
  fullName: string;
  login: string;
  password: string;
  ofo: string;
  email: string;
  lastAuthAt: string;
};

function formatLastAuth(ts: unknown): string {
  if (ts == null || ts === '') return '—';
  const n = Number(ts);
  if (Number.isNaN(n) || n <= 0) return '—';
  return new Date(n * 1000).toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function useUsersData() {
  // shared cache (single fetch/parse per session)
  const loading = sharedLoading;
  const error = sharedError;
  const users = sharedUsers;

  async function load() {
    if (sharedLoaded.value) return;
    if (sharedLoadPromise) return sharedLoadPromise;

    sharedLoading.value = true;
    sharedError.value = null;

    sharedLoadPromise = (async () => {
      try {
        const res = await fetch('/data/users.json');
        if (!res.ok) throw new Error(`Не удалось загрузить users.json: ${res.status}`);
        const data = await res.json();
        const rows = extractTableData(data, 'users') as Record<string, unknown>[];
        sharedUsers.value = rows.map((r) => {
          const id = Number(r.id);
          return {
            id: Number.isFinite(id) ? id : 0,
            status: String(r.active ?? '0') === '1' ? ('Активен' as const) : ('Заблокирован' as const),
            fullName: String(r.fio ?? '').trim() || '—',
            login: String(r.username ?? ''),
            password: '••••••••',
            ofo: String(r.ofo ?? '').trim(),
            email: String(r.email ?? '').trim() || '—',
            lastAuthAt: formatLastAuth(r.last_login),
          } satisfies AdminUserRow;
        });
        sharedLoaded.value = true;
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки пользователей';
        sharedUsers.value = [];
      } finally {
        sharedLoading.value = false;
        sharedLoadPromise = null;
      }
    })();

    return sharedLoadPromise;
  }

  function ensureLoaded() {
    void load();
  }

  onMounted(() => {
    // no auto-load by default; caller can call ensureLoaded()
  });

  return { loading, error, users, load, ensureLoaded };
}

// module-scope shared state (keeps data between composable calls)
const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedUsers = ref<AdminUserRow[]>([]);
const sharedLoaded = ref(false);
let sharedLoadPromise: Promise<void> | null = null;
