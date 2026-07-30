<?php
/**
 * POST /api/course_admin_results.php
 * Body: {courseId|versionId?, status?, ofoId?, q?, limit?, offset?}
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_section(\, 'courses');
cs_mark_overdue($pdo);
$body = cs_body();

$where = ['c.deleted_at IS NULL'];
$params = [];

if (!empty($body['versionId'])) {
    $where[] = 'e.course_version_id = :v';
    $params[':v'] = (int)$body['versionId'];
} elseif (!empty($body['courseId'])) {
    $where[] = 'v.course_id = :c';
    $params[':c'] = (int)$body['courseId'];
}
if (!empty($body['status'])) {
    $where[] = 'e.status = :st';
    $params[':st'] = (string)$body['status'];
}
if (!empty($body['ofoId'])) {
    // Выбранное ОФО + все нижестоящие (как при назначении курса)
    $ofoIds = cs_ofo_descendants($pdo, [(int)$body['ofoId']]);
    if (!$ofoIds) {
        $where[] = '1=0';
    } else {
        $ph = [];
        foreach ($ofoIds as $i => $oid) {
            $k = ':ofo' . $i;
            $ph[] = $k;
            $params[$k] = (string)$oid;
        }
        $where[] = 'u.ofo IN (' . implode(', ', $ph) . ')';
    }
}
if (!empty($body['q'])) {
    $where[] = "(u.surname ILIKE :q OR u.firstname ILIKE :q OR u.login ILIKE :q OR c.title ILIKE :q)";
    $params[':q'] = '%' . trim((string)$body['q']) . '%';
}

$wsql = implode(' AND ', $where);
$limit = min(5000, max(1, (int)($body['limit'] ?? 50)));
$offset = max(0, (int)($body['offset'] ?? 0));

$agg = $pdo->prepare(
    "SELECT
        COUNT(*) AS total,
        COUNT(*) FILTER (WHERE e.status = 'not_started') AS not_started,
        COUNT(*) FILTER (WHERE e.status = 'in_progress') AS in_progress,
        COUNT(*) FILTER (WHERE e.status = 'completed') AS completed,
        COUNT(*) FILTER (WHERE e.status = 'failed') AS failed,
        COUNT(*) FILTER (WHERE e.status = 'overdue') AS overdue,
        COUNT(*) FILTER (WHERE e.status = 'cancelled') AS cancelled,
        AVG(e.final_score) FILTER (WHERE e.final_score IS NOT NULL) AS avg_score
     FROM public.course_enrollments e
     JOIN public.course_versions v ON v.id = e.course_version_id
     JOIN public.course_courses c ON c.id = v.course_id
     JOIN public.user_info u ON u.id = e.user_id
     WHERE $wsql"
);
$agg->execute($params);
$a = $agg->fetch() ?: [];

$rows = $pdo->prepare(
    "SELECT e.*, c.id AS course_id, c.title AS course_title, v.version_number,
            u.firstname, u.surname, u.lastname, u.role, u.ofo, u.login,
            o.name AS ofo_name
     FROM public.course_enrollments e
     JOIN public.course_versions v ON v.id = e.course_version_id
     JOIN public.course_courses c ON c.id = v.course_id
     JOIN public.user_info u ON u.id = e.user_id
     LEFT JOIN public.ofo_unit o ON u.ofo ~ '^[0-9]+$' AND o.id = CAST(u.ofo AS integer)
     WHERE $wsql
     ORDER BY e.assigned_at DESC
     LIMIT $limit OFFSET $offset"
);
$rows->execute($params);

$items = [];
foreach ($rows->fetchAll() as $r) {
    $prog = cs_enrollment_progress($pdo, (int)$r['id']);
    $rawOfo = $r['ofo'] ?? null;
    $ofoId = null;
    if ($rawOfo !== null && $rawOfo !== '' && $rawOfo !== '-1' && ctype_digit((string)$rawOfo)) {
        $ofoId = (int)$rawOfo;
    }
    $items[] = [
        'enrollment' => cs_map_enrollment($r, $prog),
        'courseId' => (int)$r['course_id'],
        'courseTitle' => (string)$r['course_title'],
        'versionNumber' => (int)$r['version_number'],
        'user' => [
            'id' => (int)$r['user_id'],
            'fio' => cs_user_fio($r),
            'login' => (string)$r['login'],
            'role' => (string)($r['role'] ?? ''),
            'ofo' => $r['ofo'],
            'ofoId' => $ofoId,
            'ofoName' => $r['ofo_name'] !== null ? (string)$r['ofo_name'] : null,
        ],
    ];
}

jsonOk([
    'aggregates' => [
        'total' => (int)($a['total'] ?? 0),
        'notStarted' => (int)($a['not_started'] ?? 0),
        'inProgress' => (int)($a['in_progress'] ?? 0),
        'completed' => (int)($a['completed'] ?? 0),
        'failed' => (int)($a['failed'] ?? 0),
        'overdue' => (int)($a['overdue'] ?? 0),
        'cancelled' => (int)($a['cancelled'] ?? 0),
        'avgScore' => $a['avg_score'] !== null ? round((float)$a['avg_score'], 2) : null,
    ],
    'items' => $items,
    'limit' => $limit,
    'offset' => $offset,
]);
