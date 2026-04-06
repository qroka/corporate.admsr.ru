<?php
/**
 * API: /api/chat.php
 *
 * POST /api/chat.php — прокси к локальному Python AI-серверу (FastAPI + Ollama + Qwen2.5:7b)
 *
 * Body (JSON):
 *   { "messages": [ { "role": "user"|"assistant", "content": "..." }, ... ] }
 *
 * Response:
 *   { "success": true,  "message": "...", "model": "...", ... }
 *   { "success": false, "error": "..."   }
 *
 * Архитектура:
 *   Браузер → /api/chat.php (PHP прокси) → http://127.0.0.1:8000/chat (Python FastAPI) → Ollama → Qwen2.5:7b
 *
 * Настройка Python-сервера: см. python-ai/install.sh
 */

error_reporting(0);
ini_set('display_errors', '0');
set_time_limit(320);  // снимаем стандартный 30-секундный лимит PHP

// ─── CORS / Headers ───────────────────────────────────────────────────────────
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
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'error' => 'Метод не разрешён']);
    exit;
}

// ─── Конфигурация ─────────────────────────────────────────────────────────────
define('AI_SERVER_URL', 'http://127.0.0.1:8000/chat');
define('AI_TIMEOUT',    300);  // секунд — Qwen 7B на CPU может отвечать 1-2 минуты

// ─── Входные данные ───────────────────────────────────────────────────────────
$rawBody = file_get_contents('php://input');
$body    = json_decode($rawBody, true);

if (json_last_error() !== JSON_ERROR_NONE || !is_array($body)) {
    http_response_code(400);
    echo json_encode(['success' => false, 'error' => 'Некорректный JSON']);
    exit;
}

$inputMessages = $body['messages'] ?? [];

if (empty($inputMessages) || !is_array($inputMessages)) {
    http_response_code(400);
    echo json_encode(['success' => false, 'error' => 'Поле messages обязательно']);
    exit;
}

// Валидация — только разрешённые роли и непустой контент
$validMessages = [];
foreach ($inputMessages as $msg) {
    if (
        isset($msg['role'], $msg['content'])
        && in_array($msg['role'], ['user', 'assistant'], true)
        && is_string($msg['content'])
        && trim($msg['content']) !== ''
    ) {
        $validMessages[] = [
            'role'    => $msg['role'],
            'content' => mb_substr(trim($msg['content']), 0, 4096),
        ];
    }
}

if (empty($validMessages)) {
    http_response_code(400);
    echo json_encode(['success' => false, 'error' => 'Нет валидных сообщений']);
    exit;
}

// ─── Запрос к Python AI-серверу ───────────────────────────────────────────────
$payload = json_encode(['messages' => $validMessages], JSON_UNESCAPED_UNICODE);

$ch = curl_init(AI_SERVER_URL);
curl_setopt_array($ch, [
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_POST           => true,
    CURLOPT_POSTFIELDS     => $payload,
    CURLOPT_HTTPHEADER     => ['Content-Type: application/json'],
    CURLOPT_TIMEOUT        => AI_TIMEOUT,
    CURLOPT_CONNECTTIMEOUT => 5,
    CURLOPT_SSL_VERIFYPEER => false,  // localhost — SSL не нужен
]);

$response = curl_exec($ch);
$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
$curlErr  = curl_error($ch);
curl_close($ch);

// ─── Обработка ответа ─────────────────────────────────────────────────────────
if ($curlErr) {
    // Python-сервер не запущен или недоступен
    http_response_code(503);
    echo json_encode([
        'success' => false,
        'error'   => 'AI-сервер недоступен. Обратитесь к администратору.',
    ]);
    exit;
}

$data = json_decode($response, true);

if ($httpCode !== 200 || !isset($data['success'])) {
    $errMsg = $data['detail'] ?? $data['error'] ?? 'Неизвестная ошибка AI-сервера';
    http_response_code(502);
    echo json_encode(['success' => false, 'error' => $errMsg]);
    exit;
}

// Возвращаем ответ клиенту как есть
echo $response;
