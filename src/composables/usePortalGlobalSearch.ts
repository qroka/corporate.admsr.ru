import { computed, ref, watch, type Ref } from 'vue';
import type { CommandPaletteGroup, CommandPaletteItem } from '@nuxt/ui';
import { useNewsData } from './useNewsData';
import { useUsersData } from './useUsersData';
import { useAppToast } from './useAppToast';
import { apiSessionFetch } from './useAuthSession';

type SearchHit = CommandPaletteItem & {
  label: string;
  description?: string;
  suffix?: string;
  icon?: string;
  to?: string;
  avatar?: { src?: string; alt?: string };
  onSelect?: (e?: Event) => void;
};

type EventRow = {
  id: number;
  title: string;
  description: string;
  date: string;
  badge?: string;
};

type GalleryAlbum = {
  id: number;
  name: string;
  description?: string;
  date?: string;
};

type FormRow = {
  id: number;
  title: string;
  description?: string;
  kind?: string;
  list_id?: number;
};

type CourseRow = {
  id: number;
  title: string;
  status?: string;
  category?: string;
};

const PAGE_ITEMS: SearchHit[] = [
  { label: 'Рабочий стол', icon: 'i-lucide-grip', to: '/', description: 'Главная страница портала' },
  { label: 'Новости', icon: 'i-lucide-newspaper', to: '/news', description: 'Лента корпоративных новостей' },
  { label: 'Календарь', icon: 'i-lucide-calendar-days', to: '/calendar', description: 'Встречи, мероприятия и дни рождения' },
  { label: 'Сервисы', icon: 'i-lucide-layout-grid', to: '/services', description: 'Рабочие инструменты портала' },
  { label: 'Журнал отсутствия', icon: 'i-lucide-calendar-off', to: '/absence-journal', description: 'Отметки об отсутствии' },
  { label: 'Формы', icon: 'i-lucide-clipboard-list', to: '/tests', description: 'Опросы, анкеты и тесты' },
  { label: 'Мероприятия', icon: 'i-lucide-calendar', to: '/events', description: 'Афиша корпоративных событий' },
  { label: 'Фотогалерея', icon: 'i-lucide-images', to: '/gallery', description: 'Альбомы и фотографии' },
  { label: 'Заявки', icon: 'i-lucide-file-text', to: '/applications', description: 'Сервисы · сервисные заявки' },
  { label: 'Обучение', icon: 'i-lucide-graduation-cap', to: '/courses', description: 'Мои курсы и материалы' },
  { label: 'Документация', icon: 'i-lucide-book-open', to: '/documentation', description: 'Справка по порталу' },
  { label: 'Обратная связь', icon: 'i-lucide-message-square-more', to: '/feedback', description: 'Вопросы и предложения' },
  { label: 'Профиль', icon: 'i-lucide-user', to: '/profile', description: 'Личные данные и стена' },
];

const DOC_SECTIONS: SearchHit[] = [
  { label: 'Рабочий стол', icon: 'i-lucide-book-open', to: '/documentation', description: 'Документация · главная' },
  { label: 'Навигация', icon: 'i-lucide-book-open', to: '/documentation', description: 'Документация · меню и шапка' },
  { label: 'Новости и мероприятия', icon: 'i-lucide-book-open', to: '/documentation', description: 'Документация · контент' },
  { label: 'Сервисы и формы', icon: 'i-lucide-book-open', to: '/documentation', description: 'Документация · сервисы' },
  { label: 'Обучение', icon: 'i-lucide-book-open', to: '/documentation', description: 'Документация · курсы' },
  { label: 'Профиль и тема', icon: 'i-lucide-book-open', to: '/documentation', description: 'Документация · настройки' },
];

const PER_GROUP = 8;
const MIN_QUERY = 2;

const eventsCache = ref<EventRow[]>([]);
const galleryCache = ref<GalleryAlbum[]>([]);
const formsCache = ref<FormRow[]>([]);
const coursesCache = ref<CourseRow[]>([]);
const indexReady = ref(false);
const indexLoading = ref(false);
let indexPromise: Promise<void> | null = null;

