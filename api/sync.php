<?php
/**
 * API: /api/sync.php
 *
 * POST — синхронизация пользователей из ASU (MySQL → PostgreSQL)
 * Body: { "action": "create", "user": { id, name, login, password, email, phone } }
 */

header('Content-Type: application/json; charset=utf-8');

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'message' => 'Method not allowed']);
    exit;
}

$body = json_decode(file_get_contents('php://input'), true);
$action = $body['action'] ?? null;
$user   = $body['user']   ?? null;

if ($action !== 'create' || !$user) {
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => 'Invalid request']);
    exit;
}

// Разбиваем ФИО: "Фамилия Имя Отчество"
$parts     = explode(' ', trim($user['name'] ?? ''), 3);
$surname   = $parts[0] ?? '';
$firstname = $parts[1] ?? '';
$lastname  = $parts[2] ?? '';

try {
    $pdo = new PDO(
        'pgsql:host=localhost;port=5432;dbname=corporate_portal',
        'myuser',
        'VZAIMno4753',
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );
    $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'DB connection error']);
    exit;
}

$stmt = $pdo->prepare("
    INSERT INTO public.user_info (id, status, login, password, firstname, surname, lastname, phone, email, ofo, user_group, role, auth)
    VALUES (:id, true, :login, :password, :firstname, :surname, :lastname, :phone, :email, -1, 'user', '', false)
    ON CONFLICT (id) DO NOTHING
");
$stmt->execute([
    ':id'        => (int)($user['id']),
    ':login'     => $user['login']    ?? '',
    ':password'  => $user['password'] ?? '',
    ':firstname' => $firstname,
    ':surname'   => $surname,
    ':lastname'  => $lastname,
    ':phone'     => $user['phone']    ?? '',
    ':email'     => $user['email']    ?? '',
]);

echo json_encode(['success' => true]);
