import { CalendarDate, parseDate } from '@internationalized/date';

function isValidDate(d: Date): boolean {
  return d instanceof Date && !Number.isNaN(d.getTime());
}

function parseDateLike(value: unknown): Date | null {
  if (value == null) return null;
  if (value instanceof Date) return isValidDate(value) ? value : null;

  if (typeof value === 'number') {
    const d = new Date(value);
    return isValidDate(d) ? d : null;
  }

  const s = String(value).trim();
  if (!s) return null;

  // ISO with Z / offset → настенные часы Asia/Yekaterinburg
  if (/[zZ]|[+-]\d{2}(?::?\d{2})?\s*$/.test(s)) {
    const abs = new Date(s);
    if (!isValidDate(abs)) return null;
    const parts = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Yekaterinburg',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hourCycle: 'h23',
    }).formatToParts(abs);
    const get = (type: string) => Number(parts.find((p) => p.type === type)?.value ?? NaN);
    const d = new Date(get('year'), get('month') - 1, get('day'), get('hour'), get('minute'), get('second'), 0);
    return isValidDate(d) ? d : null;
  }

  // Local wall-clock datetime from DB/API: "YYYY-MM-DD HH:MM[:SS]" or with T
  const dt = /^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})(?::(\d{2}))?/.exec(s);
  if (dt) {
    const d = new Date(
      Number(dt[1]),
      Number(dt[2]) - 1,
      Number(dt[3]),
      Number(dt[4]),
      Number(dt[5]),
      Number(dt[6] ?? 0),
      0,
    );
    return isValidDate(d) ? d : null;
  }

  // Fast path for ISO date "YYYY-MM-DD" (avoid timezone shifts)
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (m) {
    const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
    return isValidDate(d) ? d : null;
  }

  const d = new Date(s);
  return isValidDate(d) ? d : null;
}

/**
 * Разбор даты/времени с сервера как локального «настенного» времени (Екб),
 * без сдвига UTC / Europe/Moscow.
 */
export function parseLocalDateTime(value: unknown): Date | null {
  return parseDateLike(value);
}

/** Округление минут до шага (по умолчанию 5), секунды обнуляются. */
export function roundDateToMinuteStep(date: Date, step = 5): Date {
  const d = new Date(date.getTime());
  d.setSeconds(0, 0);
  const minutes = d.getMinutes();
  let rounded = Math.round(minutes / step) * step;
  if (rounded >= 60) {
    d.setHours(d.getHours() + 1);
    rounded = 0;
  }
  d.setMinutes(rounded);
  return d;
}

export function toCalendarDate(value: unknown): CalendarDate | null {
  const s = String(value ?? '').trim();
  if (!s) return null;

  const iso = /^(\d{4})-(\d{2})-(\d{2})/.exec(s);
  if (iso) {
    try {
      return parseDate(`${iso[1]}-${iso[2]}-${iso[3]}`);
    } catch {
      // fall through
    }
  }

  const d = parseDateLike(value);
  if (!d) return null;
  return new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
}

export function formatDateRuShort(value: unknown): string {
  const d = parseDateLike(value);
  if (!d) return '';
  return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(d);
}

export function formatDateRuShortOrUndefined(value: unknown): string | undefined {
  const s = formatDateRuShort(value);
  return s ? s : undefined;
}

export function formatDateRuLong(value: unknown): string {
  const d = parseDateLike(value);
  if (!d) return '';
  return new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' }).format(d);
}

export function formatDateRuLongOrUndefined(value: unknown): string | undefined {
  const s = formatDateRuLong(value);
  return s ? s : undefined;
}

export function formatDateTimeRuShort(value: unknown): string {
  const d = parseDateLike(value);
  if (!d) return '';
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(d);
}

export function formatUnixSecondsRuDate(unixSeconds: number | null): string | undefined {
  if (!unixSeconds || !Number.isFinite(unixSeconds)) return undefined;
  return formatDateRuLongOrUndefined(new Date(unixSeconds * 1000));
}

/** Относительное время на русском («6 часов назад», «вчера»). */
export function formatRelativeRu(value: unknown): string {
  const d = parseDateLike(value);
  if (!d) return '';

  const diffSec = Math.round((d.getTime() - Date.now()) / 1000);
  const rtf = new Intl.RelativeTimeFormat('ru', { numeric: 'auto' });
  const abs = Math.abs(diffSec);

  if (abs < 60) return rtf.format(diffSec, 'second');

  const diffMin = Math.round(diffSec / 60);
  if (Math.abs(diffMin) < 60) return rtf.format(diffMin, 'minute');

  const diffHour = Math.round(diffSec / 3600);
  if (Math.abs(diffHour) < 24) return rtf.format(diffHour, 'hour');

  const diffDay = Math.round(diffSec / 86400);
  if (Math.abs(diffDay) < 30) return rtf.format(diffDay, 'day');

  const diffMonth = Math.round(diffSec / 2592000);
  if (Math.abs(diffMonth) < 12) return rtf.format(diffMonth, 'month');

  return rtf.format(Math.round(diffSec / 31536000), 'year');
}

