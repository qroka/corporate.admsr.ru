<?php
/**
 * API: /api/forms_list.php
 *
 * GET /api/forms_list.php?status=draft|published|archived&q=...
 *
 * Доступ:
 * - admin: видит все (с фильтрами)
 * - user/anon: только published
 *
 * Admin определяется сервером по X-User-Id (public.user_info.role='admin' AND auth=true AND status=true)
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
header('Access-Control-Allow-Headers: Content-Type, X-User-Id');
header('Access-Control-Max-Age: 86400');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk($data) { header('Content-Type: application/json; charset=utf-8'); echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); exit; }
function jsonError(int $code, string $message) { header('Content-Type: application/json; charset=utf-8'); http_response_code($code); echo json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_UNICODE); exit; }

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

function isAdmin(PDO $pdo): bool {
  $uid = $_SERVER['HTTP_X_USER_ID'] ?? '';
  $id = (int)$uid;
  if ($id <= 0) return false;
  $stmt = $pdo->prepare("SELECT role, auth, status FROM public.user_info WHERE id = :id LIMIT 1");
  $stmt->execute([':id' => $id]);
  $row = $stmt->fetch();
  if (!$row) return false;
  return ((string)($row['role'] ?? '') === 'admin') && ((bool)$row['auth'] === true) && ((bool)$row['status'] === true);
}

$admin = isAdmin($pdo);

$status = isset($_GET['status']) ? trim((string)$_GET['status']) : null;
$q = isset($_GET['q']) ? trim((string)$_GET['q']) : null;

$where = [];
$params = [];

if ($admin) {
  if ($status !== null && $status !== '') {
    if (!in_array($status, ['draft','published','archived'], true)) jsonError(400, 'Некорректный status');
    $where[] = "f.status = :status";
    $params[':status'] = $status;
  }
} else {
  $where[] = "f.status = 'published'";
}

if ($q) {
  $where[] = "f.title ILIKE :q";
  $params[':q'] = '%' . $q . '%';
}

$whereSql = count($where) ? ('WHERE ' . implode(' AND ', $where)) : '';

$sql = "
  SELECT
    f.id,
    f.title,
    f.description,
    f.cover_url,
    f.status,
    f.mode,
    f.created_at,
    f.updated_at,
    (SELECT COUNT(*)::int FROM public.questions q WHERE q.form_id = f.id) AS questions_count
  FROM public.forms f
  $whereSql
  ORDER BY f.updated_at DESC
  LIMIT 500
";

$stmt = $pdo->prepare($sql);
$stmt->execute($params);
$rows = $stmt->fetchAll();

$out = array_map(function($r) {
  return [
    'id' => $r['id'],
    'title' => $r['title'],
    'description' => $r['description'] ?? '',
    'coverUrl' => $r['cover_url'],
    'status' => $r['status'],
    'mode' => $r['mode'],
    'createdAt' => $r['created_at'],
    'updatedAt' => $r['updated_at'],
    'questionsCount' => (int)($r['questions_count'] ?? 0),
  ];
}, $rows);

jsonOk($out);

