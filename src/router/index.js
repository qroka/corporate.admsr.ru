import { createRouter, createWebHistory } from 'vue-router';
import { currentRole } from '../stores/role';
import { useSectionAccess } from '../composables/useSectionAccess';
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
import TestsBlankPage from '../pages/TestsBlankPage.vue';
import HrDepartmentPage from '../pages/HrDepartmentPage.vue';
import MunicipalServiceDepartmentPage from '../pages/MunicipalServiceDepartmentPage.vue';
import DevelopmentMotivationDepartmentPage from '../pages/DevelopmentMotivationDepartmentPage.vue';
import AdminDashboardPage from '../pages/Admin/AdminDashboardPage.vue';
import BirthdaysPage from '../pages/BirthdaysPage.vue';
import LoginPage from '../pages/login.vue';
import OnboardingPage from '../pages/OnboardingPage.vue';
import ChatBotPage from '../pages/ChatBotPage.vue';
import TestLinkPage from '../pages/TestLinkPage.vue';
import CoursesListPage from '../pages/Courses/admin/CoursesListPage.vue';
import CourseCreatePage from '../pages/Courses/admin/CourseCreatePage.vue';
import CourseWorkspacePage from '../pages/Courses/admin/CourseWorkspacePage.vue';
import CourseSettingsPage from '../pages/Courses/admin/CourseSettingsPage.vue';
import TopicFormPage from '../pages/Courses/admin/TopicFormPage.vue';
import MaterialFormPage from '../pages/Courses/admin/MaterialFormPage.vue';
import TopicTestPage from '../pages/Courses/admin/TopicTestPage.vue';
import FinalTestPage from '../pages/Courses/admin/FinalTestPage.vue';
import CoursePublishPage from '../pages/Courses/admin/CoursePublishPage.vue';
import CourseAssignPage from '../pages/Courses/admin/CourseAssignPage.vue';
import CourseResultsPage from '../pages/Courses/admin/CourseResultsPage.vue';
import MyCoursesPage from '../pages/Courses/employee/MyCoursesPage.vue';
import CourseEnrollmentPage from '../pages/Courses/employee/CourseEnrollmentPage.vue';
import CourseTopicPage from '../pages/Courses/employee/CourseTopicPage.vue';
import CourseTestPage from '../pages/Courses/employee/CourseTestPage.vue';
import CourseResultPage from '../pages/Courses/employee/CourseResultPage.vue';
import { userNeedsOnboarding } from '../composables/useOnboarding';

