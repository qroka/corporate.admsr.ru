import type { TestForm } from '../pages/Tests/testForm';
import type { Question, QType } from '../pages/Tests/questionTypes';

// ⚠️ ДЕМО-ДАННЫЕ: детерминированно генерируются по id формы.
// При появлении бэкенда заменить на реальные агрегаты прохождений.

export type OptionStat = { label: string; count: number; percent: number; correct?: boolean };
export type QuestionStat = {
  id: string;
  title: string;
  type: QType;
  answered: number;
  skipped: number;
  options: OptionStat[] | null;
  correctRate?: number; // % правильных (тесты)
  isText?: boolean;
  avgNumber?: number;
};
export type OfoStat = { name: string; count: number; percent: number };
export type Participant = { id: number; name: string; guest?: boolean };
export type FormStats = {
  completions: number;
  started: number;
  completionRate: number;
  avgTimeSec: number;
  lastAt: string;
  avgScore?: number;
  passRate?: number;
  hardest: { title: string; correctRate: number } | null;
  byOfo: OfoStat[];
  participants: Participant[] | null; // null = анонимно
  guestCompletions?: number;
  questions: QuestionStat[];
};

function mulberry32(a: number) {
  return function () {
    a |= 0; a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function distribute(total: number, k: number, rng: () => number): number[] {
  if (k <= 0) return [];
  const w = Array.from({ length: k }, () => rng() + 0.15);
  const sum = w.reduce((a, b) => a + b, 0);
  const counts = w.map((x) => Math.round((total * x) / sum));
  counts[0] += total - counts.reduce((a, b) => a + b, 0);
  return counts.map((c) => Math.max(0, c));
}

function sample<T>(arr: T[], n: number, rng: () => number): T[] {
  const a = [...arr];
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rng() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a.slice(0, Math.max(0, n));
}

function pct(part: number, whole: number): number {
  return whole > 0 ? Math.round((part / whole) * 100) : 0;
}

function questionStat(q: Question, completions: number, rng: () => number): QuestionStat {
  const ri = (min: number, max: number) => Math.floor(min + rng() * (max - min + 1));
  const skipped = ri(0, Math.round(completions * 0.2));
  const answered = completions - skipped;

  const optionBased = q.type === 'single' || q.type === 'multiple' || q.type === 'dropdown';

  if (optionBased) {
    const labels = (q.options ?? []).map((o, i) => o.text || `Вариант ${i + 1}`);
    let counts: number[];
    if (q.type === 'multiple') {
      counts = labels.map(() => ri(0, answered)); // независимый выбор
    } else {
      counts = distribute(answered, labels.length, rng);
    }
    const correctIds = q.type === 'multiple'
      ? new Set(Array.isArray(q.correct) ? q.correct : [])
      : new Set(q.correct != null ? [q.correct as string] : []);
    const options: OptionStat[] = labels.map((label, i) => ({
      label,
      count: counts[i] ?? 0,
      percent: pct(counts[i] ?? 0, completions),
      correct: (q.options?.[i] && correctIds.has(q.options[i].id)) || undefined,
    }));
    const correctRate = correctIds.size ? ri(35, 95) : undefined;
    return { id: q.id, title: q.title, type: q.type, answered, skipped, options, correctRate };
  }

  if (q.type === 'yesno') {
    const counts = distribute(answered, 2, rng);
    const options: OptionStat[] = [
      { label: 'Да', count: counts[0], percent: pct(counts[0], completions), correct: q.correct === 'yes' || undefined },
      { label: 'Нет', count: counts[1], percent: pct(counts[1], completions), correct: q.correct === 'no' || undefined },
    ];
    const correctRate = q.correct != null ? ri(35, 95) : undefined;
    return { id: q.id, title: q.title, type: q.type, answered, skipped, options, correctRate };
  }

  if (q.type === 'scale') {
    const min = Math.min(q.scaleMin, q.scaleMax);
    const max = Math.max(q.scaleMin, q.scaleMax);
    const nums = Array.from({ length: Math.min(max - min + 1, 50) }, (_, i) => min + i);
    const counts = distribute(answered, nums.length, rng);
    const options: OptionStat[] = nums.map((n, i) => ({ label: String(n), count: counts[i], percent: pct(counts[i], completions) }));
    return { id: q.id, title: q.title, type: q.type, answered, skipped, options };
  }

  // text / textarea / number / date
  const avgNumber = q.type === 'number' ? ri(1, 100) : undefined;
  return { id: q.id, title: q.title, type: q.type, answered, skipped, options: null, isText: true, avgNumber };
}

export function buildStats(
  form: TestForm,
  users: { id: number; fullName: string }[],
  ofoUnits: { name: string }[],
): FormStats {
  const seed = ((form.listId ?? form.id ?? 1) * 2654435761) >>> 0;
  const rng = mulberry32(seed ^ 0x9e3779b9);
  const ri = (min: number, max: number) => Math.floor(min + rng() * (max - min + 1));

  const completions = ri(8, 90);
  const started = completions + ri(0, Math.round(completions * 0.5));
  const completionRate = pct(completions, started);
  const avgTimeSec = ri(45, 18 * 60);
  const daysAgo = ri(0, 20);
  const lastAt = new Date(Date.now() - daysAgo * 86400000 - ri(0, 86400) * 1000).toISOString();

  const questions = form.questions.map((q) => questionStat(q, completions, rng));

  // По ОФО
  const names = sample(ofoUnits.map((u) => u.name), ri(3, 6), rng);
  const ofoCounts = distribute(completions, names.length, rng);
  const byOfo: OfoStat[] = names
    .map((name, i) => ({ name, count: ofoCounts[i], percent: pct(ofoCounts[i], completions) }))
    .sort((a, b) => b.count - a.count);

  // Участники (если не анонимно)
  const participants: Participant[] | null = form.anonymous
    ? null
    : sample(users, Math.min(completions, users.length), rng).map((u) => ({ id: u.id, name: u.fullName }));

  // Тесты: средний балл и проходимость
  const avgScore = form.kind === 'test' ? ri(40, 95) : undefined;
  const passRate = form.kind === 'test' && form.usePassingScore ? ri(30, 95) : undefined;

  // Самый сложный вопрос
  const withRate = questions.filter((q) => q.correctRate != null) as (QuestionStat & { correctRate: number })[];
  const hardest = withRate.length
    ? withRate.reduce((m, q) => (q.correctRate < m.correctRate ? q : m))
    : null;

  return {
    completions,
    started,
    completionRate,
    avgTimeSec,
    lastAt,
    avgScore,
    passRate,
    hardest: hardest ? { title: hardest.title || 'Без названия', correctRate: hardest.correctRate } : null,
    byOfo,
    participants,
    questions,
  };
}

export function fmtDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  if (m >= 60) {
    const h = Math.floor(m / 60);
    return `${h} ч ${m % 60} мин`;
  }
  return m > 0 ? `${m} мин ${s} с` : `${s} с`;
}
