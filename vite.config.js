import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import ui from '@nuxt/ui/vite';

export default defineConfig({
  plugins: [
    vue(),
    ui({
      ui: {
        colors: {
          primary: 'emerald',
          neutral: 'zinc'
        },
        container: {
          base: 'p-0 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0',
        },
        main: {
          base: 'min-h-[calc(100vh-var(--ui-header-height))] w-full max-w-[1600px] mx-auto',
        },
        pageHeader: {
          slots: {
            root: 'relative border-b border-default py-4',
          },
        },
        // Год в сегментах date picker: дефолтный w-11 слишком узкий для text-base / кастомного шрифта
        inputDate: {
          slots: {
            segment: [
              'rounded-sm text-center outline-hidden whitespace-nowrap data-placeholder:text-dimmed data-[segment=literal]:text-muted data-invalid:text-error data-disabled:cursor-not-allowed data-disabled:opacity-75',
              'transition-colors',
            ],
          },
          variants: {
            size: {
              xs: {
                segment: 'data-[segment=day]:w-8 data-[segment=month]:w-8 data-[segment=year]:w-12',
              },
              sm: {
                segment: 'data-[segment=day]:w-8 data-[segment=month]:w-8 data-[segment=year]:w-12',
              },
              md: {
                segment: 'data-[segment=day]:w-9 data-[segment=month]:w-9 data-[segment=year]:w-14',
              },
              lg: {
                segment: 'data-[segment=day]:w-10 data-[segment=month]:w-10 data-[segment=year]:w-14',
              },
              xl: {
                segment: 'data-[segment=day]:w-10 data-[segment=month]:w-10 data-[segment=year]:w-16',
              },
            },
          },
        },
      },
    }),
  ],
  server: {
    port: 5173,
    proxy: {
      // Бэкенд теперь редиректит 80→443, поэтому ходим сразу по https.
      // secure:false — игнорируем несовпадение wildcard-сертификата с IP (только для dev).
      //
      // Migrated endpoints → local Go API; everything else → PHP on test server.
      '/api/health.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/auth.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/logout.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/check-auth.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/heartbeat.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/session_bootstrap.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/news.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/events.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/gallery.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/gallery_base.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/Upload/upload.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/users.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/profile.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/feedback.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/ofo.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/ofo_seats.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/ofo_tree.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/ofo_positions.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/absence_journal.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/portal_groups.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/portal_my_permissions.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/portal_services.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/sync.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      // ДР: xlsx лежат на testing в uploads; локальный public/birthdays_xlsx пустой
      '/api/birthdays.php': {
        target: 'https://172.17.4.21',
        changeOrigin: true,
        secure: false,
      },
      '/api/tests_list.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_save.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_publish.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_unpublish.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_delete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_direct.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_submit.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_stats.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_participant.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_by_token.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_attempt_start.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_attempt_save.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_attempt_get.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/tests_attempt_finish.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms_list.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms_publish.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms_submit.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms_report.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms_archive.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/forms_delete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_list.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_get.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_create.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_update.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_delete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_publish.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_unpublish.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_topics_create.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_topics_update.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_topics_delete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_topics_order.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_materials_create.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_materials_update.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_materials_delete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_materials_upload.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_tests_create.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_tests_get.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_tests_update.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_tests_delete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_assign_preview.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_assign.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_admin_results.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_admin_participant.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_admin_attempt.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_enrollment_reset.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/courses_for_me.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_enrollment_get.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_start.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_topic_get.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_material_open.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_material_heartbeat.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_material_complete.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_next_action.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api/course_result.php': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/api': {
        target: 'https://172.17.4.21',
        changeOrigin: true,
        secure: false,
      },
      // Static uploads (FullPic/SmallPic) are served by backend web root in dev
      '/img': {
        target: 'https://172.17.4.21',
        changeOrigin: true,
        secure: false,
      },
    },
  },
});