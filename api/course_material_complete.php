<?php
/** POST /api/course_material_complete.php — Body: {enrollmentId, materialId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
$materialId = (int)($body['materialId'] ?? 0);
if ($enrollmentId <= 0 || $materialId <= 0) jsonError(400, 'Нужны enrollmentId и materialId');

$enr = cs_require_enrollment_access($pdo, $enrollmentId, $user, false);
$m = cs_material_version_row($pdo, $materialId);
if (!$m || (int)$m['course_version_id'] !== (int)$enr['course_version_id']) {
    jsonError(404, 'Материал не найден');
}
$topicId = (int)$m['topic_id'];

$mp = $pdo->prepare('SELECT * FROM public.course_material_progress WHERE enrollment_id = :e AND material_id = :m');
$mp->execute([':e' => $enrollmentId, ':m' => $materialId]);
$prog = $mp->fetch();
$min = (int)$m['minimum_active_seconds'];
$active = (int)($prog['active_seconds'] ?? 0);
if ($min > 0 && $active < $min) {
    jsonError(409, "Недостаточно активного времени (нужно {$min} сек, есть {$active})");
}

$pdo->beginTransaction();
try {
    $pdo->prepare(
        "INSERT INTO public.course_material_progress (enrollment_id, material_id, status, opened_at, completed_at)
         VALUES (:e, :m, 'completed', now(), now())
         ON CONFLICT (enrollment_id, material_id) DO UPDATE SET
           status = 'completed',
           completed_at = COALESCE(course_material_progress.completed_at, now()),
           updated_at = now()"
    )->execute([':e' => $enrollmentId, ':m' => $materialId]);

    // Закрыть learning sessions
    $pdo->prepare(
        'UPDATE public.course_learning_sessions SET finished_at = now()
         WHERE enrollment_id = :e AND material_id = :m AND finished_at IS NULL'
    )->execute([':e' => $enrollmentId, ':m' => $materialId]);

    // Автозавершение темы при выполнении условий
    if (cs_check_topic_complete($pdo, $enrollmentId, $topicId)) {
        $pdo->prepare(
            "UPDATE public.course_topic_progress
             SET status = 'completed', completed_at = COALESCE(completed_at, now()), updated_at = now()
             WHERE enrollment_id = :e AND topic_id = :t"
        )->execute([':e' => $enrollmentId, ':t' => $topicId]);
        cs_recalculate_locks($pdo, $enrollmentId);
    }

    $pdo->prepare(
        'UPDATE public.course_enrollments SET last_activity_at = now(), updated_at = now() WHERE id = :id'
    )->execute([':id' => $enrollmentId]);

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка завершения материала');
}

try {
    cs_try_complete_enrollment($pdo, $enrollmentId);
} catch (Throwable $e) {
    // ignore
}

jsonOk([
    'materialId' => $materialId,
    'nextAction' => cs_next_action($pdo, $enrollmentId),
    'progress' => cs_enrollment_progress($pdo, $enrollmentId),
]);
