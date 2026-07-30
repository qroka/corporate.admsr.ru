<?php
/** POST /api/courses_delete.php — soft-delete курса. Body: {courseId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$courseId = (int)($body['courseId'] ?? 0);
if ($courseId <= 0) jsonError(400, 'Не передан courseId');

cs_require_course_admin($pdo, $courseId);

$pdo->prepare(
    'UPDATE public.course_courses SET deleted_at = now(), updated_at = now() WHERE id = :id AND deleted_at IS NULL'
)->execute([':id' => $courseId]);

cs_audit($pdo, (int)$user['id'], 'course.delete', 'course', $courseId, []);
jsonOk(['courseId' => $courseId]);
