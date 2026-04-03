<?php
/**
 * API: /api/users.php
 *
 * GET /api/users.php - получение списка пользователей
 * PUT /api/users.php?id=N - обновление пользователя (статус, данные)
 */

error_reporting(0);
ini_set('display_errors', '0');

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

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
} catch (PDOException $e) {
  jsonError(500, 'Ошибка подключения к БД');
}

$method = $_SERVER['REQUEST_METHOD'];
$id     = isset($_GET['id']) ? (int)$_GET['id'] : null;

switch ($method) {
  case 'GET':
    $stmt = $pdo->prepare('SELECT id, status, login, password, firstname, surname, lastname, ofo, user_group, phone, email, auth, avatar_url, role FROM public.user_info ORDER BY id ASC');
    $stmt->execute();
    $rows = $stmt->fetchAll();
    
    $formatted = array_map(function($r) {
      return [
        'id'         => (int)$r['id'],
        'status'     => (string)$r['status'],
        'login'      => (string)$r['login'],
        'password'   => (string)$r['password'],
        'firstname'  => (string)$r['firstname'],
        'surname'    => (string)$r['surname'],
        'lastname'   => (string)$r['lastname'],
        'ofo'        => (string)$r['ofo'],
        'user_group' => (string)$r['user_group'],
        'phone'      => (string)$r['phone'],
        'email'      => (string)$r['email'],
        'auth'       => (string)$r['auth'],
        'avatar_url' => (string)$r['avatar_url'],
        'role'       => (string)$r['role'],
      ];
    }, $rows);
    
    jsonOk($formatted);
    break;

  case 'PUT':
    if (!$id) jsonError(400, 'Укажите ?id=...');
    $raw = file_get_contents('php://input');
    if (!$raw) jsonError(400, 'Тело запроса пустое');
    $d = json_decode($raw, true);
    if (!is_array($d)) jsonError(400, 'Некорректный JSON');
    
    $check = $pdo->prepare('SELECT id FROM public.user_info WHERE id = :id');
    $check->execute([':id' => $id]);
    if (!$check->fetch()) jsonError(404, 'Пользователь не найден');

    // Формируем запрос на обновление
    $fields = [];
    $params = [':id' => $id];
    
    $updatable = ['status', 'login', 'password', 'firstname', 'surname', 'lastname', 'ofo', 'user_group', 'phone', 'email', 'auth', 'avatar_url', 'role'];
    
    foreach ($updatable as $field) {
        if (array_key_exists($field, $d)) {
            $fields[] = "$field = :$field";
            $params[":$field"] = $d[$field];
        }
    }
    
    if (count($fields) > 0) {
        $sql = "UPDATE public.user_info SET " . implode(', ', $fields) . " WHERE id = :id";
        $stmt = $pdo->prepare($sql);
        $stmt->execute($params);
    }
    
    $stmt2 = $pdo->prepare('SELECT id, status, login, firstname, surname, lastname, ofo, user_group, phone, email, auth, avatar_url, role FROM public.user_info WHERE id = :id');
    $stmt2->execute([':id' => $id]);
    $updatedRow = $stmt2->fetch();
    
    jsonOk($updatedRow, 'Пользователь обновлен');
    break;

  default:
    jsonError(405, 'Метод не поддерживается');
}

function jsonOk($data, string $msg = 'OK')
{
  echo json_encode(['success' => true, 'message' => $msg, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
  exit;
}

function jsonError(int $code, string $msg)
{
  http_response_code($code);
  echo json_encode(['success' => false, 'message' => $msg, 'data' => null], JSON_UNESCAPED_UNICODE);
  exit;
}
