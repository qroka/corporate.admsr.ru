<?php
/**
 * API: /api/news.php
 *
 * GET    /api/news.php                    — список новостей (legacy: полный массив)
 * GET    /api/news.php?limit=8&cursor=…   — cursor-страница ленты
 *        → data: { items, nextCursor, hasMore }
 * GET    /api/news.php?id=N               — одна новость
 * POST   /api/news.php                    — создать новость
 * PUT    /api/news.php?id=N               — обновить новость
 * DELETE /api/news.php?id=N               — удалить новость
 *
 * Table: public.news
 *   id SERIAL PK, title VARCHAR(255), category VARCHAR(100),
 *   description TEXT, date DATE, image_path VARCHAR(500),
 *   created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
 */

error_reporting(0);
ini_set('display_errors', '0');

// ─── Подключение ──────────────────────────────────────────────────────────────
define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

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
    DB_USER,
    DB_PASS,
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
$action = isset($_GET['action']) ? trim(strtolower((string)$_GET['action'])) : '';

switch ($method) {

  case 'GET':
    if ($id !== null) {
      $stmt = $pdo->prepare('SELECT * FROM public.news WHERE id = :id');
      $stmt->execute([':id' => $id]);
      $row = $stmt->fetch();
      if (!$row) jsonError(404, 'Новость не найдена');
      jsonOk(fmt($row));
    } else {
      // Cursor / limit mode: ?limit=8&cursor=...
      // Legacy (no limit): full array — for admin list, kiosk, detail sidebar.
      if (isset($_GET['limit']) || array_key_exists('cursor', $_GET)) {
        jsonOk(fetchNewsCursorPage($pdo, $_GET));
      } else {
        [$sql, $params] = buildQuery($_GET);
        $stmt = $pdo->prepare($sql);
        $stmt->execute($params);
        jsonOk(array_map('fmt', $stmt->fetchAll()));
      }
    }
    break;

  case 'POST':
    if ($id !== null && !empty($action)) {
      if ($action === 'view') {
        $pdo->prepare(
          'UPDATE public.news
           SET views = COALESCE(views, 0) + 1,
               updated_at = NOW()
           WHERE id = :id'
        )->execute([':id' => $id]);

        $stmt2 = $pdo->prepare('SELECT * FROM public.news WHERE id = :id');
        $stmt2->execute([':id' => $id]);
        $row = $stmt2->fetch();
        if (!$row) jsonError(404, 'Новость не найдена');
        jsonOk(fmt($row), 'Просмотр учтён');
        break;
      }

      if ($action === 'like') {
        $d = jsonBody();
        $likedRaw = $d['liked'] ?? null;
        $liked = filter_var($likedRaw, FILTER_VALIDATE_BOOLEAN, FILTER_NULL_ON_FAILURE);
        if ($liked === null) jsonError(422, 'Поле «liked» обязательно (true/false)');

        $pdo->prepare(
          'UPDATE public.news
           SET likes = CASE
             WHEN :liked = 1 THEN COALESCE(likes, 0) + 1
             ELSE GREATEST(COALESCE(likes, 0) - 1, 0)
           END,
           updated_at = NOW()
           WHERE id = :id'
        )->execute([
          ':id'    => $id,
          ':liked' => $liked ? 1 : 0,
        ]);

        $stmt2 = $pdo->prepare('SELECT * FROM public.news WHERE id = :id');
        $stmt2->execute([':id' => $id]);
        $row = $stmt2->fetch();
        if (!$row) jsonError(404, 'Новость не найдена');
        jsonOk(fmt($row), 'Лайк обновлён');
        break;
      }

      jsonError(400, 'Неизвестное действие');
    }

    auth_require_section($pdo, 'news');
    $d = jsonBody();
    required($d, ['title', 'date']);

    $stmt = $pdo->prepare(
      "INSERT INTO public.news (title, category, description, date, image_path)
       VALUES (:title, :category, :description, :date, :image_path)
       RETURNING id"
    );
    $stmt->execute([
      ':title'       => trim($d['title']),
      ':category'    => isset($d['category'])    ? trim($d['category'])    : '',
      ':description' => isset($d['description']) ? trim($d['description']) : '',
      ':date'        => $d['date'],
      ':image_path'  => !empty($d['image_path']) ? trim($d['image_path'])  : null,
    ]);

    $newId = (int)$stmt->fetchColumn();
    $stmt2 = $pdo->prepare('SELECT * FROM public.news WHERE id = :id');
    $stmt2->execute([':id' => $newId]);
    http_response_code(201);
    jsonOk(fmt($stmt2->fetch()), 'Новость создана');
    break;

  case 'PUT':
    auth_require_section($pdo, 'news');
    if (!$id) jsonError(400, 'Укажите ?id=...');
    $d = jsonBody();
    required($d, ['title', 'date']);

    $check = $pdo->prepare('SELECT id FROM public.news WHERE id = :id');
    $check->execute([':id' => $id]);
    if (!$check->fetch()) jsonError(404, 'Новость не найдена');

    $stmt = $pdo->prepare(
      "UPDATE public.news
       SET title = :title, category = :category, description = :description,
           date = :date, image_path = :image_path, updated_at = NOW()
       WHERE id = :id"
    );
    $stmt->execute([
      ':title'       => trim($d['title']),
      ':category'    => isset($d['category'])    ? trim($d['category'])    : '',
      ':description' => isset($d['description']) ? trim($d['description']) : '',
      ':date'        => $d['date'],
      ':image_path'  => !empty($d['image_path']) ? trim($d['image_path'])  : null,
      ':id'          => $id,
    ]);

    $stmt2 = $pdo->prepare('SELECT * FROM public.news WHERE id = :id');
    $stmt2->execute([':id' => $id]);
    jsonOk(fmt($stmt2->fetch()), 'Новость обновлена');
    break;

  case 'DELETE':
    auth_require_section($pdo, 'news');
    if (!$id) jsonError(400, 'Укажите ?id=...');
    $check = $pdo->prepare('SELECT id FROM public.news WHERE id = :id');
    $check->execute([':id' => $id]);
    if (!$check->fetch()) jsonError(404, 'Новость не найдена');
    $pdo->prepare('DELETE FROM public.news WHERE id = :id')->execute([':id' => $id]);
    jsonOk(null, 'Новость удалена');
    break;

  default:
    jsonError(405, 'Метод не поддерживается');
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Cursor page for social-style feed.
 * Sort: date DESC, id DESC. Cursor = last item (date + id).
 *
 * Response data shape:
 *   { "items": [...], "nextCursor": "..."|null, "hasMore": bool }
 */
function fetchNewsCursorPage(PDO $pdo, array $get): array
{
  $limit = isset($get['limit']) ? (int)$get['limit'] : 8;
  if ($limit < 1) $limit = 8;
  if ($limit > 30) $limit = 30;

  $cond   = [];
  $params = [];

  if (!empty($get['search'])) {
    $cond[]            = '(title ILIKE :search OR description ILIKE :search OR category ILIKE :search)';
    $params[':search'] = '%' . trim((string)$get['search']) . '%';
  }

  if (!empty($get['category'])) {
    $cond[]              = 'category ILIKE :category';
    $params[':category'] = '%' . trim((string)$get['category']) . '%';
  }

  $cursorRaw = isset($get['cursor']) ? trim((string)$get['cursor']) : '';
  if ($cursorRaw !== '') {
    $cursor = decodeNewsCursor($cursorRaw);
    if ($cursor === null) {
      jsonError(400, 'Некорректный cursor');
    }
    // Next page for ORDER BY date DESC, id DESC
    $cond[]                 = '(date < :cursor_date OR (date = :cursor_date AND id < :cursor_id))';
    $params[':cursor_date'] = $cursor['date'];
    $params[':cursor_id']   = $cursor['id'];
  }

  $where = $cond ? (' WHERE ' . implode(' AND ', $cond)) : '';
  // Fetch one extra row to know if there is another page
  $fetchLimit = $limit + 1;

  $sql =
    "SELECT * FROM public.news" .
    $where .
    " ORDER BY date DESC, id DESC" .
    " LIMIT :fetch_limit";

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
    $nextCursor = encodeNewsCursor((string)($last['date'] ?? ''), (int)$last['id']);
  }

  return [
    'items'      => $items,
    'nextCursor' => $nextCursor,
    'hasMore'    => $hasMore,
  ];
}

