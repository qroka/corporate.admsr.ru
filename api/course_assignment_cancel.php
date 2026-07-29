<?php
/**
 * POST /api/course_assignment_cancel.php
 * Body: {assignmentId} — отмена assignment + not_started enrollments.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_admin($pdo);
$body = cs_body();
$assignmentId = (int)($body['assignmentId'] ?? 0);
if ($assignmentId <= 0) jsonError(400, 'Не передан assignmentId');

$st = $pdo->prepare('SELECT * FROM public.course_assignments WHERE id = :id');
$st->execute([':id' => $assignmentId]);
$a = $st->fetch();
if (!$a) jsonError(404, 'Назначение не найдено');
if ($a['cancelled_at']) jsonError(409, 'Уже отменено');

$pdo->beginTransaction();
try {
    $pdo->prepare(
        'UPDATE public.course_assignments SET cancelled_at = now() WHERE id = :id'
    )->execute([':id' => $assignmentId]);

    $upd = $pdo->prepare(
        "UPDATE public.course_enrollments
         SET status = 'cancelled', updated_at = now()
         WHERE assignment_id = :a AND status = 'not_started'"
    );
    $upd->execute([':a' => $assignmentId]);
    $cancelledEnrollments = $upd->rowCount();

    cs_audit($pdo, (int)$user['id'], 'course.assignment.cancel', 'course_assignment', $assignmentId, [
        'cancelledEnrollments' => $cancelledEnrollments,
    ]);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка отмены');
}

jsonOk(['assignmentId' => $assignmentId, 'cancelledEnrollments' => $cancelledEnrollments]);
