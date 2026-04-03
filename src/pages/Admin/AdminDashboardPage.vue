<script setup lang="ts">
import { computed, h, nextTick, onUnmounted, reactive, ref, resolveComponent, watch } from 'vue';
import type { TableColumn } from '@nuxt/ui';
import { useAppToast } from '../../composables/useAppToast';
import { useOfoData, type OfoStat } from '../../composables/useOfoData';
import { useGroupsData } from '../../composables/useGroupsData';
import { useUsersData, type AdminUserRow } from '../../composables/useUsersData';
import { currentRole } from '../../stores/role';

const { toast, adminUserSaved, adminOfoSaved } = useAppToast();

const UBadge = resolveComponent('UBadge');
const UButton = resolveComponent('UButton');
const UDropdownMenu = resolveComponent('UDropdownMenu');

const isAdmin = computed(() => currentRole.value === 'admin');

const {
  loading: ofoLoading,
  error: ofoError,
  ofoStats,
  officeSeats,
  ensureDirectoryLoaded: ensureOfoDirectoryLoaded,
  ensureSeatsLoaded: ensureOfoSeatsLoaded,
} = useOfoData();
const {
  loading: usersLoading,
  error: usersError,
  users,
  ensureLoaded: ensureUsersLoaded,
} = useUsersData();

const { loading: groupsLoading, error: groupsError, groups } = useGroupsData();

