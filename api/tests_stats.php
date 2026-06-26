<?php
/**
 * POST /api/tests_stats.php — агрегированная статистика по форме.
 * Body: { formId }
 * → FormStats (та же форма, что у фронтового StatsDetail).
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$formId = (int)($body['formId'] ?? 0);
if ($formId <= 0) jsonError(400, 'Не передан formId');

$st = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
$st->execute([':id' => $formId]);
$form = $st->fetch();
if (!$form) jsonError(404, 'Форма не найдена');

$isTest = $form['kind'] === 'test';
$anonymous = tf_bool($form['anonymous']);
$pct = fn($p, $w) => $w > 0 ? (int)round($p / $w * 100) : 0;

// ── Базовые метрики ───────────────────────────────────────────────────────────
$row = $pdo->prepare(
    "SELECT
       COUNT(*) FILTER (WHERE status = 'completed') AS completions,
       COUNT(*) AS started,
       ROUND(AVG(duration_sec) FILTER (WHERE status = 'completed'))::int AS avg_time,
       MAX(finished_at) AS last_at,
       ROUND(AVG(score) FILTER (WHERE score IS NOT NULL))::int AS avg_score,
       COUNT(*) FILTER (WHERE passed IS TRUE) AS passed_cnt,
       COUNT(*) FILTER (WHERE passed IS NOT NULL) AS scored_cnt
     FROM public.test_attempts WHERE form_id = :f"
);
$row->execute([':f' => $formId]);
$m = $row->fetch();
$completions = (int)$m['completions'];
$started = (int)$m['started'];

// ── По ОФО ────────────────────────────────────────────────────────────────────
$ofoNames = [];
foreach ($pdo->query('SELECT id, name FROM public.ofo_unit')->fetchAll() as $u) $ofoNames[(int)$u['id']] = $u['name'];
$byOfo = [];
$oq = $pdo->prepare(
    "SELECT u.ofo AS ofo, COUNT(*) AS c
     FROM public.test_attempts a JOIN public.user_info u ON u.id = a.user_id
     WHERE a.form_id = :f AND a.status = 'completed' AND a.user_id IS NOT NULL AND u.ofo ~ '^[0-9]+$'
     GROUP BY u.ofo ORDER BY c DESC"
);
$oq->execute([':f' => $formId]);
foreach ($oq->fetchAll() as $r) {
    $byOfo[] = ['name' => $ofoNames[(int)$r['ofo']] ?? ('ОФО #' . $r['ofo']), 'count' => (int)$r['c'], 'percent' => $pct((int)$r['c'], $completions)];
}

// ── Участники (если не анонимно) ──────────────────────────────────────────────
$participants = null;
if (!$anonymous) {
    $pq = $pdo->prepare(
        "SELECT DISTINCT u.id, TRIM(CONCAT_WS(' ', u.surname, u.firstname, u.lastname)) AS name
         FROM public.test_attempts a JOIN public.user_info u ON u.id = a.user_id
         WHERE a.form_id = :f AND a.status = 'completed' AND a.user_id IS NOT NULL
         ORDER BY name"
    );
    $pq->execute([':f' => $formId]);
    $participants = array_map(fn($r) => ['id' => (int)$r['id'], 'name' => $r['name'] ?: ('ID ' . $r['id'])], $pq->fetchAll());
}

// ── По вопросам ───────────────────────────────────────────────────────────────
$questions = [];
$qStmt = $pdo->prepare('SELECT * FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
$qStmt->execute([':f' => $formId]);

$answeredStmt = $pdo->prepare('SELECT COUNT(*) FROM public.test_answers WHERE question_id = :q AND answered = true');
$rateStmt = $pdo->prepare("SELECT COUNT(*) FILTER (WHERE is_correct) c, COUNT(*) FILTER (WHERE is_correct IS NOT NULL) t FROM public.test_answers WHERE question_id = :q");
$optAggStmt = $pdo->prepare(
    'SELECT o.text, o.is_correct, COUNT(ao.answer_id) c
     FROM public.test_options o LEFT JOIN public.test_answer_options ao ON ao.option_id = o.id
     WHERE o.question_id = :q GROUP BY o.id, o.text, o.is_correct, o.position ORDER BY o.position, o.id'
);
$ynStmt = $pdo->prepare("SELECT text_value v, COUNT(*) c FROM public.test_answers WHERE question_id = :q AND answered = true GROUP BY text_value");
$scaleStmt = $pdo->prepare("SELECT number_value v, COUNT(*) c FROM public.test_answers WHERE question_id = :q AND answered = true AND number_value IS NOT NULL GROUP BY number_value");
$avgNumStmt = $pdo->prepare('SELECT ROUND(AVG(number_value), 2) FROM public.test_answers WHERE question_id = :q AND answered = true');

$hardest = null;
foreach ($qStmt->fetchAll() as $q) {
    $qid = (int)$q['id']; $type = $q['type'];
    $answeredStmt->execute([':q' => $qid]); $answered = (int)$answeredStmt->fetchColumn(0);

    $correctRate = null;
    if ($isTest) {
        $rateStmt->execute([':q' => $qid]); $rr = $rateStmt->fetch();
        if ((int)$rr['t'] > 0) $correctRate = $pct((int)$rr['c'], (int)$rr['t']);
    }

    $options = null; $isText = false; $avgNumber = null;
    if (in_array($type, ['single', 'multiple', 'dropdown'], true)) {
        $optAggStmt->execute([':q' => $qid]); $options = [];
        foreach ($optAggStmt->fetchAll() as $o) {
            $options[] = ['label' => $o['text'], 'count' => (int)$o['c'], 'percent' => $pct((int)$o['c'], $completions), 'correct' => tf_bool($o['is_correct'])];
        }
    } elseif ($type === 'yesno') {
        $ynStmt->execute([':q' => $qid]); $cnt = ['yes' => 0, 'no' => 0];
        foreach ($ynStmt->fetchAll() as $r) { $cnt[$r['v']] = (int)$r['c']; }
        $cv = $q['correct_value'];
        $options = [
            ['label' => 'Да', 'count' => $cnt['yes'], 'percent' => $pct($cnt['yes'], $completions), 'correct' => $cv === 'yes'],
            ['label' => 'Нет', 'count' => $cnt['no'], 'percent' => $pct($cnt['no'], $completions), 'correct' => $cv === 'no'],
        ];
    } elseif ($type === 'scale') {
        $scaleStmt->execute([':q' => $qid]); $map = [];
        foreach ($scaleStmt->fetchAll() as $r) { $map[(int)round($r['v'])] = (int)$r['c']; }
        $min = (int)($q['scale_min'] ?? 1); $max = (int)($q['scale_max'] ?? 5);
        if ($max < $min) [$min, $max] = [$max, $min];
        $options = [];
        for ($n = $min; $n <= $max && count($options) < 50; $n++) {
            $c = $map[$n] ?? 0;
            $options[] = ['label' => (string)$n, 'count' => $c, 'percent' => $pct($c, $completions), 'correct' => false];
        }
    } elseif ($type === 'number') {
        $avgNumStmt->execute([':q' => $qid]); $avgNumber = $avgNumStmt->fetchColumn(0);
        $avgNumber = $avgNumber !== null ? 0 + $avgNumber : null; $isText = true;
    } else {
        $isText = true; // text / textarea / date
    }

    $questions[] = [
        'id' => (string)$qid, 'title' => $q['title'], 'type' => $type,
        'answered' => $answered, 'skipped' => max(0, $completions - $answered),
        'options' => $options, 'correctRate' => $correctRate, 'isText' => $isText, 'avgNumber' => $avgNumber,
    ];
    if ($correctRate !== null && ($hardest === null || $correctRate < $hardest['correctRate'])) {
        $hardest = ['title' => $q['title'] ?: 'Без названия', 'correctRate' => $correctRate];
    }
}

jsonOk([
    'completions' => $completions,
    'started' => $started,
    'completionRate' => $pct($completions, $started),
    'avgTimeSec' => (int)($m['avg_time'] ?? 0),
    'lastAt' => $m['last_at'],
    'avgScore' => $isTest && $m['avg_score'] !== null ? (int)$m['avg_score'] : null,
    'passRate' => $isTest && (int)$m['scored_cnt'] > 0 ? $pct((int)$m['passed_cnt'], (int)$m['scored_cnt']) : null,
    'hardest' => $hardest,
    'byOfo' => $byOfo,
    'participants' => $participants,
    'questions' => $questions,
]);
