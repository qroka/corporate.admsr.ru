export type QType =
  | 'single' | 'multiple' | 'dropdown'
  | 'text' | 'textarea'
  | 'scale' | 'yesno' | 'number' | 'date'
  | 'match' | 'classify';

export type QOption = { id: string; text: string };
/** Элемент слева у «Соответствия» / «Классификации» (понятие, утверждение). */
export type QItem = { id: string; text: string };
/** Ответ на «Соответствие» / «Классификацию»: id элемента → id варианта справа. */
export type PairingMap = Record<string, string>;

export type Question = {
  id: string;
  title: string;
  hint: string;
  type: QType;
  required: boolean;
  // Варианты ответа (для single / multiple / dropdown). У match / classify — варианты
  // справа: определения / категории.
  options: QOption[];
  // Элементы слева (только match / classify): понятия / утверждения
  items: QItem[];
  // Настройки шкалы (для scale)
  scaleMin: number;
  scaleMax: number;
  scaleMinLabel: string;
  scaleMaxLabel: string;
  // Правильный ответ (только для тестов). Для multiple — массив id вариантов,
  // для single/dropdown/yesno — одно значение, для text/number/date — введённое значение,
  // для match/classify — словарь «id элемента → id правильного варианта».
  correct: string | number | string[] | PairingMap | null;
};

export const QUESTION_TYPE_ITEMS: { label: string; value: QType }[] = [
  { label: 'Один из списка', value: 'single' },
  { label: 'Несколько из списка', value: 'multiple' },
  { label: 'Выпадающий список', value: 'dropdown' },
  { label: 'Короткий ответ', value: 'text' },
  { label: 'Развёрнутый ответ', value: 'textarea' },
  { label: 'Шкала', value: 'scale' },
  { label: 'Да / Нет', value: 'yesno' },
  { label: 'Число', value: 'number' },
  { label: 'Дата', value: 'date' },
  { label: 'Соответствие', value: 'match' },
  { label: 'Классификация', value: 'classify' },
];

export function questionTypeLabel(t: QType): string {
  return QUESTION_TYPE_ITEMS.find((i) => i.value === t)?.label ?? t;
}

// Короткое описание того, как сотрудник будет отвечать (для типов без вариантов)
export function questionTypeHint(t: QType): string {
  switch (t) {
    case 'text': return 'Сотрудник впишет короткий ответ в одну строку.';
    case 'textarea': return 'Сотрудник впишет развёрнутый ответ в несколько строк.';
    case 'number': return 'Сотрудник введёт число.';
    case 'date': return 'Сотрудник выберет дату.';
    case 'yesno': return 'Сотрудник выберет «Да» или «Нет».';
    case 'match': return 'Сотрудник подберёт каждому понятию своё определение.';
    case 'classify': return 'Сотрудник отнесёт каждое утверждение к одной из категорий.';
    default: return '';
  }
}

export function typeHasOptions(t: QType): boolean {
  return t === 'single' || t === 'multiple' || t === 'dropdown';
}

/** Вопрос, у которого ответ — словарь «элемент → вариант справа». */
export function typeIsPairing(t: QType): boolean {
  return t === 'match' || t === 'classify';
}

export function uid(prefix = 'q'): string {
  return (crypto as any)?.randomUUID?.() ?? `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
}

export function createOption(): QOption {
  return { id: uid('opt'), text: '' };
}

export function createItem(): QItem {
  return { id: uid('item'), text: '' };
}

export function createQuestion(): Question {
  return {
    id: uid('q'),
    title: '',
    hint: '',
    type: 'single',
    required: true,
    options: [createOption(), createOption()],
    items: [],
    scaleMin: 1,
    scaleMax: 5,
    scaleMinLabel: '',
    scaleMaxLabel: '',
    correct: null,
  };
}

// Достроить поля под выбранный тип (вызывается при смене типа вопроса)
export function applyTypeDefaults(q: Question): void {
  if (!Array.isArray(q.items)) q.items = [];
  if (typeIsPairing(q.type)) {
    ensurePairing(q);
    return;
  }
  if (typeHasOptions(q.type)) {
    if (!Array.isArray(q.options)) q.options = [];
    while (q.options.length < 2) q.options.push(createOption());
  }
  if (q.type === 'scale') {
    if (!q.scaleMin && q.scaleMin !== 0) q.scaleMin = 1;
    if (!q.scaleMax) q.scaleMax = 5;
  }
}


// ── Соответствие / классификация ─────────────────────────────────────────────

/** Ключ правильных ответов как словарь (у вопросов других типов и у пустого ключа — пустой). */
export function pairingCorrect(q: Question): PairingMap {
  const c = q.correct;
  return c && typeof c === 'object' && !Array.isArray(c) ? (c as PairingMap) : {};
}

/** Привести вопрос к рабочему виду: у match у каждого понятия есть своё определение. */
export function ensurePairing(q: Question): void {
  if (!Array.isArray(q.items)) q.items = [];
  if (!Array.isArray(q.options)) q.options = [];
  const map: PairingMap = { ...pairingCorrect(q) };
  if (q.type === 'match') {
    while (q.items.length < 2) q.items.push(createItem());
    for (const it of q.items) {
      if (!q.options.some((o) => o.id === map[it.id])) {
        const o = createOption();
        q.options.push(o);
        map[it.id] = o.id;
      }
    }
  } else {
    while (q.options.length < 2) q.options.push(createOption());
    if (!q.items.length) q.items.push(createItem());
  }
  q.correct = map;
}

/** Пара «понятие — определение» (match). */
export function addPair(q: Question): void {
  const item = createItem();
  const opt = createOption();
  q.items.push(item);
  q.options.push(opt);
  q.correct = { ...pairingCorrect(q), [item.id]: opt.id };
}

export function removePair(q: Question, index: number): void {
  const [item] = q.items.splice(index, 1);
  if (!item) return;
  const map = { ...pairingCorrect(q) };
  const optId = map[item.id];
  delete map[item.id];
  // Определение уходит вместе с понятием, если им больше никто не пользуется.
  if (optId && !Object.values(map).includes(optId)) {
    q.options = q.options.filter((o) => o.id !== optId);
  }
  q.correct = map;
}

/** Варианты справа, к которым не привязано ни одно понятие (лишние определения). */
export function unusedTargets(q: Question): QOption[] {
  const used = new Set(Object.values(pairingCorrect(q)));
  return q.options.filter((o) => !used.has(o.id));
}

/** Классификация: убрать категорию и сбросить ответы, которые на неё ссылались. */
export function removeCategory(q: Question, index: number): void {
  const [opt] = q.options.splice(index, 1);
  if (!opt) return;
  const map = { ...pairingCorrect(q) };
  for (const k of Object.keys(map)) if (map[k] === opt.id) delete map[k];
  q.correct = map;
}

/** Готов ли ключ: у каждого непустого элемента есть правильный вариант. */
export function pairingHasKey(q: Question): boolean {
  const map = pairingCorrect(q);
  const filled = (q.items ?? []).filter((it) => it.text.trim());
  // Вариант справа без текста сервер отбрасывает вместе со ссылкой на него — такой ключ не считаем.
  return (
    filled.length > 0 &&
    filled.every((it) => Boolean(q.options.find((o) => o.id === map[it.id])?.text.trim()))
  );
}

/** Достаточно ли заполнен вопрос: непустые элементы и не меньше двух непустых вариантов. */
export function pairingIsFilled(q: Question): boolean {
  return (q.items ?? []).some((it) => it.text.trim()) && (q.options ?? []).filter((o) => o.text.trim()).length >= 2;
}
