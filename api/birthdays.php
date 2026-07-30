<?php
/**
 * API: /api/birthdays.php — дни рождения коллег из месячных xlsx-файлов.
 *
 * Формат файла (один файл = один месяц):
 *   A1 — название месяца (напр. «Июнь») или номер 1–12
 *   B1 — год отображения (напр. 2026)
 *   A2..  — ФИО именинника
 *   B2..  — дата рождения (Excel-дата; на главной показывается без года)
 *
 * GET  /api/birthdays.php             — все записи для отображения: [{fio, month, day}]
 * GET  /api/birthdays.php?manifest=1  — статус по месяцам: { "6": {filename, year, uploaded_at}, ... }
 * POST /api/birthdays.php  (multipart: file=<xlsx>) — загрузить/заменить файл месяца (только админ-UI)
 *
 * Хранение:
 *   public/birthdays_xlsx/<NN>.xlsx        — текущий файл месяца (NN = 01..12)
 *   public/birthdays_xlsx/manifest.json    — карта месяц → {filename, year, uploaded_at}
 *   public/birthdays_xlsx_old/             — заменённые файлы (с таймстампом)
 */

error_reporting(0);
ini_set('display_errors', '0');

define('PROJECT_PUBLIC', '/var/www/corporate.admsr.ru/public/');
define('BDAY_DIR',     PROJECT_PUBLIC . 'birthdays_xlsx/');
define('BDAY_OLD_DIR', PROJECT_PUBLIC . 'birthdays_xlsx_old/');
define('MANIFEST',     BDAY_DIR . 'manifest.json');

define('DB_HOST', 'localhost');
define('DB_PORT', '5432');
define('DB_NAME', 'corporate_portal');
define('DB_USER', 'myuser');
define('DB_PASS', 'VZAIMno4753');

header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

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

const MONTH_NAMES = [
    1 => 'январь', 2 => 'февраль', 3 => 'март', 4 => 'апрель', 5 => 'май', 6 => 'июнь',
    7 => 'июль', 8 => 'август', 9 => 'сентябрь', 10 => 'октябрь', 11 => 'ноябрь', 12 => 'декабрь',
];

// ─── Парсер xlsx (ZipArchive + XML, без сторонних библиотек) ──────────────────

function xlsxSharedStrings(ZipArchive $zip): array
{
    $out = [];
    $xml = $zip->getFromName('xl/sharedStrings.xml');
    if ($xml === false) return $out;
    $doc = new DOMDocument();
    if (!@$doc->loadXML($xml)) return $out;
    foreach ($doc->getElementsByTagName('si') as $si) {
        $text = '';
        foreach ($si->getElementsByTagName('t') as $t) $text .= $t->textContent;
        $out[] = $text;
    }
    return $out;
}

/** Возвращает строки листа: [номер_строки => ['A'=>значение, 'B'=>...]] */
function xlsxRows(ZipArchive $zip, array $shared): array
{
    $xml = $zip->getFromName('xl/worksheets/sheet1.xml');
    if ($xml === false) return [];
    $doc = new DOMDocument();
    if (!@$doc->loadXML($xml)) return [];
    $rows = [];
    foreach ($doc->getElementsByTagName('row') as $rowEl) {
        $rn = (int) $rowEl->getAttribute('r');
        $cells = [];
        foreach ($rowEl->getElementsByTagName('c') as $c) {
            $ref = $c->getAttribute('r');
            $col = preg_replace('/[0-9]+/', '', $ref);
            $t   = $c->getAttribute('t');
            $vEl = $c->getElementsByTagName('v')->item(0);
            $val = $vEl ? $vEl->textContent : null;
            if ($t === 's' && $val !== null) {
                $val = $shared[(int) $val] ?? '';
            } elseif ($t === 'inlineStr') {
                $isEl = $c->getElementsByTagName('t')->item(0);
                $val  = $isEl ? $isEl->textContent : '';
            }
            $cells[$col] = $val;
        }
        $rows[$rn] = $cells;
    }
    return $rows;
}

