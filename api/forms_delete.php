<?php
/**
 * API: /api/forms_delete.php
 *
 * DELETE /api/forms_delete.php?id=UUID
 *
 * Только администратор.
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
header('Access-Control-Allow-Methods: DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, X-User-Id');
header('Access-Control-Max-Age: 86400');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk($data) { echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); exit; }
function jsonError(int $code, string $message) { http_response_code($code); echo json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_UNICODE); exit; }
function isUuid($s): bool { return is_string($s) && preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i', $s); }

if ($_SERVER['REQUEST_METHOD'] !== 'DELETE') jsonError(405, 'Метод не поддерживается');

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

if (!isAdmin($pdo)) jsonError(403, 'Недостаточно прав');

$pdo->prepare("DELETE FROM public.forms WHERE id = :id")->execute([':id' => $id]);
jsonOk(null);

