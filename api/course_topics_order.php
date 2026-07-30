<?php
/** POST /api/course_topics_order.php — Body: {versionId, topicIds: number[]} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$versionId = (int)($body['versionId'] ?? 0);
$topicIds = $body['topicIds'] ?? [];
if ($versionId <= 0 || !is_array($topicIds)) jsonError(400, 'Нужны versionId и topicIds');

$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);
cs_assert_version_editable($version);

$pdo->beginTransaction();
try {
    $upd = $pdo->prepare(
        'UPDATE public.course_topics SET sort_order = :ord, updated_at = now()
         WHERE id = :id AND course_version_id = :v AND deleted_at IS NULL'
    );
    foreach (array_values($topicIds) as $i => $tid) {
        $upd->execute([':ord' => $i, ':id' => (int)$tid, ':v' => $versionId]);
    }
    cs_audit($pdo, (int)$user['id'], 'course.topic.reorder', 'course_version', $versionId, ['topicIds' => $topicIds]);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка сортировки');
}

jsonOk(['versionId' => $versionId, 'topicIds' => array_map('intval', $topicIds)]);
