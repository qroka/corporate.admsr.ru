<?php
/**
 * POST /api/tests_attempt_get.php
 * Body: {attemptId} | {formId, enrollmentId?}
 * → in_progress attempt + answers + form (без correct)
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$attemptId = (int)($body['attemptId'] ?? 0);
$formId = (int)($body['formId'] ?? 0);
$uid = (int)$user['id'];

if ($attemptId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.test_attempts WHERE id = :id');
    $st->execute([':id' => $attemptId]);
    $att = $st->fetch();
} elseif ($formId > 0) {
    $st = $pdo->prepare(
        "SELECT * FROM public.test_attempts
         WHERE form_id = :f AND user_id = :u AND status = 'in_progress'
         ORDER BY id DESC LIMIT 1"
    );
    $st->execute([':f' => $formId, ':u' => $uid]);
    $att = $st->fetch();
} else {
    jsonError(400, 'Укажите attemptId или formId');
}

if (!$att) jsonError(404, 'Попытка не найдена');
if ((int)$att['user_id'] !== $uid && !auth_is_admin($user)) {
    jsonError(403, 'Нет доступа');
}

$attemptId = (int)$att['id'];
$formId = (int)$att['form_id'];

$ansSt = $pdo->prepare('SELECT * FROM public.test_answers WHERE attempt_id = :a');
$ansSt->execute([':a' => $attemptId]);
$optSt = $pdo->prepare('SELECT option_id FROM public.test_answer_options WHERE answer_id = :id');

$answers = [];
foreach ($ansSt->fetchAll() as $a) {
    $qid = (int)$a['question_id'];
    $optSt->execute([':id' => (int)$a['id']]);
    $opts = array_map(static fn($r) => (int)$r['option_id'], $optSt->fetchAll());
    if ($opts) {
        $answers[(string)$qid] = count($opts) === 1 ? $opts[0] : $opts;
    } elseif ($a['number_value'] !== null && $a['number_value'] !== '') {
        $answers[(string)$qid] = 0 + $a['number_value'];
    } else {
        $answers[(string)$qid] = $a['text_value'];
    }
}

$form = tf_loadForm($pdo, $formId, $uid);
if ($form && !auth_is_admin($user)) {
    $form = tf_strip_correct($form);
}

$linkMeta = null;
$ls = $pdo->prepare(
    'SELECT tal.enrollment_id, tal.course_test_link_id, l.type
     FROM public.course_test_attempt_links tal
     JOIN public.course_test_links l ON l.id = tal.course_test_link_id
     WHERE tal.test_attempt_id = :a LIMIT 1'
);
$ls->execute([':a' => $attemptId]);
$lr = $ls->fetch();
if ($lr) {
    $linkMeta = [
        'enrollmentId' => (int)$lr['enrollment_id'],
        'courseTestLinkId' => (int)$lr['course_test_link_id'],
        'type' => (string)$lr['type'],
    ];
}

jsonOk([
    'attemptId' => $attemptId,
    'status' => (string)$att['status'],
    'formId' => $formId,
    'answers' => $answers,
    'form' => $form,
    'course' => $linkMeta,
]);