const routes = [
  { path: '/t/:token', name: 'test-link', component: TestLinkPage, meta: { title: 'Прохождение', public: true } },
  { path: '/login', name: 'login', component: LoginPage, meta: { title: 'Вход', layout: 'auth' } },
  { path: '/welcome', name: 'onboarding', component: OnboardingPage, meta: { title: 'Добро пожаловать', layout: 'auth' } },
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
  { path: '/tests', name: 'tests', component: TestsBlankPage, meta: { title: 'Тесты' } },
  { path: '/tests/old', name: 'tests-old', component: TestsPage, meta: { title: 'Тесты (старые)' } },
  { path: '/hr-department', name: 'hr-department', component: HrDepartmentPage, meta: { title: 'Отдел кадров' } },
  { path: '/municipal-service', name: 'municipal-service', component: MunicipalServiceDepartmentPage, meta: { title: 'Отдел муниципальной службы' } },
  { path: '/development-motivation', name: 'development-motivation', component: DevelopmentMotivationDepartmentPage, meta: { title: 'Отдел развития и мотивации' } },
  { path: '/admin', name: 'admin', component: AdminDashboardPage, meta: { title: 'Дэшборд администратора', requiresAdmin: true } },
  { path: '/chatbot', name: 'chatbot', component: ChatBotPage, meta: { title: 'AI Ассистент' } },

  // ── Курсы: админ ───────────────────────────────────────────────────────────
  { path: '/admin/courses', name: 'admin-courses', component: CoursesListPage, meta: { title: 'Управление курсами', requiresSection: 'courses' } },
  { path: '/admin/courses/create', name: 'admin-course-create', component: CourseCreatePage, meta: { title: 'Новый курс', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId', name: 'admin-course-workspace', component: CourseWorkspacePage, meta: { title: 'Курс', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/settings', name: 'admin-course-settings', component: CourseSettingsPage, meta: { title: 'Настройки курса', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/create', name: 'admin-course-topic-create', component: TopicFormPage, meta: { title: 'Новая тема', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/:topicId', name: 'admin-course-topic-edit', component: TopicFormPage, meta: { title: 'Тема', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/:topicId/materials/create', name: 'admin-course-material-create', component: MaterialFormPage, meta: { title: 'Новый материал', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/:topicId/materials/:materialId', name: 'admin-course-material-edit', component: MaterialFormPage, meta: { title: 'Материал', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/:topicId/test', name: 'admin-course-topic-test', component: TopicTestPage, meta: { title: 'Тест темы', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/:topicId/test/questions/create', name: 'admin-course-topic-test-q-create', component: TopicTestPage, meta: { title: 'Тест темы', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/topics/:topicId/test/questions/:questionId', name: 'admin-course-topic-test-q-edit', component: TopicTestPage, meta: { title: 'Тест темы', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/final-test', name: 'admin-course-final-test', component: FinalTestPage, meta: { title: 'Итоговый тест', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/final-test/questions/create', name: 'admin-course-final-test-q-create', component: FinalTestPage, meta: { title: 'Итоговый тест', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/final-test/questions/:questionId', name: 'admin-course-final-test-q-edit', component: FinalTestPage, meta: { title: 'Итоговый тест', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/publish', name: 'admin-course-publish', component: CoursePublishPage, meta: { title: 'Публикация курса', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/review', redirect: (to) => ({ name: 'admin-course-publish', params: { courseId: to.params.courseId } }) },

  { path: '/admin/courses/:courseId/assign', name: 'admin-course-assign', component: CourseAssignPage, meta: { title: 'Назначение курса', requiresSection: 'courses' } },
  { path: '/admin/courses/:courseId/results', name: 'admin-course-results', component: CourseResultsPage, meta: { title: 'Результаты курса', requiresSection: 'courses' } },

  // ── Курсы: сотрудник ───────────────────────────────────────────────────────
  { path: '/courses', name: 'courses', component: MyCoursesPage, meta: { title: 'Мои курсы' } },
  { path: '/courses/history', redirect: { name: 'courses' } },
  { path: '/courses/:enrollmentId', name: 'course-enrollment', component: CourseEnrollmentPage, meta: { title: 'Курс' } },
  { path: '/courses/:enrollmentId/topics/:topicId', name: 'course-topic', component: CourseTopicPage, meta: { title: 'Тема курса' } },
  { path: '/courses/:enrollmentId/tests/:courseTestLinkId', name: 'course-test', component: CourseTestPage, meta: { title: 'Тест курса' } },
  { path: '/courses/:enrollmentId/result', name: 'course-result', component: CourseResultPage, meta: { title: 'Результат курса' } },
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

const AUTH_CHECK_INTERVAL = 15 * 60 * 1000; // 15 минут

function getStoredUser() {
  try {
    return JSON.parse(localStorage.getItem('auth-user'));
  } catch {
    return null;
  }
}

async function checkAuthInDb(id) {
  try {
    const res = await fetch('/api/check-auth.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id }),
    });
    const data = await res.json();
    if (data.success && data.auth === true) {
      localStorage.setItem('auth-last-check', Date.now().toString());
      return true;
    }
    return false;
  } catch {
    return false;
  }
}

function needsDbCheck() {
  const last = parseInt(localStorage.getItem('auth-last-check') || '0', 10);
  return Date.now() - last >= AUTH_CHECK_INTERVAL;
}

router.beforeEach(async (to, from, next) => {
  // Публичные страницы (например, прохождение формы по ссылке) — без авторизации
  if (to.meta?.public) return next();

  const toIsKiosk = to.matched?.some((r) => r.meta?.kiosk);
  const fromIsKiosk = from.matched?.some((r) => r.meta?.kiosk);
  const isAuth = to.meta?.layout === 'auth';

  if (!toIsKiosk) {
    const user = getStoredUser();

    if (!isAuth) {
      if (!user) return next({ name: 'login' });

      if (needsDbCheck()) {
        const valid = await checkAuthInDb(user.id);
        if (!valid) {
          localStorage.removeItem('auth-user');
          localStorage.removeItem('auth-last-check');
          return next({ name: 'login' });
        }
      }
    }

    if (isAuth && user) {
      if (needsDbCheck()) {
        const valid = await checkAuthInDb(user.id);
        if (!valid) {
          localStorage.removeItem('auth-user');
          localStorage.removeItem('auth-last-check');
          return next();
        }
      }
      if (to.name === 'onboarding') return next();
      const needsWelcome = await userNeedsOnboarding(user.id);
      return next(needsWelcome ? { name: 'onboarding' } : { name: 'home' });
    }

    if (user && to.name !== 'onboarding') {
      const needsWelcome = await userNeedsOnboarding(user.id);
      if (needsWelcome) return next({ name: 'onboarding' });
    }
  }

  // Киоск всегда работает как "user" (без админ-режима).
  if (toIsKiosk && currentRole.value !== 'user') {
    currentRole.value = 'user';
  }

  // На киоске админ-страницы недоступны даже если есть прямой URL.
  if (toIsKiosk && (to.meta?.requiresAdmin || to.meta?.requiresSection)) {
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

  // Дашборд /admin — только суперadmin с включённым UI-тогглом.
  if (to.meta?.requiresAdmin) {
    const { isSuperAdmin, ensureLoaded } = useSectionAccess();
    await ensureLoaded();
    if (!isSuperAdmin.value || currentRole.value !== 'admin') {
      return next({ name: 'profile' });
    }
  }

  // Разделы вроде /admin/courses* — по праву секции (группа или суперadmin+toggle).
  if (to.meta?.requiresSection) {
    const { ensureLoaded, canEditSection } = useSectionAccess();
    await ensureLoaded();
    if (!canEditSection(String(to.meta.requiresSection))) {
      return next({ name: 'courses' });
    }
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

