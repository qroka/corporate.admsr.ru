<?php
/**
 * API: /api/absence_journal.php
 *
 * GET    /api/absence_journal.php              — список записей (с фильтрами)
 * GET    /api/absence_journal.php?id=N         — одна запись
 * POST   /api/absence_journal.php              — создать запись (начало отсутствия)
 * PUT    /api/absence_journal.php?id=N         — обновить запись (завершить / редактировать)
 * DELETE /api/absence_journal.php?id=N         — удалить запись
 *
 * GET-фильтры:
 *   ?user_id=N          — по сотруднику
 *   ?ofo=N              — по ОФО
 *   ?status=active      — только незавершённые (end_datetime IS NULL)
 *   ?status=completed   — только завершённые
 *   ?q=строка           — поиск по ФИО и причине (ILIKE)
 *   ?period=today|week|month|all  — по дате начала
 *   ?limit=N            — кол-во записей (по умолчанию 500)
 *   ?offset=N           — смещение (пагинация)
 *
 * Формат ответа (все эндпоинты):
 *   { "success": true, "data": {...} }   или   { "success": false, "error": "..." }
 *
 * Table: public.absence_journal
 *   id SERIAL PK, user_id INT, fio VARCHAR(255), ofo INT, role TEXT,
 *   start_datetime TIMESTAMP, end_datetime TIMESTAMP|NULL,
 *   reason TEXT|NULL, created_at TIMESTAMP
 */

error_reporting(0);
ini_set('display_errors', '0');

// ─── Подключение к БД ─────────────────────────────────────────────────────────
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

// ─── Вспомогательные функции ──────────────────────────────────────────────────
function jsonOk($data): void {
    echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function jsonError(int $code, string $message): void {
    http_response_code($code);
    echo json_encode(['success' => false, 'error' => $message], JSON_UNESCAPED_UNICODE);
    exit;
}

/** Форматирует строку из БД в нужный клиенту вид */
function fmt(array $row): array {
    return [
        'id'             => (int) $row['id'],
        'user_id'        => (int) $row['user_id'],
        'fio'            => $row['fio'],
        'ofo'            => (int) $row['ofo'],
        'role'           => $row['role'],
        'start_datetime' => $row['start_datetime'],
        'end_datetime'   => $row['end_datetime'],   // null или строка ISO
        'reason'         => $row['reason'],
        'created_at'     => $row['created_at'],
        'status'         => $row['end_datetime'] === null ? 'active' : 'completed',
    ];
}

/** Валидирует строку datetime (YYYY-MM-DD HH:MM:SS или ISO 8601) */
function validDatetime(?string $v): bool {
    if ($v === null || $v === '') return false;
    $d = DateTime::createFromFormat('Y-m-d H:i:s', $v)
      ?: DateTime::createFromFormat('Y-m-d\TH:i:s', $v)
      ?: DateTime::createFromFormat('Y-m-d\TH:i', $v);
    return $d !== false;
}

/** Нормализует дату от клиента (T → пробел, убирает секунды 00 если нет) */
function normDt(string $v): string {
    $v = str_replace('T', ' ', trim($v));
    // Если формат HH:MM без секунд — добавляем :00
    if (preg_match('/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/', $v)) {
        $v .= ':00';
    }
    return $v;
}

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
    $pdo->exec("SET TIME ZONE 'Europe/Moscow'");
} catch (PDOException $e) {
    jsonError(500, 'Ошибка подключения к БД');
}

$method = $_SERVER['REQUEST_METHOD'];
$id     = isset($_GET['id']) ? (int)$_GET['id'] : null;

// ─── GET ──────────────────────────────────────────────────────────────────────
if ($method === 'GET') {

    // Одна запись по id
    if ($id !== null) {
        $stmt = $pdo->prepare('SELECT * FROM public.absence_journal WHERE id = :id');
        $stmt->execute([':id' => $id]);
        $row = $stmt->fetch();
        if (!$row) jsonError(404, 'Запись не найдена');
        jsonOk(fmt($row));
    }

    // Список с фильтрами
    $where  = [];
    $params = [];

    // Фильтр по сотруднику
    if (!empty($_GET['user_id'])) {
        $where[]            = 'user_id = :user_id';
        $params[':user_id'] = (int)$_GET['user_id'];
    }

    // Фильтр по ОФО
    if (!empty($_GET['ofo'])) {
        $where[]       = 'ofo = :ofo';
        $params[':ofo'] = (int)$_GET['ofo'];
    }

    // Фильтр по статусу
    $status = trim(strtolower($_GET['status'] ?? ''));
    if ($status === 'active') {
        $where[] = 'end_datetime IS NULL';
    } elseif ($status === 'completed') {
        $where[] = 'end_datetime IS NOT NULL';
    }

    // Поиск
    $q = trim((string)($_GET['q'] ?? ''));
    if ($q !== '') {
        $where[]        = "(fio ILIKE :q OR COALESCE(reason,'') ILIKE :q)";
        $params[':q']   = '%' . $q . '%';
    }

    // Фильтр по периоду (дата начала отсутствия)
    $period = trim(strtolower($_GET['period'] ?? 'all'));
    switch ($period) {
        case 'today':
            $where[] = "start_datetime::date = CURRENT_DATE";
            break;
        case 'week':
            $where[] = "start_datetime >= CURRENT_DATE - INTERVAL '7 days'";
            break;
        case 'month':
            $where[] = "start_datetime >= CURRENT_DATE - INTERVAL '30 days'";
            break;
        // 'all' — без ограничений
    }

    $limit  = max(1, min(5000, (int)($_GET['limit']  ?? 500)));
    $offset = max(0, (int)($_GET['offset'] ?? 0));

    $sql = 'SELECT * FROM public.absence_journal';
    if ($where) {
        $sql .= ' WHERE ' . implode(' AND ', $where);
    }
    $sql .= ' ORDER BY start_datetime DESC LIMIT :limit OFFSET :offset';

    $stmt = $pdo->prepare($sql);
    foreach ($params as $k => $v) {
        $stmt->bindValue($k, $v, $k === ':q' ? PDO::PARAM_STR : PDO::PARAM_INT);
    }
    $stmt->bindValue(':limit',  $limit,  PDO::PARAM_INT);
    $stmt->bindValue(':offset', $offset, PDO::PARAM_INT);
    $stmt->execute();

    jsonOk(array_map('fmt', $stmt->fetchAll()));
}

