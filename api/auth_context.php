<?php
/**
 * auth_context.php — общий серверный auth context.
 * Не отдаёт данные сам по себе. Подключается через require.
 *
 * Источник личности: session token (Bearer / X-Session-Token / cookie).
 * НЕ доверяет userId из тела запроса.
 */

if (defined('AUTH_CONTEXT_LOADED')) {
    return;
}
define('AUTH_CONTEXT_LOADED', true);

if (!defined('DB_HOST')) {
    define('DB_HOST', 'localhost');
    define('DB_PORT', '5432');
    define('DB_NAME', 'corporate_portal');
    define('DB_USER', 'myuser');
    define('DB_PASS', 'VZAIMno4753');
}

if (!isset($pdo) || !($pdo instanceof PDO)) {
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
        header('Content-Type: application/json; charset=utf-8');
        echo json_encode(['success' => false, 'message' => 'Ошибка подключения к БД', 'data' => null], JSON_UNESCAPED_UNICODE);
        exit;
    }
}

const AUTH_SESSION_TTL_HOURS = 24;
const AUTH_SESSION_COOKIE = 'corp_session';

/**
 * Извлечь сырой токен сессии из заголовков / cookie.
 */
function auth_extract_token(): ?string
{
    $hdr = $_SERVER['HTTP_AUTHORIZATION'] ?? $_SERVER['REDIRECT_HTTP_AUTHORIZATION'] ?? '';
    if (preg_match('/^\s*Bearer\s+(\S+)\s*$/i', $hdr, $m)) {
        return $m[1];
    }
    $x = $_SERVER['HTTP_X_SESSION_TOKEN'] ?? '';
    if (is_string($x) && $x !== '') {
        return $x;
    }
    if (!empty($_COOKIE[AUTH_SESSION_COOKIE])) {
        return (string)$_COOKIE[AUTH_SESSION_COOKIE];
    }
    return null;
}

function auth_hash_token(string $token): string
{
    return hash('sha256', $token);
}

/**
 * Создать сессию после успешного логина. Возвращает plaintext token (один раз).
 */
