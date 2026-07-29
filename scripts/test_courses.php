#!/usr/bin/env php
<?php
/**
 * CLI smoke-тест модуля учебных курсов.
 *
 * Проверяет:
 *  - подключение к БД (те же DB_* что у API);
 *  - наличие таблиц course_* и user_sessions в information_schema;
 *  - опционально HTTP POST courses_list с Bearer, если заданы
 *    COURSE_TEST_TOKEN и COURSE_TEST_BASE.
 *
 * Exit: 0 = успех, 1 = есть FAIL.
 *
 * Запуск: php scripts/test_courses.php
 *         npm run test:courses
 */

declare(strict_types=1);

$failed = 0;

function pass(string $msg): void
{
    echo "PASS  {$msg}\n";
}

function fail(string $msg): void
{
    global $failed;
    $failed++;
    echo "FAIL  {$msg}\n";
}

function info(string $msg): void
{
    echo "INFO  {$msg}\n";
}

// ── DB credentials (как у API) ───────────────────────────────────────────────

$root = dirname(__DIR__);
$configLocal = $root . DIRECTORY_SEPARATOR . 'api' . DIRECTORY_SEPARATOR . 'config.local.php';
if (is_readable($configLocal)) {
    require $configLocal;
}

if (!defined('DB_HOST')) {
    define('DB_HOST', 'localhost');
}
if (!defined('DB_PORT')) {
    define('DB_PORT', '5432');
}
if (!defined('DB_NAME')) {
    define('DB_NAME', 'corporate_portal');
}
if (!defined('DB_USER')) {
    define('DB_USER', 'myuser');
}
if (!defined('DB_PASS')) {
    // Совпадает с дефолтом api/auth_context.php и api/tests_common.php
    define('DB_PASS', 'VZAIMno4753');
}

echo "=== courses smoke test ===\n";
info('DB ' . DB_USER . '@' . DB_HOST . ':' . DB_PORT . '/' . DB_NAME);

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
    pass('PostgreSQL connection');
} catch (Throwable $e) {
    fail('PostgreSQL connection: ' . $e->getMessage());
    echo "\nResult: FAIL ({$failed})\n";
    exit(1);
}

// ── Required tables ──────────────────────────────────────────────────────────

$requiredTables = [
    'user_sessions',
    'course_courses',
    'course_versions',
    'course_topics',
    'course_materials',
    'course_test_links',
    'course_assignments',
    'course_enrollments',
    'course_topic_progress',
    'course_material_progress',
    'course_learning_sessions',
    'course_test_attempt_links',
    'course_completions',
    'course_audit_logs',
];

$st = $pdo->prepare(
    "SELECT table_name
     FROM information_schema.tables
     WHERE table_schema = 'public' AND table_name = :t
     LIMIT 1"
);

foreach ($requiredTables as $table) {
    $st->execute([':t' => $table]);
    if ($st->fetchColumn()) {
        pass("table public.{$table}");
    } else {
        fail("table public.{$table} missing (apply db/migration/V4__courses_module.sql)");
    }
}

// ── Optional HTTP check ──────────────────────────────────────────────────────

$token = getenv('COURSE_TEST_TOKEN') ?: '';
$base = rtrim(getenv('COURSE_TEST_BASE') ?: '', '/');

if ($token === '' || $base === '') {
    info('HTTP courses_list skipped (set COURSE_TEST_TOKEN and COURSE_TEST_BASE to enable)');
} else {
    $url = $base . '/api/courses_list.php';
    info("HTTP POST {$url}");

    $body = '{}';
    $ok = false;
    $detail = '';

    if (!function_exists('curl_init')) {
        fail('courses_list skipped: ext-curl required for HTTP check');
    } else {
        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_POST => true,
            CURLOPT_POSTFIELDS => $body,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => 30,
            CURLOPT_HTTPHEADER => [
                'Content-Type: application/json',
                'Authorization: Bearer ' . $token,
                'X-Session-Token: ' . $token,
            ],
        ]);
        $raw = curl_exec($ch);
        $errno = curl_errno($ch);
        $code = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);
        if ($errno) {
            $detail = 'curl error ' . $errno;
        } else {
            $json = is_string($raw) ? json_decode($raw, true) : null;
            if ($code === 200 && is_array($json) && !empty($json['success'])) {
                $ok = true;
                $n = isset($json['data']['items']) && is_array($json['data']['items'])
                    ? count($json['data']['items'])
                    : 0;
                $detail = "HTTP {$code}, items={$n}";
            } else {
                $msg = is_array($json) ? (string)($json['message'] ?? '') : substr((string)$raw, 0, 200);
                $detail = "HTTP {$code}" . ($msg !== '' ? ": {$msg}" : '');
            }
        }

        if ($ok) {
            pass("courses_list {$detail}");
        } else {
            fail("courses_list {$detail}");
        }
    }
}

echo "\n";
if ($failed > 0) {
    echo "Result: FAIL ({$failed} check(s))\n";
    exit(1);
}

echo "Result: PASS\n";
exit(0);
