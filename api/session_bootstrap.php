<?php
/**
 * POST /api/session_bootstrap.php
 * Выдать sessionToken уже авторизованному в портале пользователю
 * (после деплоя модуля курсов, если вход был до появления сессий).
 *
 * Body: { "id": <user_info.id> } — как check-auth.php.
 * Проверяет status, auth и last_activity (24ч), затем создаёт user_sessions.
 */
define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : ($origin ?: '*')));
header('Access-Control-Allow-Methods: POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
header('Access-Control-Allow-Credentials: true');

if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') {
    http_response_code(204);
    exit;
}

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'message' => 'Метод не поддерживается', 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

$body = json_decode(file_get_contents('php://input'), true);
$id = isset($body['id']) ? (int)$body['id'] : 0;
if ($id <= 0) {
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => 'Некорректный id', 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

try {
    $pdo = new PDO(
        sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
        DB_USER,
        DB_PASS,
        [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::ATTR_EMULATE_PREPARES => false,
        ]
    );
    $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Ошибка подключения к БД', 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

$stmt = $pdo->prepare(
    "SELECT id, user_group, auth, status, last_activity,
            (auth = true AND last_activity IS NOT NULL AND last_activity > now() - interval '24 hours') AS session_ok
     FROM public.user_info
     WHERE id = :id AND status = true
     LIMIT 1"
);
$stmt->execute([':id' => $id]);
$row = $stmt->fetch();

if (!$row || !($row['session_ok'] === true || $row['session_ok'] === 't' || $row['session_ok'] === '1')) {
    http_response_code(401);
    echo json_encode(['success' => false, 'message' => 'Сессия портала недействительна. Войдите снова.', 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

require_once __DIR__ . '/auth_context.php';

try {
    $token = auth_create_session($pdo, (int)$row['id']);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Не удалось создать сессию (проверьте миграцию V4)', 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

echo json_encode([
    'success' => true,
    'message' => 'OK',
    'data' => [
        'sessionToken' => $token,
        'userGroup' => (string)($row['user_group'] ?? ''),
    ],
], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
