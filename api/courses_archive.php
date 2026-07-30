<?php
/** POST /api/courses_archive.php — архивировать опубликованную версию. Body: {courseId|versionId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$versionId = cs_resolve_version_id($pdo, $body);
$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);

if ($version['status'] !== 'published') {
    jsonError(409, 'Архивировать можно только опубликованную версию');
}

$pdo->prepare(
    "UPDATE public.course_versions
     SET status = 'archived', archived_at = now(), updated_at = now()
     WHERE id = :id"
)->execute([':id' => $versionId]);

cs_audit($pdo, (int)$user['id'], 'course.version.archive', 'course_version', $versionId, [
    'courseId' => $version['courseId'],
]);

jsonOk(['version' => cs_assemble_version($pdo, $versionId, true)]);
