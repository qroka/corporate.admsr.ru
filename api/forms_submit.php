<?php
/**
 * API: /api/forms_submit.php
 *
 * POST /api/forms_submit.php?id=UUID — отправить ответы.
 *
 * Безопасность:
 * - anonymous session: sessionId UUID обязателен
 * - защита от повторной отправки: UNIQUE(form_id, session_id)
 * - rate limit: max 5 req/min per IP (на форме)
 */

error_reporting(0);
ini_set('display_errors', '0');

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

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
header('Access-Control-Allow-Methods: POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk($data) { echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); exit; }
function jsonError(int $code, string $message) { http_response_code($code); echo json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_UNICODE); exit; }
function isUuid($s): bool { return is_string($s) && preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i', $s); }
function uuidv4(): string {
  $data = random_bytes(16);
  $data[6] = chr((ord($data[6]) & 0x0f) | 0x40);
  $data[8] = chr((ord($data[8]) & 0x3f) | 0x80);
  return vsprintf('%s%s-%s-%s-%s-%s%s%s', str_split(bin2hex($data), 4));
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$formId = $_GET['id'] ?? null;
if (!$formId || !isUuid($formId)) jsonError(400, 'Укажите ?id=UUID');

$raw = file_get_contents('php://input');
$body = json_decode($raw, true);
if (!is_array($body)) jsonError(400, 'Некорректный JSON');

$sessionId = $body['sessionId'] ?? null;
if (!$sessionId || !isUuid($sessionId)) jsonError(400, 'sessionId обязателен (UUID)');

$answers = $body['answers'] ?? [];
if (!is_array($answers)) jsonError(400, 'answers должен быть массивом');

$ip = $_SERVER['REMOTE_ADDR'] ?? null;
$ua = $_SERVER['HTTP_USER_AGENT'] ?? null;

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

// rate limit: 5 per minute per IP
if ($ip) {
  $rl = $pdo->prepare('
    SELECT COUNT(*)::int AS cnt
    FROM public.form_responses
    WHERE form_id = :fid AND ip = :ip AND created_at > (now() - interval \'1 minute\')
  ');
  $rl->execute([':fid' => $formId, ':ip' => $ip]);
  $cnt = (int)($rl->fetch()['cnt'] ?? 0);
  if ($cnt >= 5) jsonError(429, 'Слишком много запросов. Попробуйте позже.');
}

$formStmt = $pdo->prepare('SELECT id, status, mode, settings FROM public.forms WHERE id = :id LIMIT 1');
$formStmt->execute([':id' => $formId]);
$form = $formStmt->fetch();
if (!$form) jsonError(404, 'Форма не найдена');
if ($form['status'] !== 'published') jsonError(409, 'Форма не опубликована');

$settings = json_decode($form['settings'] ?? '{}', true) ?: [];
$showMode = (string)($settings['showResultMode'] ?? 'immediate'); // immediate | after_review
$showResult = $showMode === 'immediate';

// load questions & options
$qStmt = $pdo->prepare('SELECT id, type, required FROM public.questions WHERE form_id = :id ORDER BY "order" ASC');
$qStmt->execute([':id' => $formId]);
$questions = $qStmt->fetchAll();
if (!count($questions)) jsonError(409, 'У формы нет вопросов');

$qById = [];
foreach ($questions as $q) $qById[$q['id']] = $q;

$qIds = array_keys($qById);
$optionsByQ = [];
if (count($qIds)) {
  $in = implode(',', array_fill(0, count($qIds), '?'));
  $oStmt = $pdo->prepare("SELECT id, question_id, is_correct FROM public.question_options WHERE question_id IN ($in)");
  $oStmt->execute($qIds);
  foreach ($oStmt->fetchAll() as $o) {
    $qid = $o['question_id'];
    if (!isset($optionsByQ[$qid])) $optionsByQ[$qid] = [];
    $optionsByQ[$qid][] = ['id' => $o['id'], 'is_correct' => (bool)$o['is_correct']];
  }
}

// normalize answers map
$answerMap = [];
foreach ($answers as $a) {
  if (!is_array($a)) continue;
  $qid = $a['questionId'] ?? null;
  if (!$qid || !isUuid($qid) || !isset($qById[$qid])) continue;
  $value = $a['value'] ?? null;
  if (!is_array($value) || !isset($value['type'])) continue;
  $answerMap[$qid] = $value;
}

// required validation
foreach ($questions as $q) {
  if (!(bool)$q['required']) continue;
  $qid = $q['id'];
  $v = $answerMap[$qid] ?? null;
  if (!$v) jsonError(400, 'Заполните все обязательные вопросы');

  $t = (string)$v['type'];
  if (($t === 'single' || $t === 'select') && empty($v['optionId'])) jsonError(400, 'Заполните все обязательные вопросы');
  if ($t === 'multiple' && (!isset($v['optionIds']) || !is_array($v['optionIds']) || count($v['optionIds']) < 1)) jsonError(400, 'Заполните все обязательные вопросы');
  if (($t === 'short_text' || $t === 'long_text') && trim((string)($v['text'] ?? '')) === '') jsonError(400, 'Заполните все обязательные вопросы');
  if ($t === 'rating_1_10' && (int)($v['value'] ?? 0) < 1) jsonError(400, 'Заполните все обязательные вопросы');
  if ($t === 'file' && empty($v['base64'])) jsonError(400, 'Заполните все обязательные вопросы');
}

// scoring
$mode = $form['mode']; // survey|test
$score = null;
$maxScore = null;
$correctPercent = null;
$perQuestion = [];

$scorableTotal = 0;
$scorableCorrect = 0;

function setify($arr): array {
  $out = [];
  foreach ($arr as $x) { $out[(string)$x] = true; }
  ksort($out);
  return array_keys($out);
}

if ($mode === 'test') {
  foreach ($questions as $q) {
    $qid = $q['id'];
    $qt = $q['type'];
    $v = $answerMap[$qid] ?? null;

    $isScorable = in_array($qt, ['single_choice','multiple_choice','select'], true);
    if (!$isScorable) {
      $perQuestion[] = ['questionId' => $qid, 'isCorrect' => null, 'earned' => null, 'possible' => null];
      continue;
    }
    $scorableTotal += 1;

    $correctOpts = array_values(array_filter($optionsByQ[$qid] ?? [], fn($o) => $o['is_correct']));
    $correctIds = array_map(fn($o) => $o['id'], $correctOpts);
    $ok = false;

    if (!$v) {
      $ok = false;
    } else if (($qt === 'single_choice' || $qt === 'select') && ($v['type'] === 'single' || $v['type'] === 'select')) {
      $ok = in_array((string)($v['optionId'] ?? ''), $correctIds, true);
    } else if ($qt === 'multiple_choice' && $v['type'] === 'multiple') {
      $given = setify($v['optionIds'] ?? []);
      $corr = setify($correctIds);
      $ok = ($given === $corr);
    }

    if ($ok) $scorableCorrect += 1;
    $perQuestion[] = ['questionId' => $qid, 'isCorrect' => $ok, 'earned' => $ok ? 1 : 0, 'possible' => 1];
  }

  $score = $scorableCorrect;
  $maxScore = $scorableTotal;
  $correctPercent = $scorableTotal > 0 ? round(($scorableCorrect / $scorableTotal) * 100.0, 2) : null;
}

// insert response with unique (form_id, session_id)
$responseId = uuidv4();
$meta = [
  'respondent' => is_array($body['respondent'] ?? null) ? $body['respondent'] : new stdClass(),
];

$pdo->beginTransaction();
try {
  $insR = $pdo->prepare('
    INSERT INTO public.form_responses
      (id, form_id, session_id, ip, user_agent, status, completed_at, score, max_score, correct_percent, meta)
    VALUES
      (:id, :form_id, :session_id, :ip, :ua, :status, now(), :score, :max_score, :correct_percent, :meta::jsonb)
  ');
  $insR->execute([
    ':id' => $responseId,
    ':form_id' => $formId,
    ':session_id' => $sessionId,
    ':ip' => $ip,
    ':ua' => $ua,
    ':status' => 'completed',
    ':score' => $score,
    ':max_score' => $maxScore,
    ':correct_percent' => $correctPercent,
    ':meta' => json_encode($meta, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
  ]);
} catch (PDOException $e) {
  $pdo->rollBack();
  // unique violation -> already submitted
  if ((string)$e->getCode() === '23505') jsonError(409, 'Эта сессия уже отправляла ответы по этой форме');
  jsonError(500, 'Ошибка сохранения результата');
}

// answers insert
$insA = $pdo->prepare('
  INSERT INTO public.response_answers
    (id, response_id, question_id, answer_json, text_value, number_value, option_ids)
  VALUES
    (:id, :response_id, :question_id, :answer_json::jsonb, :text_value, :number_value, :option_ids)
  ON CONFLICT (response_id, question_id)
  DO UPDATE SET answer_json = EXCLUDED.answer_json, text_value = EXCLUDED.text_value, number_value = EXCLUDED.number_value, option_ids = EXCLUDED.option_ids
');

foreach ($answerMap as $qid => $v) {
  $qt = $qById[$qid]['type'];
  $isCorrect = null;
  foreach ($perQuestion as $pq) { if ($pq['questionId'] === $qid) { $isCorrect = $pq['isCorrect']; break; } }

  $answerJson = ['value' => $v, 'isCorrect' => $isCorrect];
  $textValue = null;
  $numberValue = null;
  $optionIds = null;

  $t = (string)($v['type'] ?? '');
  if ($t === 'short_text' || $t === 'long_text') {
    $textValue = (string)($v['text'] ?? '');
  } else if ($t === 'rating_1_10') {
    $numberValue = (int)($v['value'] ?? 0);
  } else if ($t === 'single' || $t === 'select') {
    $optionIds = '{' . (string)($v['optionId'] ?? '') . '}';
  } else if ($t === 'multiple') {
    $ids = array_map('strval', is_array($v['optionIds'] ?? null) ? $v['optionIds'] : []);
    $optionIds = '{' . implode(',', $ids) . '}';
  } else if ($t === 'file') {
    $textValue = (string)($v['fileName'] ?? '');
  }

  $insA->execute([
    ':id' => uuidv4(),
    ':response_id' => $responseId,
    ':question_id' => $qid,
    ':answer_json' => json_encode($answerJson, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
    ':text_value' => $textValue,
    ':number_value' => $numberValue,
    ':option_ids' => $optionIds,
  ]);
}

$pdo->commit();

jsonOk([
  'responseId' => $responseId,
  'score' => $score,
  'maxScore' => $maxScore,
  'correctPercent' => $correctPercent,
  'showResult' => $showResult,
  'perQuestion' => $perQuestion,
]);

