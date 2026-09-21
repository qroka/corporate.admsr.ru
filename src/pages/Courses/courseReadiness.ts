export type CourseReadinessCheck = {
  id: string;
  label: string;
  ok: boolean;
};

export type CourseReadiness = {
  ready: boolean;
  errors: string[];
  warnings: string[];
  checks: CourseReadinessCheck[];
};

type AnyRow = Record<string, any>;

/** Клиентский чеклист — те же правила, что VersionReadiness на Go API. */
export function courseReadiness(version: AnyRow | null | undefined): CourseReadiness {
  const errors: string[] = [];
  const warnings: string[] = [];
  const checks: CourseReadinessCheck[] = [];
  if (!version) {
    return {
      ready: false,
      errors: ['Версия курса не загружена'],
      warnings,
      checks: [{ id: 'version', label: 'Версия курса загружена', ok: false }],
    };
  }

  const topics: AnyRow[] = Array.isArray(version.topics) ? version.topics : [];
  const hasTopics = topics.length > 0;
  checks.push({ id: 'topics', label: 'Есть хотя бы одна тема', ok: hasTopics });
  if (!hasTopics) errors.push('Нет тем в версии курса');

  let materialsOk = true;
  let testsOk = true;
  for (const topic of topics) {
    const title = String(topic?.title || 'Тема').trim() || 'Тема';
    const mats: AnyRow[] = Array.isArray(topic?.materials) ? topic.materials : [];
    const matCount = Number(topic?.materialsCount ?? mats.length ?? 0);
    const test = topic?.topicTest || topic?.testLink || null;
    if (topic?.isRequired === false) {
      if (!matCount && !test) warnings.push(`Тема «${title}» необязательна и пуста`);
      continue;
    }
    if (!matCount) {
      materialsOk = false;
      errors.push(`Обязательная тема «${title}» без материалов`);
    }
    const qCount = Number(test?.questionCount ?? test?.form?.questionCount ?? 0);
    if (test && test.isRequired !== false && qCount < 1) {
      testsOk = false;
      errors.push(`Тест темы «${title}» без вопросов`);
    }
  }
  if (hasTopics) {
    checks.push({
      id: 'materials',
      label: 'У обязательных тем есть материалы',
      ok: materialsOk,
    });
    checks.push({
      id: 'topic-tests',
      label: 'Тесты тем с вопросами (если подключены)',
      ok: testsOk,
    });
  }

  const finalTest = version.finalTest || null;
  if (version.requireFinalTest) {
    let finalOk = false;
    if (!finalTest) {
      errors.push('Требуется итоговый тест, но он не создан');
    } else {
      const qCount = Number(finalTest.questionCount ?? finalTest.form?.questionCount ?? 0);
      if (qCount < 1) errors.push('Итоговый тест без вопросов');
      else finalOk = true;
    }
    checks.push({
      id: 'final-test',
      label: 'Итоговый тест с вопросами',
      ok: finalOk,
    });
  } else if (finalTest) {
    warnings.push('Итоговый тест есть, но требование итогового теста выключено');
    checks.push({
      id: 'final-test',
      label: 'Итоговый тест (необязателен)',
      ok: true,
    });
  }

  return { ready: errors.length === 0, errors, warnings, checks };
}

/** Следующий шаг автора по чеклисту готовности. */
export function courseAuthorNextStep(
  courseId: number,
  version: AnyRow | null | undefined,
): { label: string; to: { name: string; params: Record<string, number | string> } } | null {
  if (!version) return null;
  const topics: AnyRow[] = Array.isArray(version.topics) ? version.topics : [];
  if (!topics.length) {
    return {
      label: 'Добавить первую тему',
      to: { name: 'admin-course-topic-create', params: { courseId } },
    };
  }

  for (const topic of topics) {
    if (topic?.isRequired === false) continue;
    const mats: AnyRow[] = Array.isArray(topic?.materials) ? topic.materials : [];
    const matCount = Number(topic?.materialsCount ?? mats.length ?? 0);
    const topicId = Number(topic.id);
    if (!matCount && topicId) {
      return {
        label: `Добавить материалы: «${String(topic.title || 'Тема')}»`,
        to: {
          name: 'admin-course-material-create',
          params: { courseId, topicId },
        },
      };
    }
    const test = topic?.topicTest || topic?.testLink || null;
    const qCount = Number(test?.questionCount ?? test?.form?.questionCount ?? 0);
    if (test && test.isRequired !== false && qCount < 1 && topicId) {
      return {
        label: `Добавить вопросы теста: «${String(topic.title || 'Тема')}»`,
        to: {
          name: 'admin-course-topic-test',
          params: { courseId, topicId },
        },
      };
    }
  }

  if (version.requireFinalTest) {
    const finalTest = version.finalTest || null;
    const qCount = Number(finalTest?.questionCount ?? finalTest?.form?.questionCount ?? 0);
    if (!finalTest || qCount < 1) {
      return {
        label: finalTest ? 'Дополнить итоговый тест' : 'Создать итоговый тест',
        to: { name: 'admin-course-final-test', params: { courseId } },
      };
    }
  }

  return {
    label: 'К публикации',
    to: { name: 'admin-course-publish', params: { courseId } },
  };
}
