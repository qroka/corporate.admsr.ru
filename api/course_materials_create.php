<?php
/** POST /api/course_materials_create.php */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$topicId = (int)($body['topicId'] ?? 0);
if ($topicId <= 0) jsonError(400, 'Не передан topicId');

$topic = cs_topic_version_row($pdo, $topicId);
if (!$topic) jsonError(404, 'Тема не найдена');
cs_require_course_admin($pdo, (int)$topic['course_id']);
if (!in_array((string)$topic['version_status'], ['draft', 'published'], true)) {
    jsonError(409, 'Версия недоступна для редактирования (только черновик/опубликовано)');
}

$allowed = ['rich_text', 'file', 'link'];
$type = (string)($body['type'] ?? 'rich_text');
if (in_array($type, ['pdf', 'image', 'video'], true)) $type = 'file';
if (!in_array($type, $allowed, true)) jsonError(400, 'Недопустимый тип материала');

$title = trim((string)($body['title'] ?? ''));
if ($title === '') jsonError(400, 'Не указано название');

$max = $pdo->prepare('SELECT COALESCE(MAX(sort_order), -1) + 1 FROM public.course_materials WHERE topic_id = :t AND deleted_at IS NULL');
$max->execute([':t' => $topicId]);
$ord = (int)$max->fetchColumn(0);

$ins = $pdo->prepare(
    'INSERT INTO public.course_materials (
        topic_id, type, title, description, content_html, file_url, external_url,
        mime_type, file_size, original_filename, sort_order, is_required, minimum_active_seconds
     ) VALUES (
        :tid, :type, :title, :desc, :html, :fu, :eu, :mime, :fs, :ofn, :ord, :req, :min
     ) RETURNING *'
);
$ins->execute([
    ':tid' => $topicId,
    ':type' => $type,
    ':title' => $title,
    ':desc' => (string)($body['description'] ?? ''),
    ':html' => cs_sanitize_html((string)($body['contentHtml'] ?? '')),
    ':fu' => $body['fileUrl'] ?? null,
    ':eu' => $body['externalUrl'] ?? null,
    ':mime' => $body['mimeType'] ?? null,
    ':fs' => isset($body['fileSize']) ? (int)$body['fileSize'] : null,
    ':ofn' => $body['originalFilename'] ?? null,
    ':ord' => $ord,
    ':req' => cs_bool($body['isRequired'] ?? true) ? 't' : 'f',
    ':min' => max(0, (int)($body['minimumActiveSeconds'] ?? 0)),
]);
$m = $ins->fetch();

cs_audit($pdo, (int)$user['id'], 'course.material.create', 'course_material', (int)$m['id'], ['topicId' => $topicId]);

jsonOk([
    'material' => [
        'id' => (int)$m['id'],
        'topicId' => $topicId,
        'type' => (string)$m['type'],
        'title' => (string)$m['title'],
        'description' => (string)$m['description'],
        'contentHtml' => (string)($m['content_html'] ?? ''),
        'fileUrl' => $m['file_url'],
        'externalUrl' => $m['external_url'],
        'mimeType' => $m['mime_type'],
        'fileSize' => $m['file_size'] !== null ? (int)$m['file_size'] : null,
        'originalFilename' => $m['original_filename'],
        'sortOrder' => (int)$m['sort_order'],
        'isRequired' => cs_bool($m['is_required']),
        'minimumActiveSeconds' => (int)$m['minimum_active_seconds'],
    ],
]);
