<?php
/** POST /api/course_materials_update.php — Body: {materialId, ...} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$materialId = (int)($body['materialId'] ?? 0);
if ($materialId <= 0) jsonError(400, 'Не передан materialId');

$m = cs_material_version_row($pdo, $materialId);
if (!$m) jsonError(404, 'Материал не найден');
cs_require_course_admin($pdo, (int)$m['course_id']);
if (!in_array((string)$m['version_status'], ['draft', 'published'], true)) {
    jsonError(409, 'Версия недоступна для редактирования (только черновик/опубликовано)');
}

$map = [
    'title' => 'title',
    'description' => 'description',
    'fileUrl' => 'file_url',
    'externalUrl' => 'external_url',
    'mimeType' => 'mime_type',
    'originalFilename' => 'original_filename',
];
$sets = [];
$params = [':id' => $materialId];

if (isset($body['title'])) {
    $t = trim((string)$body['title']);
    if ($t === '') jsonError(400, 'Пустое название');
    $sets[] = 'title = :title';
    $params[':title'] = $t;
}
if (array_key_exists('description', $body)) {
    $sets[] = 'description = :description';
    $params[':description'] = (string)$body['description'];
}
if (array_key_exists('contentHtml', $body)) {
    $sets[] = 'content_html = :html';
    $params[':html'] = cs_sanitize_html((string)$body['contentHtml']);
}
if (array_key_exists('fileUrl', $body)) {
    $sets[] = 'file_url = :fu';
    $params[':fu'] = $body['fileUrl'];
}
if (array_key_exists('externalUrl', $body)) {
    $sets[] = 'external_url = :eu';
    $params[':eu'] = $body['externalUrl'];
}
if (array_key_exists('mimeType', $body)) {
    $sets[] = 'mime_type = :mime';
    $params[':mime'] = $body['mimeType'];
}
if (array_key_exists('fileSize', $body)) {
    $sets[] = 'file_size = :fs';
    $params[':fs'] = $body['fileSize'] !== null ? (int)$body['fileSize'] : null;
}
if (array_key_exists('originalFilename', $body)) {
    $sets[] = 'original_filename = :ofn';
    $params[':ofn'] = $body['originalFilename'];
}
if (array_key_exists('isRequired', $body)) {
    $sets[] = 'is_required = :req';
    $params[':req'] = cs_bool($body['isRequired']) ? 't' : 'f';
}
if (array_key_exists('minimumActiveSeconds', $body)) {
    $sets[] = 'minimum_active_seconds = :min';
    $params[':min'] = max(0, (int)$body['minimumActiveSeconds']);
}
if (isset($body['type'])) {
    $allowed = ['rich_text', 'file', 'link'];
    $type = (string)$body['type'];
    if (in_array($type, ['pdf', 'image', 'video'], true)) $type = 'file';
    if (!in_array($type, $allowed, true)) jsonError(400, 'Недопустимый тип');
    $sets[] = 'type = :type';
    $params[':type'] = $type;
}
if (!$sets) jsonError(400, 'Нет полей для обновления');

$pdo->prepare(
    'UPDATE public.course_materials SET ' . implode(', ', $sets) . ', updated_at = now() WHERE id = :id'
)->execute($params);

cs_audit($pdo, (int)$user['id'], 'course.material.update', 'course_material', $materialId, []);
$m = cs_material_version_row($pdo, $materialId);
jsonOk([
    'material' => [
        'id' => (int)$m['id'],
        'topicId' => (int)$m['topic_id'],
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
