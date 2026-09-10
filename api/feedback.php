<?php
/**
 * API: /api/feedback.php
 *
 * POST — отправить обращение обратной связи (требуется авторизация).
 * Body JSON: { category, subject, message }
 */

error_reporting(0);
ini_set('display_errors', '0');

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
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

function jsonOk($data = null): void
{
    echo json_encode(['success' => true, 'data' => $data, 'message' => null], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function jsonError(int $code, string $message): void
{
    http_response_code($code);
    echo json_encode(['success' => false, 'message' => $message, 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    jsonError(405, 'Метод не поддерживается');
}

require_once __DIR__ . '/auth_context.php';

$user = auth_require_user($pdo);
$userId = (int)($user['id'] ?? 0);
$fio = trim((string)($user['fio'] ?? ''));

$raw = file_get_contents('php://input');
$body = json_decode($raw, true);
if (!is_array($body)) {
    jsonError(400, 'Некорректный JSON');
}

$allowedCategories = ['question', 'bug', 'idea', 'other'];
$category = trim((string)($body['category'] ?? 'other'));
if (!in_array($category, $allowedCategories, true)) {
    jsonError(400, 'Некорректный тип обращения');
}

$subject = trim((string)($body['subject'] ?? ''));
$message = trim((string)($body['message'] ?? ''));

if ($subject === '') {
    jsonError(400, 'Укажите тему');
}
if (mb_strlen($subject) > 200) {
    jsonError(400, 'Тема слишком длинная');
}
if (mb_strlen($message) < 10) {
    jsonError(400, 'Сообщение слишком короткое');
}
if (mb_strlen($message) > 5000) {
    jsonError(400, 'Сообщение слишком длинное');
}

try {
    $pdo->exec(
        "CREATE TABLE IF NOT EXISTS public.portal_feedback (
            id BIGSERIAL PRIMARY KEY,
            user_id INTEGER NOT NULL,
            fio TEXT,
            category TEXT NOT NULL,
            subject TEXT NOT NULL,
            message TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )"
    );

    $stmt = $pdo->prepare(
        'INSERT INTO public.portal_feedback (user_id, fio, category, subject, message)
         VALUES (:user_id, :fio, :category, :subject, :message)
         RETURNING id, created_at'
    );
    $stmt->execute([
        ':user_id' => $userId,
        ':fio' => $fio !== '' ? $fio : null,
        ':category' => $category,
        ':subject' => $subject,
        ':message' => $message,
    ]);
    $row = $stmt->fetch();
    jsonOk([
        'id' => (int)($row['id'] ?? 0),
        'created_at' => $row['created_at'] ?? null,
    ]);
} catch (Throwable $e) {
    jsonError(500, 'Не удалось сохранить обращение');
}
