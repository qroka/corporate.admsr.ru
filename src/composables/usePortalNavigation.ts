import { computed, ref } from 'vue';
import { useRoute, useRouter, type RouteLocationRaw, type RouteParamsGeneric } from 'vue-router';
import type { BreadcrumbItem, NavigationMenuItem } from '@nuxt/ui';

/** Динамическая подпись текущего сегмента крошек (например, название альбома). */
const breadcrumbCurrentLabel = ref<string | null>(null);

export function useBreadcrumbCurrentLabel() {
  return breadcrumbCurrentLabel;
}

type NavRouteName = string;

function isRouteMatch(routeName: unknown, names: NavRouteName[]): boolean {
  if (typeof routeName !== 'string') return false;
  return names.some((n) => routeName === n || routeName.startsWith(`${n}-`) || routeName.startsWith(`admin-${n}`));
}

/** Заголовок страницы для Navbar (как в Figma). */
export function usePageTitle() {
  const route = useRoute();

  return computed(() => {
    const title = typeof route.meta?.title === 'string' ? route.meta.title : '';
    if (route.name === 'home') return 'Рабочий стол';
    return title || 'Рабочий стол';
  });
}

type ParentCrumb = {
  name: string;
  params?: (params: RouteParamsGeneric) => Record<string, string | number>;
};

/**
 * Родительские сегменты для вложенных маршрутов (без «Главная»).
 * Текущая страница добавляется отдельно и остаётся без `to`.
 */
const BREADCRUMB_PARENTS: Record<string, ParentCrumb[]> = {
  'news-details': [{ name: 'news' }],
  'event-details': [{ name: 'events' }],
  'gallery-album': [{ name: 'gallery' }],
  'absence-journal': [{ name: 'services' }],
  tests: [{ name: 'services' }],
  'tests-old': [{ name: 'services' }, { name: 'tests' }],
  'test-link': [{ name: 'services' }, { name: 'tests' }],
  applications: [{ name: 'services' }],
  'course-enrollment': [{ name: 'courses' }],
  'course-result': [{ name: 'courses' }],
  'course-topic': [
    { name: 'courses' },
    {
      name: 'course-enrollment',
      params: (p) => ({ enrollmentId: String(p.enrollmentId) }),
    },
  ],
  'course-test': [
    { name: 'courses' },
    {
      name: 'course-enrollment',
      params: (p) => ({ enrollmentId: String(p.enrollmentId) }),
    },
  ],
  'admin-course-create': [{ name: 'admin-courses' }],
  'admin-course-workspace': [{ name: 'admin-courses' }],
  'admin-course-settings': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-publish': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-assign': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-results': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-topic-create': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-topic-edit': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-material-create': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-material-edit': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-topic-test': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-topic-test-q-create': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-topic-test-q-edit': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-final-test': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-final-test-q-create': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
  'admin-course-final-test-q-edit': [
    { name: 'admin-courses' },
    {
      name: 'admin-course-workspace',
      params: (p) => ({ courseId: String(p.courseId) }),
    },
  ],
};

function titleForRouteName(router: ReturnType<typeof useRouter>, name: string): string {
  const found = router.getRoutes().find((r) => r.name === name);
  const title = found?.meta?.title;
  return typeof title === 'string' && title ? title : name;
}

/** Иконки крошек — те же Lucide, что в сайдбаре / поиске, где возможно. */
const BREADCRUMB_ICONS: Record<string, string> = {
  home: 'i-lucide-grip',
  news: 'i-lucide-newspaper',
  'news-details': 'i-lucide-newspaper',
  events: 'i-lucide-calendar',
  'event-details': 'i-lucide-calendar',
  services: 'i-lucide-layout-grid',
  'absence-journal': 'i-lucide-calendar-off',
  gallery: 'i-lucide-images',
  'gallery-album': 'i-lucide-images',
  calendar: 'i-lucide-calendar-days',
  documentation: 'i-lucide-book-open',
  feedback: 'i-lucide-message-square-more',
  tests: 'i-lucide-clipboard-list',
  'tests-old': 'i-lucide-clipboard-list',
  'test-link': 'i-lucide-clipboard-list',
  applications: 'i-lucide-file-text',
  courses: 'i-lucide-graduation-cap',
  'course-enrollment': 'i-lucide-graduation-cap',
  'course-topic': 'i-lucide-book-open',
  'course-test': 'i-lucide-clipboard-list',
  'course-result': 'i-lucide-award',
  'admin-courses': 'i-lucide-graduation-cap',
  'admin-course-create': 'i-lucide-graduation-cap',
  'admin-course-workspace': 'i-lucide-graduation-cap',
  'admin-course-settings': 'i-lucide-settings',
  'admin-course-publish': 'i-lucide-upload',
  'admin-course-assign': 'i-lucide-user-plus',
  'admin-course-results': 'i-lucide-chart-column',
  'admin-course-topic-create': 'i-lucide-book-open',
  'admin-course-topic-edit': 'i-lucide-book-open',
  'admin-course-material-create': 'i-lucide-file-text',
  'admin-course-material-edit': 'i-lucide-file-text',
  'admin-course-topic-test': 'i-lucide-clipboard-list',
  'admin-course-topic-test-q-create': 'i-lucide-clipboard-list',
  'admin-course-topic-test-q-edit': 'i-lucide-clipboard-list',
  'admin-course-final-test': 'i-lucide-clipboard-list',
  'admin-course-final-test-q-create': 'i-lucide-clipboard-list',
  'admin-course-final-test-q-edit': 'i-lucide-clipboard-list',
  profile: 'i-lucide-user',
  admin: 'i-lucide-layout-dashboard',
  'personnel-reserve': 'i-lucide-users-round',
  'development-motivation': 'i-lucide-trending-up',
  kiosk: 'i-lucide-monitor',
};

