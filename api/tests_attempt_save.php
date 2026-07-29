<?php
/**
 * POST /api/tests_attempt_save.php
 * Body: {attemptId, answers: {[questionId]: value}}
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$attemptId = (int)($body['attemptId'] ?? 0);
$answers = $body['answers'] ?? [];
if ($attemptId <= 0) jsonError(400, 'Не передан attemptId');
if (!is_array($answers)) $answers = [];

$st = $pdo->prepare('SELECT * FROM public.test_attempts WHERE id = :id');
$st->execute([':id' => $attemptId]);
$att = $st->fetch();
if (!$att) jsonError(404, 'Попытка не найдена');
if ((int)$att['user_id'] !== (int)$user['id'] && !auth_is_admin($user)) {
    jsonError(403, 'Нет доступа');
}
if ($att['status'] !== 'in_progress') {
    jsonError(409, 'Попытка уже завершена');
}

$qStmt = $pdo->prepare('SELECT id, type FROM public.test_questions WHERE form_id = :f');
$qStmt->execute([':f' => (int)$att['form_id']]);
$qTypes = [];
foreach ($qStmt->fetchAll() as $q) {
    $qTypes[(int)$q['id']] = (string)$q['type'];
}

$pdo->beginTransaction();
try {
    $delOpts = $pdo->prepare(
        'DELETE FROM public.test_answer_options WHERE answer_id IN
         (SELECT id FROM public.test_answers WHERE attempt_id = :a AND question_id = :q)'
    );
    $delAns = $pdo->prepare('DELETE FROM public.test_answers WHERE attempt_id = :a AND question_id = :q');
    $ansIns = $pdo->prepare(
        'INSERT INTO public.test_answers (attempt_id, question_id, text_value, number_value, is_correct, answered)
         VALUES (:a, :q, :tv, :nv, NULL, :answered) RETURNING id'
    );
    $aoIns = $pdo->prepare('INSERT INTO public.test_answer_options (answer_id, option_id) VALUES (:ans, :opt) ON CONFLICT DO NOTHING');

    foreach ($answers as $qidRaw => $val) {
        $qid = (int)$qidRaw;
        if (!isset($qTypes[$qid])) continue;
        $type = $qTypes[$qid];

        $textVal = null;
        $numVal = null;
        $selected = [];
        $answered = false;

        if (in_array($type, ['single', 'dropdown'], true)) {
            if ($val !== null && $val !== '') {
                $selected = [(int)$val];
                $answered = true;
            }
        } elseif ($type === 'multiple') {
            if (is_array($val) && count($val)) {
                $selected = array_map('intval', $val);
                $answered = true;
            }
        } elseif ($type === 'scale' || $type === 'number') {
            if ($val !== null && $val !== '') {
                $numVal = 0 + $val;
                $textVal = (string)$val;
                $answered = true;
            }
        } else {
            if ($val !== null && $val !== '') {
                $textVal = (string)$val;
                $answered = true;
            }
        }

        $delOpts->execute([':a' => $attemptId, ':q' => $qid]);
        $delAns->execute([':a' => $attemptId, ':q' => $qid]);
        $ansIns->execute([
            ':a' => $attemptId,
            ':q' => $qid,
            ':tv' => $textVal,
            ':nv' => $numVal,
            ':answered' => $answered ? 't' : 'f',
        ]);
        $ansId = (int)$ansIns->fetchColumn(0);
        foreach ($selected as $optId) {
            $aoIns->execute([':ans' => $ansId, ':opt' => $optId]);
        }
    }

    $pdo->prepare('UPDATE public.test_attempts SET updated_at = now() WHERE id = :id')
        ->execute([':id' => $attemptId]);

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка сохранения ответов');
}

jsonOk(['attemptId' => $attemptId, 'saved' => count($answers)]);
