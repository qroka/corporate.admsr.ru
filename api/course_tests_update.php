<?php
/**
 * POST /api/course_tests_update.php
 * Body: {courseTestLinkId|testFormId, form: TestForm, isRequired?}
 * Только draft-версия.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$linkId = (int)($body['courseTestLinkId'] ?? 0);
$formId = (int)($body['testFormId'] ?? $body['formId'] ?? 0);

if ($linkId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE id = :id');
    $st->execute([':id' => $linkId]);
    $link = $st->fetch();
} elseif ($formId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE test_form_id = :f LIMIT 1');
    $st->execute([':f' => $formId]);
    $link = $st->fetch();
} else {
    jsonError(400, 'Укажите courseTestLinkId или testFormId');
}
if (!$link) jsonError(404, 'Связь теста не найдена');

$version = cs_get_version($pdo, (int)$link['course_version_id']);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);
if (!cs_version_editable($version)) {
    jsonError(409, 'Архивированная версия: правки вопросов недоступны.');
}

$formPayload = $body['form'] ?? null;
if (!is_array($formPayload)) jsonError(400, 'Не передана форма');
$formPayload['id'] = (int)$link['test_form_id'];
$formPayload['visibility'] = 'private';
$formPayload['kind'] = $formPayload['kind'] ?? 'test';

try {
    tf_persistForm($pdo, $formPayload, (int)$user['id']);
} catch (RuntimeException $e) {
    jsonError(403, $e->getMessage());
} catch (Throwable $e) {
    jsonError(500, 'Ошибка сохранения теста');
}

if (array_key_exists('isRequired', $body)) {
    $pdo->prepare(
        'UPDATE public.course_test_links SET is_required = :r, updated_at = now() WHERE id = :id'
    )->execute([
        ':r' => cs_bool($body['isRequired']) ? 't' : 'f',
        ':id' => (int)$link['id'],
    ]);
}

$st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE id = :id');
$st->execute([':id' => (int)$link['id']]);
$link = $st->fetch();

cs_audit($pdo, (int)$user['id'], 'course.test.update', 'course_test_link', (int)$link['id'], []);

jsonOk([
    'link' => cs_map_test_link($link, cs_test_form_summary($pdo, (int)$link['test_form_id'])),
    'form' => tf_loadForm($pdo, (int)$link['test_form_id'], (int)$user['id']),
]);
