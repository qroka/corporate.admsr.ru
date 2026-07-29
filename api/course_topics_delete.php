<?php
/** POST /api/course_topics_delete.php — soft-delete. Body: {topicId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_admin($pdo);
$body = cs_body();
$topicId = (int)($body['topicId'] ?? 0);
if ($topicId <= 0) jsonError(400, 'Не передан topicId');

$topic = cs_topic_version_row($pdo, $topicId);
if (!$topic) jsonError(404, 'Тема не найдена');
cs_require_course_admin($pdo, (int)$topic['course_id']);
if ($topic['version_status'] !== 'draft') {
    jsonError(409, 'Версия недоступна для редактирования (только черновик)');
}

$pdo->prepare(
    'UPDATE public.course_topics SET deleted_at = now(), updated_at = now() WHERE id = :id'
)->execute([':id' => $topicId]);

cs_audit($pdo, (int)$user['id'], 'course.topic.delete', 'course_topic', $topicId, []);
jsonOk(['topicId' => $topicId]);
