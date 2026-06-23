import { computed, ref } from 'vue';
import { resolveNewsImageSrc } from './useNewsData';
import { avatarUrlFromFilename } from '../constants/profileAvatars';

/** avatar_url из user_info → URL картинки. Пусто/нет файла → заглушка. */
function resolveBdayAvatar(raw: unknown): string {
  const p = String(raw ?? '').replace(/\\/g, '/').trim();
  if (!p || p.toLowerCase().includes('no-avatar')) return PLACEHOLDER_AVATAR;
  if (p.startsWith('/')) return p;          // уже готовый путь (/img/...)
  return avatarUrlFromFilename(p);          // имя файла → /img/FullPic/avatars/<file>
}

function extractTableData(exportJson: unknown, tableName: string): Record<string, unknown>[] {
  if (!Array.isArray(exportJson)) return [];
  for (const item of exportJson) {
    if (item && typeof item === 'object') {
      const maybe = item as { type?: string; name?: string; data?: unknown[] };
      if (maybe.type === 'table' && maybe.name === tableName && Array.isArray(maybe.data)) {
        return maybe.data as Record<string, unknown>[];
      }
    }
  }
  return [];
}

const PLACEHOLDER_AVATAR = '/src/img/sticker 1.png';

function parseBirthMonthDay(s: unknown): { m: number; d: number } | null {
  if (s == null) return null;
  const str = String(s).trim();
  if (!str || str === '0000-00-00') return null;
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(str);
  if (!m) return null;
  const month = Number(m[2]);
  const day = Number(m[3]);
  if (month < 1 || month > 12 || day < 1 || day > 31) return null;
  return { m: month, d: day };
}

function resolveUserAvatarSrc(raw: unknown): string {
  const p = String(raw ?? '')
    .replace(/\\/g, '/')
    .trim();
  if (!p || p.toLowerCase().includes('no-avatar')) return PLACEHOLDER_AVATAR;
  // В users.json мы храним локальные аватарки вида "/img/FullPic/avatars/...".
  // Их нельзя принудительно превращать в https://corporate.admsr.ru/... — иначе в dev/proxy они не загрузятся.
  if (p.startsWith('/img/')) return p;
  if (p.startsWith('/')) return p;
  const resolved = resolveNewsImageSrc(p);
  return resolved ?? PLACEHOLDER_AVATAR;
}

type RawUserForBirthday = {
  id: number;
  fio: string;
  md: { m: number; d: number };
  avatar: string;
  /** Должность из `office_seats.title` по `users.pos` = `office_seats.id` */
  positionTitle: string;
};

function buildSeatTitleById(seatRows: Record<string, unknown>[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const row of seatRows) {
    const id = String(row.id ?? '').trim();
    if (!id) continue;
    const title = String(row.title ?? '')
      .replace(/\t/g, '')
      .trim();
    if (title) map.set(id, title);
  }
  return map;
}

function positionFromPos(posRaw: unknown, seatTitles: Map<string, string>): string {
  const key = String(posRaw ?? '').trim();
  if (!key || key === '0' || key === '-1') return 'Сотрудник';
  return seatTitles.get(key) ?? 'Сотрудник';
}

function mapRows(rows: Record<string, unknown>[], seatTitles: Map<string, string>): RawUserForBirthday[] {
  const out: RawUserForBirthday[] = [];
  for (const r of rows) {
    if (String(r.active ?? '0') !== '1') continue;
    const id = Number(r.id);
    if (!Number.isFinite(id) || id <= 0) continue;
    const md = parseBirthMonthDay(r.birthdate);
    if (!md) continue;
    const fio = String(r.fio ?? '').trim();
    if (!fio) continue;
    out.push({
      id,
      fio,
      md,
      avatar: resolveUserAvatarSrc(r.avatar),
      positionTitle: positionFromPos(r.pos, seatTitles),
    });
  }
  return out;
}

export type BirthdayPerson = {
  id: string;
  name: string;
  role: string;
  avatar: string;
};

export type BirthdayGroup = {
  id: string;
  dateLabel: string;
  dayLabel: string;
  dayColor: 'primary' | 'neutral';
  people: BirthdayPerson[];
};

export type UpcomingBirthdayItem = {
  person: BirthdayPerson;
  daysUntil: number;
  dateLabel: string;
};

function nextOccurrenceOfMonthDay(md: { m: number; d: number }, from: Date): Date {
  const y = from.getFullYear();
  let t = new Date(y, md.m - 1, md.d);
  t.setHours(0, 0, 0, 0);
  const cur = new Date(from);
  cur.setHours(0, 0, 0, 0);
  if (t < cur) t = new Date(y + 1, md.m - 1, md.d);
  return t;
}

function formatCalendarDayTitle(d: Date): string {
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
}

