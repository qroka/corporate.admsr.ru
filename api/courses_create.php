<?php
/** POST /api/courses_create.php — создать курс. */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();

try {
    $result = cs_create_course($pdo, $user, $body);
} catch (Throwable $e) {
    jsonError(500, 'Ошибка создания курса');
}

jsonOk($result);
