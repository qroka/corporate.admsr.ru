<?php
/**
 * API: /api/news.php
 *
 * GET    /api/news.php              — список новостей
 * GET    /api/news.php?id=123       — одна новость по ID
 * POST   /api/news.php              — создать новость
 * PUT    /api/news.php?id=123       — обновить новость
 * DELETE /api/news.php?id=123       — удалить новость
 *
 * Примечание по времени:
 * - В ответе поле `timestamp` возвращается как UNIX секунды (int).
 * - В БД хранится `published_at` как TIMESTAMPTZ.
 *
 * SQL для создания таблицы (PostgreSQL):
 * -------------------------------------------------------
 * CREATE TABLE public.news (
 *   id                 SERIAL        PRIMARY KEY,
 *   title              VARCHAR(255)  NOT NULL,
 *   html               TEXT          NOT NULL DEFAULT '',
 *   short_html         TEXT          NOT NULL DEFAULT '',
 *   author_id          INTEGER       NULL,
 *   announce_image_path VARCHAR(500) NULL,
 *   likes              INTEGER       NOT NULL DEFAULT 0,
 *   views              INTEGER       NOT NULL DEFAULT 0,
 *   published_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
 *   created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
 *   updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
 * );
 *
 * CREATE INDEX news_published_at_idx ON public.news (published_at DESC);
 * -------------------------------------------------------
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
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

// ─── PDO ──────────────────────────────────────────────────────────────────────
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
  jsonError(500, 'Ошибка подключения к БД');
}

$method = $_SERVER['REQUEST_METHOD'];
$id = isset($_GET['id']) ? (int)$_GET['id'] : null;
$incView = isset($_GET['inc_view']) && (string)$_GET['inc_view'] !== '0';

switch ($method) {

  case 'GET':
    if ($id !== null) {
      if ($incView) {
        $pdo->prepare('UPDATE public.news SET views = views + 1, updated_at = NOW() WHERE id = :id')
          ->execute([':id' => $id]);
      }
      $stmt = $pdo->prepare(
        "SELECT id, title, html, short_html, author_id, announce_image_path, likes, views,
                EXTRACT(EPOCH FROM published_at)::bigint AS timestamp,
                created_at, updated_at
         FROM public.news
         WHERE id = :id"
      );
      $stmt->execute([':id' => $id]);
      $row = $stmt->fetch();
      if (!$row) jsonError(404, 'Новость не найдена');
      jsonOk(fmtNews($row));
    } else {
      [$sql, $params] = buildQuery($_GET);
      $stmt = $pdo->prepare($sql);
      $stmt->execute($params);
      jsonOk(array_map('fmtNews', $stmt->fetchAll()));
    }
    break;

  case 'POST':
    $d = jsonBody();
    required($d, ['title']);

    $publishedAt = parsePublishedAt($d);
    $stmt = $pdo->prepare(
      "INSERT INTO public.news (title, html, short_html, author_id, announce_image_path, likes, views, published_at)
       VALUES (:title, :html, :short_html, :author_id, :announce_image_path, :likes, :views, :published_at)
       RETURNING id"
    );
    $stmt->execute([
      ':title' => trim($d['title']),
      ':html' => isset($d['html']) ? (string)$d['html'] : '',
      ':short_html' => isset($d['short_html']) ? (string)$d['short_html'] : '',
      ':author_id' => isset($d['author_id']) && (int)$d['author_id'] > 0 ? (int)$d['author_id'] : null,
      ':announce_image_path' => !empty($d['announce_image_path']) ? trim((string)$d['announce_image_path']) : null,
      ':likes' => isset($d['likes']) ? max(0, (int)$d['likes']) : 0,
      ':views' => isset($d['views']) ? max(0, (int)$d['views']) : 0,
      ':published_at' => $publishedAt,
    ]);

    $newId = (int)$stmt->fetchColumn();
    $stmt2 = $pdo->prepare(
      "SELECT id, title, html, short_html, author_id, announce_image_path, likes, views,
              EXTRACT(EPOCH FROM published_at)::bigint AS timestamp,
              created_at, updated_at
       FROM public.news
       WHERE id = :id"
    );
    $stmt2->execute([':id' => $newId]);
    http_response_code(201);
    jsonOk(fmtNews($stmt2->fetch()), 'Новость создана');
    break;

  case 'PUT':
    if (!$id) jsonError(400, 'Укажите ?id=...');
    $d = jsonBody();
    required($d, ['title']);

    $check = $pdo->prepare('SELECT id FROM public.news WHERE id = :id');
    $check->execute([':id' => $id]);
    if (!$check->fetch()) jsonError(404, 'Новость не найдена');

    $publishedAt = parsePublishedAt($d);
    $stmt = $pdo->prepare(
      "UPDATE public.news
       SET title = :title,
           html = :html,
           short_html = :short_html,
           author_id = :author_id,
           announce_image_path = :announce_image_path,
           likes = :likes,
           views = :views,
           published_at = :published_at,
           updated_at = NOW()
       WHERE id = :id"
    );
    $stmt->execute([
      ':title' => trim($d['title']),
      ':html' => isset($d['html']) ? (string)$d['html'] : '',
      ':short_html' => isset($d['short_html']) ? (string)$d['short_html'] : '',
      ':author_id' => isset($d['author_id']) && (int)$d['author_id'] > 0 ? (int)$d['author_id'] : null,
      ':announce_image_path' => !empty($d['announce_image_path']) ? trim((string)$d['announce_image_path']) : null,
      ':likes' => isset($d['likes']) ? max(0, (int)$d['likes']) : 0,
      ':views' => isset($d['views']) ? max(0, (int)$d['views']) : 0,
      ':published_at' => $publishedAt,
      ':id' => $id,
    ]);

    $stmt2 = $pdo->prepare(
      "SELECT id, title, html, short_html, author_id, announce_image_path, likes, views,
              EXTRACT(EPOCH FROM published_at)::bigint AS timestamp,
              created_at, updated_at
       FROM public.news
       WHERE id = :id"
    );
    $stmt2->execute([':id' => $id]);
    jsonOk(fmtNews($stmt2->fetch()), 'Новость обновлена');
    break;

  case 'DELETE':
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

function buildQuery(array $get): array
{
  $cond = [];
  $params = [];

  if (!empty($get['search'])) {
    $cond[] = '(title ILIKE :search OR html ILIKE :search OR short_html ILIKE :search)';
    $params[':search'] = '%' . trim($get['search']) . '%';
  }

  if (!empty($get['author_id'])) {
    $cond[] = 'author_id = :author_id';
    $params[':author_id'] = (int)$get['author_id'];
  }

  $allowed = ['published_at', 'created_at', 'id', 'title'];
  $order = in_array($get['order'] ?? '', $allowed, true) ? $get['order'] : 'published_at';
  $dir = strtolower($get['dir'] ?? '') === 'asc' ? 'ASC' : 'DESC';

  $sql =
    "SELECT id, title, html, short_html, author_id, announce_image_path, likes, views,
            EXTRACT(EPOCH FROM published_at)::bigint AS timestamp,
            created_at, updated_at
     FROM public.news" .
    ($cond ? ' WHERE ' . implode(' AND ', $cond) : '') .
    " ORDER BY $order $dir, id DESC";

  return [$sql, $params];
}

function fmtNews(array $r): array
{
  return [
    'id' => (int)$r['id'],
    'title' => $r['title'] ?? '',
    'html' => $r['html'] ?? '',
    'short_html' => $r['short_html'] ?? '',
    'author_id' => isset($r['author_id']) ? (int)$r['author_id'] : null,
    'announce_image_path' => $r['announce_image_path'] ?? null,
    'likes' => isset($r['likes']) ? (int)$r['likes'] : 0,
    'views' => isset($r['views']) ? (int)$r['views'] : 0,
    'timestamp' => isset($r['timestamp']) ? (int)$r['timestamp'] : null,
    'created_at' => $r['created_at'] ?? null,
    'updated_at' => $r['updated_at'] ?? null,
  ];
}

function parsePublishedAt(array $data): string
{
  // Priority:
  // 1) data.published_at ISO/string
  // 2) data.timestamp unix seconds
  // 3) NOW()
  if (!empty($data['published_at'])) {
    $s = trim((string)$data['published_at']);
    if ($s) return $s;
  }
  if (isset($data['timestamp'])) {
    $n = (int)$data['timestamp'];
    if ($n > 0) return gmdate('c', $n); // ISO UTC
  }
  return gmdate('c');
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
    if (!isset($data[$f]) || trim((string)$data[$f]) === '') jsonError(422, 'Поле обязательно');
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

