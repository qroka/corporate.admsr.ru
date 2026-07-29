<?php
/**
 * POST /api/course_material_heartbeat.php
 * Body: {enrollmentId, materialId, sessionId?, clientGapSec?}
 * Сервер считает delta от last_heartbeat, max 60s; игнор если gap > 90s.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
$materialId = (int)($body['materialId'] ?? 0);
$sessionId = (int)($body['sessionId'] ?? 0);
$clientGap = isset($body['clientGapSec']) ? (int)$body['clientGapSec'] : null;

if ($enrollmentId <= 0 || $materialId <= 0) jsonError(400, 'Нужны enrollmentId и materialId');

$enr = cs_require_enrollment_access($pdo, $enrollmentId, $user, false);
$m = cs_material_version_row($pdo, $materialId);
if (!$m || (int)$m['course_version_id'] !== (int)$enr['course_version_id']) {
    jsonError(404, 'Материал не найден');
}
$topicId = (int)$m['topic_id'];

if ($clientGap !== null && $clientGap > 90) {
    jsonOk(['ignored' => true, 'reason' => 'client_gap', 'addedSeconds' => 0]);
}

if ($sessionId > 0) {
    $st = $pdo->prepare(
        'SELECT * FROM public.course_learning_sessions
         WHERE id = :id AND enrollment_id = :e AND finished_at IS NULL'
    );
    $st->execute([':id' => $sessionId, ':e' => $enrollmentId]);
} else {
    $st = $pdo->prepare(
        'SELECT * FROM public.course_learning_sessions
         WHERE enrollment_id = :e AND material_id = :m AND finished_at IS NULL
         ORDER BY id DESC LIMIT 1'
    );
    $st->execute([':e' => $enrollmentId, ':m' => $materialId]);
}
$session = $st->fetch();
if (!$session) jsonError(404, 'Сессия обучения не найдена');

$lastHb = strtotime($session['last_heartbeat_at']);
$now = time();
$delta = max(0, $now - $lastHb);
if ($delta > 90) {
    // слишком большой разрыв — обновляем heartbeat без начисления
    $pdo->prepare(
        'UPDATE public.course_learning_sessions SET last_heartbeat_at = now() WHERE id = :id'
    )->execute([':id' => (int)$session['id']]);
    jsonOk(['ignored' => true, 'reason' => 'server_gap', 'addedSeconds' => 0, 'sessionId' => (int)$session['id']]);
}

$add = min(60, $delta);
if ($add < 1) {
    jsonOk(['ignored' => false, 'addedSeconds' => 0, 'sessionId' => (int)$session['id'], 'activeSeconds' => (int)$session['active_seconds']]);
}

$pdo->beginTransaction();
try {
    $pdo->prepare(
        'UPDATE public.course_learning_sessions
         SET active_seconds = active_seconds + :a, last_heartbeat_at = now()
         WHERE id = :id'
    )->execute([':a' => $add, ':id' => (int)$session['id']]);

    $pdo->prepare(
        "INSERT INTO public.course_material_progress (enrollment_id, material_id, status, opened_at, active_seconds)
         VALUES (:e, :m, 'in_progress', now(), :a)
         ON CONFLICT (enrollment_id, material_id) DO UPDATE SET
           active_seconds = course_material_progress.active_seconds + :a2,
           updated_at = now()"
    )->execute([':e' => $enrollmentId, ':m' => $materialId, ':a' => $add, ':a2' => $add]);

    $pdo->prepare(
        'UPDATE public.course_topic_progress
         SET active_seconds = active_seconds + :a, updated_at = now()
         WHERE enrollment_id = :e AND topic_id = :t'
    )->execute([':a' => $add, ':e' => $enrollmentId, ':t' => $topicId]);

    $pdo->prepare(
        'UPDATE public.course_enrollments SET last_activity_at = now(), updated_at = now() WHERE id = :id'
    )->execute([':id' => $enrollmentId]);

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка heartbeat');
}

$mp = $pdo->prepare('SELECT active_seconds, status FROM public.course_material_progress WHERE enrollment_id = :e AND material_id = :m');
$mp->execute([':e' => $enrollmentId, ':m' => $materialId]);
$row = $mp->fetch() ?: ['active_seconds' => 0, 'status' => 'in_progress'];

jsonOk([
    'ignored' => false,
    'addedSeconds' => $add,
    'sessionId' => (int)$session['id'],
    'activeSeconds' => (int)$row['active_seconds'],
    'status' => (string)$row['status'],
    'minimumActiveSeconds' => (int)$m['minimum_active_seconds'],
]);
