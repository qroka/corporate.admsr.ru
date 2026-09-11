import { computed, ref } from 'vue';
import { useBirthdayColleagues } from './useBirthdayColleagues';
import { useCoursesStore, type EnrollmentSummary } from './useCoursesStore';

export type CalendarSource = 'event' | 'meeting' | 'birthday' | 'learning' | 'personal';

export type CalendarItem = {
  id: string;
  source: CalendarSource;
  /** YYYY-MM-DD */
  dateKey: string;
  title: string;
  timeStart?: string;
  timeEnd?: string;
  /** Short label for grid chips */
  timeLabel: string;
  location?: string;
  avatar?: string;
  role?: string;
  progress?: number;
  href?: string;
  actionLabel: string;
  isJoined?: boolean;
};

export type LocalCalendarEntry = {
  id: string;
  source: 'meeting' | 'personal';
  dateKey: string;
  title: string;
  timeStart?: string;
  timeEnd?: string;
  location?: string;
};

const LOCAL_KEY = 'portal-calendar-local:v1';
const RSVP_STORAGE_KEY = 'events-rsvp:v1';

export const CALENDAR_SOURCE_META: Record<
  CalendarSource,
  { label: string; icon: string; chipClass: string; barClass: string; softClass: string }
> = {
  event: {
    label: 'Мероприятия',
    icon: 'i-lucide-calendar',
    chipClass: 'bg-emerald-500/15 text-emerald-400 ring-1 ring-inset ring-emerald-500/25',
    barClass: 'bg-emerald-500',
    softClass: 'border-emerald-500/30 bg-emerald-500/10',
  },
  meeting: {
    label: 'Встречи',
    icon: 'i-lucide-users',
    chipClass: 'bg-info/15 text-info ring-1 ring-inset ring-info/25',
    barClass: 'bg-info',
    softClass: 'border-info/30 bg-info/5',
  },
  birthday: {
    label: 'Дни рождения',
    icon: 'i-lucide-cake',
    chipClass: 'bg-violet-500/15 text-violet-400 ring-1 ring-inset ring-violet-500/25',
    barClass: 'bg-violet-500',
    softClass: 'border-violet-500/30 bg-violet-500/5',
  },
  learning: {
    label: 'Обучение',
    icon: 'i-lucide-graduation-cap',
    chipClass: 'bg-warning/15 text-warning ring-1 ring-inset ring-warning/25',
    barClass: 'bg-warning',
    softClass: 'border-warning/30 bg-warning/5',
  },
  personal: {
    label: 'Личное',
    icon: 'i-lucide-circle',
    chipClass: 'bg-muted/40 text-toned ring-1 ring-inset ring-default',
    barClass: 'bg-muted',
    softClass: 'border-default bg-elevated/40',
  },
};

function pad2(n: number) {
  return String(n).padStart(2, '0');
}

