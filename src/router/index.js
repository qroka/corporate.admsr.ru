import { createRouter, createWebHistory } from 'vue-router';
import { currentRole } from '../stores/role';
import HomePage from '../pages/HomePage.vue';
import EventsPage from '../pages/Events/EventsPage.vue';
import EventDetailsPage from '../pages/Events/EventDetailsPage.vue';
import GalleryPage from '../pages/Gallery/GalleryPage.vue';
import GalleryAlbumPage from '../pages/Gallery/GalleryAlbumPage.vue';
import NewsPage from '../pages/News/NewsPage.vue';
import NewsDetailsPage from '../pages/News/NewsDetailsPage.vue';
import AppKiosk from '../AppKiosk.vue';
import KioskHomePage from '../pages/Kiosk/KioskHomePage.vue';
import NewcomersPage from '../pages/NewcomersPage.vue';
import CulturePage from '../pages/CulturePage.vue';
import ProfilePage from '../pages/ProfilePage.vue';
import AbsenceJournalPage from '../pages/AbsenceJournalPage.vue';
import ApplicationsPage from '../pages/ApplicationsPage.vue';
import KnowledgeBasePage from '../pages/KnowledgeBasePage.vue';
import PersonnelReservePage from '../pages/PersonnelReservePage.vue';
import TestsPage from '../pages/TestsPage.vue';
import HrDepartmentPage from '../pages/HrDepartmentPage.vue';
import MunicipalServiceDepartmentPage from '../pages/MunicipalServiceDepartmentPage.vue';
import DevelopmentMotivationDepartmentPage from '../pages/DevelopmentMotivationDepartmentPage.vue';
import AdminDashboardPage from '../pages/Admin/AdminDashboardPage.vue';
import BirthdaysPage from '../pages/BirthdaysPage.vue';

