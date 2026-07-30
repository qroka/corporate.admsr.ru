/** Разделы портала, которыми можно управлять через группы доступа. */
export const PORTAL_SECTIONS = [
  { key: 'news', label: 'Новости' },
  { key: 'events', label: 'Мероприятия' },
  { key: 'gallery', label: 'Фотогалерея' },
  { key: 'courses', label: 'Курсы' },
  { key: 'tests', label: 'Тесты' },
  { key: 'absence_journal', label: 'Журнал отсутствий' },
  { key: 'birthdays', label: 'Дни рождения' },
] as const;

export type PortalSectionKey = (typeof PORTAL_SECTIONS)[number]['key'];

export const PORTAL_SECTION_KEYS: PortalSectionKey[] = PORTAL_SECTIONS.map((s) => s.key);

export function portalSectionLabel(key: string): string {
  return PORTAL_SECTIONS.find((s) => s.key === key)?.label || key;
}
