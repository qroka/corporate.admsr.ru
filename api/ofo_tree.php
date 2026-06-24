<?php
/**
 * API: /api/ofo_tree.php — дерево ОФО для выбора (категории + подразделения).
 *
 * GET → {
 *   categories: [{id, name, sort_order}],
 *   units:      [{id, name, category_id, parent_id, level, unit_number, family_number, sort_order}]
 * }
 * Фронт строит дерево: категория (level 1, некликабельна) → unit level 2..4 по parent_id.
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
header('Access-Control-Allow-Methods: GET, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

function jsonOk($d, $m = 'OK') { echo json_encode(['success' => true, 'message' => $m, 'data' => $d], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); exit; }
function jsonError($c, $m) { http_response_code($c); echo json_encode(['success' => false, 'message' => $m, 'data' => null], JSON_UNESCAPED_UNICODE); exit; }

try {
    $pdo = new PDO(
        sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
        DB_USER, DB_PASS,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
    );
    $pdo->exec("SET client_encoding = 'UTF8'");
} catch (PDOException) {
    jsonError(500, 'Ошибка подключения к БД');
}

$categories = $pdo->query('SELECT id, name, sort_order FROM public.ofo_category ORDER BY sort_order, id')->fetchAll();
$units = $pdo->query(
    'SELECT id, name, category_id, parent_id, level, unit_number, family_number, sort_order
     FROM public.ofo_unit ORDER BY category_id, level, name'
)->fetchAll();

// Счётчики: должностей на подразделение (по unit_number) и пользователей (user_info.ofo = ofo_unit.id)
$posCounts = [];
foreach ($pdo->query('SELECT unit_number, COUNT(*) c FROM public.ofo_unit_position GROUP BY unit_number')->fetchAll() as $r) {
    $posCounts[(int)$r['unit_number']] = (int)$r['c'];
}
$userCounts = [];
foreach ($pdo->query("SELECT ofo, COUNT(*) c FROM public.user_info WHERE ofo ~ '^[0-9]+$' GROUP BY ofo")->fetchAll() as $r) {
    $userCounts[(int)$r['ofo']] = (int)$r['c'];
}

// приводим числовые поля к int
foreach ($categories as &$c) { $c['id'] = (int)$c['id']; $c['sort_order'] = (int)$c['sort_order']; }
unset($c);
foreach ($units as &$u) {
    $u['id'] = (int)$u['id'];
    $u['category_id'] = (int)$u['category_id'];
    $u['parent_id'] = $u['parent_id'] !== null ? (int)$u['parent_id'] : null;
    $u['level'] = (int)$u['level'];
    $u['unit_number'] = (int)$u['unit_number'];
    $u['family_number'] = $u['family_number'] !== null ? (int)$u['family_number'] : null;
    $u['sort_order'] = (int)$u['sort_order'];
    $u['position_count'] = $posCounts[$u['unit_number']] ?? 0;
    $u['user_count'] = $userCounts[$u['id']] ?? 0;
}
unset($u);

jsonOk(['categories' => $categories, 'units' => $units]);
