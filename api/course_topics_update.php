<?php
/** POST /api/course_topics_update.php — Body: {topicId, title?, description?, isRequired?, minimumActiveSeconds?} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$topicId = (int)($body['topicId'] ?? 0);
if ($topicId <= 0) jsonError(400, 'Не передан topicId');

$topic = cs_topic_version_row($pdo, $topicId);
if (!$topic) jsonError(404, 'Тема не найдена');
cs_require_course_admin($pdo, (int)$topic['course_id']);
if (!in_array((string)$topic['version_status'], ['draft', 'published'], true)) {
    jsonError(409, 'Версия недоступна для редактирования (только черновик/опубликовано)');
}

$sets = [];
$params = [':id' => $topicId];
if (isset($body['title'])) {
    $t = trim((string)$body['title']);
    if ($t === '') jsonError(400, 'Пустое название');
    $sets[] = 'title = :t';
    $params[':t'] = $t;
}
if (array_key_exists('description', $body)) {
    $sets[] = 'description = :d';
    $params[':d'] = (string)$body['description'];
}
if (array_key_exists('isRequired', $body)) {
    $sets[] = 'is_required = :req';
    $params[':req'] = cs_bool($body['isRequired']) ? 't' : 'f';
}
if (array_key_exists('minimumActiveSeconds', $body)) {
    $sets[] = 'minimum_active_seconds = :min';
    $params[':min'] = max(0, (int)$body['minimumActiveSeconds']);
}
if (array_key_exists('completionRule', $body)) {
    $sets[] = 'completion_rule = :cr';
    $params[':cr'] = (string)$body['completionRule'];
}
if (!$sets) jsonError(400, 'Нет полей для обновления');

$pdo->prepare(
    'UPDATE public.course_topics SET ' . implode(', ', $sets) . ', updated_at = now() WHERE id = :id'
)->execute($params);

cs_audit($pdo, (int)$user['id'], 'course.topic.update', 'course_topic', $topicId, []);
$topic = cs_topic_version_row($pdo, $topicId);
jsonOk([
    'topic' => [
        'id' => (int)$topic['id'],
        'courseVersionId' => (int)$topic['course_version_id'],
        'title' => (string)$topic['title'],
        'description' => (string)$topic['description'],
        'sortOrder' => (int)$topic['sort_order'],
        'isRequired' => cs_bool($topic['is_required']),
        'minimumActiveSeconds' => (int)$topic['minimum_active_seconds'],
        'completionRule' => (string)$topic['completion_rule'],
    ],
]);
