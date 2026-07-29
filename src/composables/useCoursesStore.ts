import { computed, ref } from 'vue';
import { apiSessionFetch, apiSessionUpload, type ApiResult } from './useAuthSession';

export type CourseListItem = {
  id: number;
  title: string;
  category?: string | null;
  ownerId?: number;
  updatedAt?: string;
  currentVersionId?: number | null;
  versionNumber?: number | null;
  status?: string | null;
  topicsCount?: number;
  publishedAt?: string | null;
};

export type CourseMaterial = {
  id: number;
  topicId: number;
  type: 'rich_text' | 'file' | 'pdf' | 'image' | 'video' | 'link';
  title: string;
  description?: string;
  contentHtml?: string | null;
  fileUrl?: string | null;
  externalUrl?: string | null;
  mimeType?: string | null;
  fileSize?: number | null;
  sortOrder?: number;
  isRequired?: boolean;
  minimumActiveSeconds?: number;
};

export type CourseTopic = {
  id: number;
  title: string;
  description?: string;
  sortOrder?: number;
  isRequired?: boolean;
  minimumActiveSeconds?: number;
  materials?: CourseMaterial[];
  materialsCount?: number;
  testLink?: CourseTestLink | null;
  questionCount?: number;
  ready?: boolean;
};

export type CourseTestLink = {
  id: number;
  type: 'topic' | 'final';
  topicId?: number | null;
  testFormId: number;
  isRequired?: boolean;
  title?: string;
  status?: string;
  questionCount?: number;
};

export type CourseVersion = {
  id: number;
  courseId: number;
  versionNumber: number;
  status: 'draft' | 'published' | 'archived';
  shortDescription?: string;
  fullDescription?: string;
  coverUrl?: string | null;
  sequentialProgress?: boolean;
  completionRule?: string;
  defaultDeadlineDays?: number | null;
  finalPassingScore?: number | null;
  requireFinalTest?: boolean;
  topics?: CourseTopic[];
  finalTest?: CourseTestLink | null;
  updatedAt?: string;
  publishedAt?: string | null;
};

export type CourseDetail = {
  id: number;
  title: string;
  category?: string | null;
  ownerId?: number;
  currentVersionId?: number | null;
  version?: CourseVersion | null;
  readiness?: { ready: boolean; errors: string[]; warnings: string[] };
};

export type EnrollmentSummary = {
  id: number;
  courseId?: number;
  courseTitle: string;
  versionNumber?: number;
  status: string;
  progressPercent?: number;
  topicsCompleted?: number;
  topicsTotal?: number;
  deadlineAt?: string | null;
  lastActivityAt?: string | null;
  currentTopicTitle?: string | null;
  nextAction?: { type: string; label: string; topicId?: number; courseTestLinkId?: number };
};

