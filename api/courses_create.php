<?php
/** POST /api/courses_create.php — создать курс. */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$category = isset($body['category']) ? trim((string)$body['category']) : '';
if ($category === '') {
    jsonError(400, 'Укажите категорию курса');
}
auth_require_course_category($pdo, $category);

try {
    $result = cs_create_course($pdo, $user, $body);
} catch (Throwable $e) {
    jsonError(500, 'Ошибка создания курса');
}

jsonOk($result);