function iconForRouteName(name: string): string {
  if (BREADCRUMB_ICONS[name]) return BREADCRUMB_ICONS[name];
  if (name.startsWith('admin-course') || name.startsWith('course-')) {
    return 'i-lucide-graduation-cap';
  }
  if (name.startsWith('kiosk-')) {
    return iconForRouteName(name.slice('kiosk-'.length));
  }
  return 'i-lucide-file';
}

/** Хлебные крошки тулбара: всегда «Рабочий стол», затем цепочка раздела / текущей страницы. */
export function usePortalBreadcrumbs() {
  const route = useRoute();
  const router = useRouter();

  return computed<BreadcrumbItem[]>(() => {
    if (route.name === 'home' || route.name === 'kiosk') {
      return [{
        label: 'Рабочий стол',
        icon: 'i-lucide-grip',
        ui: { linkLeadingIcon: 'text-primary' },
      }];
    }

    const items: BreadcrumbItem[] = [
      {
        label: 'Рабочий стол',
        icon: 'i-lucide-grip',
        to: { name: 'home' } satisfies RouteLocationRaw,
      },
    ];

    const routeName = typeof route.name === 'string' ? route.name : '';
    const parents = BREADCRUMB_PARENTS[routeName] ?? [];

    for (const parent of parents) {
      const to: RouteLocationRaw = parent.params
        ? { name: parent.name, params: parent.params(route.params) }
        : { name: parent.name };
      items.push({
        label: titleForRouteName(router, parent.name),
        icon: iconForRouteName(parent.name),
        to,
      });
    }

    const currentTitle =
      (breadcrumbCurrentLabel.value && breadcrumbCurrentLabel.value.trim()) ||
      (typeof route.meta?.title === 'string' && route.meta.title
        ? route.meta.title
        : titleForRouteName(router, routeName) || 'Страница');

    items.push({
      label: currentTitle,
      icon: iconForRouteName(routeName || 'home'),
      ui: { linkLeadingIcon: 'text-primary' },
    });

    return items;
  });
}

/** Основная навигация сайдбара по макету Figma + привязка к существующим маршрутам. */
export function useSidebarNavItems() {
  const route = useRoute();
  const router = useRouter();

  const mainItems = computed<NavigationMenuItem[]>(() => [
    {
      label: 'Рабочий стол',
      icon: 'i-lucide-grip',
      to: '/',
      active: route.name === 'home',
    },
    {
      label: 'Новости',
      icon: 'i-lucide-newspaper',
      to: '/news',
      active: isRouteMatch(route.name, ['news', 'news-details']),
    },
    {
      label: 'Календарь',
      icon: 'i-lucide-calendar-days',
      to: '/calendar',
      active: route.name === 'calendar',
    },
    {
      label: 'Сервисы',
      icon: 'i-lucide-layout-grid',
      to: '/services',
      defaultOpen: true,
      active: isRouteMatch(route.name, [
        'services',
        'absence-journal',
        'tests',
        'tests-old',
        'test-link',
        'applications',
      ]),
      onSelect: () => {
        void router.push({ name: 'services' });
      },
      children: [
        {
          label: 'Журнал отсутствия',
          icon: 'i-lucide-calendar-off',
          to: '/absence-journal',
          active: route.name === 'absence-journal',
        },
        {
          label: 'Формы',
          icon: 'i-lucide-clipboard-list',
          to: '/tests',
          active: isRouteMatch(route.name, ['tests', 'tests-old', 'test-link']),
        },
        {
          label: 'Заявки',
          icon: 'i-lucide-file-text',
          to: '/applications',
          active: route.name === 'applications',
        },
      ],
    },
    {
      label: 'Мероприятия',
      icon: 'i-lucide-calendar',
      to: '/events',
      active: isRouteMatch(route.name, ['events', 'event-details']),
    },
    {
      label: 'Фотогалерея',
      icon: 'i-lucide-images',
      to: '/gallery',
      active: isRouteMatch(route.name, ['gallery', 'gallery-album']),
    },
    {
      label: 'Обучение',
      icon: 'i-lucide-graduation-cap',
      to: '/courses',
      active:
        typeof route.name === 'string'
        && (route.name === 'courses' || route.name.startsWith('course-') || route.name.startsWith('admin-course')),
    },
  ]);

  const footerItems = computed<NavigationMenuItem[]>(() => [
    {
      label: 'Обратная связь',
      icon: 'i-lucide-message-square-more',
      to: '/feedback',
      trailingIcon: 'i-lucide-arrow-up-right',
    },
    {
      label: 'Документация',
      icon: 'i-lucide-book-open',
      to: '/documentation',
      trailingIcon: 'i-lucide-arrow-up-right',
    },
  ]);

  return { mainItems, footerItems };
}
