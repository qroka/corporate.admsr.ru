<?php
/**
 * POST /api/course_enrollment_reset.php
 * Body: {enrollmentId}
 * Админ: обнуляет прогресс, попытки тестов и completion — enrollment снова not_started.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
if ($enrollmentId <= 0) jsonError(400, 'Не передан enrollmentId');

$enr = cs_enrollment_row($pdo, $enrollmentId);
if (!$enr) jsonError(404, 'Запись не найдена');

$version = cs_get_version($pdo, (int)$enr['course_version_id']);
if (!$version) jsonError(404, 'Версия курса не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);

$pdo->beginTransaction();
try {
    // Попытки тестов курса
    $att = $pdo->prepare(
        'SELECT test_attempt_id FROM public.course_test_attempt_links WHERE enrollment_id = :e'
    );
    $att->execute([':e' => $enrollmentId]);
    $attemptIds = array_map('intval', array_column($att->fetchAll(), 'test_attempt_id'));

    $pdo->prepare('DELETE FROM public.course_test_attempt_links WHERE enrollment_id = :e')
        ->execute([':e' => $enrollmentId]);

    if ($attemptIds) {
        $in = implode(',', array_fill(0, count($attemptIds), '?'));
        // test_answers каскадом удалятся с attempt
        $pdo->prepare("DELETE FROM public.test_attempts WHERE id IN ($in)")
            ->execute($attemptIds);
    }

    $pdo->prepare('DELETE FROM public.course_learning_sessions WHERE enrollment_id = :e')
        ->execute([':e' => $enrollmentId]);
    $pdo->prepare('DELETE FROM public.course_material_progress WHERE enrollment_id = :e')
        ->execute([':e' => $enrollmentId]);
    $pdo->prepare('DELETE FROM public.course_topic_progress WHERE enrollment_id = :e')
        ->execute([':e' => $enrollmentId]);
    $pdo->prepare('DELETE FROM public.course_completions WHERE enrollment_id = :e')
        ->execute([':e' => $enrollmentId]);

    $pdo->prepare(
        "UPDATE public.course_enrollments
         SET status = 'not_started',
             started_at = NULL,
             completed_at = NULL,
             last_activity_at = NULL,
             final_score = NULL,
             updated_at = now()
         WHERE id = :id"
    )->execute([':id' => $enrollmentId]);

    cs_ensure_topic_progress_rows($pdo, $enrollmentId, (int)$enr['course_version_id']);
    cs_recalculate_locks($pdo, $enrollmentId);

    cs_audit($pdo, (int)$user['id'], 'course.enrollment.reset', 'course_enrollment', $enrollmentId, [
        'courseId' => (int)$version['courseId'],
        'userId' => (int)$enr['user_id'],
        'deletedAttempts' => count($attemptIds),
    ]);

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Не удалось обнулить результат');
}

$enr = cs_enrollment_row($pdo, $enrollmentId);
jsonOk([
    'enrollment' => cs_map_enrollment($enr, cs_enrollment_progress($pdo, $enrollmentId)),
]);
