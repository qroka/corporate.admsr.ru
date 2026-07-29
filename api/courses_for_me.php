<?php
/** POST /api/courses_for_me.php — мои записи на курсы. */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
cs_mark_overdue($pdo);
$uid = (int)$user['id'];

$st = $pdo->prepare(
    "SELECT e.*, c.id AS course_id, c.title AS course_title, c.category,
            v.short_description, v.cover_url, v.version_number, v.status AS version_status
     FROM public.course_enrollments e
     JOIN public.course_versions v ON v.id = e.course_version_id
     JOIN public.course_courses c ON c.id = v.course_id
     WHERE e.user_id = :u AND e.status <> 'cancelled' AND c.deleted_at IS NULL
     ORDER BY e.assigned_at DESC"
);
$st->execute([':u' => $uid]);

$groups = [
    'active' => [],
    'completed' => [],
    'overdue' => [],
    'failed' => [],
];

foreach ($st->fetchAll() as $r) {
    $prog = cs_enrollment_progress($pdo, (int)$r['id']);
    $item = [
        'enrollment' => cs_map_enrollment($r, $prog),
        'course' => [
            'id' => (int)$r['course_id'],
            'title' => (string)$r['course_title'],
            'category' => $r['category'],
            'shortDescription' => (string)($r['short_description'] ?? ''),
            'coverUrl' => $r['cover_url'],
            'versionNumber' => (int)$r['version_number'],
            'versionStatus' => (string)$r['version_status'],
        ],
    ];
    $status = (string)$r['status'];
    if ($status === 'completed') $groups['completed'][] = $item;
    elseif ($status === 'overdue') $groups['overdue'][] = $item;
    elseif ($status === 'failed') $groups['failed'][] = $item;
    else $groups['active'][] = $item;
}

jsonOk($groups);
