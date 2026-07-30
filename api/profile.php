<?php
/**
 * API: /api/profile.php
 *
 * GET  ?id=N      — получить данные профиля
 * POST { id, firstname, surname, lastname, phone, email, ofo, role, avatar_url }
 *                 — сохранить данные профиля
 */

error_reporting(0);
ini_set('display_errors', '0');

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk(mixed $data): never
{
    echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function jsonError(int $code, string $msg): never
{
    http_response_code($code);
    echo json_encode(['success' => false, 'message' => $msg], JSON_UNESCAPED_UNICODE);
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
} catch (PDOException) {
    jsonError(500, 'Ошибка подключения к БД');
}

// --- GET: загрузить профиль ---
if ($_SERVER['REQUEST_METHOD'] === 'GET') {
    $id = isset($_GET['id']) ? (int)$_GET['id'] : 0;
    if ($id <= 0) jsonError(400, 'Некорректный id');

    $stmt = $pdo->prepare(
        'SELECT id, firstname, surname, lastname, ofo, user_group, phone, email, role, avatar_url
         FROM public.user_info WHERE id = :id LIMIT 1'
    );
    $stmt->execute([':id' => $id]);
    $row = $stmt->fetch();

    if (!$row) jsonError(404, 'Пользователь не найден');

    $isAdmin = (($row['user_group'] ?? '') === 'admin');
    $sections = [];
    try {
        require_once __DIR__ . '/auth_context.php';
        $sections = auth_user_sections($pdo, [
            'id' => (int)$row['id'],
            'user_group' => $row['user_group'] ?? '',
        ]);
    } catch (Throwable $e) {
        $sections = $isAdmin
            ? ['news', 'events', 'gallery', 'courses', 'tests', 'absence_journal', 'birthdays']
            : [];
    }

    jsonOk([
        'id'         => (int)$row['id'],
        'firstname'  => $row['firstname']  ?? '',
        'surname'    => $row['surname']    ?? '',
        'lastname'   => $row['lastname']   ?? '',
        'ofo'        => $row['ofo']        ?? '',
        'user_group' => $row['user_group'] ?? '',
        'phone'      => $row['phone']      ?? '',
        'email'      => $row['email']      ?? '',
        'role'       => $row['role']       ?? '',
        'avatar_url' => $row['avatar_url'] ?? '',
        'isAdmin'    => $isAdmin,
        'sections'   => $sections,
    ]);
}

// --- POST: сохранить профиль ---
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $body = json_decode(file_get_contents('php://input'), true);

    $id         = isset($body['id'])         ? (int)$body['id']               : 0;
    $firstname  = isset($body['firstname'])  ? trim((string)$body['firstname'])  : null;
    $surname    = isset($body['surname'])    ? trim((string)$body['surname'])    : null;
    $lastname   = isset($body['lastname'])   ? trim((string)$body['lastname'])   : null;
    $phone      = isset($body['phone'])      ? trim((string)$body['phone'])      : null;
    $email      = isset($body['email'])      ? trim((string)$body['email'])      : null;
    $ofo        = isset($body['ofo'])        ? trim((string)$body['ofo'])        : null;
    $role       = isset($body['role'])       ? trim((string)$body['role'])       : null;
    $avatar_url = isset($body['avatar_url']) ? trim((string)$body['avatar_url']) : null;

    if ($id <= 0) jsonError(400, 'Некорректный id');

    $stmt = $pdo->prepare(
        'UPDATE public.user_info
         SET firstname  = :firstname,
             surname    = :surname,
             lastname   = :lastname,
             phone      = :phone,
             email      = :email,
             ofo        = :ofo,
             role       = :role,
             avatar_url = :avatar_url
         WHERE id = :id'
    );
    $stmt->execute([
        ':id'         => $id,
        ':firstname'  => $firstname,
        ':surname'    => $surname,
        ':lastname'   => $lastname,
        ':phone'      => $phone,
        ':email'      => $email,
        ':ofo'        => $ofo,
        ':role'       => $role,
        ':avatar_url' => $avatar_url,
    ]);

    jsonOk(null);
}

jsonError(405, 'Метод не поддерживается');