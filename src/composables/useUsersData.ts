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
  last_activity?: string;
  avatar_url: string;
  role: string;
};

function mapUser(r: any): AdminUserRow {
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
    last_activity: r.last_activity || '',
    avatar_url: r.avatar_url || '',
    role: r.role || ''
  } satisfies AdminUserRow;
}

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

        sharedUsers.value = json.data.map(mapUser);
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

  /** Принудительный перезапрос (для авто-поллинга в админке). */
  async function reload() {
    sharedLoaded.value = false;
    sharedLoadPromise = null;
    return load();
  }

  /**
   * Тихое обновление без флага загрузки: патчит строки на месте по id
   * (перерисовываются только изменившиеся ячейки, без морганий таблицы).
   * Массив пересоздаётся только если изменился состав пользователей.
   */
  async function refresh() {
    try {
      const res = await fetch('/api/users.php');
      if (!res.ok) return;
      const json = await res.json();
      if (!json.success || !Array.isArray(json.data)) return;

      const mapped: AdminUserRow[] = json.data.map(mapUser);
      const byId = new Map(sharedUsers.value.map((u) => [u.id, u] as const));

      let structural = mapped.length !== sharedUsers.value.length;
      for (const m of mapped) {
        const existing = byId.get(m.id);
        if (existing) Object.assign(existing, m); // мутируем на месте → точечная перерисовка
        else structural = true;
      }
      if (structural) sharedUsers.value = mapped;
    } catch {
      /* офлайн — оставляем текущие данные */
    }
  }

  onMounted(() => {});

  return { loading, error, users, load, ensureLoaded, reload, refresh };
}

// module-scope shared state (keeps data between composable calls)
const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedUsers = ref<AdminUserRow[]>([]);
const sharedLoaded = ref(false);
let sharedLoadPromise: Promise<void> | null = null;