function buildBirthdayGroups(users: RawUserForBirthday[]): BirthdayGroup[] {
  const base = new Date();
  base.setHours(0, 0, 0, 0);

  const dayLabels = ['Сегодня', 'Завтра', 'Послезавтра'] as const;
  const colors: ('primary' | 'neutral')[] = ['primary', 'neutral', 'neutral'];

  const groups: BirthdayGroup[] = [];

  for (let i = 0; i < 3; i++) {
    const d = new Date(base);
    d.setDate(base.getDate() + i);
    const m = d.getMonth() + 1;
    const day = d.getDate();

    const people = users
      .filter((u) => u.md.m === m && u.md.d === day)
      .sort((a, b) => a.fio.localeCompare(b.fio, 'ru'))
      .map(
        (u): BirthdayPerson => ({
          id: `u-${u.id}`,
          name: u.fio,
          role: u.positionTitle,
          avatar: u.avatar,
        }),
      );

    groups.push({
      id: `bday-offset-${i}`,
      dateLabel: formatCalendarDayTitle(d),
      dayLabel: dayLabels[i],
      dayColor: colors[i],
      people,
    });
  }

  return groups;
}

/** Ключ месяц-день (1–12, 1–31) для сопоставления с ячейками календаря */
function monthDayKey(m: number, d: number): string {
  return `${m}-${d}`;
}

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedUsers = ref<RawUserForBirthday[]>([]);
const sharedLoaded = ref(false);
let sharedPromise: Promise<void> | null = null;

export function useBirthdayColleagues() {
  const birthdayGroups = computed(() => buildBirthdayGroups(sharedUsers.value));

  /** Именинники по ключу `${month}-${day}` (месяц и день как в календаре) */
  const byMonthDay = computed(() => {
    const map: Record<string, RawUserForBirthday[]> = {};
    for (const u of sharedUsers.value) {
      const k = monthDayKey(u.md.m, u.md.d);
      if (!map[k]) map[k] = [];
      map[k].push(u);
    }
    for (const k of Object.keys(map)) {
      map[k].sort((a, b) => a.fio.localeCompare(b.fio, 'ru'));
    }
    return map;
  });

  const birthdayColleaguesCount = computed(() => sharedUsers.value.length);

  const upcomingBirthdays = computed((): UpcomingBirthdayItem[] => {
    const from = new Date();
    from.setHours(0, 0, 0, 0);
    const items: UpcomingBirthdayItem[] = [];
    for (const u of sharedUsers.value) {
      const next = nextOccurrenceOfMonthDay(u.md, from);
      const daysUntil = Math.round((next.getTime() - from.getTime()) / 86400000);
      items.push({
        person: {
          id: `u-${u.id}`,
          name: u.fio,
          role: u.positionTitle,
          avatar: u.avatar,
        },
        daysUntil,
        dateLabel: next.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' }),
      });
    }
    items.sort((a, b) => a.daysUntil - b.daysUntil);
    return items.slice(0, 20);
  });

  async function load() {
    if (sharedLoaded.value) return;
    if (sharedPromise) return sharedPromise;

    sharedLoading.value = true;
    sharedError.value = null;

    sharedPromise = (async () => {
      try {
        // Источник — месячные xlsx-файлы (см. /api/birthdays.php). Возвращает [{fio, month, day}].
        const res = await fetch('/api/birthdays.php', { cache: 'no-store' });
        if (!res.ok) throw new Error(`Не удалось загрузить дни рождения (${res.status})`);
        const json = await res.json();
        if (!json.success) throw new Error(json.message || 'Ошибка загрузки');
        const list = Array.isArray(json.data) ? json.data : [];
        sharedUsers.value = list
          .map((e: any, i: number): RawUserForBirthday | null => {
            const m = Number(e.month);
            const d = Number(e.day);
            const fio = String(e.fio ?? '').trim();
            if (!fio || !(m >= 1 && m <= 12) || !(d >= 1 && d <= 31)) return null;
            return { id: i + 1, fio, md: { m, d }, avatar: resolveBdayAvatar(e.avatar), positionTitle: '' };
          })
          .filter((x): x is RawUserForBirthday => x !== null);
        sharedLoaded.value = true;
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки';
        sharedUsers.value = [];
      } finally {
        sharedLoading.value = false;
        sharedPromise = null;
      }
    })();

    return sharedPromise;
  }

  function ensureLoaded() {
    void load();
  }

  /** Перезагрузить данные (после загрузки нового xlsx в админке). */
  async function reload() {
    sharedLoaded.value = false;
    sharedPromise = null;
    return load();
  }

  return {
    loading: sharedLoading,
    error: sharedError,
    birthdayGroups,
    byMonthDay,
    upcomingBirthdays,
    birthdayColleaguesCount,
    ensureLoaded,
    reload,
  };
}