function auth_create_session(PDO $pdo, int $userId): string
{
    $token = bin2hex(random_bytes(32));
    $hash = auth_hash_token($token);
    $ttl = AUTH_SESSION_TTL_HOURS;
    $stmt = $pdo->prepare(
        "INSERT INTO public.user_sessions (user_id, token_hash, expires_at, ip_address, user_agent)
         VALUES (:u, :h, now() + make_interval(hours => :ttl), :ip, :ua)"
    );
    $stmt->execute([
        ':u' => $userId,
        ':h' => $hash,
        ':ttl' => $ttl,
        ':ip' => $_SERVER['REMOTE_ADDR'] ?? null,
        ':ua' => isset($_SERVER['HTTP_USER_AGENT']) ? mb_substr((string)$_SERVER['HTTP_USER_AGENT'], 0, 500) : null,
    ]);

    // Cookie для same-origin запросов (опционально; SPA также хранит token в localStorage)
    if (!headers_sent()) {
        setcookie(AUTH_SESSION_COOKIE, $token, [
            'expires' => time() + $ttl * 3600,
            'path' => '/',
            'secure' => (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off'),
            'httponly' => true,
            'samesite' => 'Lax',
        ]);
    }

    return $token;
}

/**
 * Отозвать сессию по plaintext token.
 */
function auth_revoke_session(PDO $pdo, ?string $token): void
{
    if (!$token) {
        return;
    }
    $pdo->prepare(
        'UPDATE public.user_sessions SET revoked_at = now() WHERE token_hash = :h AND revoked_at IS NULL'
    )->execute([':h' => auth_hash_token($token)]);

    if (!headers_sent()) {
        setcookie(AUTH_SESSION_COOKIE, '', [
            'expires' => time() - 3600,
            'path' => '/',
            'httponly' => true,
            'samesite' => 'Lax',
        ]);
    }
}

/**
 * Отозвать все сессии пользователя.
 */
function auth_revoke_user_sessions(PDO $pdo, int $userId): void
{
    $pdo->prepare(
        'UPDATE public.user_sessions SET revoked_at = now() WHERE user_id = :u AND revoked_at IS NULL'
    )->execute([':u' => $userId]);
}

/**
 * Текущий авторизованный пользователь или null.
 * @return array{id:int,login:string,firstname:string,surname:string,lastname:string,email:?string,phone:?string,ofo:mixed,user_group:string,role:string,status:bool,auth:bool}|null
 */
function auth_current_user(PDO $pdo): ?array
{
    static $cached = false;
    static $user = null;
    if ($cached) {
        return $user;
    }
    $cached = true;

    $token = auth_extract_token();
    if (!$token) {
        return null;
    }

    $stmt = $pdo->prepare(
        "SELECT s.id AS session_id, s.expires_at, u.id, u.login, u.firstname, u.surname, u.lastname,
                u.email, u.phone, u.ofo, u.user_group, u.role, u.status, u.auth, u.last_activity
         FROM public.user_sessions s
         JOIN public.user_info u ON u.id = s.user_id
         WHERE s.token_hash = :h
           AND s.revoked_at IS NULL
         LIMIT 1"
    );
    $stmt->execute([':h' => auth_hash_token($token)]);
    $row = $stmt->fetch();
    if (!$row) {
        return null;
    }

    $statusOk = ($row['status'] === true || $row['status'] === 't' || $row['status'] === '1' || $row['status'] === 1);
    $authOk = ($row['auth'] === true || $row['auth'] === 't' || $row['auth'] === '1' || $row['auth'] === 1);
    $notExpired = strtotime($row['expires_at']) > time();

    if (!$statusOk || !$authOk || !$notExpired) {
        $pdo->prepare('UPDATE public.user_sessions SET revoked_at = now() WHERE id = :id')
            ->execute([':id' => (int)$row['session_id']]);
        return null;
    }

    // Продление last_seen + last_activity (не чаще чем раз в 5 мин — упрощённо каждый запрос)
    $pdo->prepare('UPDATE public.user_sessions SET last_seen_at = now() WHERE id = :id')
        ->execute([':id' => (int)$row['session_id']]);
    $pdo->prepare('UPDATE public.user_info SET last_activity = now() WHERE id = :id')
        ->execute([':id' => (int)$row['id']]);

    $user = [
        'id' => (int)$row['id'],
        'login' => (string)$row['login'],
        'firstname' => (string)($row['firstname'] ?? ''),
        'surname' => (string)($row['surname'] ?? ''),
        'lastname' => (string)($row['lastname'] ?? ''),
        'email' => $row['email'] ?? null,
        'phone' => $row['phone'] ?? null,
        'ofo' => $row['ofo'],
        'user_group' => (string)($row['user_group'] ?? ''),
        'role' => (string)($row['role'] ?? ''),
        'status' => true,
        'auth' => true,
    ];
    return $user;
}

function auth_require_user(PDO $pdo): array
{
    $u = auth_current_user($pdo);
    if (!$u) {
        http_response_code(401);
        header('Content-Type: application/json; charset=utf-8');
        echo json_encode(['success' => false, 'message' => 'Требуется авторизация', 'data' => null], JSON_UNESCAPED_UNICODE);
        exit;
    }
    return $u;
}

function auth_require_admin(PDO $pdo): array
{
    $u = auth_require_user($pdo);
    if (($u['user_group'] ?? '') !== 'admin') {
        http_response_code(403);
        header('Content-Type: application/json; charset=utf-8');
        echo json_encode(['success' => false, 'message' => 'Недостаточно прав', 'data' => null], JSON_UNESCAPED_UNICODE);
        exit;
    }
    return $u;
}

function auth_is_admin(array $user): bool
{
    return ($user['user_group'] ?? '') === 'admin';
}

function auth_ofo_unit_id(array $user): ?int
{
    $raw = $user['ofo'] ?? null;
    if ($raw === null || $raw === '' || $raw === -1 || $raw === '-1') {
        return null;
    }
    if (is_numeric($raw) && (int)$raw > 0) {
        return (int)$raw;
    }
    return null;
}