function monthNameToNumber($raw): int
{
    $s = function_exists('mb_strtolower') ? mb_strtolower(trim((string) $raw), 'UTF-8') : strtolower(trim((string) $raw));
    if ($s === '') return 0;
    if (is_numeric($s)) { $n = (int) $s; return ($n >= 1 && $n <= 12) ? $n : 0; }
    // «март» проверяем раньше «май», чтобы стем 'ма' не перехватил
    $stems = [1 => 'янв', 2 => 'фев', 3 => 'мар', 4 => 'апр', 6 => 'июн', 7 => 'июл',
              8 => 'авг', 9 => 'сен', 10 => 'окт', 11 => 'ноя', 12 => 'дек', 5 => 'ма'];
    foreach ($stems as $m => $stem) {
        if (mb_strpos($s, $stem) !== false) return $m;
    }
    return 0;
}

/** Excel-сериал (или текстовая дата) → ['m'=>месяц,'d'=>день] */
function excelToMd($raw): ?array
{
    if ($raw === null || $raw === '') return null;
    if (is_numeric($raw)) {
        $serial = (int) round((float) $raw);
        if ($serial <= 0) return null;
        $unix = ($serial - 25569) * 86400; // 25569 = Excel-сериал для 1970-01-01
        return ['m' => (int) gmdate('n', $unix), 'd' => (int) gmdate('j', $unix)];
    }
    $ts = strtotime((string) $raw);
    if ($ts === false) return null;
    return ['m' => (int) date('n', $ts), 'd' => (int) date('j', $ts)];
}

/** Парсит файл: ['month'=>int, 'year'=>int, 'entries'=>[['fio','month','day'],...]] | null */
function parseBirthdayFile(string $path): ?array
{
    if (!class_exists('ZipArchive')) return null;
    $zip = new ZipArchive();
    if ($zip->open($path) !== true) return null;
    if ($zip->getFromName('xl/worksheets/sheet1.xml') === false) { $zip->close(); return null; }

    $shared = xlsxSharedStrings($zip);
    $rows   = xlsxRows($zip, $shared);
    $zip->close();
    if (!$rows) return null;

    $header = $rows[1] ?? [];
    $month  = monthNameToNumber($header['A'] ?? '');
    $year   = (int) ($header['B'] ?? 0);

    $entries = [];
    foreach ($rows as $rn => $cells) {
        if ($rn < 2) continue;
        $fio = trim((string) ($cells['A'] ?? ''));
        $md  = excelToMd($cells['B'] ?? null);
        if ($fio === '' || !$md) continue;
        $entries[] = ['fio' => $fio, 'month' => $md['m'], 'day' => $md['d']];
    }

    // Если месяц из A1 не распознан — берём преобладающий месяц из дат
    if (!$month && $entries) {
        $counts = [];
        foreach ($entries as $e) $counts[$e['month']] = ($counts[$e['month']] ?? 0) + 1;
        arsort($counts);
        $month = (int) array_key_first($counts);
    }

    return ['month' => $month, 'year' => $year, 'entries' => $entries];
}

// ─── Аватарки из профилей (матчинг по ФИО) ────────────────────────────────────

function normFio(string $s): string
{
    $s = function_exists('mb_strtolower') ? mb_strtolower($s, 'UTF-8') : strtolower($s);
    $s = str_replace('ё', 'е', $s);
    return preg_replace('/\s+/u', ' ', trim($s));
}

