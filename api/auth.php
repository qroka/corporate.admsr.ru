<?php
/**
 * API: /api/auth.php
 *
 * POST /api/auth.php — проверка логина и пароля
 * Body: { "login": "...", "password": "..." }
 * Response 200: { "success": true,  "user": { id, fio, ofo, user_group } }
 * Response 401: { "success": false, "message": "..." }
 */

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

/**
 * Проверяет логин+пароль в ASU (MySQL).
 * Если найден — создаёт пользователя в PostgreSQL и возвращает его данные.
 * Если не найден — возвращает null.
 */
function asu_lookup_and_create(string $login, string $password, PDO $pdo): ?array
{
    $asuUrl    = 'https://172.17.30.42/asu_lookup.php';
    $asuSecret = 'asu_corporate_sync_key';

    $ch = curl_init($asuUrl);
    curl_setopt_array($ch, [
        CURLOPT_POST           => true,
        CURLOPT_POSTFIELDS     => json_encode(['login' => $login, 'password' => $password]),
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT        => 5,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_SSL_VERIFYHOST => false,
        CURLOPT_HTTPHEADER     => [
            'Content-Type: application/json',
            'Host: asu.admsr.ru',
            'X-Sync-Secret: ' . $asuSecret,
        ],
    ]);
    $response  = curl_exec($ch);
    $curlError = curl_error($ch);
    $httpCode  = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    // Лог для отладки — удалить после проверки
    file_put_contents(
        '/tmp/asu_sync_debug.log',
        date('Y-m-d H:i:s') . " login={$login} http={$httpCode} curl_err={$curlError} response={$response}" . PHP_EOL,
        FILE_APPEND
    );

    if (!$response) {
        return null;
    }

    $data = json_decode($response, true);
    if (empty($data['success']) || empty($data['user'])) {
        return null;
    }

    $u = $data['user'];

    // Разбиваем ФИО: "Фамилия Имя Отчество"
    $parts     = explode(' ', trim($u['name'] ?? ''), 3);
    $surname   = $parts[0] ?? '';
    $firstname = $parts[1] ?? '';
    $lastname  = $parts[2] ?? '';

    // Создаём пользователя в PostgreSQL
    $stmt = $pdo->prepare("
        INSERT INTO public.user_info
            (id, status, login, password, firstname, surname, lastname, phone, email, ofo, user_group, role, auth)
        VALUES
            (:id, true, :login, :password, :firstname, :surname, :lastname, :phone, :email, -1, 'user', '', false)
        ON CONFLICT (id) DO NOTHING
    ");
    $stmt->execute([
        ':id'        => (int)$u['id'],
        ':login'     => $u['login']    ?? '',
        ':password'  => $u['password'] ?? '',
        ':firstname' => $firstname,
        ':surname'   => $surname,
        ':lastname'  => $lastname,
        ':phone'     => $u['phone']    ?? '',
        ':email'     => $u['email']    ?? '',
    ]);

    // Возвращаем данные в формате, совместимом с основным потоком
    return [
        'id'         => (int)$u['id'],
        'password'   => $u['password'] ?? '',
        'firstname'  => $firstname,
        'surname'    => $surname,
        'lastname'   => $lastname,
        'ofo'        => -1,
        'user_group' => 'user',
        'status'     => true,
    ];
}

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'message' => 'Метод не поддерживается']);
    exit;
}

$body = json_decode(file_get_contents('php://input'), true);
$login    = trim($body['login']    ?? '');
$password = $body['password'] ?? '';

if ($login === '' || $password === '') {
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => 'Введите логин и пароль']);
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
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Ошибка подключения к БД']);
    exit;
}

$stmt = $pdo->prepare(
    'SELECT id, status, password, firstname, surname, lastname, ofo, user_group, role FROM public.user_info WHERE login = :login LIMIT 1'
);
$stmt->execute([':login' => $login]);
$user = $stmt->fetch();

if (!$user) {
    // Пользователь не найден в PostgreSQL — проверяем в ASU
    $user = asu_lookup_and_create($login, $password, $pdo);

    if (!$user) {
        http_response_code(401);
        echo json_encode(['success' => false, 'message' => 'Неверный логин или пароль']);
        exit;
    }
}

// Поддержка как password_hash(), так и открытого текста
$passwordOk = password_verify($password, $user['password']) || $user['password'] === $password;

if (!$passwordOk) {
    http_response_code(401);
    echo json_encode(['success' => false, 'message' => 'Неверный логин или пароль']);
    exit;
}

if (!(bool)$user['status']) {
    http_response_code(403);
    echo json_encode(['success' => false, 'message' => 'Учётная запись отключена. Обратитесь к администратору']);
    exit;
}

$pdo->prepare('UPDATE public.user_info SET auth = true WHERE id = :id')
    ->execute([':id' => $user['id']]);

$fio = implode(' ', array_filter([
    $user['surname'],
    $user['firstname'],
    $user['lastname'],
]));

echo json_encode([
    'success' => true,
    'user' => [
        'id'         => (int)$user['id'],
        'fio'        => $fio,
        'ofo'        => $user['ofo'],
        'user_group' => $user['user_group'],
        'role'       => $user['role'] ?? '',
    ],
]);