// ─── POST — создать запись ────────────────────────────────────────────────────
if ($method === 'POST') {
    $body = json_decode(file_get_contents('php://input'), true);
    if (!is_array($body)) jsonError(400, 'Некорректный JSON');

    // Обязательные поля
    $userId = isset($body['user_id']) ? (int)$body['user_id'] : 0;
    $fio    = trim($body['fio']    ?? '');
    $ofo    = isset($body['ofo'])  ? (int)$body['ofo']  : 0;
    $role   = trim($body['role'] ?? '');
    $start  = trim($body['start_datetime'] ?? '');

    if ($userId <= 0)        jsonError(400, 'user_id обязателен');
    if ($fio === '')         jsonError(400, 'fio обязателен');
    if ($ofo <= 0)           jsonError(400, 'ofo обязателен');
    if ($role === '')        jsonError(400, 'role обязателен');
    if ($start === '')       jsonError(400, 'start_datetime обязателен');
    if (!validDatetime($start)) jsonError(400, 'Некорректный формат start_datetime');

    // Опциональные поля
    $end    = isset($body['end_datetime']) && trim($body['end_datetime']) !== ''
              ? normDt($body['end_datetime']) : null;
    $reason = isset($body['reason']) ? trim($body['reason']) : null;

    if ($end !== null && !validDatetime($end)) jsonError(400, 'Некорректный формат end_datetime');

    $stmt = $pdo->prepare(
        'INSERT INTO public.absence_journal
            (user_id, fio, ofo, role, start_datetime, end_datetime, reason)
         VALUES
            (:user_id, :fio, :ofo, :role, :start, :end, :reason)
         RETURNING *'
    );
    $stmt->execute([
        ':user_id' => $userId,
        ':fio'     => $fio,
        ':ofo'     => $ofo,
        ':role'    => $role,
        ':start'   => normDt($start),
        ':end'     => $end,
        ':reason'  => $reason !== '' ? $reason : null,
    ]);

    $row = $stmt->fetch();
    if (!$row) jsonError(500, 'Не удалось создать запись');
    http_response_code(201);
    jsonOk(fmt($row));
}

// ─── PUT — обновить запись ────────────────────────────────────────────────────
if ($method === 'PUT') {
    if ($id === null || $id <= 0) jsonError(400, 'Не указан id');

    $body = json_decode(file_get_contents('php://input'), true);
    if (!is_array($body)) jsonError(400, 'Некорректный JSON');

    // Проверяем существование записи
    $stmt = $pdo->prepare('SELECT * FROM public.absence_journal WHERE id = :id');
    $stmt->execute([':id' => $id]);
    $existing = $stmt->fetch();
    if (!$existing) jsonError(404, 'Запись не найдена');

    // Собираем поля для обновления
    $sets   = [];
    $params = [':id' => $id];

    if (isset($body['start_datetime']) && trim($body['start_datetime']) !== '') {
        $start = normDt($body['start_datetime']);
        if (!validDatetime($start)) jsonError(400, 'Некорректный формат start_datetime');
        $sets[]            = 'start_datetime = :start';
        $params[':start']  = $start;
    }

    if (array_key_exists('end_datetime', $body)) {
        $end = trim($body['end_datetime'] ?? '');
        if ($end === '' || $end === null) {
            $sets[]          = 'end_datetime = NULL';
        } else {
            $end = normDt($end);
            if (!validDatetime($end)) jsonError(400, 'Некорректный формат end_datetime');
            $sets[]          = 'end_datetime = :end';
            $params[':end']  = $end;
        }
    }

    if (array_key_exists('reason', $body)) {
        $reason              = trim($body['reason'] ?? '');
        $sets[]              = 'reason = :reason';
        $params[':reason']   = $reason !== '' ? $reason : null;
    }

    if (empty($sets)) jsonError(400, 'Нет полей для обновления');

    $sql  = 'UPDATE public.absence_journal SET ' . implode(', ', $sets)
          . ' WHERE id = :id RETURNING *';
    $stmt = $pdo->prepare($sql);
    $stmt->execute($params);
    $row = $stmt->fetch();
    if (!$row) jsonError(500, 'Не удалось обновить запись');
    jsonOk(fmt($row));
}

// ─── DELETE — удалить запись ──────────────────────────────────────────────────
if ($method === 'DELETE') {
    if ($id === null || $id <= 0) jsonError(400, 'Не указан id');

    $stmt = $pdo->prepare('DELETE FROM public.absence_journal WHERE id = :id RETURNING id');
    $stmt->execute([':id' => $id]);
    $row = $stmt->fetch();
    if (!$row) jsonError(404, 'Запись не найдена');
    jsonOk(['deleted_id' => (int)$row['id']]);
}

jsonError(405, 'Метод не разрешён');