/** Карта нормализованное_ФИО → avatar_url из public.user_info. Пусто, если БД недоступна. */
function buildAvatarMap(): array
{
    $map = [];
    try {
        $pdo = new PDO(
            sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
            DB_USER, DB_PASS,
            [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
        );
        $pdo->exec("SET client_encoding = 'UTF8'");
        $rows = $pdo->query('SELECT surname, firstname, lastname, avatar_url FROM public.user_info')->fetchAll();
        foreach ($rows as $r) {
            $avatar = trim((string) ($r['avatar_url'] ?? ''));
            if ($avatar === '') continue;
            $full = normFio(($r['surname'] ?? '') . ' ' . ($r['firstname'] ?? '') . ' ' . ($r['lastname'] ?? ''));
            if ($full !== '' && !isset($map[$full])) $map[$full] = $avatar;
        }
    } catch (Throwable) {
        // БД недоступна — вернём пустую карту, на фронте подставится заглушка
    }
    return $map;
}

// ─── Манифест ─────────────────────────────────────────────────────────────────

function loadManifest(): array
{
    if (!is_file(MANIFEST)) return [];
    $j = json_decode((string) file_get_contents(MANIFEST), true);
    return is_array($j) ? $j : [];
}
function saveManifest(array $m): void
{
    file_put_contents(MANIFEST, json_encode($m, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT));
}

// ─── Маршруты ─────────────────────────────────────────────────────────────────

$method = $_SERVER['REQUEST_METHOD'];

if ($method === 'GET') {
    // Статус по месяцам для админ-окна
    if (isset($_GET['manifest'])) {
        jsonOk(loadManifest());
    }

    // Данные для отображения: все записи из всех загруженных месяцев + аватарка по ФИО
    $avatarMap = buildAvatarMap();
    $all = [];
    foreach (glob(BDAY_DIR . '*.xlsx') ?: [] as $file) {
        $parsed = parseBirthdayFile($file);
        if (!$parsed) continue;
        foreach ($parsed['entries'] as $e) {
            $e['avatar'] = $avatarMap[normFio($e['fio'])] ?? null;
            $all[] = $e;
        }
    }
    jsonOk($all);
}

if ($method === 'POST') {
    require_once __DIR__ . '/auth_context.php';
    // birthdays.php doesn't always have $pdo — create if needed for auth
    if (!isset($pdo) || !($pdo instanceof PDO)) {
        try {
            $pdo = new PDO(
                sprintf('pgsql:host=%s;port=%s;dbname=%s', DB_HOST, DB_PORT, DB_NAME),
                DB_USER, DB_PASS,
                [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
            );
        } catch (PDOException $e) {
            jsonError(500, 'Ошибка подключения к БД');
        }
    }
    auth_require_section($pdo, 'birthdays');
    if (!is_dir(BDAY_DIR))     mkdir(BDAY_DIR, 0755, true);
    if (!is_dir(BDAY_OLD_DIR)) mkdir(BDAY_OLD_DIR, 0755, true);

    // Превышение post_max_size обнуляет $_POST/$_FILES
    $contentLen = (int) ($_SERVER['CONTENT_LENGTH'] ?? 0);
    if ($contentLen > 0 && empty($_FILES)) {
        jsonError(413, 'Файл слишком большой — превышен лимит сервера (post_max_size).');
    }

    if (empty($_FILES['file'])) jsonError(400, 'Файл не передан');
    $file = $_FILES['file'];
    if ($file['error'] !== UPLOAD_ERR_OK) jsonError(400, 'Ошибка загрузки файла (код ' . $file['error'] . ')');

    $origName = (string) ($file['name'] ?? 'file.xlsx');
    if (!preg_match('/\.xlsx$/i', $origName)) jsonError(400, 'Допустим только файл .xlsx');

    // Парсим прямо из временного файла, чтобы определить месяц
    $parsed = parseBirthdayFile($file['tmp_name']);
    if (!$parsed)               jsonError(400, 'Не удалось прочитать xlsx-файл');
    $month = (int) $parsed['month'];
    if ($month < 1 || $month > 12) jsonError(422, 'Не удалось определить месяц (проверьте ячейку A1)');

    $nn     = str_pad((string) $month, 2, '0', STR_PAD_LEFT);
    $target = BDAY_DIR . $nn . '.xlsx';

    // Если файл месяца уже есть — переносим старый в _old с таймстампом
    if (is_file($target)) {
        $manifest = loadManifest();
        $prevName = $manifest[(string) $month]['filename'] ?? ($nn . '.xlsx');
        $safePrev = preg_replace('/[^\p{L}\p{N}._-]+/u', '_', $prevName);
        $oldName  = $nn . '_' . date('Ymd_His') . '_' . $safePrev;
        if (!preg_match('/\.xlsx$/i', $oldName)) $oldName .= '.xlsx';
        @rename($target, BDAY_OLD_DIR . $oldName);
    }

    if (!move_uploaded_file($file['tmp_name'], $target)) {
        jsonError(500, 'Не удалось сохранить файл');
    }

    $manifest = loadManifest();
    $manifest[(string) $month] = [
        'filename'    => $origName,
        'year'        => (int) $parsed['year'],
        'count'       => count($parsed['entries']),
        'uploaded_at' => date('c'),
    ];
    saveManifest($manifest);

    jsonOk($manifest, 'Файл «' . MONTH_NAMES[$month] . '» загружен');
}

jsonError(405, 'Метод не поддерживается');
