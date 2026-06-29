<?php
/**
 * POST /api/tests_submit.php — записать завершённое прохождение.
 * Body: { userId|null, formId, durationSec, answers: { [questionId]: value } }
 *   value: single/dropdown → optionId; multiple → optionId[]; yesno → 'yes'|'no';
 *          scale/number → number; text/textarea/date → string.
 * → { attemptId, score, passed, correctCount, scorable }
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);
$token = trim((string)($body['token'] ?? ''));
$answers = $body['answers'] ?? [];
$durationSec = isset($body['durationSec']) ? (int)$body['durationSec'] : null;
if (!is_array($answers)) $answers = [];

$viaLink = false;
$respondentToken = null;
$guestName = null;
$guestOfo = null;
$userId = $viewer > 0 ? $viewer : null;

if ($token !== '') {
    // ── Прохождение по ссылке ──
    $st = $pdo->prepare("SELECT * FROM public.test_forms WHERE access_token = :t AND status = 'published' AND access_by_link = true");
    $st->execute([':t' => $token]);
    $form = $st->fetch();
    if (!$form) jsonError(404, 'Ссылка недействительна');
    $formId = (int)$form['id'];
    $viaLink = true;
    $mode = $form['link_access'] ?? 'any';
    if ($viewer > 0) {
        $userId = $viewer; // авторизованный по ссылке — учитываем под аккаунтом
    } else {
        if ($mode === 'authorized') jsonError(403, 'Форма доступна только авторизованным — войдите в портал');
        $userId = null; // гость
        $respondentToken = isset($body['respondentToken']) ? substr((string)$body['respondentToken'], 0, 100) : null;
        $guestName = trim((string)($body['guestName'] ?? ''));
        $guestOfo = (int)($body['guestOfoId'] ?? 0);
        if ($guestName === '') jsonError(400, 'Укажите ФИО');
        $guestOfo = $guestOfo > 0 ? $guestOfo : null;
    }
} else {
    // ── Обычное прохождение из портала ──
    $formId = (int)($body['formId'] ?? 0);
    if ($formId <= 0) jsonError(400, 'Не передан formId');
    $st = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
    $st->execute([':id' => $formId]);
    $form = $st->fetch();
    if (!$form) jsonError(404, 'Форма не найдена');
    if ($form['status'] !== 'published') jsonError(409, 'Форма не опубликована');

    // Доступ к приватной форме: создатель, либо адресат (лично / по ОФО)
    if ($form['visibility'] === 'private') {
        $allowed = $viewer > 0 && (int)$form['owner_id'] === $viewer;
        if (!$allowed && $viewer > 0) {
            $c = $pdo->prepare('SELECT 1 FROM public.test_audience_users WHERE form_id = :f AND user_id = :u LIMIT 1');
            $c->execute([':f' => $formId, ':u' => $viewer]);
            if ($c->fetchColumn(0)) $allowed = true;
        }
        if (!$allowed && $viewer > 0) {
            $o = $pdo->prepare('SELECT ofo FROM public.user_info WHERE id = :u');
            $o->execute([':u' => $viewer]);
            $raw = $o->fetchColumn(0);
            if ($raw !== false && preg_match('/^[0-9]+$/', (string)$raw)) {
                $c = $pdo->prepare('SELECT 1 FROM public.test_audience_ofo WHERE form_id = :f AND ofo_unit_id = :o LIMIT 1');
                $c->execute([':f' => $formId, ':o' => (int)$raw]);
                if ($c->fetchColumn(0)) $allowed = true;
            }
        }
        if (!$allowed) jsonError(403, 'Нет доступа к этой форме');
    }
}

$isTest = $form['kind'] === 'test';

// Лимит попыток (для голосования с «переголосовать» не ограничиваем)
if (!($form['kind'] === 'poll' && tf_bool($form['allow_revote']))) {
    $allowedAttempts = tf_bool($form['limit_attempts']) ? max(1, (int)$form['attempts']) : 1;
    if ($userId !== null) {
        $cnt = $pdo->prepare("SELECT COUNT(*) FROM public.test_attempts WHERE form_id = :f AND user_id = :u AND status = 'completed'");
        $cnt->execute([':f' => $formId, ':u' => $userId]);
        if ((int)$cnt->fetchColumn(0) >= $allowedAttempts) jsonError(409, 'Вы уже прошли эту форму допустимое число раз');
    } elseif ($respondentToken) {
        $cnt = $pdo->prepare("SELECT COUNT(*) FROM public.test_attempts WHERE form_id = :f AND respondent_token = :r AND status = 'completed'");
        $cnt->execute([':f' => $formId, ':r' => $respondentToken]);
        if ((int)$cnt->fetchColumn(0) >= $allowedAttempts) jsonError(409, 'С этого устройства форма уже пройдена');
    }
}

// Вопросы + правильные ответы
$qStmt = $pdo->prepare('SELECT id, type, correct_value FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
$qStmt->execute([':f' => $formId]);
$questions = $qStmt->fetchAll();
$optStmt = $pdo->prepare('SELECT id, is_correct FROM public.test_options WHERE question_id = :q');

$norm = fn($s) => mb_strtolower(trim((string)$s));

$pdo->beginTransaction();
try {
    $aIns = $pdo->prepare(
        "INSERT INTO public.test_attempts
           (form_id, user_id, status, current_page, started_at, finished_at, duration_sec, ip, user_agent, via_link, respondent_token, guest_name, guest_ofo_id)
         VALUES
           (:f, :u, 'completed', 0, now() - make_interval(secs => :dur::int), now(), :dur2, :ip, :ua, :vl, :rt, :gn, :go) RETURNING id"
    );
    $dur = $durationSec ?? 0;
    $aIns->execute([
        ':f' => $formId, ':u' => $userId, ':dur' => $dur, ':dur2' => $durationSec,
        ':ip' => $_SERVER['REMOTE_ADDR'] ?? null, ':ua' => $_SERVER['HTTP_USER_AGENT'] ?? null,
        ':vl' => $viaLink ? 't' : 'f', ':rt' => $respondentToken,
        ':gn' => ($guestName !== null && $guestName !== '') ? $guestName : null, ':go' => $guestOfo,
    ]);
    $attemptId = (int)$aIns->fetchColumn(0);

    $ansIns = $pdo->prepare(
        'INSERT INTO public.test_answers (attempt_id, question_id, text_value, number_value, is_correct, answered)
         VALUES (:a, :q, :tv, :nv, :ic, :answered) RETURNING id'
    );
    $aoIns = $pdo->prepare('INSERT INTO public.test_answer_options (answer_id, option_id) VALUES (:ans, :opt) ON CONFLICT DO NOTHING');

    $scorable = 0; $correctCount = 0;

    foreach ($questions as $q) {
        $qid = (int)$q['id'];
        $type = $q['type'];
        $val = $answers[(string)$qid] ?? ($answers[$qid] ?? null);

        $textVal = null; $numVal = null; $selected = [];
        $answered = false;

        if (in_array($type, ['single', 'dropdown'], true)) {
            if ($val !== null && $val !== '') { $selected = [(int)$val]; $answered = true; }
        } elseif ($type === 'multiple') {
            if (is_array($val) && count($val)) { $selected = array_map('intval', $val); $answered = true; }
        } elseif ($type === 'scale' || $type === 'number') {
            if ($val !== null && $val !== '') { $numVal = 0 + $val; $textVal = (string)$val; $answered = true; }
        } else { // yesno / text / textarea / date
            if ($val !== null && $val !== '') { $textVal = (string)$val; $answered = true; }
        }

        // правильность (для тестов)
        $isCorrect = null;
        if ($isTest) {
            $hasCorrect = false; $correctOptIds = []; $cv = $q['correct_value'];
            if (in_array($type, ['single', 'multiple', 'dropdown'], true)) {
                $optStmt->execute([':q' => $qid]);
                foreach ($optStmt->fetchAll() as $o) { if (tf_bool($o['is_correct'])) $correctOptIds[] = (int)$o['id']; }
                $hasCorrect = count($correctOptIds) > 0;
                if ($hasCorrect) {
                    sort($correctOptIds); $sel = $selected; sort($sel);
                    $isCorrect = $type === 'multiple' ? ($sel === $correctOptIds) : (count($sel) === 1 && in_array($sel[0], $correctOptIds, true));
                }
            } elseif ($cv !== null && $cv !== '') {
                $hasCorrect = true;
                if ($type === 'scale' || $type === 'number') $isCorrect = ($numVal !== null) && (0 + $cv == $numVal);
                else $isCorrect = $norm($textVal) === $norm($cv); // yesno/text/textarea/date — без регистра/пробелов
            }
            if ($hasCorrect) { $scorable++; if ($isCorrect) $correctCount++; }
            if (!$hasCorrect) $isCorrect = null;
        }

        $ansIns->execute([
            ':a' => $attemptId, ':q' => $qid, ':tv' => $textVal, ':nv' => $numVal,
            ':ic' => $isCorrect === null ? null : ($isCorrect ? 't' : 'f'),
            ':answered' => $answered ? 't' : 'f',
        ]);
        $ansId = (int)$ansIns->fetchColumn(0);
        foreach ($selected as $optId) $aoIns->execute([':ans' => $ansId, ':opt' => $optId]);
    }

    // Балл
    $score = null; $passed = null;
    if ($isTest && $scorable > 0) {
        $score = round($correctCount / $scorable * 100, 2);
        if (tf_bool($form['use_passing_score'])) $passed = $score >= (int)$form['passing_score'];
        $pdo->prepare('UPDATE public.test_attempts SET score = :s, max_score = 100, passed = :p WHERE id = :id')
            ->execute([':s' => $score, ':p' => $passed === null ? null : ($passed ? 't' : 'f'), ':id' => $attemptId]);
    }

    $pdo->commit();
} catch (Throwable $e) {
    $pdo->rollBack();
    jsonError(500, 'Ошибка записи прохождения');
}

jsonOk(['attemptId' => $attemptId, 'score' => $score, 'passed' => $passed, 'correctCount' => $correctCount, 'scorable' => $scorable]);
