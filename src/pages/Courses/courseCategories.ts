/** Категории курсов (пока фиксированный список). */
export const COURSE_CATEGORY_ITEMS = [
  { label: 'Кадровая деятельность', value: 'Кадровая деятельность' },
  { label: 'Безопасность', value: 'Безопасность' },
] as const;

export type CourseCategory = (typeof COURSE_CATEGORY_ITEMS)[number]['value'];
