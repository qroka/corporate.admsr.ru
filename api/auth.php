<?php
/**
 * API: /api/auth.php
 *
 * POST /api/auth.php — проверка логина и пароля
 * Body: { "login": "...", "password": "..." }
 * Response 200: { "success": true,  "user": { id, fio, ofo, user_group } }
 * Response 401: { "success": false, "message": "..." }
 */

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'message' => 'Метод не поддерживается']);
    exit;
}

$body = json_decode(file_get_contents('php://input'), true);
$login    = trim($body['login']    ?? '');
$password = $body['password'] ?? '';

if ($login === '' || $password === '') {
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => 'Введите логин и пароль']);
    exit;
}

try {
    $pdo = new PDO(
        sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
        DB_USER, DB_PASS,
        [
            PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::ATTR_EMULATE_PREPARES   => false,
        ]
    );
    $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Ошибка подключения к БД']);
    exit;
}

$stmt = $pdo->prepare(
    'SELECT id, status, password, fio, ofo, user_group FROM public.authentication WHERE login = :login LIMIT 1'
);
$stmt->execute([':login' => $login]);
$user = $stmt->fetch();

if (!$user) {
    http_response_code(401);
    echo json_encode(['success' => false, 'message' => 'Неверный логин или пароль']);
    exit;
}

// Поддержка как password_hash(), так и открытого текста
$passwordOk = password_verify($password, $user['password']) || $user['password'] === $password;

if (!$passwordOk) {
    http_response_code(401);
    echo json_encode(['success' => false, 'message' => 'Неверный логин или пароль']);
    exit;
}

if (!(bool)$user['status']) {
    http_response_code(403);
    echo json_encode(['success' => false, 'message' => 'Учётная запись отключена. Обратитесь к администратору']);
    exit;
}

$pdo->prepare('UPDATE public.authentication SET auth = true WHERE id = :id')
    ->execute([':id' => $user['id']]);

echo json_encode([
    'success' => true,
    'user' => [
        'id'         => (int)$user['id'],
        'fio'        => $user['fio'],
        'ofo'        => $user['ofo'],
        'user_group' => $user['user_group'],
    ],
]);