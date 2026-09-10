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
 *   album_id    INTEGER       NULL REFERENCES public.gallery(id) ON DELETE SET NULL,
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

try {
switch ($method) {

    // ── GET ───────────────────────────────────────────────────────────────────
    case 'GET':
        if ($id !== null) {
            $row = fetchEventRow($pdo, $id);
            if (!$row) jsonError(404, 'Мероприятие не найдено');
            jsonOk(fmt($row));
        } elseif (isset($_GET['limit']) || array_key_exists('cursor', $_GET)) {
            jsonOk(fetchEventsCursorPage($pdo, $_GET));
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

        $def = '/favicon.svg';
        $albumId = parseAlbumId($pdo, $d, true);
        $stmt = $pdo->prepare(
            'INSERT INTO events (title, description, badge, date, image, image_full, album_id)
             VALUES (:title, :description, :badge, :date, :image, :image_full, :album_id)
             RETURNING id'
        );
        $stmt->execute([
            ':title'       => trim($d['title']),
            ':description' => isset($d['description']) ? trim($d['description']) : null,
            ':badge'       => isset($d['badge'])        ? trim($d['badge'])        : null,
            ':date'        => $d['date'],
            ':image'       => !empty($d['image'])       ? trim($d['image'])       : $def,
            ':image_full'  => !empty($d['image_full'])  ? trim($d['image_full'])  : (!empty($d['image']) ? trim($d['image']) : $def),
            ':album_id'    => $albumId,
        ]);

        $newId = (int) $stmt->fetchColumn();
        http_response_code(201);
        jsonOk(fmt(fetchEventRow($pdo, $newId)), 'Мероприятие создано');
        break;

    // ── PUT ───────────────────────────────────────────────────────────────────
    case 'PUT':
        auth_require_section($pdo, 'events');
        if (!$id) jsonError(400, 'Укажите ?id=...');
        $d = jsonBody();
        required($d, ['title', 'date']);

        $cur = $pdo->prepare('SELECT image, image_full, album_id FROM events WHERE id = ?');
        $cur->execute([$id]);
        $curRow = $cur->fetch();
        if (!$curRow) jsonError(404, 'Мероприятие не найдено');

        $albumId = array_key_exists('album_id', $d)
            ? parseAlbumId($pdo, $d, true)
            : (isset($curRow['album_id']) && $curRow['album_id'] !== null ? (int) $curRow['album_id'] : null);

        $stmt = $pdo->prepare(
            'UPDATE events
             SET title=:title, description=:description, badge=:badge,
                 date=:date, image=:image, image_full=:image_full, album_id=:album_id
             WHERE id=:id'
        );
        $stmt->execute([
            ':title'       => trim($d['title']),
            ':description' => isset($d['description']) ? trim($d['description']) : null,
            ':badge'       => isset($d['badge'])        ? trim($d['badge'])        : null,
            ':date'        => $d['date'],
            ':image'       => !empty($d['image'])      ? trim($d['image'])      : $curRow['image'],
            ':image_full'  => !empty($d['image_full']) ? trim($d['image_full']) : $curRow['image_full'],
            ':album_id'    => $albumId,
            ':id'          => $id,
        ]);

        jsonOk(fmt(fetchEventRow($pdo, $id)), 'Мероприятие обновлено');
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
} catch (Throwable $e) {
    $msg = $e->getMessage();
    // Частая причина: не применена миграция V7 (колонка album_id)
    if (stripos($msg, 'album_id') !== false) {
        jsonError(500, 'В БД нет колонки events.album_id. Примените миграцию db/migration/V7__events_gallery_album.sql');
    }
    jsonError(500, 'Ошибка БД: ' . $msg);
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Одна запись мероприятия + имя альбома (если колонка/таблица доступны). */
function fetchEventRow(PDO $pdo, int $id): ?array
{
    // Сначала без JOIN — страница детали работает даже до миграции V7
    $stmt = $pdo->prepare('SELECT * FROM events WHERE id = ?');
    $stmt->execute([$id]);
    $row = $stmt->fetch();
    if (!$row) return null;

    $row['album_name'] = null;
    if (!empty($row['album_id'])) {
        try {
            $g = $pdo->prepare('SELECT name FROM public.gallery WHERE id = ?');
            $g->execute([(int) $row['album_id']]);
            $name = $g->fetchColumn();
            $row['album_name'] = $name !== false ? $name : null;
        } catch (Throwable $e) {
            $row['album_name'] = null;
        }
    }
    return $row;
}

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

/**
 * Cursor page for events list.
 * Sort: date DESC, id DESC. Cursor = last item (date + id).
 */
function fetchEventsCursorPage(PDO $pdo, array $get): array
{
    $limit = isset($get['limit']) ? (int)$get['limit'] : 12;
    if ($limit < 1) $limit = 12;
    if ($limit > 48) $limit = 48;

    $cond   = [];
    $params = [];

    if (!empty($get['search'])) {
        $cond[] = '(title ILIKE :search OR description ILIKE :search)';
        $params[':search'] = '%' . trim((string)$get['search']) . '%';
    }

    if (!empty($get['badge']) && $get['badge'] !== '_all') {
        $badges = array_filter(array_map('trim', explode(',', (string)$get['badge'])));
        if ($badges) {
            $ph = [];
            foreach ($badges as $i => $b) {
                $ph[] = ":b$i";
                $params[":b$i"] = $b;
            }
            $cond[] = 'badge IN (' . implode(',', $ph) . ')';
        }
    }

    $cursorRaw = isset($get['cursor']) ? trim((string)$get['cursor']) : '';
    if ($cursorRaw !== '') {
        $cursor = decodeDateIdCursor($cursorRaw);
        if ($cursor === null) {
            jsonError(400, 'Некорректный cursor');
        }
        $cond[] = '(date < :cursor_date OR (date = :cursor_date AND id < :cursor_id))';
        $params[':cursor_date'] = $cursor['date'];
        $params[':cursor_id'] = $cursor['id'];
    }

    $where = $cond ? (' WHERE ' . implode(' AND ', $cond)) : '';
    $fetchLimit = $limit + 1;

    $sql =
        'SELECT * FROM events' .
        $where .
        ' ORDER BY date DESC, id DESC' .
        ' LIMIT :fetch_limit';

    $stmt = $pdo->prepare($sql);
    foreach ($params as $key => $value) {
        if ($key === ':cursor_id') {
            $stmt->bindValue($key, (int)$value, PDO::PARAM_INT);
        } else {
            $stmt->bindValue($key, $value);
        }
    }
    $stmt->bindValue(':fetch_limit', $fetchLimit, PDO::PARAM_INT);
    $stmt->execute();
    $rows = $stmt->fetchAll();

    $hasMore = count($rows) > $limit;
    if ($hasMore) {
        $rows = array_slice($rows, 0, $limit);
    }

    $items = array_map('fmt', $rows);
    $nextCursor = null;
    if ($hasMore && count($rows) > 0) {
        $last = $rows[count($rows) - 1];
        $nextCursor = encodeDateIdCursor((string)($last['date'] ?? ''), (int)$last['id']);
    }

    return [
        'items' => $items,
        'nextCursor' => $nextCursor,
        'hasMore' => $hasMore,
    ];
}

function encodeDateIdCursor(string $date, int $id): string
{
    $payload = json_encode(['d' => $date, 'i' => $id], JSON_UNESCAPED_UNICODE);
    return rtrim(strtr(base64_encode($payload !== false ? $payload : ''), '+/', '-_'), '=');
}

/** @return array{date: string, id: int}|null */
function decodeDateIdCursor(string $raw): ?array
{
    $b64 = strtr($raw, '-_', '+/');
    $pad = strlen($b64) % 4;
    if ($pad > 0) {
        $b64 .= str_repeat('=', 4 - $pad);
    }
    $json = base64_decode($b64, true);
    if ($json === false || $json === '') return null;

    $data = json_decode($json, true);
    if (!is_array($data)) return null;

    $date = isset($data['d']) ? trim((string)$data['d']) : '';
    $id = isset($data['i']) ? (int)$data['i'] : 0;

    if (!preg_match('/^\d{4}-\d{2}-\d{2}/', $date)) return null;
    $date = substr($date, 0, 10);
    if ($id < 1) return null;

    return ['date' => $date, 'id' => $id];
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
        'album_id'    => isset($r['album_id']) && $r['album_id'] !== null ? (int) $r['album_id'] : null,
        'album_name'  => $r['album_name'] ?? null,
        'created_at'  => $r['created_at'],
        'updated_at'  => $r['updated_at'],
    ];
}

/** @param bool $allowNull true — явный null сбрасывает альбом */
function parseAlbumId(PDO $pdo, array $d, bool $allowNull): ?int
{
    if (!array_key_exists('album_id', $d)) {
        return null;
    }
    if ($d['album_id'] === null || $d['album_id'] === '' || $d['album_id'] === false) {
        return $allowNull ? null : null;
    }
    $id = (int) $d['album_id'];
    if ($id <= 0) {
        return null;
    }
    $check = $pdo->prepare('SELECT id FROM public.gallery WHERE id = ?');
    $check->execute([$id]);
    if (!$check->fetch()) {
        jsonError(422, 'Альбом не найден');
    }
    return $id;
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