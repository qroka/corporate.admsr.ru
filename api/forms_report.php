<?php
/**
 * API: /api/forms_report.php
 *
 * GET /api/forms_report.php?id=UUID[&from=YYYY-MM-DD&to=YYYY-MM-DD&status=completed|incomplete&format=csv]
 *
 * Отчёт:
 * - summary: total, avg, correct%, median, stddev
 * - per question: distribution for charts
 * - funnel: reached per question
 * - top-3 mistakes (test mode)
 * - participants list
 */

error_reporting(0);
ini_set('display_errors', '0');

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

$allowedOrigins = [
  'http://localhost:5173',
  'http://localhost:5174',
  'http://127.0.0.1:5173',
  'https://corporate.admsr.ru',
];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
if (in_array($origin, $allowedOrigins, true)) {
  header("Access-Control-Allow-Origin: $origin");
}
header('Access-Control-Allow-Methods: GET, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk($data) { header('Content-Type: application/json; charset=utf-8'); echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); exit; }
function jsonError(int $code, string $message) { header('Content-Type: application/json; charset=utf-8'); http_response_code($code); echo json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_UNICODE); exit; }
function isUuid($s): bool { return is_string($s) && preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i', $s); }

if ($_SERVER['REQUEST_METHOD'] !== 'GET') jsonError(405, 'Метод не поддерживается');

$formId = $_GET['id'] ?? null;
if (!$formId || !isUuid($formId)) jsonError(400, 'Укажите ?id=UUID');

$from = isset($_GET['from']) ? trim((string)$_GET['from']) : null;
$to = isset($_GET['to']) ? trim((string)$_GET['to']) : null;
$status = isset($_GET['status']) ? trim((string)$_GET['status']) : null;
$format = isset($_GET['format']) ? trim((string)$_GET['format']) : null;

if ($status !== null && !in_array($status, ['completed', 'incomplete'], true)) jsonError(400, 'status должен быть completed|incomplete');

try {
  $pdo = new PDO(
    sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
    DB_USER,
    DB_PASS,
    [
      PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
      PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
      PDO::ATTR_EMULATE_PREPARES   => false,
    ]
  );
  $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException) {
  jsonError(500, 'Ошибка подключения к БД');
}

$formStmt = $pdo->prepare('SELECT id, title, description, cover_url, status, mode, settings, created_at, updated_at FROM public.forms WHERE id = :id LIMIT 1');
$formStmt->execute([':id' => $formId]);
$form = $formStmt->fetch();
if (!$form) jsonError(404, 'Форма не найдена');

$where = 'form_id = :fid';
$params = [':fid' => $formId];

if ($status) { $where .= ' AND status = :status'; $params[':status'] = $status; }
if ($from) { $where .= ' AND created_at >= :from::date'; $params[':from'] = $from; }
if ($to) { $where .= ' AND created_at < (:to::date + interval \'1 day\')'; $params[':to'] = $to; }

// Participants list
$pStmt = $pdo->prepare("
  SELECT id, session_id, started_at, completed_at, status, score, max_score, meta
  FROM public.form_responses
  WHERE $where
  ORDER BY created_at DESC
  LIMIT 5000
");
$pStmt->execute($params);
$participantsRaw = $pStmt->fetchAll();

$participants = array_map(function($r) {
  $meta = json_decode($r['meta'] ?? '{}', true) ?: [];
  $resp = $meta['respondent'] ?? [];
  return [
    'responseId' => $r['id'],
    'sessionId' => $r['session_id'],
    'userId' => isset($resp['userId']) ? (string)$resp['userId'] : null,
    'fio' => isset($resp['fio']) ? (string)$resp['fio'] : null,
    'startedAt' => $r['started_at'],
    'completedAt' => $r['completed_at'],
    'status' => $r['status'],
    'score' => $r['score'] !== null ? (float)$r['score'] : null,
    'maxScore' => $r['max_score'] !== null ? (float)$r['max_score'] : null,
  ];
}, $participantsRaw);

// Summary (completed only)
$sStmt = $pdo->prepare("
  SELECT
    COUNT(*)::int AS total,
    COUNT(*) FILTER (WHERE status = 'completed')::int AS completed,
    AVG(score) FILTER (WHERE status='completed' AND score IS NOT NULL) AS avg_score,
    AVG(correct_percent) FILTER (WHERE status='completed' AND correct_percent IS NOT NULL) AS avg_correct,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY score) FILTER (WHERE status='completed' AND score IS NOT NULL) AS median_score,
    stddev_pop(score) FILTER (WHERE status='completed' AND score IS NOT NULL) AS stddev_score
  FROM public.form_responses
  WHERE $where
");
$sStmt->execute($params);
$s = $sStmt->fetch();

$summary = [
  'totalResponses' => (int)($s['total'] ?? 0),
  'completedResponses' => (int)($s['completed'] ?? 0),
  'avgScore' => $s['avg_score'] !== null ? round((float)$s['avg_score'], 2) : null,
  'medianScore' => $s['median_score'] !== null ? round((float)$s['median_score'], 2) : null,
  'stdDevScore' => $s['stddev_score'] !== null ? round((float)$s['stddev_score'], 2) : null,
  'correctPercentAvg' => $s['avg_correct'] !== null ? round((float)$s['avg_correct'], 2) : null,
];

// Questions + options
$qStmt = $pdo->prepare('SELECT id, type, "order", title FROM public.questions WHERE form_id = :fid ORDER BY "order" ASC');
$qStmt->execute([':fid' => $formId]);
$qs = $qStmt->fetchAll();
$qIds = array_map(fn($x) => $x['id'], $qs);

$optLabels = [];
if (count($qIds)) {
  $in = implode(',', array_fill(0, count($qIds), '?'));
  $oStmt = $pdo->prepare("SELECT id, question_id, label, \"order\" FROM public.question_options WHERE question_id IN ($in) ORDER BY question_id, \"order\"");
  $oStmt->execute($qIds);
  foreach ($oStmt->fetchAll() as $o) {
    $qid = $o['question_id'];
    if (!isset($optLabels[$qid])) $optLabels[$qid] = [];
    $optLabels[$qid][$o['id']] = $o['label'];
  }
}

// Funnel: reached per question (count of responses having an answer row for that question)
$funnel = [];
foreach ($qs as $q) {
  $stmt = $pdo->prepare("
    SELECT COUNT(DISTINCT fr.id)::int AS reached
    FROM public.form_responses fr
    JOIN public.response_answers ra ON ra.response_id = fr.id AND ra.question_id = :qid
    WHERE $where
  ");
  $stmt->execute(array_merge($params, [':qid' => $q['id']]));
  $funnel[] = ['questionId' => $q['id'], 'reached' => (int)($stmt->fetch()['reached'] ?? 0)];
}

// Distributions per question
$questionsOut = [];
foreach ($qs as $q) {
  $qid = $q['id'];
  $type = $q['type'];

  if (in_array($type, ['single_choice','select','multiple_choice'], true)) {
    $stmt = $pdo->prepare("
      SELECT ra.option_ids
      FROM public.response_answers ra
      JOIN public.form_responses fr ON fr.id = ra.response_id
      WHERE ra.question_id = :qid AND $where
    ");
    $stmt->execute(array_merge($params, [':qid' => $qid]));
    $rows = $stmt->fetchAll();

    $counts = [];
    foreach ($rows as $r) {
      $raw = $r['option_ids'];
      if (!$raw) { $counts['__no_answer__'] = ($counts['__no_answer__'] ?? 0) + 1; continue; }
      // option_ids is returned as "{uuid,uuid}"
      $trim = trim($raw, '{}');
      if ($trim === '') { $counts['__no_answer__'] = ($counts['__no_answer__'] ?? 0) + 1; continue; }
      foreach (explode(',', $trim) as $oid) {
        $oid = trim($oid, '"');
        $counts[$oid] = ($counts[$oid] ?? 0) + 1;
      }
    }

    $dist = [];
    foreach (($optLabels[$qid] ?? []) as $oid => $label) {
      $dist[] = ['key' => $oid, 'label' => $label, 'count' => (int)($counts[$oid] ?? 0)];
    }
    $dist[] = ['key' => '__no_answer__', 'label' => 'Нет ответа', 'count' => (int)($counts['__no_answer__'] ?? 0)];

    $questionsOut[] = [
      'questionId' => $qid,
      'type' => $type,
      'title' => $q['title'],
      'distribution' => $dist,
    ];
  } else if ($type === 'rating_1_10') {
    $stmt = $pdo->prepare("
      SELECT ra.number_value
      FROM public.response_answers ra
      JOIN public.form_responses fr ON fr.id = ra.response_id
      WHERE ra.question_id = :qid AND $where
    ");
    $stmt->execute(array_merge($params, [':qid' => $qid]));
    $rows = $stmt->fetchAll();

    $counts = array_fill(1, 10, 0);
    $empty = 0;
    foreach ($rows as $r) {
      $v = $r['number_value'];
      if ($v === null) { $empty++; continue; }
      $n = (int)$v;
      if ($n >= 1 && $n <= 10) $counts[$n] += 1;
    }
    $dist = [];
    for ($i=1; $i<=10; $i++) $dist[] = ['key' => (string)$i, 'label' => (string)$i, 'count' => (int)$counts[$i]];
    $dist[] = ['key' => '__no_answer__', 'label' => 'Нет ответа', 'count' => (int)$empty];

    $questionsOut[] = [
      'questionId' => $qid,
      'type' => $type,
      'title' => $q['title'],
      'distribution' => $dist,
    ];
  } else {
    // text/file — answered vs empty
    $stmt = $pdo->prepare("
      SELECT ra.text_value
      FROM public.response_answers ra
      JOIN public.form_responses fr ON fr.id = ra.response_id
      WHERE ra.question_id = :qid AND $where
    ");
    $stmt->execute(array_merge($params, [':qid' => $qid]));
    $rows = $stmt->fetchAll();
    $answered = 0; $empty = 0;
    foreach ($rows as $r) {
      $t = trim((string)($r['text_value'] ?? ''));
      if ($t === '') $empty++; else $answered++;
    }
    $questionsOut[] = [
      'questionId' => $qid,
      'type' => $type,
      'title' => $q['title'],
      'distribution' => [
        ['key' => 'answered', 'label' => 'Ответили', 'count' => (int)$answered],
        ['key' => 'empty', 'label' => 'Пусто', 'count' => (int)$empty],
      ],
    ];
  }
}

// Top mistakes (test only): based on response_answers.answer_json->isCorrect
$topMistakes = [];
if ($form['mode'] === 'test') {
  $stmt = $pdo->prepare("
    SELECT
      ra.question_id,
      AVG(CASE WHEN (ra.answer_json->>'isCorrect') = 'true' THEN 1 ELSE 0 END) AS avg_correct
    FROM public.response_answers ra
    JOIN public.form_responses fr ON fr.id = ra.response_id
    JOIN public.questions q ON q.id = ra.question_id
    WHERE q.form_id = :fid AND q.type IN ('single_choice','multiple_choice','select') AND $where
    GROUP BY ra.question_id
    ORDER BY avg_correct ASC NULLS LAST
    LIMIT 3
  ");
  $stmt->execute($params);
  foreach ($stmt->fetchAll() as $r) {
    $avg = $r['avg_correct'];
    $wrong = $avg !== null ? (1.0 - (float)$avg) * 100.0 : 0.0;
    $topMistakes[] = ['questionId' => $r['question_id'], 'wrongPercent' => round($wrong, 2)];
  }
}

// CSV export (participants)
if ($format === 'csv') {
  header('Content-Type: text/csv; charset=utf-8');
  header('Content-Disposition: attachment; filename="report.csv"');
  $out = fopen('php://output', 'w');
  fputcsv($out, ['response_id','session_id','fio','user_id','status','started_at','completed_at','score','max_score']);
  foreach ($participants as $p) {
    fputcsv($out, [
      $p['responseId'],
      $p['sessionId'],
      $p['fio'],
      $p['userId'],
      $p['status'],
      $p['startedAt'],
      $p['completedAt'],
      $p['score'],
      $p['maxScore'],
    ]);
  }
  fclose($out);
  exit;
}

jsonOk([
  'form' => [
    'id' => $form['id'],
    'title' => $form['title'],
    'description' => $form['description'] ?? '',
    'coverUrl' => $form['cover_url'],
    'status' => $form['status'],
    'mode' => $form['mode'],
    'settings' => json_decode($form['settings'] ?? '{}', true) ?: [],
    'createdAt' => $form['created_at'],
    'updatedAt' => $form['updated_at'],
  ],
  'summary' => $summary,
  'funnel' => $funnel,
  'topMistakes' => $topMistakes,
  'questions' => $questionsOut,
  'participants' => $participants,
]);