const courses = ref<CourseListItem[]>([]);
const current = ref<CourseDetail | null>(null);
const myEnrollments = ref<EnrollmentSummary[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

function unwrap<T>(res: ApiResult<T>, fallbackMsg: string): T {
  if (!res.success) throw new Error(res.message || fallbackMsg);
  return res.data as T;
}

function normalizeCourseDetail(data: any): CourseDetail {
  if (data?.course) {
    const course = data.course;
    const version = data.version ?? course.version ?? course.currentVersion ?? null;
    return {
      id: Number(course.id),
      title: String(course.title ?? ''),
      category: course.category ?? null,
      ownerId: course.ownerId,
      currentVersionId: course.currentVersionId ?? version?.id ?? null,
      version: version ?? null,
      readiness: data.readiness ?? course.readiness,
    };
  }
  return data as CourseDetail;
}

export function useCoursesStore() {
  async function loadList() {
    loading.value = true;
    error.value = null;
    try {
      const res = await apiSessionFetch<CourseListItem[] | { items?: CourseListItem[] }>('/api/courses_list.php', {
        method: 'POST',
        json: {},
      });
      const data = unwrap(res, 'Не удалось загрузить курсы') as any;
      const raw = Array.isArray(data) ? data : data?.items || [];
      courses.value = raw.map((c: any) => ({
        id: Number(c.id),
        title: String(c.title ?? ''),
        category: c.category ?? null,
        ownerId: c.ownerId,
        updatedAt: c.updatedAt,
        currentVersionId: c.currentVersionId ?? c.currentVersion?.id ?? null,
        versionNumber: c.versionNumber ?? c.currentVersion?.versionNumber ?? null,
        status: c.status ?? c.currentVersion?.status ?? null,
        topicsCount: c.topicsCount,
        publishedAt: c.publishedAt ?? c.currentVersion?.publishedAt ?? null,
      }));
    } catch (e: any) {
      error.value = e.message || 'Ошибка';
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function loadCourse(courseId: number, versionId?: number) {
    loading.value = true;
    error.value = null;
    try {
      const res = await apiSessionFetch<CourseDetail>('/api/courses_get.php', {
        method: 'POST',
        json: { courseId, versionId },
      });
      current.value = normalizeCourseDetail(unwrap(res, 'Курс не найден'));
      return current.value;
    } catch (e: any) {
      error.value = e.message || 'Ошибка';
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function createCourse(payload: Record<string, unknown>) {
    const res = await apiSessionFetch<CourseDetail>('/api/courses_create.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось создать курс');
  }

  async function updateCourse(payload: Record<string, unknown>) {
    const res = await apiSessionFetch<CourseDetail>('/api/courses_update.php', {
      method: 'POST',
      json: payload,
    });
    current.value = normalizeCourseDetail(unwrap(res, 'Не удалось сохранить'));
    return current.value;
  }

  async function deleteCourse(courseId: number) {
    const res = await apiSessionFetch('/api/courses_delete.php', {
      method: 'POST',
      json: { courseId },
    });
    unwrap(res, 'Не удалось удалить');
  }

  async function publishCourse(courseId: number) {
    const res = await apiSessionFetch('/api/courses_publish.php', {
      method: 'POST',
      json: { courseId },
    });
    return unwrap(res, 'Не удалось опубликовать');
  }

  async function readiness(courseId: number) {
    const res = await apiSessionFetch<{ ready: boolean; errors: string[]; warnings: string[] }>(
      '/api/courses_readiness.php',
      { method: 'POST', json: { courseId } },
    );
    return unwrap(res, 'Не удалось проверить готовность');
  }

  async function createTopic(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_topics_create.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось создать тему');
  }

  async function updateTopic(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_topics_update.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось сохранить тему');
  }

  async function deleteTopic(topicId: number) {
    const res = await apiSessionFetch('/api/course_topics_delete.php', {
      method: 'POST',
      json: { topicId },
    });
    unwrap(res, 'Не удалось удалить тему');
  }

  async function orderTopics(versionId: number, topicIds: number[]) {
    const res = await apiSessionFetch('/api/course_topics_order.php', {
      method: 'POST',
      json: { versionId, topicIds },
    });
    unwrap(res, 'Не удалось изменить порядок');
  }

  async function createMaterial(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_materials_create.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось создать материал');
  }

  async function updateMaterial(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_materials_update.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось сохранить материал');
  }

  async function deleteMaterial(materialId: number) {
    const res = await apiSessionFetch('/api/course_materials_delete.php', {
      method: 'POST',
      json: { materialId },
    });
    unwrap(res, 'Не удалось удалить материал');
  }

  async function uploadMaterialFile(topicId: number, file: File, extra: Record<string, string> = {}) {
    const fd = new FormData();
    fd.append('file', file);
    fd.append('topicId', String(topicId));
    Object.entries(extra).forEach(([k, v]) => fd.append(k, v));
    const res = await apiSessionUpload('/api/course_materials_upload.php', fd);
    return unwrap(res, 'Не удалось загрузить файл');
  }

  async function createCourseTest(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_tests_create.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось создать тест');
  }

  async function getCourseTest(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_tests_get.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось загрузить тест');
  }

  async function updateCourseTest(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_tests_update.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось сохранить тест');
  }

  async function deleteCourseTest(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_tests_delete.php', {
      method: 'POST',
      json: payload,
    });
    unwrap(res, 'Не удалось удалить тест');
  }

  async function assignPreview(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_assign_preview.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось сформировать список');
  }

  async function assign(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_assign.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось назначить');
  }

  async function loadResults(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_admin_results.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось загрузить отчёт');
  }

  async function loadParticipant(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_admin_participant.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось загрузить карточку');
  }

  // Employee
  async function loadMyCourses() {
    loading.value = true;
    try {
      const res = await apiSessionFetch<{ items?: EnrollmentSummary[] } | EnrollmentSummary[]>(
        '/api/courses_for_me.php',
        { method: 'POST', json: {} },
      );
      const data = unwrap(res, 'Не удалось загрузить курсы') as any;
      if (Array.isArray(data)) {
        myEnrollments.value = data;
        return { all: data };
      }
      // API groups: active / completed / overdue / failed
      const mapItem = (x: any): EnrollmentSummary => {
        const enr = x?.enrollment ?? x;
        const course = x?.course ?? enr?.course;
        const prog = enr?.progress ?? x?.progress;
        return {
          id: Number(enr?.id),
          courseId: course?.id != null ? Number(course.id) : enr?.courseId,
          courseTitle: String(course?.title ?? enr?.courseTitle ?? 'Курс'),
          versionNumber: course?.versionNumber ?? enr?.versionNumber,
          status: String(enr?.status ?? ''),
          progressPercent: prog?.percent ?? enr?.progressPercent ?? 0,
          topicsCompleted: prog?.topicsCompleted ?? enr?.topicsCompleted ?? 0,
          topicsTotal: prog?.topicsTotal ?? enr?.topicsTotal ?? 0,
          deadlineAt: enr?.deadlineAt ?? null,
          lastActivityAt: enr?.lastActivityAt ?? null,
          currentTopicTitle: null,
          nextAction: prog?.nextAction ?? enr?.nextAction,
        };
      };
      const groups = {
        overdue: (data?.overdue || []).map(mapItem),
        active: (data?.active || []).map(mapItem),
        completed: (data?.completed || []).map(mapItem),
        failed: (data?.failed || []).map(mapItem),
      };
      myEnrollments.value = [
        ...groups.overdue,
        ...groups.active,
        ...groups.completed,
        ...groups.failed,
      ];
      return groups;
    } finally {
      loading.value = false;
    }
  }

  async function getEnrollment(enrollmentId: number) {
    const res = await apiSessionFetch('/api/course_enrollment_get.php', {
      method: 'POST',
      json: { enrollmentId },
    });
    return unwrap(res, 'Назначение не найдено');
  }

  async function startCourse(enrollmentId: number) {
    const res = await apiSessionFetch('/api/course_start.php', {
      method: 'POST',
      json: { enrollmentId },
    });
    return unwrap(res, 'Не удалось начать курс');
  }

  async function getTopic(enrollmentId: number, topicId: number) {
    const res = await apiSessionFetch('/api/course_topic_get.php', {
      method: 'POST',
      json: { enrollmentId, topicId },
    });
    return unwrap(res, 'Тема недоступна');
  }

  async function openMaterial(enrollmentId: number, materialId: number) {
    const res = await apiSessionFetch('/api/course_material_open.php', {
      method: 'POST',
      json: { enrollmentId, materialId },
    });
    return unwrap(res, 'Не удалось открыть материал');
  }

  async function heartbeat(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/course_material_heartbeat.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Heartbeat failed');
  }

  async function completeMaterial(enrollmentId: number, materialId: number) {
    const res = await apiSessionFetch('/api/course_material_complete.php', {
      method: 'POST',
      json: { enrollmentId, materialId },
    });
    return unwrap(res, 'Не удалось завершить материал');
  }

  async function nextAction(enrollmentId: number) {
    const res = await apiSessionFetch('/api/course_next_action.php', {
      method: 'POST',
      json: { enrollmentId },
    });
    return unwrap(res, 'Не удалось определить шаг');
  }

  async function loadHistory() {
    const res = await apiSessionFetch('/api/course_history.php', {
      method: 'POST',
      json: {},
    });
    return unwrap(res, 'Не удалось загрузить историю');
  }

  async function loadResult(enrollmentId: number) {
    const res = await apiSessionFetch('/api/course_result.php', {
      method: 'POST',
      json: { enrollmentId },
    });
    return unwrap(res, 'Результат недоступен');
  }

  // Attempts
  async function attemptStart(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/tests_attempt_start.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось начать попытку');
  }

  async function attemptSave(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/tests_attempt_save.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось сохранить ответы');
  }

  async function attemptGet(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/tests_attempt_get.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Попытка не найдена');
  }

  async function attemptFinish(payload: Record<string, unknown>) {
    const res = await apiSessionFetch('/api/tests_attempt_finish.php', {
      method: 'POST',
      json: payload,
    });
    return unwrap(res, 'Не удалось завершить попытку');
  }

  const version = computed(() => current.value?.version ?? null);
  const topics = computed(() => version.value?.topics ?? []);

  return {
    courses,
    current,
    version,
    topics,
    myEnrollments,
    loading,
    error,
    loadList,
    loadCourse,
    createCourse,
    updateCourse,
    deleteCourse,
    publishCourse,
    readiness,
    createTopic,
    updateTopic,
    deleteTopic,
    orderTopics,
    createMaterial,
    updateMaterial,
    deleteMaterial,
    uploadMaterialFile,
    createCourseTest,
    getCourseTest,
    updateCourseTest,
    deleteCourseTest,
    assignPreview,
    assign,
    loadResults,
    loadParticipant,
    loadMyCourses,
    getEnrollment,
    startCourse,
    getTopic,
    openMaterial,
    heartbeat,
    completeMaterial,
    nextAction,
    loadHistory,
    loadResult,
    attemptStart,
    attemptSave,
    attemptGet,
    attemptFinish,
  };
}
