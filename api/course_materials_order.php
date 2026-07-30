<?php
/** POST /api/course_materials_order.php — Body: {topicId, materialIds: number[]} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$topicId = (int)($body['topicId'] ?? 0);
$ids = $body['materialIds'] ?? [];
if ($topicId <= 0 || !is_array($ids)) jsonError(400, 'Нужны topicId и materialIds');

$topic = cs_topic_version_row($pdo, $topicId);
if (!$topic) jsonError(404, 'Тема не найдена');
cs_require_course_admin($pdo, (int)$topic['course_id']);
if (!in_array((string)$topic['version_status'], ['draft', 'published'], true)) {
    jsonError(409, 'Версия недоступна для редактирования (только черновик/опубликовано)');
}

$pdo->beginTransaction();
try {
    $upd = $pdo->prepare(
        'UPDATE public.course_materials SET sort_order = :ord, updated_at = now()
         WHERE id = :id AND topic_id = :t AND deleted_at IS NULL'
    );
    foreach (array_values($ids) as $i => $mid) {
        $upd->execute([':ord' => $i, ':id' => (int)$mid, ':t' => $topicId]);
    }
    cs_audit($pdo, (int)$user['id'], 'course.material.reorder', 'course_topic', $topicId, ['materialIds' => $ids]);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка сортировки');
}

jsonOk(['topicId' => $topicId, 'materialIds' => array_map('intval', $ids)]);
