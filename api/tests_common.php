<?php
/**
 * tests_common.php — общий модуль для эндпоинтов модуля «Тесты».
 * Подключается через require. Сам по себе не отдаёт данные.
 *
 * Содержит: PDO ($pdo), CORS, jsonOk/jsonError, чтение тела,
 * сериализацию формы БД↔JSON (camelCase, как TestForm на фронте).
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
header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') { http_response_code(204); exit; }

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

// ── Helpers ───────────────────────────────────────────────────────────────────
function tf_body(): array { $b = json_decode(file_get_contents('php://input'), true); return is_array($b) ? $b : []; }
function tf_viewer(array $body): int { return (int)($body['userId'] ?? $_GET['userId'] ?? 0); }
function tf_bool($v): bool { return $v === true || $v === 't' || $v === '1' || $v === 1; }
function tf_secToHms($s): string { if ($s === null || $s === '') return ''; $s = (int)$s; return sprintf('%02d:%02d:%02d', intdiv($s, 3600), intdiv($s % 3600, 60), $s % 60); }
function tf_hmsToSec($v): ?int { if (!$v) return null; $p = array_map('intval', explode(':', (string)$v)); return ($p[0] ?? 0) * 3600 + ($p[1] ?? 0) * 60 + ($p[2] ?? 0); }

// Собрать полную форму (camelCase) из строки test_forms
function tf_assembleForm(PDO $pdo, array $row, int $viewerId): array {
    $fid = (int)$row['id'];

    // Вопросы + варианты
    $qs = $pdo->prepare('SELECT * FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
    $qs->execute([':f' => $fid]);
    $optStmt = $pdo->prepare('SELECT * FROM public.test_options WHERE question_id = :q ORDER BY position, id');
    $questions = [];
    foreach ($qs->fetchAll() as $q) {
        $qid = (int)$q['id'];
        $optStmt->execute([':q' => $qid]);
        $options = []; $correctSingle = null; $correctMulti = [];
        foreach ($optStmt->fetchAll() as $o) {
            $oid = (int)$o['id'];
            $options[] = ['id' => (string)$oid, 'text' => $o['text']];
            if (tf_bool($o['is_correct'])) { $correctSingle = (string)$oid; $correctMulti[] = (string)$oid; }
        }
        $type = $q['type'];
        $cv = $q['correct_value'];
        $correct = null;
        if ($type === 'single' || $type === 'dropdown') $correct = $correctSingle;
        elseif ($type === 'multiple') $correct = $correctMulti;
        elseif ($type === 'scale' || $type === 'number') $correct = $cv !== null && $cv !== '' ? 0 + $cv : null;
        else $correct = $cv; // yesno / text / textarea / date
        $questions[] = [
            'id' => (string)$qid,
            'title' => $q['title'],
            'hint' => $q['hint'],
            'type' => $type,
            'required' => tf_bool($q['required']),
            'options' => $options,
            'scaleMin' => (int)($q['scale_min'] ?? 1),
            'scaleMax' => (int)($q['scale_max'] ?? 5),
            'scaleMinLabel' => $q['scale_min_label'] ?? '',
            'scaleMaxLabel' => $q['scale_max_label'] ?? '',
            'correct' => $correct,
        ];
    }

    // Аудитория (направления)
    $initialOfo = []; $directedOfo = [];
    $a = $pdo->prepare('SELECT ofo_unit_id, source FROM public.test_audience_ofo WHERE form_id = :f');
    $a->execute([':f' => $fid]);
    foreach ($a->fetchAll() as $r) { $id = (int)$r['ofo_unit_id']; if ($r['source'] === 'initial') $initialOfo[] = $id; else $directedOfo[] = $id; }
    $initialUsers = []; $directedUsers = [];
    $a = $pdo->prepare('SELECT user_id, source FROM public.test_audience_users WHERE form_id = :f');
    $a->execute([':f' => $fid]);
    foreach ($a->fetchAll() as $r) { $id = (int)$r['user_id']; if ($r['source'] === 'initial') $initialUsers[] = $id; else $directedUsers[] = $id; }

    // Сколько раз текущий пользователь уже завершил эту форму
    $attemptsUsed = 0;
    if ($viewerId > 0) {
        $c = $pdo->prepare("SELECT COUNT(*) FROM public.test_attempts WHERE form_id = :f AND user_id = :u AND status = 'completed'");
        $c->execute([':f' => $fid, ':u' => $viewerId]);
        $attemptsUsed = (int)$c->fetchColumn(0);
    }

    return [
        'id' => $fid,
        'listId' => $row['list_no'] !== null ? (int)$row['list_no'] : null,
        'status' => $row['status'],
        'kind' => $row['kind'],
        'visibility' => $row['visibility'],
        'title' => $row['title'],
        'description' => $row['description'],
        'completionMessage' => $row['completion_message'],
        'shuffle' => tf_bool($row['shuffle']),
        'shuffleOptions' => tf_bool($row['shuffle_options']),
        'showProgress' => tf_bool($row['show_progress']),
        'freeNavigation' => tf_bool($row['free_navigation']),
        'anonymous' => tf_bool($row['anonymous']),
        'allowChangeAnswer' => tf_bool($row['allow_change_answer']),
        'liveResults' => tf_bool($row['live_results']),
        'allowRevote' => tf_bool($row['allow_revote']),
        'notifyAdmin' => tf_bool($row['notify_creator']),
        'usePassingScore' => tf_bool($row['use_passing_score']),
        'passingScore' => (int)$row['passing_score'],
        'showCorrectAnswers' => tf_bool($row['show_correct_answers']),
        'restrictByOfo' => tf_bool($row['restrict_by_ofo']),
        'useTimeLimit' => tf_bool($row['use_time_limit']),
        'timeLimit' => tf_secToHms($row['time_limit_sec']),
        'limitAttempts' => tf_bool($row['limit_attempts']),
        'attempts' => (int)$row['attempts'],
        'useStart' => tf_bool($row['use_start']),
        'startsAt' => $row['starts_at'] ?? '',
        'useEnd' => tf_bool($row['use_end']),
        'endsAt' => $row['ends_at'] ?? '',
        'showResult' => $row['show_result'],
        'accessByLink' => tf_bool($row['access_by_link']),
        'linkAccess' => $row['link_access'] ?? 'any',
        // токен ссылки отдаём только владельцу (для кнопки «Ссылка»)
        'accessToken' => ($viewerId > 0 && (int)$row['owner_id'] === $viewerId) ? $row['access_token'] : null,
        'ownerId' => $row['owner_id'] !== null ? (int)$row['owner_id'] : null,
        'mine' => $viewerId > 0 && (int)$row['owner_id'] === $viewerId,
        'ofoIds' => $initialOfo,
        'recipients' => $initialUsers,
        'directedOfo' => $directedOfo,
        'directedUsers' => $directedUsers,
        'attemptsUsed' => $attemptsUsed,
        'questions' => $questions,
        'createdAt' => $row['created_at'],
        'updatedAt' => $row['updated_at'],
    ];
}

function tf_loadForm(PDO $pdo, int $id, int $viewerId): ?array {
    $s = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
    $s->execute([':id' => $id]);
    $row = $s->fetch();
    return $row ? tf_assembleForm($pdo, $row, $viewerId) : null;
}

/**
 * Создать/обновить форму (вопросы, варианты, начальную аудиторию).
 * Возвращает id. Не меняет status/list_no (публикация отдельно).
 * Бросает RuntimeException при отказе (нет прав).
 */
