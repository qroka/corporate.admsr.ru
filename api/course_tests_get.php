<?php
/**
 * POST /api/course_tests_get.php
 * Body: {courseTestLinkId} | {testFormId} | {topicId} | {versionId, type:'final'}
 * Admin или enrolled с доступом.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$linkId = (int)($body['courseTestLinkId'] ?? 0);
$formId = (int)($body['testFormId'] ?? $body['formId'] ?? 0);

$link = null;
if ($linkId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE id = :id');
    $st->execute([':id' => $linkId]);
    $link = $st->fetch();
} elseif ($formId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE test_form_id = :f LIMIT 1');
    $st->execute([':f' => $formId]);
    $link = $st->fetch();
} elseif (!empty($body['topicId'])) {
    $st = $pdo->prepare("SELECT * FROM public.course_test_links WHERE topic_id = :t AND type = 'topic' LIMIT 1");
    $st->execute([':t' => (int)$body['topicId']]);
    $link = $st->fetch();
} else {
    $versionId = cs_resolve_version_id($pdo, $body);
    $st = $pdo->prepare("SELECT * FROM public.course_test_links WHERE course_version_id = :v AND type = 'final' LIMIT 1");
    $st->execute([':v' => $versionId]);
    $link = $st->fetch();
}

if (!$link) jsonError(404, 'Тест курса не найден');
$formId = (int)$link['test_form_id'];
$versionId = (int)$link['course_version_id'];

$allowed = auth_is_admin($user);
if (!$allowed) {
    $chk = $pdo->prepare(
        "SELECT 1 FROM public.course_enrollments
         WHERE user_id = :u AND course_version_id = :v AND status NOT IN ('cancelled') LIMIT 1"
    );
    $chk->execute([':u' => (int)$user['id'], ':v' => $versionId]);
    $allowed = (bool)$chk->fetchColumn(0);
}
if (!$allowed) jsonError(403, 'Нет доступа');

$form = tf_loadForm($pdo, $formId, (int)$user['id']);
if (!$form) jsonError(404, 'Форма не найдена');
if (!auth_is_admin($user)) {
    $form = tf_strip_correct($form);
}

jsonOk([
    'link' => cs_map_test_link($link, cs_test_form_summary($pdo, $formId)),
    'form' => $form,
]);
