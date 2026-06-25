export type QType =
  | 'single' | 'multiple' | 'dropdown'
  | 'text' | 'textarea'
  | 'scale' | 'yesno' | 'number' | 'date';

export type Question = {
  id: string;
  title: string;
  hint: string;
  type: QType;
  required: boolean;
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
