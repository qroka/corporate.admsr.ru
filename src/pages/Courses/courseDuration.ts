/** Минимальное активное время хранится в секундах — здесь оно становится читаемым. */

/** Русское склонение по числу: plural(2, ['материал','материала','материалов']). */
export function plural(n: number, forms: [string, string, string]): string {
  const abs = Math.abs(n) % 100;
  const tail = abs % 10;
  if (abs > 10 && abs < 20) return forms[2];
  if (tail > 1 && tail < 5) return forms[1];
  if (tail === 1) return forms[0];
  return forms[2];
}

export function formatSeconds(total: number): string {
  const min = Math.floor(total / 60);
  const sec = total % 60;
  const parts: string[] = [];
  if (min) parts.push(min + ' мин');
  if (sec) parts.push(sec + ' с');
  return parts.join(' ');
}

/**
 * Подпись под полем: что именно даст введённое число.
 * @param subject винительный падеж — «тему», «материал».
 */
export function activeTimeHint(value: unknown, subject: string): string {
  const total = Number(value);
  if (!Number.isFinite(total) || total <= 0) {
    return `Без ограничения — ${subject} можно завершить сразу.`;
  }
  return `Нельзя завершить ${subject} раньше, чем через ${formatSeconds(total)}.`;
}

/** API хранит секунды, админ вводит минуты (допустимы дробные: 1,5 мин = 90 с). */
export function secondsToMinutes(seconds: unknown): number {
  const total = Number(seconds) || 0;
  return Math.round((total / 60) * 100) / 100;
}

export function minutesToSeconds(minutes: unknown): number {
  const n = Number(minutes);
  return Number.isFinite(n) ? Math.round(n * 60) : NaN;
}
