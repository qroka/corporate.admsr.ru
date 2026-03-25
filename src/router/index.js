import { createRouter, createWebHistory } from 'vue-router';
import { currentRole } from '../stores/role';
import HomePage from '../pages/HomePage.vue';
import EventsPage from '../pages/Events/EventsPage.vue';
import EventDetailsPage from '../pages/Events/EventDetailsPage.vue';
import GalleryPage from '../pages/Gallery/GalleryPage.vue';
import GalleryAlbumPage from '../pages/Gallery/GalleryAlbumPage.vue';
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

const routes = [
  { path: '/', name: 'home', component: HomePage, meta: { title: 'Главная' } },
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
  { path: '/profile', name: 'profile', component: ProfilePage, meta: { title: 'Профиль' } },
  { path: '/absence-journal', name: 'absence-journal', component: AbsenceJournalPage, meta: { title: 'Журнал отсутствия' } },
  { path: '/applications', name: 'applications', component: ApplicationsPage, meta: { title: 'Заявки' } },
  { path: '/knowledge-base', name: 'knowledge-base', component: KnowledgeBasePage, meta: { title: 'База знаний' } },
  { path: '/personnel-reserve', name: 'personnel-reserve', component: PersonnelReservePage, meta: { title: 'Кадровый резерв' } },
  { path: '/tests', name: 'tests', component: TestsPage, meta: { title: 'Тесты' } },
  { path: '/hr-department', name: 'hr-department', component: HrDepartmentPage, meta: { title: 'Отдел кадров' } },
  { path: '/municipal-service', name: 'municipal-service', component: MunicipalServiceDepartmentPage, meta: { title: 'Отдел муниципальной службы' } },
  { path: '/development-motivation', name: 'development-motivation', component: DevelopmentMotivationDepartmentPage, meta: { title: 'Отдел развития и мотивации' } },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, from, next) => {
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

