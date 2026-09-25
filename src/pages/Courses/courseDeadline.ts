/** Сроки назначений: человекочитаемая подпись + цвет срочности. */

export type DeadlineTone = 'neutral' | 'warning' | 'error';

export type DeadlineInfo = {
  label: string;
  tone: DeadlineTone;
  icon: string;
  /** Полная дата для title/подсказки. */
  full: string;
};

const DAY_FORMS: [string, string, string] = ['день', 'дня', 'дней'];

function plural(n: number, forms: [string, string, string]) {
  const abs = Math.abs(n) % 100;
  const tail = abs % 10;
  if (abs > 10 && abs < 20) return forms[2];
  if (tail > 1 && tail < 5) return forms[1];
  if (tail === 1) return forms[0];
  return forms[2];
}

function startOfDay(d: Date) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

export function formatDateTime(iso?: string | null) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const date = d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${date}, ${time}`;
}

export function formatDate(iso?: string | null) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
}

/**
 * Подпись к дедлайну. Для завершённых курсов срочность не нужна —
 * показываем нейтральную дату (`settled`).
 */
export function describeDeadline(iso?: string | null, settled = false): DeadlineInfo | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;

  const full = `Срок: ${formatDateTime(iso)}`;
  if (settled) {
    return { label: `Срок: ${formatDate(iso)}`, tone: 'neutral', icon: 'i-lucide-calendar', full };
  }

  const days = Math.round((startOfDay(d) - startOfDay(new Date())) / 86_400_000);

  if (days < 0) {
    const n = Math.abs(days);
    return {
      label: `Просрочено на ${n} ${plural(n, DAY_FORMS)}`,
      tone: 'error',
      icon: 'i-lucide-calendar-x',
      full,
    };
  }
  if (days === 0) {
    return { label: 'Сегодня последний день', tone: 'error', icon: 'i-lucide-calendar-clock', full };
  }
  if (days <= 3) {
    return {
      label: `Осталось ${days} ${plural(days, DAY_FORMS)}`,
      tone: 'warning',
      icon: 'i-lucide-calendar-clock',
      full,
    };
  }
  return { label: `До ${formatDate(iso)}`, tone: 'neutral', icon: 'i-lucide-calendar', full };
}

/** Просроченные и несданные — выше всех, дальше по близости срока. */
const ATTENTION_RANK: Record<string, number> = {
  overdue: 0,
  failed: 1,
  in_progress: 2,
  not_started: 3,
};

type RankedEnrollment = {
  status: string;
  deadlineAt?: string | null;
  courseTitle: string;
};

function deadlineTime(iso?: string | null) {
  if (!iso) return Number.POSITIVE_INFINITY;
  const t = new Date(iso).getTime();
  return Number.isNaN(t) ? Number.POSITIVE_INFINITY : t;
}

/**
 * Порядок активных назначений — один на весь раздел: список «Моё обучение»
 * и виджет на рабочем столе должны показывать одно и то же первым.
 */
export function compareByAttention(a: RankedEnrollment, b: RankedEnrollment) {
  const rank = (ATTENTION_RANK[a.status] ?? 9) - (ATTENTION_RANK[b.status] ?? 9);
  if (rank !== 0) return rank;
  const byDeadline = deadlineTime(a.deadlineAt) - deadlineTime(b.deadlineAt);
  if (byDeadline !== 0) return byDeadline;
  return a.courseTitle.localeCompare(b.courseTitle, 'ru');
}