function encodeNewsCursor(string $date, int $id): string
{
  $payload = json_encode(['d' => $date, 'i' => $id], JSON_UNESCAPED_UNICODE);
  return rtrim(strtr(base64_encode($payload !== false ? $payload : ''), '+/', '-_'), '=');
}

/** @return array{date: string, id: int}|null */
function decodeNewsCursor(string $raw): ?array
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
  $id   = isset($data['i']) ? (int)$data['i'] : 0;

  // YYYY-MM-DD
  if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $date)) return null;
  if ($id < 1) return null;

  return ['date' => $date, 'id' => $id];
}

function buildQuery(array $get): array
{
  $cond   = [];
  $params = [];

  if (!empty($get['search'])) {
    $cond[]            = '(title ILIKE :search OR description ILIKE :search OR category ILIKE :search)';
    $params[':search'] = '%' . trim($get['search']) . '%';
  }

  if (!empty($get['category'])) {
    $cond[]              = 'category ILIKE :category';
    $params[':category'] = '%' . trim((string)$get['category']) . '%';
  }

  $allowed = ['date', 'created_at', 'id', 'title'];
  $order   = in_array($get['order'] ?? '', $allowed, true) ? $get['order'] : 'date';
  $dir     = strtolower($get['dir'] ?? '') === 'asc' ? 'ASC' : 'DESC';

  $sql =
    "SELECT * FROM public.news" .
    ($cond ? ' WHERE ' . implode(' AND ', $cond) : '') .
    " ORDER BY $order $dir, id DESC";

  return [$sql, $params];
}

function fmt(array $r): array
{
  return [
    'id'          => (int)$r['id'],
    'title'       => $r['title']       ?? '',
    'category'    => $r['category']    ?? '',
    'description' => $r['description'] ?? '',
    'date'        => $r['date']        ?? null,
    'image_path'  => $r['image_path']  ?? null,
    'created_at'  => $r['created_at']  ?? null,
    'updated_at'  => $r['updated_at']  ?? null,
    'likes'       => $r['likes']       ?? 0,
    'views'       => $r['views']       ?? 0,
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
    if (!isset($data[$f]) || trim((string)$data[$f]) === '') {
      jsonError(422, 'Поле `' . $f . '` обязательно');
    }
  }
}

function jsonOk($data, string $msg = 'OK'): never
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