function tf_persistForm(PDO $pdo, array $data, int $viewerId): int {
    $id = !empty($data['id']) ? (int)$data['id'] : null;
    $b = fn($k) => !empty($data[$k]) ? 't' : 'f'; // boolean → 't'/'f' (PDO иначе шлёт '' для false)
    $enum = fn($v, $allowed, $def) => in_array($v, $allowed, true) ? $v : $def;

    $cols = [
        'owner_id' => $viewerId ?: null,
        'kind' => $enum($data['kind'] ?? 'test', ['test', 'survey', 'poll'], 'test'),
        'visibility' => $enum($data['visibility'] ?? 'public', ['public', 'private'], 'public'),
        'title' => (string)($data['title'] ?? ''),
        'description' => (string)($data['description'] ?? ''),
        'completion_message' => (string)($data['completionMessage'] ?? ''),
        'shuffle' => $b('shuffle'),
        'shuffle_options' => $b('shuffleOptions'),
        'show_progress' => $b('showProgress'),
        'free_navigation' => $b('freeNavigation'),
        'anonymous' => $b('anonymous'),
        'allow_change_answer' => $b('allowChangeAnswer'),
        'live_results' => $b('liveResults'),
        'allow_revote' => $b('allowRevote'),
        'notify_creator' => $b('notifyAdmin'),
        'use_passing_score' => $b('usePassingScore'),
        'passing_score' => max(0, min(100, (int)($data['passingScore'] ?? 70))),
        'show_correct_answers' => $b('showCorrectAnswers'),
        'restrict_by_ofo' => $b('restrictByOfo'),
        'use_time_limit' => $b('useTimeLimit'),
        'time_limit_sec' => tf_hmsToSec($data['timeLimit'] ?? ''),
        'limit_attempts' => $b('limitAttempts'),
        'attempts' => max(1, (int)($data['attempts'] ?? 1)),
        'use_start' => $b('useStart'),
        'starts_at' => !empty($data['startsAt']) ? $data['startsAt'] : null,
        'use_end' => $b('useEnd'),
        'ends_at' => !empty($data['endsAt']) ? $data['endsAt'] : null,
        'show_result' => $enum($data['showResult'] ?? 'after', ['immediate', 'after', 'never'], 'after'),
        'access_by_link' => $b('accessByLink'),
        'link_access' => $enum($data['linkAccess'] ?? 'any', ['authorized', 'guest', 'any'], 'any'),
    ];

    $pdo->beginTransaction();
    try {
        if ($id) {
            // владелец должен совпадать (или быть пустым)
            $chk = $pdo->prepare('SELECT owner_id FROM public.test_forms WHERE id = :id');
            $chk->execute([':id' => $id]);
            $own = $chk->fetchColumn(0);
            if ($own === false) throw new RuntimeException('Форма не найдена');
            if ($own !== null && (int)$own !== $viewerId) throw new RuntimeException('Нет прав на изменение');
            unset($cols['owner_id']); // владельца не меняем
            $set = implode(', ', array_map(fn($k) => "$k = :$k", array_keys($cols)));
            $stmt = $pdo->prepare("UPDATE public.test_forms SET $set, updated_at = now() WHERE id = :id");
            $stmt->execute($cols + [':id' => $id]);
        } else {
            $keys = array_keys($cols);
            $stmt = $pdo->prepare(
                'INSERT INTO public.test_forms (' . implode(', ', $keys) . ') VALUES (:' . implode(', :', $keys) . ') RETURNING id'
            );
            $stmt->execute($cols);
            $id = (int)$stmt->fetchColumn(0);
        }

        // Вопросы пересоздаём целиком
        $pdo->prepare('DELETE FROM public.test_questions WHERE form_id = :f')->execute([':f' => $id]);
        $qIns = $pdo->prepare(
            'INSERT INTO public.test_questions (form_id, position, type, title, hint, required, scale_min, scale_max, scale_min_label, scale_max_label, correct_value)
             VALUES (:f, :pos, :type, :title, :hint, :req, :smin, :smax, :slmin, :slmax, :cv) RETURNING id'
        );
        $oIns = $pdo->prepare('INSERT INTO public.test_options (question_id, position, text, is_correct) VALUES (:q, :pos, :text, :ic)');

        foreach (($data['questions'] ?? []) as $qi => $q) {
            $type = $q['type'] ?? 'single';
            $correct = $q['correct'] ?? null;
            $correctValue = null;
            if (in_array($type, ['yesno', 'text', 'textarea', 'date'], true)) {
                $correctValue = ($correct === null || $correct === '') ? null : (string)$correct;
            } elseif (in_array($type, ['scale', 'number'], true)) {
                $correctValue = ($correct === null || $correct === '') ? null : (string)$correct;
            }
            $qIns->execute([
                ':f' => $id, ':pos' => $qi, ':type' => $type,
                ':title' => (string)($q['title'] ?? ''), ':hint' => (string)($q['hint'] ?? ''),
                ':req' => !empty($q['required']) ? 't' : 'f',
                ':smin' => isset($q['scaleMin']) ? (int)$q['scaleMin'] : null,
                ':smax' => isset($q['scaleMax']) ? (int)$q['scaleMax'] : null,
                ':slmin' => (string)($q['scaleMinLabel'] ?? ''), ':slmax' => (string)($q['scaleMaxLabel'] ?? ''),
                ':cv' => $correctValue,
            ]);
            $qid = (int)$qIns->fetchColumn(0);

            if (in_array($type, ['single', 'multiple', 'dropdown'], true)) {
                $correctIds = $type === 'multiple'
                    ? array_map('strval', is_array($correct) ? $correct : [])
                    : ($correct !== null ? [(string)$correct] : []);
                foreach (($q['options'] ?? []) as $oi => $opt) {
                    $clientId = (string)($opt['id'] ?? '');
                    $oIns->execute([
                        ':q' => $qid, ':pos' => $oi, ':text' => (string)($opt['text'] ?? ''),
                        ':ic' => in_array($clientId, $correctIds, true) ? 't' : 'f',
                    ]);
                }
            }
        }

        // Начальная аудитория (source = initial): пересоздаём, directed не трогаем
        $pdo->prepare("DELETE FROM public.test_audience_ofo WHERE form_id = :f AND source = 'initial'")->execute([':f' => $id]);
        $pdo->prepare("DELETE FROM public.test_audience_users WHERE form_id = :f AND source = 'initial'")->execute([':f' => $id]);
        $aoIns = $pdo->prepare("INSERT INTO public.test_audience_ofo (form_id, ofo_unit_id, source) VALUES (:f, :t, 'initial') ON CONFLICT (form_id, ofo_unit_id) DO NOTHING");
        foreach (array_unique(array_map('intval', $data['ofoIds'] ?? [])) as $t) $aoIns->execute([':f' => $id, ':t' => $t]);
        $auIns = $pdo->prepare("INSERT INTO public.test_audience_users (form_id, user_id, source) VALUES (:f, :t, 'initial') ON CONFLICT (form_id, user_id) DO NOTHING");
        foreach (array_unique(array_map('intval', $data['recipients'] ?? [])) as $t) $auIns->execute([':f' => $id, ':t' => $t]);

        $pdo->commit();
    } catch (Throwable $e) {
        $pdo->rollBack();
        throw $e;
    }
    return $id;
}

// Защита от прямого вызова модуля
if (realpath(__FILE__) === realpath($_SERVER['SCRIPT_FILENAME'] ?? '')) {
    jsonError(404, 'Not found');
}
