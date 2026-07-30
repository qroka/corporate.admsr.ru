<?php
/** POST /api/course_material_open.php — Body: {enrollmentId, materialId} */
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
cs_ensure_topic_progress_rows($pdo, $enrollmentId, (int)$enr['course_version_id']);
cs_recalculate_locks($pdo, $enrollmentId);

$isReview = in_array((string)$enr['status'], ['completed', 'failed'], true);

$tp = $pdo->prepare('SELECT status FROM public.course_topic_progress WHERE enrollment_id = :e AND topic_id = :t');
$tp->execute([':e' => $enrollmentId, ':t' => $topicId]);
$ts = $tp->fetchColumn(0);
if (!$isReview && ($ts === false || $ts === 'locked')) {
    jsonError(403, 'Тема ещё недоступна');
}

$pdo->beginTransaction();
try {
    $pdo->prepare(
        "INSERT INTO public.course_material_progress (enrollment_id, material_id, status, opened_at)
         VALUES (:e, :m, 'in_progress', now())
         ON CONFLICT (enrollment_id, material_id) DO UPDATE SET
           status = CASE WHEN course_material_progress.status = 'completed' THEN 'completed' ELSE 'in_progress' END,
           opened_at = COALESCE(course_material_progress.opened_at, now()),
           updated_at = now()"
    )->execute([':e' => $enrollmentId, ':m' => $materialId]);

    $pdo->prepare(
        "UPDATE public.course_topic_progress
         SET status = CASE WHEN status = 'completed' THEN 'completed'
                           WHEN status = 'locked' THEN 'locked'
                           ELSE 'in_progress' END,
             opened_at = COALESCE(opened_at, now()),
             last_material_id = :m,
             updated_at = now()
         WHERE enrollment_id = :e AND topic_id = :t"
    )->execute([':m' => $materialId, ':e' => $enrollmentId, ':t' => $topicId]);

    $sessIns = $pdo->prepare(
        'INSERT INTO public.course_learning_sessions
            (enrollment_id, topic_id, material_id, user_id, ip_address, user_agent)
         VALUES (:e, :t, :m, :u, :ip, :ua) RETURNING id'
    );
    $sessIns->execute([
        ':e' => $enrollmentId,
        ':t' => $topicId,
        ':m' => $materialId,
        ':u' => (int)$user['id'],
        ':ip' => $_SERVER['REMOTE_ADDR'] ?? null,
        ':ua' => isset($_SERVER['HTTP_USER_AGENT']) ? mb_substr((string)$_SERVER['HTTP_USER_AGENT'], 0, 500) : null,
    ]);
    $sessionId = (int)$sessIns->fetchColumn(0);

    $pdo->prepare(
        "UPDATE public.course_enrollments
         SET status = CASE WHEN status = 'not_started' THEN 'in_progress' ELSE status END,
             started_at = COALESCE(started_at, now()),
             last_activity_at = now(), updated_at = now()
         WHERE id = :id"
    )->execute([':id' => $enrollmentId]);

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка открытия материала');
}

$fileServeUrl = null;
if (!empty($m['file_url'])) {
    $fileServeUrl = '/api/course_file.php?materialId=' . $materialId;
}

jsonOk([
    'material' => [
        'id' => (int)$m['id'],
        'topicId' => $topicId,
        'type' => (string)$m['type'],
        'title' => (string)$m['title'],
        'description' => (string)$m['description'],
        'contentHtml' => (string)($m['content_html'] ?? ''),
        'fileUrl' => $fileServeUrl,
        'storageKey' => $m['file_url'],
        'externalUrl' => $m['external_url'],
        'mimeType' => $m['mime_type'],
        'minimumActiveSeconds' => (int)$m['minimum_active_seconds'],
        'isRequired' => cs_bool($m['is_required']),
    ],
    'sessionId' => $sessionId,
]);
