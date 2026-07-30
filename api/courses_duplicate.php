<?php
/** POST /api/courses_duplicate.php — Body: {courseId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$courseId = (int)($body['courseId'] ?? 0);
if ($courseId <= 0) jsonError(400, 'Не передан courseId');

cs_require_course_admin($pdo, $courseId);

try {
    $result = cs_duplicate_course($pdo, $courseId, $user);
} catch (Throwable $e) {
    jsonError(500, 'Ошибка дублирования');
}

jsonOk($result);