watch(
  usersError,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить пользователей',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

watch(
  ofoError,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить ОФО',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

watch(
  groupsError,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить группы',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

type OfoViewMode = 'cards' | 'table';
const ofoViewMode = ref<OfoViewMode>('cards');
const ofoViewItems = [
  { label: 'Карточки', value: 'cards' },
  { label: 'Таблица', value: 'table' },
];

type OfoSort = 'title_asc' | 'title_desc' | 'seats_desc' | 'seats_asc';
const ofoQuery = ref('');
const ofoSort = ref<OfoSort>('title_asc');
const ofoSortItems = [
  { label: 'Название (А→Я)', value: 'title_asc' },
  { label: 'Название (Я→А)', value: 'title_desc' },
  { label: 'Должностей (больше → меньше)', value: 'seats_desc' },
  { label: 'Должностей (меньше → больше)', value: 'seats_asc' },
];

type OfoTreeNode = {
  stat: OfoStat;
  children: OfoTreeNode[];
};

type OfoFlatRow = OfoStat & { depth: number; hasChildren: boolean };

function compareOfo(a: OfoStat, b: OfoStat, sort: OfoSort): number {
  switch (sort) {
    case 'title_desc':
      return b.title.localeCompare(a.title, 'ru');
    case 'seats_desc':
      return b.seatsCount - a.seatsCount || a.title.localeCompare(b.title, 'ru');
    case 'seats_asc':
      return a.seatsCount - b.seatsCount || a.title.localeCompare(b.title, 'ru');
    default:
      return a.title.localeCompare(b.title, 'ru');
  }
}

/** Поиск: оставляем совпадения, их предков (путь) и потомков (поддерево). */
function collectOfoSearchAllowedIds(stats: OfoStat[], query: string): Set<string> {
  const allIds = new Set(stats.map((s) => s.id));
  const q = query.trim().toLowerCase();
  if (!q) return allIds;

  const byId = new Map(stats.map((s) => [s.id, s] as const));
  const matches = stats.filter((s) =>
    [s.id, s.title, s.parentTitle ?? ''].join(' ').toLowerCase().includes(q),
  );
  const allowed = new Set<string>();

  const childrenByParent = new Map<string, string[]>();
  for (const s of stats) {
    let pid = s.parentId ?? '0';
    if (!allIds.has(pid)) pid = '0';
    if (!childrenByParent.has(pid)) childrenByParent.set(pid, []);
    childrenByParent.get(pid)!.push(s.id);
  }

  function addDescendants(id: string) {
    const ch = childrenByParent.get(id) ?? [];
    for (const c of ch) {
      if (!allowed.has(c)) {
        allowed.add(c);
        addDescendants(c);
      }
    }
  }

  for (const m of matches) {
    allowed.add(m.id);
    addDescendants(m.id);
    let p: string | undefined = m.parentId;
    while (p) {
      allowed.add(p);
      const parent = byId.get(p);
      p = parent?.parentId;
    }
  }

  return allowed;
}

function buildOfoTree(stats: OfoStat[]): OfoTreeNode[] {
  const ids = new Set(stats.map((s) => s.id));
  const childrenByParent = new Map<string, OfoStat[]>();

  for (const s of stats) {
    let pid = s.parentId ?? '0';
    if (!ids.has(pid)) pid = '0';
    if (!childrenByParent.has(pid)) childrenByParent.set(pid, []);
    childrenByParent.get(pid)!.push(s);
  }

  function walk(parentId: string): OfoTreeNode[] {
    const list = childrenByParent.get(parentId) ?? [];
    return list.map((stat) => ({
      stat,
      children: walk(stat.id),
    }));
  }

  return walk('0');
}

function sortOfoTree(nodes: OfoTreeNode[], sort: OfoSort): OfoTreeNode[] {
  const sorted = [...nodes].sort((a, b) => compareOfo(a.stat, b.stat, sort));
  return sorted.map((n) => ({
    ...n,
    children: sortOfoTree(n.children, sort),
  }));
}

function collectOfoIdsWithChildren(nodes: OfoTreeNode[]): string[] {
  const ids: string[] = [];
  for (const n of nodes) {
    if (n.children.length) {
      ids.push(n.stat.id, ...collectOfoIdsWithChildren(n.children));
    }
  }
  return ids;
}

/** Только развёрнутые ветки: строка есть всегда, дети — если id в expandedIds. */
function flattenOfoTreeVisible(
  nodes: OfoTreeNode[],
  expandedIds: Set<string>,
  depth = 0,
): OfoFlatRow[] {
  const out: OfoFlatRow[] = [];
  for (const n of nodes) {
    const hasChildren = n.children.length > 0;
    out.push({ ...n.stat, depth, hasChildren });
    if (hasChildren && expandedIds.has(n.stat.id)) {
      out.push(...flattenOfoTreeVisible(n.children, expandedIds, depth + 1));
    }
  }
  return out;
}

const ofoSearchAllowedIds = computed(() => collectOfoSearchAllowedIds(ofoStats.value, ofoQuery.value));

const ofoFilteredStats = computed(() => {
  const allowed = ofoSearchAllowedIds.value;
  return ofoStats.value.filter((s) => allowed.has(s.id));
});

const ofoTreeSorted = computed(() => sortOfoTree(buildOfoTree(ofoFilteredStats.value), ofoSort.value));

const ofoExpandedIds = ref<Set<string>>(new Set());

// Performance: do not auto-expand the whole tree on every sort/search change.
// Expand once after initial OFO load; then keep user-controlled state.
const ofoAutoExpandedOnce = ref(false);
watch(
  [ofoLoading, ofoTreeSorted],
  ([loading, tree]) => {
    if (loading) return;
    if (ofoAutoExpandedOnce.value) return;
    ofoExpandedIds.value = new Set(collectOfoIdsWithChildren(tree));
    ofoAutoExpandedOnce.value = true;
  },
  { immediate: true },
);

function toggleOfoExpand(id: string) {
  const next = new Set(ofoExpandedIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  ofoExpandedIds.value = next;
}

function ofoExpandAll() {
  ofoExpandedIds.value = new Set(collectOfoIdsWithChildren(ofoTreeSorted.value));
}

function ofoCollapseAll() {
  ofoExpandedIds.value = new Set();
}

/** Навигация по карточкам: путь от корня — массив id узлов. */
const ofoCardPath = ref<string[]>([]);

watch(ofoQuery, () => {
  ofoCardPath.value = [];
});

const ofoCardBreadcrumbs = computed(() => {
  const crumbs: { id: string; title: string }[] = [{ id: '0', title: 'Все ОФО' }];
  let level = ofoTreeSorted.value;
  for (const id of ofoCardPath.value) {
    const node = level.find((n) => n.stat.id === id);
    if (!node) break;
    crumbs.push({ id: node.stat.id, title: node.stat.title });
    level = node.children;
  }
  return crumbs;
});

const ofoCardCurrentLevel = computed<OfoTreeNode[]>(() => {
  let level = ofoTreeSorted.value;
  for (const id of ofoCardPath.value) {
    const node = level.find((n) => n.stat.id === id);
    if (!node) return [];
    level = node.children;
  }
  return level;
});

function ofoCardEnter(node: OfoTreeNode) {
  if (node.children.length) {
    ofoCardPath.value = [...ofoCardPath.value, node.stat.id];
  }
}

function ofoCardGoToBreadcrumb(index: number) {
  ofoCardPath.value = ofoCardPath.value.slice(0, index);
}

const ofoTableRows = computed(() =>
  flattenOfoTreeVisible(ofoTreeSorted.value, ofoExpandedIds.value),
);

function seatsForOfo(ofoId: string) {
  return officeSeats.value.filter((s) => s.ofo === ofoId);
}

type OfoActionTarget = Pick<OfoStat, 'id' | 'title'> & { parentTitle?: string };

const ofoEditOpen = ref(false);
const ofoEditForm = reactive({
  id: '',
  title: '',
  parentTitle: '',
});

function openOfoEdit(o: OfoActionTarget) {
  Object.assign(ofoEditForm, {
    id: o.id,
    title: o.title,
    parentTitle: o.parentTitle ?? '',
  });
  ofoEditOpen.value = true;
}

function saveOfoEdit() {
  adminOfoSaved(ofoEditForm.title);
  ofoEditOpen.value = false;
}

const ofoPositionsOpen = ref(false);
const ofoPositionsContext = ref<{ id: string; title: string } | null>(null);

const seatEdits = reactive<Record<string, { title: string; priority: number }>>({});

function ensureSeatEditDefaults(rows: Array<{ id: string; title: string }>) {
  rows.forEach((r, idx) => {
    if (!seatEdits[r.id]) {
      seatEdits[r.id] = { title: r.title, priority: idx + 1 };
    }
  });
}

function setSeatTitle(id: string, title: string) {
  if (!seatEdits[id]) seatEdits[id] = { title, priority: 1 };
  seatEdits[id].title = title;
}

function setSeatPriority(id: string, priority: number) {
  const p = Number.isFinite(priority) && priority > 0 ? Math.floor(priority) : 1;
  if (!seatEdits[id]) seatEdits[id] = { title: '', priority: p };
  seatEdits[id].priority = p;
}

const ofoPositionsList = computed(() => {
  if (!ofoPositionsContext.value) return [];
  const base = seatsForOfo(ofoPositionsContext.value.id);
  ensureSeatEditDefaults(base);
  return [...base]
    .map((r) => ({
      id: r.id,
      title: seatEdits[r.id]?.title ?? r.title,
      priority: seatEdits[r.id]?.priority ?? 1,
    }))
    .sort((a, b) => a.priority - b.priority || a.title.localeCompare(b.title, 'ru'));
});

function openOfoPositions(o: OfoActionTarget) {
  ofoPositionsContext.value = { id: o.id, title: o.title };
  ofoPositionsOpen.value = true;
}

function ofoActionItems(o: OfoActionTarget) {
  return [
    {
      label: 'Редактировать',
      icon: 'i-lucide-pencil',
      onSelect() {
        openOfoEdit(o);
      },
    },
    {
      label: 'Просмотреть должности',
      icon: 'i-lucide-briefcase',
      onSelect() {
        openOfoPositions(o);
      },
    },
  ];
}

function saveOfoPositions() {
  const title = ofoPositionsContext.value?.title ?? 'ОФО';
  adminOfoSaved(`Должности: ${title}`);
  ofoPositionsOpen.value = false;
}

const tabItems = [
  { label: 'Пользователи', value: 'users' },
  { label: 'Группы пользователей', value: 'groups' },
  { label: 'ОФО', value: 'ofo' },
];
const tab = ref<(typeof tabItems)[number]['value']>('users');

// Lazy-load heavy JSON only when the tab is opened.
watch(
  [tab, isAdmin],
  ([t, admin]) => {
    if (!admin) return;
    if (t === 'users') {
      ensureUsersLoaded();
      // Needed to render OFO titles in users table immediately.
      ensureOfoDirectoryLoaded();
    }
    if (t === 'ofo') {
      ensureOfoDirectoryLoaded();
      ensureOfoSeatsLoaded();
    }
  },
  { immediate: true },
);

/** Сколько пользователей привязано к ОФО (поле `ofo` в users.json = id ОФО). */
function countUsersForOfo(ofoId: string, title: string): number {
  const t = title.trim();
  return users.value.filter((u) => u.ofo === ofoId || u.ofo === t).length;
}

const userSearchQuery = ref('');
const filterStatus = ref('');
const filterOfo = ref('');

const statusFilterItems = [
  { value: '', label: 'Все статусы' },
  { value: 'Активен', label: 'Активен' },
  { value: 'Заблокирован', label: 'Заблокирован' },
];

const ofoTitleById = computed(() => {
  const m: Record<string, string> = {};
  for (const s of ofoStats.value) m[s.id] = s.title;
  return m;
});

const ofoFilterItems = computed(() => {
  const uniq = [...new Set(users.value.map((u) => u.ofo))].sort((a, b) => a.localeCompare(b, 'ru'));
  return [
    { value: '', label: 'Все ОФО' },
    ...uniq.map((o) => {
      const title = ofoTitleById.value[o];
      return { value: o, label: title ? `${o} — ${title}` : o };
    }),
  ];
});

const filteredUsers = computed(() => {
  let list = users.value;

  if (filterStatus.value) {
    list = list.filter((u) => u.status === filterStatus.value);
  }
  if (filterOfo.value) {
    list = list.filter((u) => u.ofo === filterOfo.value);
  }

  const q = userSearchQuery.value.trim().toLowerCase();
  if (q) {
    list = list.filter((u) => {
      const hay = [
        String(u.id),
        u.status,
        u.fullName,
        u.ofo,
        u.lastAuthAt,
      ]
        .join(' ')
        .toLowerCase();
      return hay.includes(q);
    });
  }

  return list;
});

// Infinite scroll for users table: render rows in chunks to avoid heavy initial render.
const USERS_PAGE_SIZE = 100;
const usersVisibleCount = ref(USERS_PAGE_SIZE);
const usersVisible = computed(() => filteredUsers.value.slice(0, usersVisibleCount.value));
const usersScrollAreaRef = ref<any>(null);
const usersInfiniteLoading = ref(false);

function canLoadMoreUsers() {
  return !usersLoading.value && usersVisibleCount.value < filteredUsers.value.length;
}

async function loadMoreUsers() {
  if (!canLoadMoreUsers()) return;
  if (usersInfiniteLoading.value) return;
  usersInfiniteLoading.value = true;
  // Yield to let the browser paint before growing the DOM.
  await nextTick();
  usersVisibleCount.value = Math.min(
    filteredUsers.value.length,
    usersVisibleCount.value + USERS_PAGE_SIZE,
  );
  usersInfiniteLoading.value = false;
}

function handleUsersScroll() {
  if (!canLoadMoreUsers()) return;
  const el = usersScrollAreaRef.value?.$el as HTMLElement | undefined;
  if (!el) return;

  const distance = 240;
  const reachedBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - distance;
  if (reachedBottom) void loadMoreUsers();
}

let usersScrollCleanup: (() => void) | null = null;

async function attachUsersInfiniteScroll() {
  usersScrollCleanup?.();
  usersScrollCleanup = null;

  await nextTick();
  const el = usersScrollAreaRef.value?.$el as HTMLElement | undefined;
  if (!el) return;

  const onScroll = () => handleUsersScroll();
  el.addEventListener('scroll', onScroll, { passive: true });
  usersScrollCleanup = () => el.removeEventListener('scroll', onScroll);

  // In case the first chunk doesn't fill the viewport.
  handleUsersScroll();
}

watch(
  [tab, isAdmin, usersLoading],
  ([t, admin, loading]) => {
    if (!admin) return;
    if (t !== 'users') return;
    if (loading) return;
    void attachUsersInfiniteScroll();
  },
  { immediate: true },
);

watch(
  filteredUsers,
  () => {
    usersVisibleCount.value = USERS_PAGE_SIZE;
    void nextTick(() => {
      const el = usersScrollAreaRef.value?.$el as HTMLElement | undefined;
      if (el) el.scrollTop = 0;
      handleUsersScroll();
    });
  },
  { immediate: true },
);

onUnmounted(() => {
  usersScrollCleanup?.();
  usersScrollCleanup = null;
});

function resetUserFilters() {
  userSearchQuery.value = '';
  filterStatus.value = '';
  filterOfo.value = '';
}

function toggleUserStatus(user: AdminUserRow) {
  const next: AdminUserRow['status'] = user.status === 'Активен' ? 'Заблокирован' : 'Активен';
  users.value = users.value.map((u) => (u.id === user.id ? { ...u, status: next } : u));
}

const editOpen = ref(false);
const editForm = reactive<AdminUserRow>({
  id: 0,
  status: 'Активен',
  fullName: '',
  login: '',
  password: '••••••••',
  ofo: '',
  email: '',
  lastAuthAt: '',
});

function openEdit(user: AdminUserRow) {
  Object.assign(editForm, { ...user });
  editOpen.value = true;
}

function saveEdit() {
  users.value = users.value.map((u) =>
    u.id === editForm.id ? { ...editForm } : u,
  );
  adminUserSaved(editForm.fullName);
  editOpen.value = false;
}

function impersonate(user: AdminUserRow) {
  window.alert(
    `Имитация входа под пользователем «${user.fullName}» (${user.login}). В продакшене здесь будет серверный токен/сессия.`,
  );
}

type BadgeColor = 'primary' | 'success' | 'warning' | 'info' | 'neutral';

const ofoDepthById = computed(() => {
  // depth = 0 for top-level nodes (no parent in dataset / parentId missing)
  const byId = new Map(ofoStats.value.map((s) => [s.id, s] as const));
  const cache = new Map<string, number>();

  function computeDepth(id: string): number {
    if (cache.has(id)) return cache.get(id)!;

    const visited = new Set<string>();
    let depth = 0;
    let cur: string | undefined = id;

    while (cur) {
      if (visited.has(cur)) break; // guard cycles
      visited.add(cur);

      const node = byId.get(cur);
      const parentId = node?.parentId;
      if (!parentId || !byId.has(parentId)) break;

      depth += 1;
      cur = parentId;

      // safety: unrealistic depth guard
      if (depth > 50) break;
    }

    cache.set(id, depth);
    return depth;
  }

  const out: Record<string, number> = {};
  for (const s of ofoStats.value) out[s.id] = computeDepth(s.id);
  return out;
});

function ofoBadgeColorByDepth(depth: number): BadgeColor {
  const palette: BadgeColor[] = ['primary', 'success', 'warning', 'info', 'neutral'];
  const d = Number.isFinite(depth) && depth >= 0 ? Math.floor(depth) : 0;
  return palette[d % palette.length] ?? 'neutral';
}

/** Цвет бейджа ОФО зависит от уровня вложенности. */
function ofoBadgeColor(ofoId: string, depthOverride?: number): BadgeColor {
  const id = (ofoId ?? '').trim();
  if (!id || id === '-1' || id === '0') return 'neutral';
  const depth = depthOverride ?? ofoDepthById.value[id] ?? 0;
  return ofoBadgeColorByDepth(depth);
}

function rowActionItems(user: AdminUserRow) {
  return [
    {
      label: user.status === 'Активен' ? 'Отключить пользователя' : 'Включить пользователя',
      icon: user.status === 'Активен' ? 'i-lucide-user-x' : 'i-lucide-user-check',
      onSelect() {
        toggleUserStatus(user);
      },
    },
    {
      label: 'Редактировать информацию',
      icon: 'i-lucide-pencil',
      onSelect() {
        openEdit(user);
      },
    },
    {
      label: 'Авторизоваться под ним',
      icon: 'i-lucide-log-in',
      onSelect() {
        impersonate(user);
      },
    },
  ];
}

const userColumns: TableColumn<AdminUserRow>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'status',
    header: 'Статус',
    cell: ({ row }) => {
      const s = row.getValue('status') as AdminUserRow['status'];
      const color = s === 'Активен' ? 'success' : 'error';
      return h(UBadge, { variant: 'subtle', color }, () => s);
    },
  },
  { accessorKey: 'fullName', header: 'ФИО' },
  {
    accessorKey: 'ofo',
    header: 'ОФО',
    cell: ({ row }) => {
      const id = row.getValue('ofo') as string;
      const title = ofoTitleById.value[id];
      const label = title
        ? title
        : (!id || id === '-1' || id === '0')
            ? 'Не указано'
            : `ОФО #${id}`;
      return h(UBadge, {
        variant: 'subtle',
        color: ofoBadgeColor(id),
        leading: true,
        leadingIcon: 'i-lucide-building-2',
        class: 'max-w-[min(340px,100%)] min-w-0 truncate',
        title: title ? `${id} — ${title}` : (id ? `ID: ${id}` : 'Не указано'),
      }, () => label);
    },
  },
  { accessorKey: 'lastAuthAt', header: 'Последняя авторизация' },
  {
    id: 'actions',
    header: () => h('span', { class: 'sr-only' }, 'Действия'),
    enableHiding: false,
    meta: {
      class: {
        th: 'w-14 text-right',
        td: 'text-right',
      },
    },
    cell: ({ row }) => {
      const user = row.original as AdminUserRow;
      return h(
        UDropdownMenu,
        {
          content: { align: 'end' },
          items: rowActionItems(user),
          'aria-label': 'Действия с пользователем',
        },
        () =>
          h(UButton, {
            icon: 'i-lucide-ellipsis-vertical',
            color: 'neutral',
            variant: 'ghost',
            square: true,
            size: 'sm',
            'aria-label': 'Действия',
          }),
      );
    },
  },
];

const ofoColumns: TableColumn<OfoFlatRow>[] = [
  {
    accessorKey: 'id',
    header: 'ID',
    meta: {
      class: {
        th: 'w-[96px]',
        td: 'font-mono text-sm tabular-nums',
      },
    },
    cell: ({ row }) => {
      const id = row.getValue('id') as string;
      return h('span', { class: 'text-muted' }, id);
    },
  },
  {
    accessorKey: 'title',
    header: 'ОФО',
    cell: ({ row }) => {
      const o = row.original as OfoFlatRow;
      const title = o.title;
      const seats = o.seatsCount;
      const depth = o.depth;
      const hasChildren = o.hasChildren;
      const expanded = ofoExpandedIds.value.has(o.id);
      const toggle = () => toggleOfoExpand(o.id);

      const indent = h('span', { class: 'inline-block shrink-0', style: { width: `${depth * 14}px` } });

      const expander = hasChildren
        ? h(UButton, {
            color: 'neutral',
            variant: 'ghost',
            square: true,
            size: 'xs',
            icon: expanded ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right',
            'aria-label': expanded ? 'Свернуть' : 'Развернуть',
            'aria-expanded': expanded,
            onClick: (e: Event) => {
              e.stopPropagation();
              toggle();
            },
          })
        : h('span', { class: 'inline-flex w-8 shrink-0', 'aria-hidden': 'true' });

      return h(
        'div',
        {
          class: 'flex items-center gap-1 min-w-0',
        },
        [
          indent,
          expander,
          h(
            UBadge,
            {
              variant: 'subtle',
              color: ofoBadgeColor(o.id, depth),
              leading: true,
              leadingIcon: 'i-lucide-building-2',
              class: 'max-w-[min(360px,100%)] min-w-0 truncate',
              title,
            },
            () => title,
          ),
        ],
      );
    },
  },
  {
    accessorKey: 'seatsCount',
    header: 'Должностей',
    cell: ({ row }) => {
      const v = row.getValue('seatsCount') as number;
      return h('span', { class: 'font-medium text-highlighted' }, String(v));
    },
  },
  {
    id: 'usersCount',
    header: 'Пользователей',
    cell: ({ row }) => {
      const o = row.original as OfoFlatRow;
      const n = countUsersForOfo(o.id, o.title);
      return h(UBadge, { variant: 'subtle', color: n ? 'info' : 'neutral', leading: true, leadingIcon: 'i-lucide-user' }, () =>
        String(n),
      );
    },
  },
  {
    id: 'ofoActions',
    header: () => h('span', { class: 'sr-only' }, 'Действия'),
    enableHiding: false,
    meta: {
      class: {
        th: 'w-14 text-right',
        td: 'text-right',
      },
    },
    cell: ({ row }) => {
      const o = row.original as OfoFlatRow;
      return h(
        UDropdownMenu,
        {
          content: { align: 'end' },
          items: ofoActionItems(o),
          'aria-label': 'Действия с ОФО',
        },
        () =>
          h(UButton, {
            icon: 'i-lucide-ellipsis-vertical',
            color: 'neutral',
            variant: 'ghost',
            square: true,
            size: 'sm',
            'aria-label': 'Действия',
          }),
      );
    },
  },
];
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 gap-6">

    <UContainer class="flex-1 min-h-0 overflow-y-auto sm:p-px max-w-full w-full md:p-px lg:p-px xl:p-px scrollbar-hide mx-0">
      <UAlert
        v-if="!isAdmin"
        color="red"
        variant="subtle"
        icon="i-lucide-shield-alert"
        title="Недостаточно прав"
        description="Эта страница доступна только администраторам."
      />

      <div v-else class="flex flex-col gap-4">
        <UTabs v-model="tab" :items="tabItems" size="xl" />

        <div v-if="tab === 'users'" class="flex flex-col gap-4 overflow-visible">

          <div class="flex flex-col gap-4 p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 overflow-visible">
            <UContainer class="flex flex-col w-full gap-3 sm:flex-row sm:flex-wrap sm:items-end p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 max-w-full mx-0 overflow-visible">
              <UInput
                v-model="userSearchQuery"
                icon="i-lucide-search"
                size="xl"
                color="neutral"
                variant="outline"
                placeholder="Поиск по ID, ФИО, ОФО, дате…"
                class="w-full sm:flex-1 sm:min-w-[240px]"
              />
              <USelectMenu
                v-model="filterStatus"
                :items="statusFilterItems"
                size="xl"
                color="neutral"
                placeholder="Статус"
                class="w-full sm:w-52"
                :content="{ align: 'start', sideOffset: 8 }"
              />
              <USelectMenu
                v-model="filterOfo"
                :items="ofoFilterItems"
                size="xl"
                color="neutral"
                placeholder="ОФО"
                class="w-full sm:w-52"
                :content="{ align: 'start', sideOffset: 8 }"
              />
              <UButton
                color="neutral"
                variant="outline"
                size="xl"
                icon="i-lucide-rotate-ccw"
                class="w-full sm:w-auto justify-center"
                @click="resetUserFilters"
              >
                Сбросить
              </UButton>
            </UContainer>

            <p
              v-if="filteredUsers.length !== users.length || userSearchQuery.trim() || filterStatus || filterOfo"
              class="text-sm text-muted -mt-2"
            >
              Найдено: {{ filteredUsers.length }} из {{ users.length }} · показано: {{ usersVisible.length }}
            </p>

            <UScrollArea
              ref="usersScrollAreaRef"
              class="max-h-[min(70vh,720px)] w-full min-h-0 rounded-lg border border-default"
              orientation="vertical"
              :ui="{ root: 'overflow-auto' }"
            >
              <div v-if="usersLoading" class="flex items-center justify-center py-16 text-muted text-sm">
                Загрузка пользователей…
              </div>
              <div v-else class="min-w-0 pb-1">
                <UTable
                  :columns="userColumns"
                  :data="usersVisible"
                  sticky
                  class="min-w-[960px]"
                />
                <div v-if="usersInfiniteLoading" class="py-3 text-center text-xs text-muted">
                  Загрузка…
                </div>
                <div v-else-if="usersVisible.length < filteredUsers.length" class="py-3 text-center text-xs text-muted">
                  Прокрутите вниз, чтобы загрузить ещё ({{ filteredUsers.length - usersVisible.length }})
                </div>
              </div>
            </UScrollArea>
          </div>
        </div>

        <div v-else-if="tab === 'groups'" class="flex flex-col gap-4">
          <div class="flex items-center justify-between gap-3 mb-2">
            <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-users">
              Создать группу
            </UButton>
          </div>

          <UAlert
            v-if="groupsError"
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :title="groupsError"
          />

          <div v-else-if="groupsLoading" class="text-sm text-muted py-4 px-2">
            Загрузка групп...
          </div>

          <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            <UCard v-for="g in groups" :key="g.name" class="w-full">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="font-medium text-highlighted truncate">{{ g.name }}</div>
                  <div class="text-sm text-muted">{{ g.members }} участников</div>
                </div>
                <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-chevron-right" square />
              </div>
            </UCard>
          </div>
        </div>

        <div v-else class="flex flex-col gap-4">


          <div class="flex flex-col gap-4">
            
            <div class="flex flex-col gap-3">
              
              <div class="flex items-center justify-between gap-3 flex-wrap">
                <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-plus">
              Добавить ОФО
            </UButton>
                <UTabs v-model="ofoViewMode" :items="ofoViewItems" size="sm" />

              </div>

              <div class="flex flex-col sm:flex-row gap-3 sm:items-end">
                <UInput
                  v-model="ofoQuery"
                  icon="i-lucide-search"
                  size="lg"
                  color="neutral"
                  variant="outline"
                  placeholder="Поиск по названию, ID или родителю…"
                  class="w-full sm:flex-1 sm:min-w-[260px]"
                />
                <USelectMenu
                  v-model="ofoSort"
                  :items="ofoSortItems"
                  size="lg"
                  color="neutral"
                  placeholder="Сортировка"
                  class="w-full sm:w-72"
                  :content="{ align: 'start', sideOffset: 8 }"
                />
              </div>
            </div>

            <div v-if="ofoLoading" class="text-sm text-muted py-4 px-2">
              Загрузка структуры...
            </div>

            <div
              v-else-if="ofoViewMode === 'cards'"
              class="flex flex-col gap-4 max-w-5xl"
            >
              <nav class="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm" aria-label="Путь по структуре ОФО">
                <template v-for="(crumb, i) in ofoCardBreadcrumbs" :key="`${crumb.id}-${i}`">
                  <UButton
                    v-if="i < ofoCardBreadcrumbs.length - 1"
                    color="neutral"
                    variant="link"
                    size="xs"
                    class="max-w-[min(280px,100%)] min-w-0 truncate px-0.5"
                    @click="ofoCardGoToBreadcrumb(i)"
                  >
                    {{ crumb.title }}
                  </UButton>
                  <span
                    v-else
                    class="text-highlighted font-medium max-w-[min(320px,100%)] min-w-0 truncate px-0.5"
                  >
                    {{ crumb.title }}
                  </span>
                  <span v-if="i < ofoCardBreadcrumbs.length - 1" class="text-muted select-none" aria-hidden="true">/</span>
                </template>
              </nav>

              <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
                <UCard
                  v-for="node in ofoCardCurrentLevel"
                  :key="node.stat.id"
                  class="w-full border-l-2 border-primary/30"
                >
                  <div class="space-y-3">
                    <div class="flex items-start justify-between gap-3">
                      <div class="min-w-0">
                        <div class="font-medium text-highlighted truncate">{{ node.stat.title }}</div>
                        <div class="text-sm text-muted truncate">ID: {{ node.stat.id }}</div>
                      </div>
                      <div class="flex items-center gap-1 shrink-0">
                        <UDropdownMenu
                          :items="ofoActionItems(node.stat)"
                          :content="{ align: 'end' }"
                          aria-label="Действия с ОФО"
                        >
                          <UButton
                            icon="i-lucide-ellipsis-vertical"
                            color="neutral"
                            variant="ghost"
                            square
                            size="sm"
                            aria-label="Действия"
                          />
                        </UDropdownMenu>
                      </div>
                    </div>

                    <div class="flex flex-wrap gap-2">
                      <UBadge
                        variant="subtle"
                        color="primary"
                        leading
                        leadingIcon="i-lucide-users"
                        class="max-w-full min-w-0 truncate"
                        :title="`${node.stat.seatsCount} должностей`"
                      >
                        {{ node.stat.seatsCount }} должн.
                      </UBadge>
                      <UBadge
                        variant="subtle"
                        color="info"
                        leading
                        leadingIcon="i-lucide-user"
                      >
                        {{ countUsersForOfo(node.stat.id, node.stat.title) }} польз.
                      </UBadge>
                    </div>

                    <div class="flex flex-wrap items-center gap-2 pt-1">
                      <UButton
                        v-if="node.children.length"
                        color="primary"
                        variant="outline"
                        size="sm"
                        icon="i-lucide-chevron-right"
                        trailing
                        @click="ofoCardEnter(node)"
                      >
                        Подразделения ({{ node.children.length }})
                      </UButton>
                      <span v-else class="text-xs text-muted">Нет дочерних ОФО</span>
                    </div>
                  </div>
                </UCard>
              </div>
            </div>

            <div v-else class="overflow-x-auto -mx-1 px-1">
              <UTable :columns="ofoColumns" :data="ofoTableRows" sticky class="min-w-[1040px]" />
            </div>
          </div>
        </div>
      </div>
    </UContainer>

    <USlideover v-model:open="ofoEditOpen" side="right" title="Редактирование ОФО" description="">
      <template #body>
        <UForm :state="ofoEditForm" class="space-y-4" @submit.prevent="saveOfoEdit">
          <UFormField label="ID" name="id">
            <UInput v-model="ofoEditForm.id" size="xl" class="w-full" disabled />
          </UFormField>
          <UFormField label="Название" name="title">
            <UInput v-model="ofoEditForm.title" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Родитель" name="parentTitle">
            <UInput v-model="ofoEditForm.parentTitle" size="xl" class="w-full" disabled />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" size="xl" @click="ofoEditOpen = false">
            Отмена
          </UButton>
          <UButton size="xl" @click="saveOfoEdit">
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>

    <UModal
      v-model:open="ofoPositionsOpen"
      :title="ofoPositionsContext ? `Должности: ${ofoPositionsContext.title}` : 'Должности'"
      description="Записи из справочника office_seats для выбранного ОФО"
    >
      <template #body>
        <div v-if="ofoPositionsList.length" class="max-h-[min(60vh,28rem)] overflow-y-auto space-y-2 pr-1">
          <div class="grid grid-cols-[88px_1fr_84px] gap-2 px-3 text-xs text-muted">
            <span>Приоритет</span>
            <span>Должность</span>
            <span class="text-right">ID</span>
          </div>
          <div
            v-for="row in ofoPositionsList"
            :key="row.id"
            class="grid grid-cols-[88px_1fr_84px] items-center gap-2 rounded-lg border border-default px-3 py-2"
          >
            <UInput
              :model-value="String(row.priority)"
              type="number"
              size="sm"
              class="w-full"
              inputmode="numeric"
              min="1"
              @update:model-value="(v) => setSeatPriority(row.id, Number(v))"
            />
            <UInput
              :model-value="row.title"
              size="sm"
              class="w-full"
              @update:model-value="(v) => setSeatTitle(row.id, String(v))"
            />
            <span class="text-muted font-mono text-xs text-right">#{{ row.id }}</span>
          </div>
        </div>
        <p v-else class="text-sm text-muted">
          Для этого ОФО нет записей должностей в данных.
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" size="lg" @click="ofoPositionsOpen = false">
            Отмена
          </UButton>
          <UButton size="lg" @click="saveOfoPositions">
            Сохранить
          </UButton>
        </div>
      </template>
    </UModal>

    <USlideover v-model:open="editOpen" side="right" title="Редактирование пользователя" description="">
      <template #body>
        <UForm :state="editForm" class="space-y-4" @submit.prevent="saveEdit">
          <UFormField label="ФИО" name="fullName">
            <UInput v-model="editForm.fullName" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Логин" name="login">
            <UInput v-model="editForm.login" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="ОФО" name="ofo">
            <UInput v-model="editForm.ofo" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Email" name="email">
            <UInput v-model="editForm.email" size="xl" class="w-full" />
          </UFormField>
          <UFormField label="Статус" name="status">
            <USelect
              v-model="editForm.status"
              :items="[
                { value: 'Активен', label: 'Активен' },
                { value: 'Заблокирован', label: 'Заблокирован' },
              ]"
              size="xl"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Последняя авторизация" name="lastAuthAt">
            <UInput v-model="editForm.lastAuthAt" size="xl" class="w-full" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-end gap-3 w-full">
          <UButton color="neutral" variant="outline" size="xl" @click="editOpen = false">
            Отмена
          </UButton>
          <UButton size="xl" @click="saveEdit">
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>
  </UMain>
</template>
