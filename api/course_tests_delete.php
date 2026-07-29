<?php
/**
 * POST /api/course_tests_delete.php — Body: {courseTestLinkId}
 * Удаляет link; форму удаляет только если draft и нет attempts.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_admin($pdo);
$body = cs_body();
$linkId = (int)($body['courseTestLinkId'] ?? 0);
if ($linkId <= 0) jsonError(400, 'Не передан courseTestLinkId');

$st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE id = :id');
$st->execute([':id' => $linkId]);
$link = $st->fetch();
if (!$link) jsonError(404, 'Связь не найдена');

$version = cs_get_version($pdo, (int)$link['course_version_id']);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);
cs_assert_version_editable($version);

$formId = (int)$link['test_form_id'];

$pdo->beginTransaction();
try {
    $pdo->prepare('DELETE FROM public.course_test_links WHERE id = :id')->execute([':id' => $linkId]);

    $att = $pdo->prepare('SELECT COUNT(*) FROM public.test_attempts WHERE form_id = :f');
    $att->execute([':f' => $formId]);
    $hasAttempts = (int)$att->fetchColumn(0) > 0;

    $form = $pdo->prepare('SELECT status FROM public.test_forms WHERE id = :id');
    $form->execute([':id' => $formId]);
    $status = $form->fetchColumn(0);

    $deletedForm = false;
    if (!$hasAttempts && $status === 'draft') {
        $pdo->prepare('DELETE FROM public.test_forms WHERE id = :id')->execute([':id' => $formId]);
        $deletedForm = true;
    }

    if ($link['type'] === 'final') {
        $pdo->prepare(
            'UPDATE public.course_versions SET require_final_test = false, updated_at = now() WHERE id = :id'
        )->execute([':id' => (int)$link['course_version_id']]);
    }

    cs_audit($pdo, (int)$user['id'], 'course.test.delete', 'course_test_link', $linkId, [
        'testFormId' => $formId,
        'deletedForm' => $deletedForm,
    ]);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка удаления');
}

jsonOk(['courseTestLinkId' => $linkId, 'testFormId' => $formId]);
