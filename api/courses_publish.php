<?php
/** POST /api/courses_publish.php — Body: {courseId} | {versionId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$versionId = cs_resolve_version_id($pdo, $body);
$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);

try {
    $assembled = cs_publish_version($pdo, $versionId, $user);
} catch (Throwable $e) {
    // jsonError уже мог вызвать exit; иначе 500
    if (!headers_sent()) jsonError(500, 'Ошибка публикации');
    throw $e;
}

jsonOk(['version' => $assembled]);
