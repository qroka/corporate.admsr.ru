import { computed, onUnmounted, watch } from 'vue';
import { useCoursesStore } from '../../composables/useCoursesStore';
import {
  useBreadcrumbCurrentLabel,
  useBreadcrumbLabelsByRoute,
} from '../../composables/usePortalNavigation';

/**
 * Подставляет название обучения в портальные крошки:
 * — asCurrent: текущий сегмент (хаб курса);
 * — всегда: родитель «admin-course-workspace» на вложенных страницах.
 */
export function useAdminCoursePortalBreadcrumbs(options?: { asCurrent?: boolean }) {
  const asCurrent = options?.asCurrent === true;
  const store = useCoursesStore();
  const breadcrumbLabel = useBreadcrumbCurrentLabel();
  const breadcrumbByRoute = useBreadcrumbLabelsByRoute();

  const courseTitle = computed(() => String(store.current.value?.title || '').trim());

  watch(
    courseTitle,
    (title) => {
      if (title) {
        breadcrumbByRoute.value = {
          ...breadcrumbByRoute.value,
          'admin-course-workspace': title,
        };
      }
      if (asCurrent) {
        breadcrumbLabel.value = title || null;
      }
    },
    { immediate: true },
  );

  onUnmounted(() => {
    if (asCurrent) {
      breadcrumbLabel.value = null;
    }
  });
}
