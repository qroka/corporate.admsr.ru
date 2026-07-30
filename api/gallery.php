<?php
/**
 * API: /api/gallery.php
 *
 * GET    /api/gallery.php          — список альбомов (+ обложка из первого фото)
 * GET    /api/gallery.php?id=N     — один альбом
 * POST   /api/gallery.php          — создать { name, description?, date? }
 * PUT    /api/gallery.php?id=N     — обновить { name?, description?, date? }
 * DELETE /api/gallery.php?id=N     — удалить альбом + все фото из gallery_base
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
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

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
    jsonError(500, 'Ошибка подключения к БД');
}

require_once __DIR__ . '/auth_context.php';

$method = $_SERVER['REQUEST_METHOD'];
$id     = isset($_GET['id']) ? (int)$_GET['id'] : null;

switch ($method) {

    case 'GET':
        if ($id !== null) {
            $stmt = $pdo->prepare(
                'SELECT id, name, description, date FROM public.gallery WHERE id = :id'
            );
            $stmt->execute([':id' => $id]);
            $row = $stmt->fetch();
            if (!$row) jsonError(404, 'Альбом не найден');
            jsonOk(fmtAlbum($row));
        } else {
            $stmt = $pdo->query(
                "SELECT g.id, g.name, g.description, g.date,
                        (SELECT image_small_url FROM public.gallery_base
                         WHERE album_id = g.id ORDER BY id ASC LIMIT 1) AS cover
                 FROM public.gallery g
                 ORDER BY g.date DESC, g.id DESC"
            );
            jsonOk(array_map('fmtAlbum', $stmt->fetchAll()));
        }
        break;

    case 'POST':
        auth_require_section($pdo, 'gallery');
        $d = jsonBody();
        if (empty($d['name'])) jsonError(422, 'Поле «name» обязательно');

        $stmt = $pdo->prepare(
            'INSERT INTO public.gallery (name, description, date)
             VALUES (:name, :description, :date)
             RETURNING id'
        );
        $stmt->execute([
            ':name'        => trim($d['name']),
            ':description' => !empty($d['description']) ? trim($d['description']) : null,
            ':date'        => !empty($d['date']) ? $d['date'] : null,
        ]);
        $newId = (int)$stmt->fetchColumn();
        $stmt2 = $pdo->prepare('SELECT id, name, description, date FROM public.gallery WHERE id = :id');
        $stmt2->execute([':id' => $newId]);
        http_response_code(201);
        jsonOk(fmtAlbum($stmt2->fetch()), 'Альбом создан');
        break;

    case 'PUT':
        auth_require_section($pdo, 'gallery');
        if (!$id) jsonError(400, 'Укажите ?id=...');
        $d = jsonBody();

        $check = $pdo->prepare('SELECT id FROM public.gallery WHERE id = :id');
        $check->execute([':id' => $id]);
        if (!$check->fetch()) jsonError(404, 'Альбом не найден');

        $sets   = [];
        $params = [':id' => $id];
        if (!empty($d['name']))               { $sets[] = 'name = :name';               $params[':name']        = trim($d['name']); }
        if (array_key_exists('description', $d)) { $sets[] = 'description = :description'; $params[':description'] = $d['description']; }
        if (!empty($d['date']))               { $sets[] = 'date = :date';               $params[':date']        = $d['date']; }

        if ($sets) {
            $pdo->prepare('UPDATE public.gallery SET ' . implode(', ', $sets) . ' WHERE id = :id')
                ->execute($params);
        }

        $stmt2 = $pdo->prepare('SELECT id, name, description, date FROM public.gallery WHERE id = :id');
        $stmt2->execute([':id' => $id]);
        jsonOk(fmtAlbum($stmt2->fetch()), 'Альбом обновлён');
        break;

    case 'DELETE':
        auth_require_section($pdo, 'gallery');
        if (!$id) jsonError(400, 'Укажите ?id=...');
        $check = $pdo->prepare('SELECT id FROM public.gallery WHERE id = :id');
        $check->execute([':id' => $id]);
        if (!$check->fetch()) jsonError(404, 'Альбом не найден');

        $pdo->prepare('DELETE FROM public.gallery_base WHERE album_id = :id')->execute([':id' => $id]);
        $pdo->prepare('DELETE FROM public.gallery WHERE id = :id')->execute([':id' => $id]);
        jsonOk(null, 'Альбом удалён');
        break;

    default:
        jsonError(405, 'Метод не поддерживается');
}

function fmtAlbum(array $r): array
{
    return [
        'id'          => (int)$r['id'],
        'name'        => $r['name'] ?? '',
        'description' => $r['description'] ?? '',
        'date'        => isset($r['date']) ? substr($r['date'], 0, 10) : '',
        'cover'       => $r['cover'] ?? null,
    ];
}

function jsonBody(): array
{
    $raw = file_get_contents('php://input');
    if (!$raw) jsonError(400, 'Тело запроса пустое');
    $data = json_decode($raw, true);
    if (!is_array($data)) jsonError(400, 'Некорректный JSON');
    return $data;
}

function jsonOk(mixed $data, string $msg = 'OK'): never
{
    echo json_encode(['success' => true, 'message' => $msg, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function jsonError(int $code, string $msg): never
{
    http_response_code($code);
    echo json_encode(['success' => false, 'message' => $msg, 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}