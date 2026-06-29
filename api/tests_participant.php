<?php
/**
 * POST /api/tests_participant.php — ответы конкретного участника (для создателя).
 * Body: { userId (создатель), formId, participantId }
 * Доступно только владельцу формы и только если форма НЕ анонимна.
 * → { attempt: {score, passed, finishedAt, durationSec}, answers: [{title, type, userAnswer, correctAnswer, isCorrect, answered}] }
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$requester = tf_viewer($body);
$formId = (int)($body['formId'] ?? 0);
$participantId = (int)($body['participantId'] ?? 0);
if ($formId <= 0 || $participantId <= 0) jsonError(400, 'Нужны formId и participantId');

$st = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
$st->execute([':id' => $formId]);
$form = $st->fetch();
if (!$form) jsonError(404, 'Форма не найдена');
if ($form['owner_id'] === null || (int)$form['owner_id'] !== $requester) jsonError(403, 'Доступно только создателю формы');
if (tf_bool($form['anonymous'])) jsonError(403, 'Форма анонимна — ответы участников скрыты');

// последнее завершённое прохождение участника
$as = $pdo->prepare(
    "SELECT * FROM public.test_attempts
     WHERE form_id = :f AND user_id = :p AND status = 'completed'
     ORDER BY finished_at DESC NULLS LAST, id DESC LIMIT 1"
);
$as->execute([':f' => $formId, ':p' => $participantId]);
$attempt = $as->fetch();
if (!$attempt) jsonError(404, 'У участника нет завершённых прохождений');
$attemptId = (int)$attempt['id'];

$qStmt = $pdo->prepare('SELECT id, type, title, correct_value FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
$qStmt->execute([':f' => $formId]);
$optStmt = $pdo->prepare('SELECT id, text, is_correct FROM public.test_options WHERE question_id = :q ORDER BY position, id');
$ansStmt = $pdo->prepare('SELECT id, text_value, number_value, is_correct, answered FROM public.test_answers WHERE attempt_id = :a AND question_id = :q');
$selStmt = $pdo->prepare('SELECT option_id FROM public.test_answer_options WHERE answer_id = :ans');

$yn = fn($v) => $v === 'yes' ? 'Да' : ($v === 'no' ? 'Нет' : (string)$v);

$answers = [];
foreach ($qStmt->fetchAll() as $q) {
    $qid = (int)$q['id'];
    $type = $q['type'];

    $ansStmt->execute([':a' => $attemptId, ':q' => $qid]);
    $a = $ansStmt->fetch();
    $answered = $a ? tf_bool($a['answered']) : false;
    $isCorrect = ($a && $a['is_correct'] !== null) ? tf_bool($a['is_correct']) : null;

    // карта вариантов
    $optText = []; $correctOptIds = [];
    if (in_array($type, ['single', 'multiple', 'dropdown'], true)) {
        $optStmt->execute([':q' => $qid]);
        foreach ($optStmt->fetchAll() as $o) { $optText[(int)$o['id']] = $o['text']; if (tf_bool($o['is_correct'])) $correctOptIds[] = (int)$o['id']; }
    }

    // ответ пользователя
    $userAnswer = '— нет ответа';
    if ($answered && $a) {
        if (in_array($type, ['single', 'multiple', 'dropdown'], true)) {
            $selStmt->execute([':ans' => (int)$a['id']]);
            $sel = array_map(fn($r) => $optText[(int)$r['option_id']] ?? '?', $selStmt->fetchAll());
            $userAnswer = $sel ? implode(', ', $sel) : '— нет ответа';
        } elseif ($type === 'yesno') {
            $userAnswer = $yn($a['text_value']);
        } elseif ($type === 'scale' || $type === 'number') {
            $userAnswer = (string)($a['number_value'] ?? $a['text_value'] ?? '');
        } else {
            $userAnswer = (string)($a['text_value'] ?? '');
        }
    }

    // правильный ответ
    $correctAnswer = '';
    if (in_array($type, ['single', 'multiple', 'dropdown'], true)) {
        $correctAnswer = implode(', ', array_map(fn($id) => $optText[$id] ?? '?', $correctOptIds));
    } elseif ($type === 'yesno') {
        $correctAnswer = $q['correct_value'] !== null ? $yn($q['correct_value']) : '';
    } elseif (in_array($type, ['scale', 'number', 'text', 'textarea', 'date'], true)) {
        $correctAnswer = (string)($q['correct_value'] ?? '');
    }

    $answers[] = [
        'title' => $q['title'],
        'type' => $type,
        'userAnswer' => $userAnswer,
        'correctAnswer' => $correctAnswer,
        'isCorrect' => $isCorrect,
        'answered' => $answered,
    ];
}

jsonOk([
    'attempt' => [
        'score' => $attempt['score'] !== null ? (float)$attempt['score'] : null,
        'passed' => $attempt['passed'] !== null ? tf_bool($attempt['passed']) : null,
        'finishedAt' => $attempt['finished_at'],
        'durationSec' => $attempt['duration_sec'] !== null ? (int)$attempt['duration_sec'] : null,
    ],
    'answers' => $answers,
]);
