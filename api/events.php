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
 * Требования:
 *   — XAMPP / локальный Apache + PHP + MySQL
 *   — создайте БД и таблицу (SQL ниже в комментарии)
 *   — поместите файл в папку htdocs/corporate/api/events.php
 *     или настройте Virtual Host, чтобы корень проекта был в htdocs
 *
 * SQL для создания таблицы:
 * -------------------------------------------------------
 * CREATE DATABASE IF NOT EXISTS corporate_portal
 *   CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
 *
 * USE corporate_portal;
 *
 * CREATE TABLE IF NOT EXISTS events (
 *   id          INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
 *   title       VARCHAR(255)  NOT NULL,
 *   description TEXT          NULL,
 *   badge       VARCHAR(100)  NULL,
 *   date        DATE          NULL,
 *   image       VARCHAR(500)  NOT NULL DEFAULT '/src/img/tailwindcss-v4.svg',
 *   link        VARCHAR(500)  NOT NULL DEFAULT '#',
 *   created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
 *   updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
 *                             ON UPDATE CURRENT_TIMESTAMP
 * ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
 * -------------------------------------------------------
 */

// ─── Настройки подключения ────────────────────────────────────────────────────
define('DB_HOST', 'localhost');
define('DB_PORT', '3306');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'root');
define('DB_PASS', '');          // стандартный пароль XAMPP — пустой
// ─────────────────────────────────────────────────────────────────────────────

// ─── Заголовки ───────────────────────────────────────────────────────────────
header('Content-Type: application/json; charset=utf-8');

// Разрешаем запросы с dev-сервера Vite (localhost:5173) и со своего домена
$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
if (in_array($origin, $allowedOrigins, true)) {
    header("Access-Control-Allow-Origin: $origin");
} else {
    header('Access-Control-Allow-Origin: http://localhost:5173');
}
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');

// Preflight (OPTIONS) — браузер отправляет перед реальным запросом
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}
// ─────────────────────────────────────────────────────────────────────────────

// ─── Подключение к БД ────────────────────────────────────────────────────────
try {
    $dsn = sprintf(
        'mysql:host=%s;port=%s;dbname=%s;charset=utf8mb4',
        DB_HOST, DB_PORT, DB_NAME
    );
    $pdo = new PDO($dsn, DB_USER, DB_PASS, [
        PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        PDO::ATTR_EMULATE_PREPARES   => false,
    ]);
} catch (PDOException $e) {
    jsonError(500, 'Ошибка подключения к БД: ' . $e->getMessage());
}
// ─────────────────────────────────────────────────────────────────────────────

$method = $_SERVER['REQUEST_METHOD'];
$id     = isset($_GET['id']) ? (int) $_GET['id'] : null;

switch ($method) {

    // ── GET ──────────────────────────────────────────────────────────────────
    case 'GET':
        if ($id !== null) {
            // Одно мероприятие
            $stmt = $pdo->prepare('SELECT * FROM events WHERE id = ?');
            $stmt->execute([$id]);
            $event = $stmt->fetch();
            if (!$event) {
                jsonError(404, 'Мероприятие не найдено');
            }
            jsonOk(formatEvent($event));
        } else {
            // Список с опциональными фильтрами
            [$sql, $params] = buildListQuery($_GET);
            $stmt = $pdo->prepare($sql);
            $stmt->execute($params);
            $rows = $stmt->fetchAll();
            jsonOk(array_map('formatEvent', $rows));
        }
        break;

    // ── POST ─────────────────────────────────────────────────────────────────
    case 'POST':
        $data = jsonBody();
        validateRequired($data, ['title', 'date']);

        $stmt = $pdo->prepare(
            'INSERT INTO events (title, description, badge, date, image, link)
             VALUES (:title, :description, :badge, :date, :image, :link)'
        );
        $stmt->execute([
            ':title'       => trim($data['title']),
            ':description' => isset($data['description']) ? trim($data['description']) : null,
            ':badge'       => isset($data['badge'])       ? trim($data['badge'])       : null,
            ':date'        => $data['date'],
            ':image'       => !empty($data['image']) ? trim($data['image']) : '/src/img/tailwindcss-v4.svg',
            ':link'        => !empty($data['link'])  ? trim($data['link'])  : '#',
        ]);

        $newId = (int) $pdo->lastInsertId();

        $stmt2 = $pdo->prepare('SELECT * FROM events WHERE id = ?');
        $stmt2->execute([$newId]);

        http_response_code(201);
        jsonOk(formatEvent($stmt2->fetch()), 'Мероприятие создано');
        break;

    // ── PUT ──────────────────────────────────────────────────────────────────
    case 'PUT':
        if (!$id) {
            jsonError(400, 'Укажите id в параметрах запроса (?id=...)');
        }

        $data = jsonBody();
        validateRequired($data, ['title', 'date']);

        // Проверяем существование записи
        $check = $pdo->prepare('SELECT id FROM events WHERE id = ?');
        $check->execute([$id]);
        if (!$check->fetch()) {
            jsonError(404, 'Мероприятие не найдено');
        }

        $stmt = $pdo->prepare(
            'UPDATE events
             SET title       = :title,
                 description = :description,
                 badge       = :badge,
                 date        = :date,
                 image       = :image,
                 link        = :link
             WHERE id = :id'
        );
        $stmt->execute([
            ':title'       => trim($data['title']),
            ':description' => isset($data['description']) ? trim($data['description']) : null,
            ':badge'       => isset($data['badge'])       ? trim($data['badge'])       : null,
            ':date'        => $data['date'],
            ':image'       => !empty($data['image']) ? trim($data['image']) : '/src/img/tailwindcss-v4.svg',
            ':link'        => !empty($data['link'])  ? trim($data['link'])  : '#',
            ':id'          => $id,
        ]);

        $stmt2 = $pdo->prepare('SELECT * FROM events WHERE id = ?');
        $stmt2->execute([$id]);
        jsonOk(formatEvent($stmt2->fetch()), 'Мероприятие обновлено');
        break;

    // ── DELETE ───────────────────────────────────────────────────────────────
    case 'DELETE':
        if (!$id) {
            jsonError(400, 'Укажите id в параметрах запроса (?id=...)');
        }

        $check = $pdo->prepare('SELECT id FROM events WHERE id = ?');
        $check->execute([$id]);
        if (!$check->fetch()) {
            jsonError(404, 'Мероприятие не найдено');
        }

        $stmt = $pdo->prepare('DELETE FROM events WHERE id = ?');
        $stmt->execute([$id]);
        jsonOk(null, 'Мероприятие удалено');
        break;

    default:
        jsonError(405, 'Метод не поддерживается');
}


