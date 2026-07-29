<?php
/** POST /api/course_start.php — Body: {enrollmentId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
if ($enrollmentId <= 0) jsonError(400, 'Не передан enrollmentId');

$enr = cs_require_enrollment_access($pdo, $enrollmentId, $user, false);

if (in_array($enr['status'], ['completed', 'cancelled', 'failed'], true)) {
    jsonError(409, 'Нельзя начать этот курс');
}
if ($enr['starts_at'] && strtotime($enr['starts_at']) > time()) {
    jsonError(409, 'Курс ещё не доступен');
}

$pdo->beginTransaction();
try {
    if ($enr['status'] === 'not_started' || $enr['status'] === 'overdue') {
        $pdo->prepare(
            "UPDATE public.course_enrollments
             SET status = 'in_progress',
                 started_at = COALESCE(started_at, now()),
                 last_activity_at = now(),
                 updated_at = now()
             WHERE id = :id"
        )->execute([':id' => $enrollmentId]);
    } else {
        $pdo->prepare(
            'UPDATE public.course_enrollments SET last_activity_at = now(), updated_at = now() WHERE id = :id'
        )->execute([':id' => $enrollmentId]);
    }
    cs_ensure_topic_progress_rows($pdo, $enrollmentId, (int)$enr['course_version_id']);
    cs_recalculate_locks($pdo, $enrollmentId);
    cs_audit($pdo, (int)$user['id'], 'course.enrollment.start', 'course_enrollment', $enrollmentId, []);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка запуска');
}

$enr = cs_enrollment_row($pdo, $enrollmentId);
jsonOk([
    'enrollment' => cs_map_enrollment($enr, cs_enrollment_progress($pdo, $enrollmentId)),
    'nextAction' => cs_next_action($pdo, $enrollmentId),
]);
