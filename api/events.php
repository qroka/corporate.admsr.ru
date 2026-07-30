<?php
/**
 * API: /api/events.php
 *
 * GET    /api/events.php          — список всех мероприятий
 * GET    /api/events.php?id=1     — одно мероприятие по ID
 * POST   /api/events.php          — создать мероприятие
 * PUT    /api/events.php?id=1     — обновить мероприятие
 * DELETE /api/events.php?id=1     — удалить мероприятие
 *
 * SQL для создания таблицы:
 * -------------------------------------------------------
 * CREATE TABLE events (
 *   id          SERIAL        PRIMARY KEY,
 *   title       VARCHAR(255)  NOT NULL,
 *   description TEXT,
 *   badge       VARCHAR(100),
 *   date        DATE          NOT NULL,
 *   image       VARCHAR(500)  NOT NULL DEFAULT '/favicon.svg',
 *   image_full  VARCHAR(500)  NOT NULL DEFAULT '/favicon.svg',
 *   created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
 *   updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
 * );
 * -------------------------------------------------------
 */

// ─── Подключение ──────────────────────────────────────────────────────────────
define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');   // ← укажите ваш пароль PostgreSQL

// ─── Заголовки ────────────────────────────────────────────────────────────────
header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

// ─── PDO ──────────────────────────────────────────────────────────────────────
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
    jsonError(500, 'Ошибка подключения к БД: ' . $e->getMessage());
}

require_once __DIR__ . '/auth_context.php';

$method = $_SERVER['REQUEST_METHOD'];
$id     = isset($_GET['id']) ? (int) $_GET['id'] : null;

switch ($method) {

    // ── GET ───────────────────────────────────────────────────────────────────
    case 'GET':
        if ($id !== null) {
            $stmt = $pdo->prepare('SELECT * FROM events WHERE id = ?');
            $stmt->execute([$id]);
            $row = $stmt->fetch();
            if (!$row) jsonError(404, 'Мероприятие не найдено');
            jsonOk(fmt($row));
        } else {
            [$sql, $params] = buildQuery($_GET);
            $stmt = $pdo->prepare($sql);
            $stmt->execute($params);
            jsonOk(array_map('fmt', $stmt->fetchAll()));
        }
        break;

    // ── POST ──────────────────────────────────────────────────────────────────
    case 'POST':
        auth_require_section($pdo, 'events');
        $d = jsonBody();
        required($d, ['title', 'date']);

        $stmt = $pdo->prepare(
            'INSERT INTO events (title, description, badge, date, image, image_full)
             VALUES (:title, :description, :badge, :date, :image, :image_full)
             RETURNING id'
        );
        $def = '/favicon.svg';
        $stmt->execute([
            ':title'       => trim($d['title']),
            ':description' => isset($d['description']) ? trim($d['description']) : null,
            ':badge'       => isset($d['badge'])        ? trim($d['badge'])        : null,
            ':date'        => $d['date'],
            ':image'       => !empty($d['image'])       ? trim($d['image'])       : $def,
            ':image_full'  => !empty($d['image_full'])  ? trim($d['image_full'])  : (!empty($d['image']) ? trim($d['image']) : $def),
        ]);

        $newId = (int) $stmt->fetchColumn();
        $stmt2 = $pdo->prepare('SELECT * FROM events WHERE id = ?');
        $stmt2->execute([$newId]);
        http_response_code(201);
        jsonOk(fmt($stmt2->fetch()), 'Мероприятие создано');
        break;

    // ── PUT ───────────────────────────────────────────────────────────────────
    case 'PUT':
        auth_require_section($pdo, 'events');
        if (!$id) jsonError(400, 'Укажите ?id=...');
        $d = jsonBody();
        required($d, ['title', 'date']);

        $cur = $pdo->prepare('SELECT image, image_full FROM events WHERE id = ?');
        $cur->execute([$id]);
        $curRow = $cur->fetch();
        if (!$curRow) jsonError(404, 'Мероприятие не найдено');

        $stmt = $pdo->prepare(
            'UPDATE events
             SET title=:title, description=:description, badge=:badge,
                 date=:date, image=:image, image_full=:image_full
             WHERE id=:id'
        );
        $stmt->execute([
            ':title'       => trim($d['title']),
            ':description' => isset($d['description']) ? trim($d['description']) : null,
            ':badge'       => isset($d['badge'])        ? trim($d['badge'])        : null,
            ':date'        => $d['date'],
            ':image'       => !empty($d['image'])      ? trim($d['image'])      : $curRow['image'],
            ':image_full'  => !empty($d['image_full']) ? trim($d['image_full']) : $curRow['image_full'],
            ':id'          => $id,
        ]);

        $stmt2 = $pdo->prepare('SELECT * FROM events WHERE id = ?');
        $stmt2->execute([$id]);
        jsonOk(fmt($stmt2->fetch()), 'Мероприятие обновлено');
        break;

    // ── DELETE ────────────────────────────────────────────────────────────────
    case 'DELETE':
        auth_require_section($pdo, 'events');
        if (!$id) jsonError(400, 'Укажите ?id=...');
        $check = $pdo->prepare('SELECT id FROM events WHERE id = ?');
        $check->execute([$id]);
        if (!$check->fetch()) jsonError(404, 'Мероприятие не найдено');
        $pdo->prepare('DELETE FROM events WHERE id = ?')->execute([$id]);
        jsonOk(null, 'Мероприятие удалено');
        break;

    default:
        jsonError(405, 'Метод не поддерживается');
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function buildQuery(array $get): array
{
    $cond   = [];
    $params = [];

    if (!empty($get['search'])) {
        $cond[] = '(title ILIKE :search OR description ILIKE :search)';
        $params[':search'] = '%' . trim($get['search']) . '%';
    }

    if (!empty($get['badge'])) {
        $badges = array_filter(array_map('trim', explode(',', $get['badge'])));
        if ($badges) {
            $ph = [];
            foreach ($badges as $i => $b) { $ph[] = ":b$i"; $params[":b$i"] = $b; }
            $cond[] = 'badge IN (' . implode(',', $ph) . ')';
        }
    }

    if (!empty($get['date_from'])) { $cond[] = 'date >= :date_from'; $params[':date_from'] = $get['date_from']; }
    if (!empty($get['date_to']))   { $cond[] = 'date <= :date_to';   $params[':date_to']   = $get['date_to'];   }

    $allowed = ['date', 'title', 'created_at', 'badge'];
    $order   = in_array($get['order'] ?? '', $allowed, true) ? $get['order'] : 'date';
    $dir     = strtolower($get['dir'] ?? '') === 'desc' ? 'DESC' : 'ASC';

    $sql = 'SELECT * FROM events' . ($cond ? ' WHERE ' . implode(' AND ', $cond) : '') . " ORDER BY $order $dir";
    return [$sql, $params];
}

function fmt(array $r): array
{
    return [
        'id'          => (int) $r['id'],
        'title'       => $r['title'],
        'description' => $r['description'] ?? '',
        'badge'       => $r['badge'] ?? null,
        'date'        => $r['date'],
        'image'       => $r['image'],
        'image_full'  => $r['image_full'],
        'created_at'  => $r['created_at'],
        'updated_at'  => $r['updated_at'],
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

function required(array $data, array $fields): void
{
    foreach ($fields as $f) {
        if (empty($data[$f])) jsonError(422, "Поле «$f» обязательно");
    }
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