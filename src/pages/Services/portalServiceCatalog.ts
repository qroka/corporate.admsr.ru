/** Каталог внутренних страниц портала для выбора сервиса. */

export type PortalServiceCatalogItem = {
  key: string;
  label: string;
  description: string;
  icon: string;
  path: string;
};

export const PORTAL_SERVICE_CATALOG: PortalServiceCatalogItem[] = [
  {
    key: 'home',
    label: 'Рабочий стол',
    description: 'Главная страница портала',
    icon: 'i-lucide-grip',
    path: '/',
  },
  {
    key: 'news',
    label: 'Новости',
    description: 'Лента корпоративных новостей',
    icon: 'i-lucide-newspaper',
    path: '/news',
  },
  {
    key: 'calendar',
    label: 'Календарь',
    description: 'События и встречи',
    icon: 'i-lucide-calendar-days',
    path: '/calendar',
  },
  {
    key: 'absence-journal',
    label: 'Журнал отсутствия',
    description: 'Отметки об отсутствии сотрудника',
    icon: 'i-lucide-calendar-off',
    path: '/absence-journal',
  },
  {
    key: 'tests',
    label: 'Формы',
    description: 'Опросы, анкеты и тесты',
    icon: 'i-lucide-clipboard-list',
    path: '/tests',
  },
  {
    key: 'applications',
    label: 'Заявки',
    description: 'Подача и отслеживание сервисных заявок',
    icon: 'i-lucide-file-text',
    path: '/applications',
  },
  {
    key: 'events',
    label: 'Мероприятия',
    description: 'Корпоративные мероприятия',
    icon: 'i-lucide-calendar',
    path: '/events',
  },
  {
    key: 'gallery',
    label: 'Фотогалерея',
    description: 'Альбомы и фото',
    icon: 'i-lucide-images',
    path: '/gallery',
  },
  {
    key: 'courses',
    label: 'Обучение',
    description: 'Назначенные программы и материалы',
    icon: 'i-lucide-graduation-cap',
    path: '/courses',
  },
  {
    key: 'feedback',
    label: 'Обратная связь',
    description: 'Сообщения в поддержку',
    icon: 'i-lucide-message-square-more',
    path: '/feedback',
  },
  {
    key: 'documentation',
    label: 'Документация',
    description: 'Справка по разделам портала',
    icon: 'i-lucide-book-open',
    path: '/documentation',
  },
  {
    key: 'profile',
    label: 'Профиль',
    description: 'Личные данные сотрудника',
    icon: 'i-lucide-user',
    path: '/profile',
  },
];

export const PORTAL_SERVICE_ICON_OPTIONS = [
  'i-lucide-layout-grid',
  'i-lucide-link',
  'i-lucide-external-link',
  'i-lucide-globe',
  'i-lucide-building-2',
  'i-lucide-briefcase',
  'i-lucide-file-text',
  'i-lucide-clipboard-list',
  'i-lucide-calendar-off',
  'i-lucide-graduation-cap',
  'i-lucide-newspaper',
  'i-lucide-calendar',
  'i-lucide-images',
  'i-lucide-message-square-more',
  'i-lucide-book-open',
  'i-lucide-user',
] as const;

export function catalogItemByKey(key: string | null | undefined): PortalServiceCatalogItem | undefined {
  if (!key) return undefined;
  return PORTAL_SERVICE_CATALOG.find((c) => c.key === key);
}

export function catalogItemByPath(path: string | null | undefined): PortalServiceCatalogItem | undefined {
  if (!path) return undefined;
  return PORTAL_SERVICE_CATALOG.find((c) => c.path === path);
}
