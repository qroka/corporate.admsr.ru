<?php
/**
 * POST /api/logout.php — сброс сессии пользователя
 * Body (legacy): { "id": 1 }
 * Сессия отзывается по Bearer / X-Session-Token / cookie.
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
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'message' => 'Метод не поддерживается']);
    exit;
}

$body = json_decode(file_get_contents('php://input'), true);
$id = isset($body['id']) ? (int)$body['id'] : 0;

try {
    $pdo = new PDO(
        sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
        DB_USER, DB_PASS,
        [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::ATTR_EMULATE_PREPARES => false,
        ]
    );
    $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Ошибка подключения к БД']);
    exit;
}

require_once __DIR__ . '/auth_context.php';

$token = auth_extract_token();
$userIdFromSession = null;
if ($token) {
    $st = $pdo->prepare(
        'SELECT user_id FROM public.user_sessions
         WHERE token_hash = :h AND revoked_at IS NULL LIMIT 1'
    );
    $st->execute([':h' => auth_hash_token($token)]);
    $userIdFromSession = $st->fetchColumn(0);
    if ($userIdFromSession !== false) {
        $userIdFromSession = (int)$userIdFromSession;
    } else {
        $userIdFromSession = null;
    }
}

auth_revoke_session($pdo, $token);

$uid = $id > 0 ? $id : ($userIdFromSession ?? 0);
if ($uid > 0) {
    $pdo->prepare('UPDATE public.user_info SET auth = false WHERE id = :id')->execute([':id' => $uid]);
}

echo json_encode(['success' => true]);
