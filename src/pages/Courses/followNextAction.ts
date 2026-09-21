import type { Router } from 'vue-router';

export type CourseNextAction = {
  type?: string;
  label?: string;
  topicId?: number;
  courseTestLinkId?: number;
  materialId?: number;
};

/** Куда вести после успешного шага (тест, тема). Неизвестный шаг — карта курса. */
export async function followCourseNextAction(
  router: Router,
  enrollmentId: number,
  action: CourseNextAction | null | undefined,
) {
  const type = String(action?.type ?? '');
  const topicId = Number(action?.topicId || 0);
  const linkId = Number(action?.courseTestLinkId || 0);

  if ((type === 'topic_test' || type === 'final_test') && linkId > 0) {
    await router.push({
      name: 'course-test',
      params: { enrollmentId, courseTestLinkId: linkId },
    });
    return;
  }

  if (topicId > 0 && (type === 'material' || type === 'topic' || type === 'complete_topic')) {
    await router.push({
      name: 'course-topic',
      params: { enrollmentId, topicId },
    });
    return;
  }

  if (type === 'done' || type === 'complete_course' || type === 'failed') {
    await router.push({ name: 'course-result', params: { enrollmentId } });
    return;
  }

  await router.push({ name: 'course-enrollment', params: { enrollmentId } });
}
