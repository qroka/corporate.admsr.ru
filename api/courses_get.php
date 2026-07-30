<?php
/** POST /api/courses_get.php — полный курс + версия. Body: {courseId, versionId?} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_section(\, 'courses');
$body = cs_body();
$courseId = (int)($body['courseId'] ?? 0);
if ($courseId <= 0) jsonError(400, 'Не передан courseId');

$course = cs_get_course($pdo, $courseId);
if (!$course) jsonError(404, 'Курс не найден');

$versionId = (int)($body['versionId'] ?? 0);
if ($versionId <= 0) {
    $versionId = (int)($course['currentVersionId'] ?? 0);
}
if ($versionId <= 0) jsonError(404, 'Версия не найдена');

$version = cs_assemble_version($pdo, $versionId, true);
if (!$version || (int)$version['courseId'] !== $courseId) {
    jsonError(404, 'Версия не найдена');
}

jsonOk(['course' => $course, 'version' => $version]);
