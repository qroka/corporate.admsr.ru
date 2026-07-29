<?php
/** POST /api/course_next_action.php — Body: {enrollmentId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
if ($enrollmentId <= 0) jsonError(400, 'Не передан enrollmentId');

cs_require_enrollment_access($pdo, $enrollmentId, $user);
jsonOk([
    'nextAction' => cs_next_action($pdo, $enrollmentId),
    'progress' => cs_enrollment_progress($pdo, $enrollmentId),
]);
