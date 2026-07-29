<?php
/** POST /api/course_result.php — Body: {enrollmentId} | {completionId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$completionId = (int)($body['completionId'] ?? 0);
$enrollmentId = (int)($body['enrollmentId'] ?? 0);

if ($completionId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_completions WHERE id = :id');
    $st->execute([':id' => $completionId]);
    $comp = $st->fetch();
} elseif ($enrollmentId > 0) {
    cs_require_enrollment_access($pdo, $enrollmentId, $user);
    $st = $pdo->prepare('SELECT * FROM public.course_completions WHERE enrollment_id = :e ORDER BY id DESC LIMIT 1');
    $st->execute([':e' => $enrollmentId]);
    $comp = $st->fetch();
} else {
    jsonError(400, 'Укажите enrollmentId или completionId');
}

if (!$comp) jsonError(404, 'Результат не найден');

// доступ: свой или админ
if ((int)$comp['user_id'] !== (int)$user['id'] && !auth_is_admin($user)) {
    jsonError(403, 'Нет доступа');
}

$snap = $comp['result_snapshot'];
if (is_string($snap)) {
    $snap = json_decode($snap, true) ?: [];
}

jsonOk([
    'completion' => [
        'id' => (int)$comp['id'],
        'enrollmentId' => (int)$comp['enrollment_id'],
        'courseId' => (int)$comp['course_id'],
        'courseVersionId' => (int)$comp['course_version_id'],
        'completionNumber' => (int)$comp['completion_number'],
        'assignedAt' => $comp['assigned_at'],
        'startedAt' => $comp['started_at'],
        'completedAt' => $comp['completed_at'],
        'totalActiveSeconds' => (int)$comp['total_active_seconds'],
        'finalScore' => $comp['final_score'] !== null ? (float)$comp['final_score'] : null,
        'passed' => cs_bool($comp['passed']),
        'resultSnapshot' => $snap,
    ],
]);
