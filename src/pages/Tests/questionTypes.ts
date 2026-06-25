export type QType =
  | 'single' | 'multiple' | 'dropdown'
  | 'text' | 'textarea'
  | 'scale' | 'yesno' | 'number' | 'date';

export type QOption = { id: string; text: string };

export type Question = {
  id: string;
  title: string;
  hint: string;
  type: QType;
  required: boolean;
  // Варианты ответа (для single / multiple / dropdown)
  options: QOption[];
  // Настройки шкалы (для scale)
  scaleMin: number;
  scaleMax: number;
  scaleMinLabel: string;
  scaleMaxLabel: string;
  // Правильный ответ (только для тестов). Для multiple — массив id вариантов,
  // для single/dropdown/yesno — одно значение, для text/number/date — введённое значение.
  correct: string | number | string[] | null;
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
    default: return '';
  }
}

export function typeHasOptions(t: QType): boolean {
  return t === 'single' || t === 'multiple' || t === 'dropdown';
}

export function uid(prefix = 'q'): string {
  return (crypto as any)?.randomUUID?.() ?? `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
}

export function createOption(): QOption {
  return { id: uid('opt'), text: '' };
}

export function createQuestion(): Question {
  return {
    id: uid('q'),
    title: '',
    hint: '',
    type: 'single',
    required: true,
    options: [createOption(), createOption()],
    scaleMin: 1,
    scaleMax: 5,
    scaleMinLabel: '',
    scaleMaxLabel: '',
    correct: null,
  };
}

// Достроить поля под выбранный тип (вызывается при смене типа вопроса)
export function applyTypeDefaults(q: Question): void {
  if (typeHasOptions(q.type)) {
    if (!Array.isArray(q.options)) q.options = [];
    while (q.options.length < 2) q.options.push(createOption());
  }
  if (q.type === 'scale') {
    if (!q.scaleMin && q.scaleMin !== 0) q.scaleMin = 1;
    if (!q.scaleMax) q.scaleMax = 5;
  }
}
