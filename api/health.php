<?php
/**
 * Health-check для деплоя и мониторинга.
 * GET /api/health.php
 */

error_reporting(0);
ini_set('display_errors', '0');

header('Content-Type: application/json; charset=utf-8');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

$payload = [
    'ok'      => true,
    'service' => 'corporate-portal',
    'time'    => gmdate('c'),
];

$configLocal = __DIR__ . '/config.local.php';
if (is_readable($configLocal)) {
    require $configLocal;
    if (defined('DB_HOST') && defined('DB_NAME') && defined('DB_USER') && defined('DB_PASS')) {
        try {
            $port = defined('DB_PORT') ? DB_PORT : '5432';
            $pdo = new PDO(
                sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, $port, DB_NAME),
                DB_USER,
                DB_PASS,
                [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
            );
            $pdo->query('SELECT 1');
            $payload['database'] = 'ok';
        } catch (Throwable $e) {
            http_response_code(503);
            $payload['ok'] = false;
            $payload['database'] = 'error';
        }
    }
}

echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
