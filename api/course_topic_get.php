<?php
/** POST /api/course_topic_get.php — Body: {enrollmentId, topicId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
$topicId = (int)($body['topicId'] ?? 0);
if ($enrollmentId <= 0 || $topicId <= 0) jsonError(400, 'Нужны enrollmentId и topicId');

$enr = cs_require_enrollment_access($pdo, $enrollmentId, $user);
cs_ensure_topic_progress_rows($pdo, $enrollmentId, (int)$enr['course_version_id']);
cs_recalculate_locks($pdo, $enrollmentId);

$topic = cs_topic_version_row($pdo, $topicId);
if (!$topic || (int)$topic['course_version_id'] !== (int)$enr['course_version_id']) {
    jsonError(404, 'Тема не найдена');
}

$isReview = in_array((string)$enr['status'], ['completed', 'failed'], true);

$ps = $pdo->prepare('SELECT * FROM public.course_topic_progress WHERE enrollment_id = :e AND topic_id = :t');
$ps->execute([':e' => $enrollmentId, ':t' => $topicId]);
$prog = $ps->fetch();
if (!$isReview && (!$prog || $prog['status'] === 'locked')) {
    jsonError(403, 'Тема ещё недоступна');
}

// Открыть тему (не трогаем прогресс в режиме просмотра завершённого курса)
if (!$isReview && $prog && $prog['status'] === 'available') {
    $pdo->prepare(
        "UPDATE public.course_topic_progress
         SET status = 'in_progress', opened_at = COALESCE(opened_at, now()), updated_at = now()
         WHERE id = :id"
    )->execute([':id' => (int)$prog['id']]);
}

$assembled = cs_assemble_version($pdo, (int)$enr['course_version_id'], true);
$topicData = null;
foreach ($assembled['topics'] ?? [] as $t) {
    if ((int)$t['id'] === $topicId) {
        $topicData = $t;
        break;
    }
}
if (!$topicData) jsonError(404, 'Тема не найдена');

$mp = $pdo->prepare('SELECT material_id, status, active_seconds FROM public.course_material_progress WHERE enrollment_id = :e');
$mp->execute([':e' => $enrollmentId]);
$matStatus = [];
foreach ($mp->fetchAll() as $p) {
    $matStatus[(int)$p['material_id']] = [
        'status' => (string)$p['status'],
        'activeSeconds' => (int)$p['active_seconds'],
    ];
}
foreach ($topicData['materials'] as &$m) {
    $m['progress'] = $matStatus[$m['id']] ?? ['status' => 'not_started', 'activeSeconds' => 0];
}
unset($m);

$ps->execute([':e' => $enrollmentId, ':t' => $topicId]);
$prog = $ps->fetch();
$topicData['progress'] = [
    'status' => (string)($prog['status'] ?? 'locked'),
    'activeSeconds' => (int)($prog['active_seconds'] ?? 0),
    'openedAt' => $prog['opened_at'] ?? null,
    'completedAt' => $prog['completed_at'] ?? null,
];

$nextTopic = null;
$nextSt = $pdo->prepare(
    'SELECT t.id, t.title, t.sort_order, COALESCE(p.status, \'locked\') AS progress_status
     FROM public.course_topics t
     LEFT JOIN public.course_topic_progress p
       ON p.topic_id = t.id AND p.enrollment_id = :e
     WHERE t.course_version_id = :v
       AND t.deleted_at IS NULL
       AND (
         t.sort_order > :so
         OR (t.sort_order = :so AND t.id > :id)
       )
     ORDER BY t.sort_order, t.id
     LIMIT 1'
);
$nextSt->execute([
    ':e' => $enrollmentId,
    ':v' => (int)$enr['course_version_id'],
    ':so' => (int)$topic['sort_order'],
    ':id' => $topicId,
]);
$nextRow = $nextSt->fetch();
if ($nextRow) {
    $nextStatus = (string)$nextRow['progress_status'];
    if ($isReview || $nextStatus !== 'locked') {
        $nextTopic = [
            'id' => (int)$nextRow['id'],
            'title' => (string)$nextRow['title'],
            'status' => $nextStatus,
        ];
    }
}

jsonOk([
    'topic' => $topicData,
    'nextAction' => cs_next_action($pdo, $enrollmentId),
    'nextTopic' => $nextTopic,
    'reviewMode' => $isReview,
    'enrollmentStatus' => (string)$enr['status'],
]);
