<?php
/** POST /api/course_topics_create.php */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$versionId = cs_resolve_version_id($pdo, $body);
$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);
cs_assert_version_editable($version);

$title = trim((string)($body['title'] ?? ''));
if ($title === '') jsonError(400, 'Не указано название темы');

$max = $pdo->prepare('SELECT COALESCE(MAX(sort_order), -1) + 1 FROM public.course_topics WHERE course_version_id = :v AND deleted_at IS NULL');
$max->execute([':v' => $versionId]);
$ord = (int)$max->fetchColumn(0);

$ins = $pdo->prepare(
    'INSERT INTO public.course_topics (
        course_version_id, title, description, sort_order, is_required, minimum_active_seconds
     ) VALUES (:v, :t, :d, :ord, :req, :min) RETURNING *'
);
$ins->execute([
    ':v' => $versionId,
    ':t' => $title,
    ':d' => (string)($body['description'] ?? ''),
    ':ord' => $ord,
    ':req' => cs_bool($body['isRequired'] ?? true) ? 't' : 'f',
    ':min' => max(0, (int)($body['minimumActiveSeconds'] ?? 0)),
]);
$row = $ins->fetch();

cs_audit($pdo, (int)$user['id'], 'course.topic.create', 'course_topic', (int)$row['id'], ['versionId' => $versionId]);

jsonOk([
    'topic' => [
        'id' => (int)$row['id'],
        'courseVersionId' => $versionId,
        'title' => (string)$row['title'],
        'description' => (string)$row['description'],
        'sortOrder' => (int)$row['sort_order'],
        'isRequired' => cs_bool($row['is_required']),
        'minimumActiveSeconds' => (int)$row['minimum_active_seconds'],
        'completionRule' => (string)$row['completion_rule'],
        'materials' => [],
        'topicTest' => null,
    ],
]);
