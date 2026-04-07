<?php
/**
 * API: /api/forms_publish.php
 *
 * POST /api/forms_publish.php?id=UUID — опубликовать форму
 *
 * Серверная валидация:
 * - минимум 1 вопрос
 * - single/multiple/select: минимум 2 варианта
 * - если mode=test: хотя бы 1 правильный вариант
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

if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$id = $_GET['id'] ?? null;
if (!$id || !isUuid($id)) jsonError(400, 'Укажите ?id=UUID');

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

$f = $pdo->prepare('SELECT id, status, mode FROM public.forms WHERE id = :id LIMIT 1');
$f->execute([':id' => $id]);
$form = $f->fetch();
if (!$form) jsonError(404, 'Форма не найдена');
if ($form['status'] === 'archived') jsonError(409, 'Форма в архиве');

$q = $pdo->prepare('SELECT id, type FROM public.questions WHERE form_id = :id ORDER BY "order" ASC');
$q->execute([':id' => $id]);
$questions = $q->fetchAll();
if (count($questions) < 1) jsonError(400, 'Перед публикацией добавьте хотя бы 1 вопрос');

foreach ($questions as $question) {
  $type = $question['type'];
  $needsOptions = in_array($type, ['single_choice','multiple_choice','select'], true);
  if (!$needsOptions) continue;

  $o = $pdo->prepare('SELECT is_correct FROM public.question_options WHERE question_id = :qid');
  $o->execute([':qid' => $question['id']]);
  $opts = $o->fetchAll();
  if (count($opts) < 2) jsonError(400, 'Вопросы с выбором должны иметь минимум 2 варианта');
  if ($form['mode'] === 'test') {
    $any = false;
    foreach ($opts as $r) { if ((bool)$r['is_correct']) { $any = true; break; } }
    if (!$any) jsonError(400, 'В тесте у вопроса должен быть отмечен хотя бы один правильный вариант');
  }
}

$pdo->prepare("UPDATE public.forms SET status = 'published' WHERE id = :id")->execute([':id' => $id]);
jsonOk(null);

