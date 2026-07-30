<?php
/**
 * GET /api/portal_my_permissions.php
 * Текущие права пользователя: { isAdmin, sections: string[] }
 */
require_once __DIR__ . '/auth_context.php';

header('Content-Type: application/json; charset=utf-8');
$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : '*'));
header('Access-Control-Allow-Methods: GET, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}
if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
    http_response_code(405);
    echo json_encode(['success' => false, 'message' => 'Метод не поддерживается', 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

$user = auth_require_user($pdo);
$isAdmin = auth_is_admin($user);
$sections = auth_user_sections($pdo, $user);

echo json_encode([
    'success' => true,
    'data' => [
        'isAdmin' => $isAdmin,
        'sections' => $sections,
    ],
], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