function stripHtml(html: string): string {
  return String(html ?? '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function includesQuery(q: string, ...parts: unknown[]): boolean {
  const hay = parts
    .map((p) => String(p ?? '').toLowerCase())
    .join(' ');
  return hay.includes(q);
}

function getAuthUserId(): string | null {
  try {
    const raw = localStorage.getItem('auth-user');
    const u = raw ? JSON.parse(raw) : null;
    const id = u?.id;
    if (id === null || id === undefined) return null;
    const s = String(id).trim();
    return s || null;
  } catch {
    return null;
  }
}

async function fetchJson(url: string, init?: RequestInit): Promise<unknown> {
  const uid = getAuthUserId();
  const headers = new Headers(init?.headers);
  if (uid && !headers.has('X-User-Id')) headers.set('X-User-Id', uid);
  const res = await fetch(url, { cache: 'no-store', ...init, headers });
  if (!res.ok) return null;
  return res.json();
}

async function loadEvents() {
  try {
    const json = (await fetchJson('/api/events.php')) as { success?: boolean; data?: any[] } | null;
    if (!json?.success || !Array.isArray(json.data)) return;
    eventsCache.value = json.data
      .map((r) => ({
        id: Number(r.id),
        title: String(r.title ?? '').trim(),
        description: stripHtml(String(r.description ?? '')),
        date: String(r.date ?? '').trim(),
        badge: r.badge ? String(r.badge) : undefined,
      }))
      .filter((e) => e.id && e.title);
  } catch {
    /* ignore */
  }
}

async function loadGallery() {
  try {
    const json = (await fetchJson('/api/gallery.php')) as { success?: boolean; data?: any[] } | null;
    if (!json?.success || !Array.isArray(json.data)) return;
    galleryCache.value = json.data
      .map((r) => ({
        id: Number(r.id),
        name: String(r.name ?? '').trim(),
        description: String(r.description ?? '').trim(),
        date: String(r.date ?? '').trim(),
      }))
      .filter((a) => a.id && a.name);
  } catch {
    /* ignore */
  }
}

async function loadForms() {
  try {
    const json = (await fetchJson('/api/forms_list.php?status=published')) as {
      success?: boolean;
      data?: any[];
    } | null;
    if (!json?.success || !Array.isArray(json.data)) return;
    formsCache.value = json.data
      .map((r) => ({
        id: Number(r.id),
        title: String(r.title ?? '').trim(),
        description: String(r.description ?? '').trim(),
        kind: String(r.kind ?? '').trim(),
        list_id: Number(r.list_id ?? r.listId ?? 0) || undefined,
      }))
      .filter((f) => f.id && f.title);
  } catch {
    /* ignore */
  }
}

async function loadCourses() {
  try {
    const json = await apiSessionFetch<{ items?: any[] } | any[]>('/api/courses_for_me.php', {
      method: 'POST',
      json: {},
    });
    if (!json?.success) return;
    const raw = json.data as any;
    const mapItem = (x: any) => {
      const enr = x?.enrollment ?? x;
      const course = x?.course ?? enr?.course;
      return {
        id: Number(enr?.id ?? x?.id ?? 0),
        title: String(course?.title ?? enr?.courseTitle ?? x?.courseTitle ?? x?.title ?? '').trim(),
        status: String(enr?.status ?? x?.status ?? '').trim() || undefined,
        category: String(course?.category ?? x?.category ?? '').trim() || undefined,
      };
    };
    const list: any[] = Array.isArray(raw)
      ? raw
      : [
          ...(raw?.overdue || []),
          ...(raw?.active || []),
          ...(raw?.completed || []),
          ...(raw?.failed || []),
          ...(raw?.items || []),
        ];
    coursesCache.value = list.map(mapItem).filter((c) => c.id && c.title);
  } catch {
    /* ignore */
  }
}

export async function ensurePortalSearchIndex() {
  if (indexReady.value) return;
  if (indexPromise) return indexPromise;

  indexLoading.value = true;
  indexPromise = (async () => {
    const newsApi = useNewsData();
    const usersApi = useUsersData();
    await Promise.allSettled([
      newsApi.load(),
      usersApi.load(),
      loadEvents(),
      loadGallery(),
      loadForms(),
      loadCourses(),
    ]);
    indexReady.value = true;
  })().finally(() => {
    indexLoading.value = false;
    indexPromise = null;
  });

  return indexPromise;
}

function kindLabel(kind?: string) {
  return ({ test: 'Тест', survey: 'Опрос', poll: 'Голосование' } as Record<string, string>)[String(kind ?? '')] ?? 'Форма';
}

function buildContentGroups(q: string, toast: ReturnType<typeof useAppToast>['toast']): CommandPaletteGroup[] {
  const groups: CommandPaletteGroup[] = [];
  const { news } = useNewsData();
  const { users } = useUsersData();

  const people: SearchHit[] = [];
  for (const u of users.value) {
    if (u.status !== 'Активен') continue;
    if (!includesQuery(q, u.fullName, u.login, u.email, u.phone, u.role, u.ofo)) continue;
    people.push({
      id: `user-${u.id}`,
      label: u.fullName,
      description: [u.role, u.email, u.phone].filter(Boolean).join(' · ') || u.login,
      icon: 'i-lucide-user',
      avatar: u.avatar_url ? { src: u.avatar_url, alt: u.fullName } : undefined,
      onSelect(e?: Event) {
        e?.preventDefault?.();
        const lines = [u.email, u.phone, u.role].filter(Boolean);
        toast.add({
          title: u.fullName,
          description: lines.join(' · ') || 'Контакты не указаны',
          icon: 'i-lucide-user',
          color: 'neutral',
        });
      },
    });
    if (people.length >= PER_GROUP) break;
  }
  if (people.length) {
    groups.push({ id: 'people', label: 'Сотрудники', items: people, ignoreFilter: true });
  }

  const newsItems: SearchHit[] = [];
  for (const n of news.value) {
    if (!includesQuery(q, n.title, n.category, stripHtml(n.description))) continue;
    newsItems.push({
      id: `news-${n.id}`,
      label: n.title,
      description: [n.category, n.date].filter(Boolean).join(' · ') || 'Новость',
      icon: 'i-lucide-newspaper',
      to: `/news/${n.id}`,
    });
    if (newsItems.length >= PER_GROUP) break;
  }
  if (newsItems.length) {
    groups.push({ id: 'news', label: 'Новости', items: newsItems, ignoreFilter: true });
  }

  const eventItems: SearchHit[] = [];
  for (const e of eventsCache.value) {
    if (String(e.badge ?? '').toLowerCase() === 'архив') continue;
    if (!includesQuery(q, e.title, e.description, e.date)) continue;
    eventItems.push({
      id: `event-${e.id}`,
      label: e.title,
      description: e.date || 'Мероприятие',
      icon: 'i-lucide-calendar',
      to: `/events/${e.id}`,
    });
    if (eventItems.length >= PER_GROUP) break;
  }
  if (eventItems.length) {
    groups.push({ id: 'events', label: 'Мероприятия', items: eventItems, ignoreFilter: true });
  }

  const albums: SearchHit[] = [];
  for (const a of galleryCache.value) {
    if (!includesQuery(q, a.name, a.description, a.date)) continue;
    albums.push({
      id: `album-${a.id}`,
      label: a.name,
      description: a.date || a.description || 'Альбом',
      icon: 'i-lucide-images',
      to: `/gallery/${a.id}`,
    });
    if (albums.length >= PER_GROUP) break;
  }
  if (albums.length) {
    groups.push({ id: 'gallery', label: 'Фотогалерея', items: albums, ignoreFilter: true });
  }

  const formItems: SearchHit[] = [];
  for (const f of formsCache.value) {
    if (!includesQuery(q, f.title, f.description, f.kind, f.list_id)) continue;
    formItems.push({
      id: `form-${f.id}`,
      label: f.title,
      description: kindLabel(f.kind),
      suffix: f.list_id ? `#${f.list_id}` : undefined,
      icon: 'i-lucide-clipboard-list',
      to: '/tests',
    });
    if (formItems.length >= PER_GROUP) break;
  }
  if (formItems.length) {
    groups.push({ id: 'forms', label: 'Формы', items: formItems, ignoreFilter: true });
  }

  const courseItems: SearchHit[] = [];
  for (const c of coursesCache.value) {
    if (!includesQuery(q, c.title, c.status, c.category)) continue;
    courseItems.push({
      id: `course-${c.id}`,
      label: c.title,
      description: [c.category, c.status].filter(Boolean).join(' · ') || 'Курс',
      icon: 'i-lucide-graduation-cap',
      to: `/courses/${c.id}`,
    });
    if (courseItems.length >= PER_GROUP) break;
  }
  if (courseItems.length) {
    groups.push({ id: 'courses', label: 'Обучение', items: courseItems, ignoreFilter: true });
  }

  const docs: SearchHit[] = [];
  for (const d of DOC_SECTIONS) {
    if (!includesQuery(q, d.label, d.description)) continue;
    docs.push({ ...d, id: `doc-${d.label}` });
    if (docs.length >= PER_GROUP) break;
  }
  if (docs.length) {
    groups.push({ id: 'docs', label: 'Документация', items: docs, ignoreFilter: true });
  }

  return groups;
}

/**
 * Глобальный поиск портала: разделы + сотрудники, новости, мероприятия,
 * галерея, формы, курсы, документация (как в CMDB — клиентский индекс по контенту).
 */
export function usePortalGlobalSearch(searchTerm: Ref<string>, open: Ref<boolean>) {
  const { toast } = useAppToast();

  watch(
    open,
    (isOpen) => {
      if (isOpen) void ensurePortalSearchIndex();
      else searchTerm.value = '';
    },
    { immediate: true },
  );

  const groups = computed<CommandPaletteGroup[]>(() => {
    const q = searchTerm.value.trim().toLowerCase();
    const result: CommandPaletteGroup[] = [
      {
        id: 'pages',
        label: 'Разделы',
        items: PAGE_ITEMS,
      },
    ];

    if (q.length >= MIN_QUERY) {
      result.push(...buildContentGroups(q, toast));
    }

    return result;
  });

  return {
    groups,
    loading: indexLoading,
    ensureIndex: ensurePortalSearchIndex,
  };
}
