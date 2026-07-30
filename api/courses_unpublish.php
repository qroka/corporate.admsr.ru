<?php
/**
 * POST /api/courses_unpublish.php — снять публикацию курса.
 * Body: {courseId} | {versionId}
 *
 * Логика: курс возвращается в статус "draft".
 * Примечание: тестовые формы не трогаем, чтобы участники могли
 * продолжить текущие попытки/прохождение.
 */
require_once __DIR__ . '/courses_common.php';

if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$versionId = cs_resolve_version_id($pdo, $body);
$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);

if ($version['status'] !== 'published') {
    jsonError(409, 'Снимать публикацию можно только опубликованной версии');
}

$uid = (int)$user['id'];

$pdo->beginTransaction();
try {
    // Снимаем публикацию версии курса
    $pdo->prepare(
        "UPDATE public.course_versions
         SET status = 'draft',
             published_at = NULL,
             archived_at = NULL,
             updated_at = now()
         WHERE id = :id"
    )->execute([':id' => $versionId]);

    cs_audit($pdo, $uid, 'course.version.unpublish', 'course_version', $versionId, [
        'courseId' => $version['courseId'],
    ]);

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) {
        $pdo->rollBack();
    }
    throw $e;
}

jsonOk(['version' => cs_assemble_version($pdo, $versionId, true)]);

