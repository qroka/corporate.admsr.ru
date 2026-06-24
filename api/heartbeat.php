<?php
/**
 * API: /api/heartbeat.php — отметка активности пользователя.
 *
 * POST { "id": N } — обновляет last_activity = now() (только если пользователь авторизован).
 * Вызывается фронтом при реальной активности (клик/ввод/навигация), не чаще раза в неск. минут.
 * Если 24 часа без heartbeat — check-auth.php снимет авторизацию.
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
$id = isset($body['id']) ? (int)$body['id'] : 0;
if ($id <= 0) {
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => 'Некорректный id']);
    exit;
}

try {
    $pdo = new PDO(
        sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
        DB_USER, DB_PASS,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
    );
    $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Ошибка подключения к БД']);
    exit;
}

// Обновляем активность только у авторизованных и активных учёток.
$stmt = $pdo->prepare(
    'UPDATE public.user_info SET last_activity = now() WHERE id = :id AND auth = true AND status = true'
);
$stmt->execute([':id' => $id]);

echo json_encode(['success' => true, 'updated' => $stmt->rowCount() > 0]);
