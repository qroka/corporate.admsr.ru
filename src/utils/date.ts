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

  // Fast path for ISO date "YYYY-MM-DD" (avoid timezone shifts)
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s);
  if (m) {
    const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
    return isValidDate(d) ? d : null;
  }

  const d = new Date(s);
  return isValidDate(d) ? d : null;
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