// ─── Вспомогательные функции ─────────────────────────────────────────────────

/**
 * Формирует SQL-запрос для GET-списка с поддержкой фильтров:
 *   ?badge=Опрос
 *   ?date_from=2026-03-01&date_to=2026-04-30
 *   ?search=семинар
 *   ?order=date&dir=asc   (по умолчанию: date ASC)
 */
function buildListQuery(array $get): array
{
    $conditions = [];
    $params     = [];

    if (!empty($get['search'])) {
        $conditions[] = '(title LIKE :search OR description LIKE :search)';
        $params[':search'] = '%' . trim($get['search']) . '%';
    }

    if (!empty($get['badge'])) {
        // Поддержка нескольких значений через запятую: ?badge=Опрос,Праздник
        $badges = array_filter(array_map('trim', explode(',', $get['badge'])));
        if ($badges) {
            $placeholders = [];
            foreach ($badges as $i => $b) {
                $key = ":badge$i";
                $placeholders[] = $key;
                $params[$key] = $b;
            }
            $conditions[] = 'badge IN (' . implode(', ', $placeholders) . ')';
        }
    }

    if (!empty($get['date_from'])) {
        $conditions[] = 'date >= :date_from';
        $params[':date_from'] = $get['date_from'];
    }

    if (!empty($get['date_to'])) {
        $conditions[] = 'date <= :date_to';
        $params[':date_to'] = $get['date_to'];
    }

    $allowedOrder = ['date', 'title', 'created_at', 'badge'];
    $orderBy      = in_array($get['order'] ?? '', $allowedOrder, true) ? $get['order'] : 'date';
    $dir          = strtolower($get['dir'] ?? '') === 'desc' ? 'DESC' : 'ASC';

    $sql = 'SELECT * FROM events';
    if ($conditions) {
        $sql .= ' WHERE ' . implode(' AND ', $conditions);
    }
    $sql .= " ORDER BY $orderBy $dir";

    return [$sql, $params];
}

/**
 * Приводит строку БД к формату, ожидаемому фронтендом.
 * Поле `link` → `to` (BlogPostProps), числа — к int.
 */
function formatEvent(array $row): array
{
    return [
        'id'          => (int) $row['id'],
        'title'       => $row['title'],
        'description' => $row['description'] ?? '',
        'badge'       => $row['badge'] ?? null,
        'date'        => $row['date'],           // формат YYYY-MM-DD
        'image'       => $row['image'],
        'to'          => $row['link'],            // фронтенд ожидает `to`
        'link'        => $row['link'],            // дублируем для удобства
        'created_at'  => $row['created_at'],
        'updated_at'  => $row['updated_at'],
    ];
}

/**
 * Читает и декодирует тело запроса как JSON.
 */
function jsonBody(): array
{
    $raw = file_get_contents('php://input');
    if (!$raw) {
        jsonError(400, 'Тело запроса пустое');
    }
    $data = json_decode($raw, true);
    if (!is_array($data)) {
        jsonError(400, 'Некорректный JSON');
    }
    return $data;
}

/**
 * Проверяет наличие обязательных полей.
 */
function validateRequired(array $data, array $fields): void
{
    foreach ($fields as $field) {
        if (empty($data[$field])) {
            jsonError(422, "Поле «$field» обязательно");
        }
    }
}

/**
 * Отправляет успешный JSON-ответ.
 */
function jsonOk(mixed $data, string $message = 'OK'): never
{
    echo json_encode(
        ['success' => true, 'message' => $message, 'data' => $data],
        JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES
    );
    exit;
}

/**
 * Отправляет JSON с ошибкой и нужным HTTP-кодом.
 */
function jsonError(int $code, string $message): never
{
    http_response_code($code);
    echo json_encode(
        ['success' => false, 'message' => $message, 'data' => null],
        JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES
    );
    exit;
}