const routes = [
  { path: '/', name: 'home', component: HomePage, meta: { title: 'Главная' } },
  {
    path: '/kiosk',
    component: AppKiosk,
    meta: { title: 'Киоск', kiosk: true },
    children: [
      { path: '', name: 'kiosk', component: KioskHomePage },
      { path: 'news', name: 'kiosk-news', component: NewsPage, meta: { title: 'Новости', kiosk: true } },
      { path: 'news/:id', name: 'kiosk-news-details', component: NewsDetailsPage, meta: { title: 'Новость', kiosk: true } },
      { path: 'events', name: 'kiosk-events', component: EventsPage, meta: { title: 'Мероприятия', kiosk: true } },
      { path: 'events/:id', name: 'kiosk-event-details', component: EventDetailsPage, meta: { title: 'Мероприятие', kiosk: true } },
      { path: 'gallery', name: 'kiosk-gallery', component: GalleryPage, meta: { title: 'Фотогалерея', kiosk: true } },
      { path: 'gallery/:albumId', name: 'kiosk-gallery-album', component: GalleryAlbumPage, meta: { title: 'Альбом', kiosk: true } },
      { path: 'newcomers', name: 'kiosk-newcomers', component: NewcomersPage, meta: { title: 'Новичкам', kiosk: true } },
      { path: 'culture', name: 'kiosk-culture', component: CulturePage, meta: { title: 'Корпоративная культура', kiosk: true } },
      { path: 'birthdays', name: 'kiosk-birthdays', component: BirthdaysPage, meta: { title: 'Дни рождения коллег', kiosk: true } },
      { path: 'profile', name: 'kiosk-profile', component: ProfilePage, meta: { title: 'Профиль', kiosk: true } },
      { path: 'absence-journal', name: 'kiosk-absence-journal', component: AbsenceJournalPage, meta: { title: 'Журнал отсутствия', kiosk: true } },
      { path: 'applications', name: 'kiosk-applications', component: ApplicationsPage, meta: { title: 'Заявки', kiosk: true } },
      { path: 'knowledge-base', name: 'kiosk-knowledge-base', component: KnowledgeBasePage, meta: { title: 'База знаний', kiosk: true } },
      { path: 'personnel-reserve', name: 'kiosk-personnel-reserve', component: PersonnelReservePage, meta: { title: 'Кадровый резерв', kiosk: true } },
      { path: 'tests', name: 'kiosk-tests', component: TestsPage, meta: { title: 'Тесты', kiosk: true } },
      { path: 'hr-department', name: 'kiosk-hr-department', component: HrDepartmentPage, meta: { title: 'Отдел кадров', kiosk: true } },
      { path: 'municipal-service', name: 'kiosk-municipal-service', component: MunicipalServiceDepartmentPage, meta: { title: 'Отдел муниципальной службы', kiosk: true } },
      { path: 'development-motivation', name: 'kiosk-development-motivation', component: DevelopmentMotivationDepartmentPage, meta: { title: 'Отдел развития и мотивации', kiosk: true } },
      { path: 'admin', name: 'kiosk-admin', component: AdminDashboardPage, meta: { title: 'Дэшборд администратора', requiresAdmin: true, kiosk: true } },
    ],
  },
  { path: '/news', name: 'news', component: NewsPage, meta: { title: 'Новости' } },
  { path: '/news/:id', name: 'news-details', component: NewsDetailsPage, meta: { title: 'Новость' } },
  { path: '/events', name: 'events', component: EventsPage, meta: { title: 'Мероприятия' } },
  {
    path: '/events/:id',
    name: 'event-details',
    component: EventDetailsPage,
    meta: { title: 'Мероприятие' },
  },
  { path: '/gallery', name: 'gallery', component: GalleryPage, meta: { title: 'Фотогалерея' } },
  { path: '/gallery/:albumId', name: 'gallery-album', component: GalleryAlbumPage, meta: { title: 'Альбом' } },
  { path: '/newcomers', name: 'newcomers', component: NewcomersPage, meta: { title: 'Новичкам' } },
  { path: '/culture', name: 'culture', component: CulturePage, meta: { title: 'Корпоративная культура' } },
  { path: '/birthdays', name: 'birthdays', component: BirthdaysPage, meta: { title: 'Дни рождения коллег' } },
  { path: '/profile', name: 'profile', component: ProfilePage, meta: { title: 'Профиль' } },
  { path: '/absence-journal', name: 'absence-journal', component: AbsenceJournalPage, meta: { title: 'Журнал отсутствия' } },
  { path: '/applications', name: 'applications', component: ApplicationsPage, meta: { title: 'Заявки' } },
  { path: '/knowledge-base', name: 'knowledge-base', component: KnowledgeBasePage, meta: { title: 'База знаний' } },
  { path: '/personnel-reserve', name: 'personnel-reserve', component: PersonnelReservePage, meta: { title: 'Кадровый резерв' } },
  { path: '/tests', name: 'tests', component: TestsPage, meta: { title: 'Тесты' } },
  { path: '/hr-department', name: 'hr-department', component: HrDepartmentPage, meta: { title: 'Отдел кадров' } },
  { path: '/municipal-service', name: 'municipal-service', component: MunicipalServiceDepartmentPage, meta: { title: 'Отдел муниципальной службы' } },
  { path: '/development-motivation', name: 'development-motivation', component: DevelopmentMotivationDepartmentPage, meta: { title: 'Отдел развития и мотивации' } },
  { path: '/admin', name: 'admin', component: AdminDashboardPage, meta: { title: 'Дэшборд администратора', requiresAdmin: true } },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

const kioskRouteNameByName = {
  home: 'kiosk',
  news: 'kiosk-news',
  'news-details': 'kiosk-news-details',
  events: 'kiosk-events',
  'event-details': 'kiosk-event-details',
  gallery: 'kiosk-gallery',
  'gallery-album': 'kiosk-gallery-album',
  newcomers: 'kiosk-newcomers',
  culture: 'kiosk-culture',
  birthdays: 'kiosk-birthdays',
  profile: 'kiosk-profile',
  'absence-journal': 'kiosk-absence-journal',
  applications: 'kiosk-applications',
  'knowledge-base': 'kiosk-knowledge-base',
  'personnel-reserve': 'kiosk-personnel-reserve',
  tests: 'kiosk-tests',
  'hr-department': 'kiosk-hr-department',
  'municipal-service': 'kiosk-municipal-service',
  'development-motivation': 'kiosk-development-motivation',
  admin: 'kiosk-admin',
};

router.beforeEach((to, from, next) => {
  const toIsKiosk = to.matched?.some((r) => r.meta?.kiosk);
  const fromIsKiosk = from.matched?.some((r) => r.meta?.kiosk);

  // Киоск всегда работает как "user" (без админ-режима).
  if (toIsKiosk && currentRole.value !== 'user') {
    currentRole.value = 'user';
  }

  // На киоске админ-страницы недоступны даже если есть прямой URL.
  if (toIsKiosk && to.meta?.requiresAdmin) {
    return next({ name: 'kiosk' });
  }

  // Если уже находимся внутри киоска, то любые переходы ведём в kiosk-shell,
  // чтобы страницы открывались внутри `AppKiosk.vue`.
  if (fromIsKiosk && !toIsKiosk) {
    const rawName = to.name ? String(to.name) : '';
    const kioskName = kioskRouteNameByName[rawName];
    if (kioskName) {
      return next({ name: kioskName, params: to.params, query: to.query, hash: to.hash });
    }
  }

  if (to.meta?.requiresAdmin && currentRole.value !== 'admin') {
    return next({ name: 'profile' });
  }
  return next();
});

const DEFAULT_TITLE = 'Корпоративный портал';

router.afterEach((to) => {
  const pageTitle = to.meta?.title;
  if (typeof pageTitle === 'string' && pageTitle.length) {
    document.title = `${pageTitle} · ${DEFAULT_TITLE}`;
  } else {
    document.title = DEFAULT_TITLE;
  }
});

