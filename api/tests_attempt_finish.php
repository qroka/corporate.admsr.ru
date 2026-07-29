<?php
/**
 * POST /api/tests_attempt_finish.php
 * Body: {attemptId, answers?, durationSec?}
 * Дозаписывает ответы, оценивает через tf_evaluate_answers, завершает попытку.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$attemptId = (int)($body['attemptId'] ?? 0);
if ($attemptId <= 0) jsonError(400, 'Не передан attemptId');

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

$formId = (int)$att['form_id'];
$fSt = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
$fSt->execute([':id' => $formId]);
$formRow = $fSt->fetch();
if (!$formRow) jsonError(404, 'Форма не найдена');

// Если переданы answers — сохранить как в save
$incoming = $body['answers'] ?? null;
if (is_array($incoming) && $incoming) {
    // делегируем логику через повторное использование: inline upsert
    $_POST_SAVE = true; // marker unused
    $qStmt = $pdo->prepare('SELECT id, type FROM public.test_questions WHERE form_id = :f');
    $qStmt->execute([':f' => $formId]);
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
        foreach ($incoming as $qidRaw => $val) {
            $qid = (int)$qidRaw;
            if (!isset($qTypes[$qid])) continue;
            $type = $qTypes[$qid];
            $textVal = null; $numVal = null; $selected = []; $answered = false;
            if (in_array($type, ['single', 'dropdown'], true)) {
                if ($val !== null && $val !== '') { $selected = [(int)$val]; $answered = true; }
            } elseif ($type === 'multiple') {
                if (is_array($val) && count($val)) { $selected = array_map('intval', $val); $answered = true; }
            } elseif ($type === 'scale' || $type === 'number') {
                if ($val !== null && $val !== '') { $numVal = 0 + $val; $textVal = (string)$val; $answered = true; }
            } else {
                if ($val !== null && $val !== '') { $textVal = (string)$val; $answered = true; }
            }
            $delOpts->execute([':a' => $attemptId, ':q' => $qid]);
            $delAns->execute([':a' => $attemptId, ':q' => $qid]);
            $ansIns->execute([
                ':a' => $attemptId, ':q' => $qid, ':tv' => $textVal, ':nv' => $numVal,
                ':answered' => $answered ? 't' : 'f',
            ]);
            $ansId = (int)$ansIns->fetchColumn(0);
            foreach ($selected as $optId) $aoIns->execute([':ans' => $ansId, ':opt' => $optId]);
        }
        $pdo->commit();
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) $pdo->rollBack();
        jsonError(500, 'Ошибка сохранения ответов');
    }
}

// Собрать answers map из БД
$ansSt = $pdo->prepare('SELECT * FROM public.test_answers WHERE attempt_id = :a');
$ansSt->execute([':a' => $attemptId]);
$optSt = $pdo->prepare('SELECT option_id FROM public.test_answer_options WHERE answer_id = :id');
$answers = [];
foreach ($ansSt->fetchAll() as $a) {
    $qid = (int)$a['question_id'];
    $optSt->execute([':id' => (int)$a['id']]);
    $opts = array_map(static fn($r) => (int)$r['option_id'], $optSt->fetchAll());
    if ($opts) {
        $answers[$qid] = count($opts) === 1 ? $opts[0] : $opts;
    } elseif ($a['number_value'] !== null && $a['number_value'] !== '') {
        $answers[$qid] = 0 + $a['number_value'];
    } else {
        $answers[$qid] = $a['text_value'];
    }
}

$qStmt = $pdo->prepare('SELECT id, type, correct_value FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
$qStmt->execute([':f' => $formId]);
$questions = $qStmt->fetchAll();

$eval = tf_evaluate_answers($pdo, $formRow, $questions, $answers);
$durationSec = isset($body['durationSec']) ? (int)$body['durationSec'] : null;

$pdo->beginTransaction();
try {
    // обновить is_correct на answers
    $upd = $pdo->prepare('UPDATE public.test_answers SET is_correct = :ic WHERE attempt_id = :a AND question_id = :q');
    foreach ($eval['details'] as $d) {
        $upd->execute([
            ':ic' => $d['isCorrect'] === null ? null : ($d['isCorrect'] ? 't' : 'f'),
            ':a' => $attemptId,
            ':q' => $d['questionId'],
        ]);
    }

    $pdo->prepare(
        "UPDATE public.test_attempts
         SET status = 'completed', finished_at = now(),
             duration_sec = COALESCE(:dur, duration_sec),
             score = :s, max_score = 100, passed = :p
         WHERE id = :id"
    )->execute([
        ':dur' => $durationSec,
        ':s' => $eval['score'],
        ':p' => $eval['passed'] === null ? null : ($eval['passed'] ? 't' : 'f'),
        ':id' => $attemptId,
    ]);

    // Курс: обновить прогресс темы / completion
    $ls = $pdo->prepare(
        'SELECT enrollment_id, course_test_link_id FROM public.course_test_attempt_links WHERE test_attempt_id = :a'
    );
    $ls->execute([':a' => $attemptId]);
    $cl = $ls->fetch();
    $enrollmentId = null;
    if ($cl) {
        $enrollmentId = (int)$cl['enrollment_id'];
        $linkId = (int)$cl['course_test_link_id'];
        $linkSt = $pdo->prepare('SELECT * FROM public.course_test_links WHERE id = :id');
        $linkSt->execute([':id' => $linkId]);
        $link = $linkSt->fetch();
        if ($link && $link['type'] === 'topic' && $link['topic_id']) {
            $tid = (int)$link['topic_id'];
            if (cs_check_topic_complete($pdo, $enrollmentId, $tid)) {
                $pdo->prepare(
                    "UPDATE public.course_topic_progress
                     SET status = 'completed', completed_at = COALESCE(completed_at, now()), updated_at = now()
                     WHERE enrollment_id = :e AND topic_id = :t"
                )->execute([':e' => $enrollmentId, ':t' => $tid]);
                cs_recalculate_locks($pdo, $enrollmentId);
            }
        }
        $pdo->prepare(
            'UPDATE public.course_enrollments SET last_activity_at = now(), updated_at = now() WHERE id = :id'
        )->execute([':id' => $enrollmentId]);
    }

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка завершения попытки');
}

if ($enrollmentId) {
    try {
        cs_try_complete_enrollment($pdo, $enrollmentId);
    } catch (Throwable $e) {
        // не блокируем ответ по тесту
    }
}

jsonOk([
    'attemptId' => $attemptId,
    'score' => $eval['score'],
    'passed' => $eval['passed'],
    'correctCount' => $eval['correctCount'],
    'scorable' => $eval['scorable'],
    'enrollmentId' => $enrollmentId,
    'nextAction' => $enrollmentId ? cs_next_action($pdo, $enrollmentId) : null,
]);
