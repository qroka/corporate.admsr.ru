<?php
/**
 * API: /api/ofo_positions.php?unit_number=N — должности для выбранного ОФО.
 *
 * GET ?unit_number=N → [{id, name, is_head}]
 * Должности берутся через связь ofo_unit_position. is_head=true — руководящие
 * (начальник/директор/глава/председатель и их «и.о.»), для выбора пары на фронте.
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

$unitNumber = isset($_GET['unit_number']) ? (int)$_GET['unit_number'] : 0;
if ($unitNumber <= 0) jsonError(400, 'Укажите unit_number');

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

$stmt = $pdo->prepare(
    'SELECT p.id, p.name, p.is_head
     FROM public.ofo_position p
     JOIN public.ofo_unit_position oup ON oup.position_id = p.id
     WHERE oup.unit_number = :un
     ORDER BY p.is_head DESC, p.sort_order, p.name'
);
$stmt->execute([':un' => $unitNumber]);
$rows = $stmt->fetchAll();
foreach ($rows as &$r) { $r['id'] = (int)$r['id']; $r['is_head'] = (bool)$r['is_head']; }
unset($r);

jsonOk($rows);
