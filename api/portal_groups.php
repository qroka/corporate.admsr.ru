<?php
/**
 * API: /api/portal_groups.php — CRUD групп доступа (только суперadmin).
 *
 * GET    /api/portal_groups.php           — список групп
 * GET    /api/portal_groups.php?id=N      — одна группа с правами и участниками
 * POST   /api/portal_groups.php           — создать { name, description?, permissions?, memberIds? }
 * PUT    /api/portal_groups.php?id=N      — обновить
 * DELETE /api/portal_groups.php?id=N      — удалить
 */
require_once __DIR__ . '/auth_context.php';

header('Content-Type: application/json; charset=utf-8');
$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : '*'));
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type, Authorization, X-Session-Token');
header('Access-Control-Max-Age: 86400');
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

function pg_json_ok(mixed $data): never
{
    echo json_encode(['success' => true, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function pg_json_error(int $code, string $msg): never
{
    http_response_code($code);
    echo json_encode(['success' => false, 'message' => $msg, 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

function pg_body(): array
{
    $raw = file_get_contents('php://input');
    if ($raw === false || $raw === '') {
        return [];
    }
    $j = json_decode($raw, true);
    return is_array($j) ? $j : [];
}

function pg_normalize_permissions(mixed $raw): array
{
    if (!is_array($raw)) {
        return [];
    }
    $out = [];
    foreach ($raw as $k) {
        $key = trim((string)$k);
        if ($key !== '' && in_array($key, AUTH_PORTAL_SECTIONS, true)) {
            $out[] = $key;
        }
    }
    return array_values(array_unique($out));
}

function pg_normalize_course_categories(mixed $raw): array
{
    if (!is_array($raw)) {
        return [];
    }
    $out = [];
    foreach ($raw as $k) {
        if (is_array($k)) {
            $key = trim((string)($k['value'] ?? $k['label'] ?? ''));
        } else {
            $key = trim((string)$k);
        }
        if ($key !== '' && in_array($key, AUTH_COURSE_CATEGORIES, true)) {
            $out[] = $key;
        }
    }
    return array_values(array_unique($out));
}

function pg_normalize_member_ids(mixed $raw): array
{
    if (!is_array($raw)) {
        return [];
    }
    $out = [];
    foreach ($raw as $id) {
        $n = (int)$id;
        if ($n > 0) {
            $out[] = $n;
        }
    }
    return array_values(array_unique($out));
}

function pg_map_group_list_row(array $r): array
{
    $perms = [];
    if (!empty($r['permissions'])) {
        $perms = array_values(array_filter(explode(',', (string)$r['permissions'])));
    }
    $cats = [];
    if (!empty($r['course_categories'])) {
        $cats = array_values(array_filter(explode(',', (string)$r['course_categories'])));
    }
    return [
        'id' => (int)$r['id'],
        'name' => (string)$r['name'],
        'description' => (string)($r['description'] ?? ''),
        'memberCount' => (int)($r['member_count'] ?? 0),
        'permissions' => $perms,
        'courseCategories' => $cats,
        'createdAt' => $r['created_at'] ?? null,
        'updatedAt' => $r['updated_at'] ?? null,
    ];
}

function pg_fetch_group(PDO $pdo, int $id): ?array
{
    $st = $pdo->prepare(
        'SELECT id, name, description, created_at, updated_at
         FROM public.portal_access_groups WHERE id = :id'
    );
    $st->execute([':id' => $id]);
    $g = $st->fetch();
    if (!$g) {
        return null;
    }

    $p = $pdo->prepare('SELECT section_key FROM public.portal_group_permissions WHERE group_id = :g ORDER BY section_key');
    $p->execute([':g' => $id]);
    $permissions = array_map(static fn($r) => (string)$r['section_key'], $p->fetchAll());

    $courseCategories = [];
    try {
        $c = $pdo->prepare(
            'SELECT category_key FROM public.portal_group_course_categories WHERE group_id = :g ORDER BY category_key'
        );
        $c->execute([':g' => $id]);
        $courseCategories = array_map(static fn($r) => (string)$r['category_key'], $c->fetchAll());
    } catch (Throwable $e) {
        $courseCategories = [];
    }

    $m = $pdo->prepare(
        'SELECT m.user_id, u.surname, u.firstname, u.lastname, u.login
         FROM public.portal_group_members m
         LEFT JOIN public.user_info u ON u.id = m.user_id
         WHERE m.group_id = :g
         ORDER BY u.surname NULLS LAST, u.firstname NULLS LAST, m.user_id'
    );
    $m->execute([':g' => $id]);
    $members = [];
    $memberIds = [];
    foreach ($m->fetchAll() as $row) {
        $uid = (int)$row['user_id'];
        $memberIds[] = $uid;
        $fio = trim(implode(' ', array_filter([
            $row['surname'] ?? '',
            $row['firstname'] ?? '',
            $row['lastname'] ?? '',
        ])));
        $members[] = [
            'id' => $uid,
            'fio' => $fio !== '' ? $fio : ('ID ' . $uid),
            'login' => (string)($row['login'] ?? ''),
        ];
    }

    return [
        'id' => (int)$g['id'],
        'name' => (string)$g['name'],
        'description' => (string)($g['description'] ?? ''),
        'permissions' => $permissions,
        'courseCategories' => $courseCategories,
        'memberIds' => $memberIds,
        'members' => $members,
        'memberCount' => count($memberIds),
        'createdAt' => $g['created_at'] ?? null,
        'updatedAt' => $g['updated_at'] ?? null,
    ];
}

function pg_replace_permissions(PDO $pdo, int $groupId, array $permissions): void
{
    $pdo->prepare('DELETE FROM public.portal_group_permissions WHERE group_id = :g')->execute([':g' => $groupId]);
    if (!$permissions) {
        return;
    }
    $ins = $pdo->prepare('INSERT INTO public.portal_group_permissions (group_id, section_key) VALUES (:g, :s)');
    foreach ($permissions as $s) {
        $ins->execute([':g' => $groupId, ':s' => $s]);
    }
}

function pg_replace_course_categories(PDO $pdo, int $groupId, array $categories): void
{
    $pdo->prepare('DELETE FROM public.portal_group_course_categories WHERE group_id = :g')->execute([':g' => $groupId]);
    if (!$categories) {
        return;
    }
    $ins = $pdo->prepare(
        'INSERT INTO public.portal_group_course_categories (group_id, category_key) VALUES (:g, :c)'
    );
    foreach ($categories as $cat) {
        $ins->execute([':g' => $groupId, ':c' => $cat]);
    }
}

function pg_replace_members(PDO $pdo, int $groupId, array $memberIds): void
{
    $pdo->prepare('DELETE FROM public.portal_group_members WHERE group_id = :g')->execute([':g' => $groupId]);
    if (!$memberIds) {
        return;
    }
    $ins = $pdo->prepare('INSERT INTO public.portal_group_members (group_id, user_id) VALUES (:g, :u)');
    foreach ($memberIds as $uid) {
        $ins->execute([':g' => $groupId, ':u' => $uid]);
    }
}

auth_require_admin($pdo);

$method = $_SERVER['REQUEST_METHOD'];
$id = isset($_GET['id']) ? (int)$_GET['id'] : 0;

try {
    if ($method === 'GET') {
        if ($id > 0) {
            $g = pg_fetch_group($pdo, $id);
            if (!$g) {
                pg_json_error(404, 'Группа не найдена');
            }
            pg_json_ok($g);
        }

        $rows = $pdo->query(
            "SELECT g.id, g.name, g.description, g.created_at, g.updated_at,
                    (SELECT COUNT(*) FROM public.portal_group_members m WHERE m.group_id = g.id) AS member_count,
                    (SELECT string_agg(p.section_key, ',' ORDER BY p.section_key)
                     FROM public.portal_group_permissions p WHERE p.group_id = g.id) AS permissions,
                    (SELECT string_agg(c.category_key, ',' ORDER BY c.category_key)
                     FROM public.portal_group_course_categories c WHERE c.group_id = g.id) AS course_categories
             FROM public.portal_access_groups g
             ORDER BY g.name"
        )->fetchAll();
        pg_json_ok(array_map('pg_map_group_list_row', $rows));
    }

    if ($method === 'POST') {
        $body = pg_body();
        $name = trim((string)($body['name'] ?? ''));
        if ($name === '') {
            pg_json_error(400, 'Укажите название группы');
        }
        $description = trim((string)($body['description'] ?? ''));
        $permissions = pg_normalize_permissions($body['permissions'] ?? []);
        $courseCategories = in_array('courses', $permissions, true)
            ? pg_normalize_course_categories($body['courseCategories'] ?? [])
            : [];
        if (in_array('courses', $permissions, true) && !$courseCategories) {
            pg_json_error(400, 'Для права «Курсы» выберите хотя бы одну категорию');
        }
        $memberIds = pg_normalize_member_ids($body['memberIds'] ?? []);

        $pdo->beginTransaction();
        try {
            $ins = $pdo->prepare(
                'INSERT INTO public.portal_access_groups (name, description)
                 VALUES (:n, :d) RETURNING id'
            );
            $ins->execute([':n' => $name, ':d' => $description]);
            $newId = (int)$ins->fetchColumn(0);
            pg_replace_permissions($pdo, $newId, $permissions);
            pg_replace_course_categories($pdo, $newId, $courseCategories);
            pg_replace_members($pdo, $newId, $memberIds);
            $pdo->commit();
        } catch (PDOException $e) {
            $pdo->rollBack();
            if (($e->getCode() === '23505') || str_contains($e->getMessage(), 'portal_access_groups_name')) {
                pg_json_error(409, 'Группа с таким названием уже есть');
            }
            throw $e;
        }
        pg_json_ok(pg_fetch_group($pdo, $newId));
    }

    if ($method === 'PUT') {
        if ($id <= 0) {
            pg_json_error(400, 'Не передан id');
        }
        $existing = pg_fetch_group($pdo, $id);
        if (!$existing) {
            pg_json_error(404, 'Группа не найдена');
        }
        $body = pg_body();
        $name = array_key_exists('name', $body) ? trim((string)$body['name']) : $existing['name'];
        if ($name === '') {
            pg_json_error(400, 'Укажите название группы');
        }
        $description = array_key_exists('description', $body)
            ? trim((string)$body['description'])
            : $existing['description'];
        $permissions = array_key_exists('permissions', $body)
            ? pg_normalize_permissions($body['permissions'])
            : $existing['permissions'];
        $courseCategories = array_key_exists('courseCategories', $body)
            ? pg_normalize_course_categories($body['courseCategories'])
            : ($existing['courseCategories'] ?? []);
        if (!in_array('courses', $permissions, true)) {
            $courseCategories = [];
        } elseif (!$courseCategories) {
            pg_json_error(400, 'Для права «Курсы» выберите хотя бы одну категорию');
        }
        $memberIds = array_key_exists('memberIds', $body)
            ? pg_normalize_member_ids($body['memberIds'])
            : $existing['memberIds'];

        $pdo->beginTransaction();
        try {
            $upd = $pdo->prepare(
                'UPDATE public.portal_access_groups
                 SET name = :n, description = :d, updated_at = now()
                 WHERE id = :id'
            );
            $upd->execute([':n' => $name, ':d' => $description, ':id' => $id]);
            pg_replace_permissions($pdo, $id, $permissions);
            pg_replace_course_categories($pdo, $id, $courseCategories);
            pg_replace_members($pdo, $id, $memberIds);
            $pdo->commit();
        } catch (PDOException $e) {
            $pdo->rollBack();
            if (($e->getCode() === '23505') || str_contains($e->getMessage(), 'portal_access_groups_name')) {
                pg_json_error(409, 'Группа с таким названием уже есть');
            }
            throw $e;
        }
        pg_json_ok(pg_fetch_group($pdo, $id));
    }

    if ($method === 'DELETE') {
        if ($id <= 0) {
            pg_json_error(400, 'Не передан id');
        }
        $st = $pdo->prepare('DELETE FROM public.portal_access_groups WHERE id = :id RETURNING id');
        $st->execute([':id' => $id]);
        if (!$st->fetch()) {
            pg_json_error(404, 'Группа не найдена');
        }
        pg_json_ok(['id' => $id]);
    }

    pg_json_error(405, 'Метод не поддерживается');
  } catch (Throwable $e) {
    if (str_contains($e->getMessage(), 'portal_access_groups')
        || str_contains($e->getMessage(), 'portal_group_course_categories')
        || str_contains($e->getMessage(), 'does not exist')) {
        pg_json_error(503, 'Нужны миграции V5__portal_access_groups.sql и V6__portal_group_course_categories.sql');
    }
    pg_json_error(500, 'Ошибка сервера');
}