export function toDateKey(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

export function parseDateKey(key: string): Date {
  const [y, m, d] = key.split('-').map(Number);
  return new Date(y, m - 1, d);
}

function extractTimeRange(text: string): { start?: string; end?: string; label: string } {
  const range = String(text ?? '').match(/(\d{1,2}:\d{2})\s*[-–—]\s*(\d{1,2}:\d{2})/);
  if (range) return { start: range[1], end: range[2], label: `${range[1]}-${range[2]}` };
  const single = String(text ?? '').match(/\b(\d{1,2}:\d{2})\b/);
  if (single) return { start: single[1], label: single[1] };
  return { label: '' };
}

function extractPlace(text: string): string | undefined {
  const m = String(text ?? '').match(/(?:адрес|место|зал|переговорн)[:\s]+([^.\n]+)/i);
  return m?.[1]?.trim() || undefined;
}

function readLocalEntries(): LocalCalendarEntry[] {
  try {
    const raw = window.localStorage.getItem(LOCAL_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function writeLocalEntries(items: LocalCalendarEntry[]) {
  try {
    window.localStorage.setItem(LOCAL_KEY, JSON.stringify(items));
  } catch {
    // ignore
  }
}

function getRsvpMap(): Record<string, boolean> {
  try {
    const raw = window.localStorage.getItem(RSVP_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Record<string, boolean>) : {};
  } catch {
    return {};
  }
}

function formatHm(d: Date): string {
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

export function useCalendarFeed() {
  const {
    loading: birthdaysLoading,
    error: birthdaysError,
    ensureLoaded: ensureBirthdays,
    byMonthDay,
  } = useBirthdayColleagues();

  const courses = useCoursesStore();

  const eventsLoading = ref(false);
  const eventsError = ref<string | null>(null);
  const rawEvents = ref<any[]>([]);
  const localEntries = ref<LocalCalendarEntry[]>([]);
  const learningLoading = ref(false);

  const loading = computed(
    () => birthdaysLoading.value || eventsLoading.value || learningLoading.value,
  );
  const error = computed(() => birthdaysError.value || eventsError.value);

  async function loadEvents() {
    eventsLoading.value = true;
    eventsError.value = null;
    try {
      const res = await fetch('/api/events.php', { cache: 'no-store' });
      if (!res.ok) throw new Error(`Не удалось загрузить мероприятия (${res.status})`);
      const json = await res.json();
      if (!json.success) throw new Error(json.message || 'Ошибка загрузки мероприятий');
      rawEvents.value = Array.isArray(json.data) ? json.data : [];
    } catch (e) {
      eventsError.value = e instanceof Error ? e.message : 'Ошибка загрузки мероприятий';
      rawEvents.value = [];
    } finally {
      eventsLoading.value = false;
    }
  }

  async function loadLearning() {
    learningLoading.value = true;
    try {
      await courses.loadMyCourses();
    } catch {
      // LMS may be unavailable — calendar still works without deadlines
    } finally {
      learningLoading.value = false;
    }
  }

  function loadLocal() {
    localEntries.value = readLocalEntries();
  }

  async function ensureLoaded() {
    ensureBirthdays();
    loadLocal();
    await Promise.all([loadEvents(), loadLearning()]);
  }

  function addLocalEntry(entry: Omit<LocalCalendarEntry, 'id'> & { id?: string }) {
    const next: LocalCalendarEntry = {
      id: entry.id ?? `local-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      source: entry.source,
      dateKey: entry.dateKey,
      title: entry.title.trim(),
      timeStart: entry.timeStart,
      timeEnd: entry.timeEnd,
      location: entry.location,
    };
    localEntries.value = [...localEntries.value, next];
    writeLocalEntries(localEntries.value);
    return next;
  }

  function removeLocalEntry(id: string) {
    localEntries.value = localEntries.value.filter((e) => e.id !== id);
    writeLocalEntries(localEntries.value);
  }

  const items = computed((): CalendarItem[] => {
    const out: CalendarItem[] = [];
    const rsvp = getRsvpMap();

    for (const ev of rawEvents.value) {
      const dateRaw = String(ev?.date ?? '').trim();
      const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(dateRaw);
      if (!m) continue;
      const dateKey = `${m[1]}-${m[2]}-${m[3]}`;
      const text = `${ev?.description ?? ''} ${ev?.title ?? ''}`;
      const time = extractTimeRange(text);
      const id = String(ev.id);
      out.push({
        id: `event-${id}`,
        source: 'event',
        dateKey,
        title: String(ev.title ?? 'Мероприятие'),
        timeStart: time.start,
        timeEnd: time.end,
        timeLabel: time.label || time.start || '',
        location: extractPlace(text),
        href: `/events/${id}`,
        actionLabel: 'Открыть',
        isJoined: Boolean(rsvp[id]),
      });
    }

    for (const entry of localEntries.value) {
      const label =
        entry.timeStart && entry.timeEnd
          ? `${entry.timeStart}-${entry.timeEnd}`
          : entry.timeStart || '';
      out.push({
        id: entry.id,
        source: entry.source,
        dateKey: entry.dateKey,
        title: entry.title,
        timeStart: entry.timeStart,
        timeEnd: entry.timeEnd,
        timeLabel: label,
        location: entry.location,
        actionLabel: 'Открыть',
      });
    }

    const yearHint = new Date().getFullYear();
    for (const [key, people] of Object.entries(byMonthDay.value)) {
      const [mm, dd] = key.split('-').map(Number);
      if (!mm || !dd) continue;
      // Birthdays are year-agnostic — emit for current + next calendar year span
      for (const year of [yearHint - 1, yearHint, yearHint + 1]) {
        const dateKey = `${year}-${pad2(mm)}-${pad2(dd)}`;
        for (const person of people) {
          out.push({
            id: `bday-${year}-${person.id}`,
            source: 'birthday',
            dateKey,
            title: person.fio,
            timeLabel: '',
            avatar: person.avatar,
            role: person.positionTitle || undefined,
            actionLabel: 'Поздравить',
          });
        }
      }
    }

    for (const enr of courses.myEnrollments.value as EnrollmentSummary[]) {
      const raw = enr.deadlineAt;
      if (!raw) continue;
      const d = new Date(raw);
      if (Number.isNaN(d.getTime())) continue;
      const dateKey = toDateKey(d);
      const hm = formatHm(d);
      out.push({
        id: `learn-${enr.id}`,
        source: 'learning',
        dateKey,
        title: enr.courseTitle ? `Завершить: ${enr.courseTitle}` : 'Срок обучения',
        timeStart: hm,
        timeLabel: hm !== '00:00' ? `до ${hm}` : 'Срок',
        progress: Number(enr.progressPercent ?? 0),
        href: `/courses/${enr.id}`,
        actionLabel: 'Продолжить',
      });
    }

    out.sort((a, b) => {
      if (a.dateKey !== b.dateKey) return a.dateKey.localeCompare(b.dateKey);
      const ta = a.timeStart || '99:99';
      const tb = b.timeStart || '99:99';
      if (ta !== tb) return ta.localeCompare(tb);
      return a.title.localeCompare(b.title, 'ru');
    });

    return out;
  });

  return {
    loading,
    error,
    items,
    localEntries,
    ensureLoaded,
    reloadEvents: loadEvents,
    reloadLearning: loadLearning,
    addLocalEntry,
    removeLocalEntry,
  };
}
