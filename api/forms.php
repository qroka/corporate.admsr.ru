<?php
/**
 * API: /api/forms.php
 *
 * POST /api/forms.php
 *   Создать форму (черновик). Body: { title, description, coverUrl, mode, settings, questions? }
 *
 * GET /api/forms.php?id=UUID
 *   Получить форму с вопросами и вариантами.
 *
 * PUT /api/forms.php?id=UUID
 *   Обновить форму (метаданные + вопросы). Body: { ...как в конструкторе... }
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
header('Access-Control-Allow-Methods: GET, POST, PUT, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk($data) { echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); exit; }
function jsonError(int $code, string $message) { http_response_code($code); echo json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_UNICODE); exit; }

function uuidv4(): string {
  $data = random_bytes(16);
  $data[6] = chr((ord($data[6]) & 0x0f) | 0x40);
  $data[8] = chr((ord($data[8]) & 0x3f) | 0x80);
  return vsprintf('%s%s-%s-%s-%s-%s%s%s', str_split(bin2hex($data), 4));
}

function isUuid($s): bool {
  return is_string($s) && preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i', $s);
}

function getJsonBody(): array {
  $raw = file_get_contents('php://input');
  $d = json_decode($raw, true);
  if (!is_array($d)) jsonError(400, 'Некорректный JSON');
  return $d;
}

function validateFormPayload(array $d): array {
  $title = trim((string)($d['title'] ?? ''));
  if ($title === '') jsonError(400, 'title обязателен');
  if (mb_strlen($title) > 200) jsonError(400, 'title слишком длинный');

  $mode = (string)($d['mode'] ?? 'survey');
  if ($mode !== 'survey' && $mode !== 'test') jsonError(400, 'mode должен быть survey|test');

  $status = (string)($d['status'] ?? 'draft');
  if (!in_array($status, ['draft', 'published', 'archived'], true)) jsonError(400, 'Некорректный status');

  $description = (string)($d['description'] ?? '');
  if (mb_strlen($description) > 2000) jsonError(400, 'description слишком длинный');

  $coverUrl = $d['coverUrl'] ?? null;
  if (!is_null($coverUrl) && (!is_string($coverUrl) || mb_strlen(trim($coverUrl)) < 5)) {
    jsonError(400, 'coverUrl должен быть URL или null');
  }

  $settings = $d['settings'] ?? [];
  if (!is_array($settings)) jsonError(400, 'settings должен быть объектом');

  $questions = $d['questions'] ?? [];
  if (!is_array($questions)) jsonError(400, 'questions должен быть массивом');

  $allowedTypes = [
    'single_choice',
    'multiple_choice',
    'short_text',
    'long_text',
    'rating_1_10',
    'select',
    'file',
  ];

  $normQuestions = [];
  foreach ($questions as $idx => $q) {
    if (!is_array($q)) jsonError(400, 'Некорректный question');
    $qid = $q['id'] ?? null;
    if ($qid !== null && !isUuid($qid)) jsonError(400, 'question.id должен быть UUID');
    $type = (string)($q['type'] ?? '');
    if (!in_array($type, $allowedTypes, true)) jsonError(400, "Некорректный type у вопроса #".($idx+1));
    $qTitle = trim((string)($q['title'] ?? ''));
    if ($qTitle === '') jsonError(400, "Пустой title у вопроса #".($idx+1));
    $hint = (string)($q['hint'] ?? '');
    $required = (bool)($q['required'] ?? false);
    $order = (int)($q['order'] ?? $idx);

    $options = $q['options'] ?? null;
    $normOptions = [];
    $needsOptions = in_array($type, ['single_choice','multiple_choice','select'], true);
    if ($needsOptions) {
      if (!is_array($options)) jsonError(400, "options обязателен у вопроса #".($idx+1));
      if (count($options) < 2) jsonError(400, "Минимум 2 варианта у вопроса #".($idx+1));
      $anyCorrect = false;
      foreach ($options as $oIdx => $o) {
        if (!is_array($o)) jsonError(400, 'Некорректный option');
        $oid = $o['id'] ?? null;
        if ($oid !== null && !isUuid($oid)) jsonError(400, 'option.id должен быть UUID');
        $label = trim((string)($o['label'] ?? ''));
        if ($label === '') jsonError(400, "Пустой label у варианта #".($oIdx+1));
        $isCorrect = (bool)($o['isCorrect'] ?? false);
        if ($isCorrect) $anyCorrect = true;
        $normOptions[] = [
          'id' => $oid,
          'label' => $label,
          'order' => (int)($o['order'] ?? $oIdx),
          'isCorrect' => $isCorrect,
        ];
      }
      if ($mode === 'test' && !$anyCorrect) {
        jsonError(400, "В тесте у вопроса #".($idx+1)." должен быть отмечен хотя бы один правильный вариант");
      }
    } else {
      $normOptions = [];
    }

    $normQuestions[] = [
      'id' => $qid,
      'type' => $type,
      'order' => $order,
      'title' => $qTitle,
      'hint' => $hint,
      'required' => $required,
      'options' => $normOptions,
    ];
  }

  return [
    'title' => $title,
    'description' => $description,
    'cover_url' => is_string($coverUrl) ? trim($coverUrl) : null,
    'mode' => $mode,
    'status' => $status,
    'settings' => $settings,
    'questions' => $normQuestions,
  ];
}

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

$method = $_SERVER['REQUEST_METHOD'];
$id = $_GET['id'] ?? null;
if ($id !== null && !isUuid($id)) jsonError(400, 'Некорректный id');

if ($method === 'POST') {
  $d = getJsonBody();
  $p = validateFormPayload($d);

  $formId = uuidv4();
  $stmt = $pdo->prepare('
    INSERT INTO public.forms (id, title, description, cover_url, status, mode, settings)
    VALUES (:id, :title, :description, :cover_url, :status, :mode, :settings::jsonb)
  ');
  $stmt->execute([
    ':id' => $formId,
    ':title' => $p['title'],
    ':description' => $p['description'],
    ':cover_url' => $p['cover_url'],
    ':status' => 'draft',
    ':mode' => $p['mode'],
    ':settings' => json_encode($p['settings'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
  ]);

  jsonOk(['id' => $formId]);
}

if ($method === 'GET') {
  if (!$id) jsonError(400, 'Укажите ?id=UUID');

  $formStmt = $pdo->prepare('SELECT id, title, description, cover_url, status, mode, settings, created_at, updated_at FROM public.forms WHERE id = :id LIMIT 1');
  $formStmt->execute([':id' => $id]);
  $form = $formStmt->fetch();
  if (!$form) jsonError(404, 'Форма не найдена');

  $qStmt = $pdo->prepare('SELECT id, form_id, type, "order", title, hint, required, config FROM public.questions WHERE form_id = :id ORDER BY "order" ASC');
  $qStmt->execute([':id' => $id]);
  $qs = $qStmt->fetchAll();

  $qIds = array_map(fn($r) => $r['id'], $qs);
  $optionsByQ = [];
  if (count($qIds)) {
    $in = implode(',', array_fill(0, count($qIds), '?'));
    $oStmt = $pdo->prepare("SELECT id, question_id, label, \"order\", is_correct FROM public.question_options WHERE question_id IN ($in) ORDER BY question_id ASC, \"order\" ASC");
    $oStmt->execute($qIds);
    foreach ($oStmt->fetchAll() as $o) {
      $qid = $o['question_id'];
      if (!isset($optionsByQ[$qid])) $optionsByQ[$qid] = [];
      $optionsByQ[$qid][] = [
        'id' => $o['id'],
        'label' => $o['label'],
        'order' => (int)$o['order'],
        'isCorrect' => (bool)$o['is_correct'],
      ];
    }
  }

  $out = [
    'id' => $form['id'],
    'title' => $form['title'],
    'description' => $form['description'] ?? '',
    'coverUrl' => $form['cover_url'],
    'status' => $form['status'],
    'mode' => $form['mode'],
    'settings' => json_decode($form['settings'] ?? '{}', true) ?: [],
    'createdAt' => $form['created_at'],
    'updatedAt' => $form['updated_at'],
    'questions' => array_map(function($q) use ($optionsByQ) {
      $qid = $q['id'];
      return [
        'id' => $qid,
        'formId' => $q['form_id'],
        'type' => $q['type'],
        'order' => (int)$q['order'],
        'title' => $q['title'],
        'hint' => $q['hint'] ?? '',
        'required' => (bool)$q['required'],
        'options' => $optionsByQ[$qid] ?? [],
      ];
    }, $qs),
  ];

  jsonOk($out);
}

if ($method === 'PUT') {
  if (!$id) jsonError(400, 'Укажите ?id=UUID');
  $d = getJsonBody();
  $p = validateFormPayload($d);

  // Ensure form exists
  $check = $pdo->prepare('SELECT id, status FROM public.forms WHERE id = :id LIMIT 1');
  $check->execute([':id' => $id]);
  $existing = $check->fetch();
  if (!$existing) jsonError(404, 'Форма не найдена');

  $pdo->beginTransaction();
  try {
    $upd = $pdo->prepare('
      UPDATE public.forms
      SET title = :title,
          description = :description,
          cover_url = :cover_url,
          mode = :mode,
          settings = :settings::jsonb
      WHERE id = :id
    ');
    $upd->execute([
      ':id' => $id,
      ':title' => $p['title'],
      ':description' => $p['description'],
      ':cover_url' => $p['cover_url'],
      ':mode' => $p['mode'],
      ':settings' => json_encode($p['settings'], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
    ]);

    // Rebuild questions/options (simple & consistent)
    $pdo->prepare('DELETE FROM public.questions WHERE form_id = :id')->execute([':id' => $id]);

    foreach ($p['questions'] as $q) {
      $qid = $q['id'] && isUuid($q['id']) ? $q['id'] : uuidv4();
      $insQ = $pdo->prepare('
        INSERT INTO public.questions (id, form_id, type, "order", title, hint, required, config)
        VALUES (:id, :form_id, :type, :order, :title, :hint, :required, :config::jsonb)
      ');
      $insQ->execute([
        ':id' => $qid,
        ':form_id' => $id,
        ':type' => $q['type'],
        ':order' => (int)$q['order'],
        ':title' => $q['title'],
        ':hint' => $q['hint'],
        ':required' => $q['required'] ? 't' : 'f',
        ':config' => json_encode(new stdClass(), JSON_UNESCAPED_UNICODE),
      ]);

      if (in_array($q['type'], ['single_choice','multiple_choice','select'], true)) {
        foreach ($q['options'] as $opt) {
          $oid = $opt['id'] && isUuid($opt['id']) ? $opt['id'] : uuidv4();
          $insO = $pdo->prepare('
            INSERT INTO public.question_options (id, question_id, label, "order", is_correct)
            VALUES (:id, :question_id, :label, :order, :is_correct)
          ');
          $insO->execute([
            ':id' => $oid,
            ':question_id' => $qid,
            ':label' => $opt['label'],
            ':order' => (int)$opt['order'],
            ':is_correct' => $opt['isCorrect'] ? 't' : 'f',
          ]);
        }
      }
    }

    $pdo->commit();
    jsonOk(null);
  } catch (Exception $e) {
    $pdo->rollBack();
    jsonError(500, 'Ошибка сохранения формы');
  }
}

jsonError(405, 'Метод не поддерживается');

