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
  status: string;
  login: string;
  password?: string;
  firstname: string;
  surname: string;
  lastname: string;
  fullName: string;
  ofo: string;
  user_group: string;
  phone: string;
  email: string;
  auth: string;
  avatar_url: string;
  role: string;
};

export function useUsersData() {
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
        const res = await fetch('/api/users.php');
        if (!res.ok) throw new Error(`Не удалось загрузить пользователей: ${res.status}`);
        const json = await res.json();
        if (!json.success) throw new Error(json.message || 'Ошибка загрузки пользователей');
        
        sharedUsers.value = json.data.map((r: any) => {
          const id = Number(r.id);
          const fname = r.firstname || '';
          const sname = r.surname || '';
          const lname = r.lastname || '';
          const fullName = [sname, fname, lname].filter(Boolean).join(' ') || '—';
          
          return {
            id: Number.isFinite(id) ? id : 0,
            status: r.status === 'Активен' || r.status === '1' ? 'Активен' : 'Заблокирован',
            login: r.login || '',
            password: r.password || '',
            firstname: fname,
            surname: sname,
            lastname: lname,
            fullName,
            ofo: String(r.ofo || '').trim(),
            user_group: r.user_group || '',
            phone: r.phone || '',
            email: r.email || '',
            auth: r.auth || '',
            avatar_url: r.avatar_url || '',
            role: r.role || ''
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

  onMounted(() => {});

  return { loading, error, users, load, ensureLoaded };
}

// module-scope shared state (keeps data between composable calls)
const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedUsers = ref<AdminUserRow[]>([]);
const sharedLoaded = ref(false);
let sharedLoadPromise: Promise<void> | null = null;
