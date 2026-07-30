<?php
/** POST /api/course_assignments_list.php — Body: {courseId|versionId?} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_section(\, 'courses');
$body = cs_body();

$sql = 'SELECT a.*, v.course_id, c.title AS course_title
        FROM public.course_assignments a
        JOIN public.course_versions v ON v.id = a.course_version_id
        JOIN public.course_courses c ON c.id = v.course_id
        WHERE 1=1';
$params = [];

if (!empty($body['versionId'])) {
    $sql .= ' AND a.course_version_id = :v';
    $params[':v'] = (int)$body['versionId'];
} elseif (!empty($body['courseId'])) {
    $sql .= ' AND v.course_id = :c';
    $params[':c'] = (int)$body['courseId'];
}
if (!empty($body['activeOnly'])) {
    $sql .= ' AND a.cancelled_at IS NULL';
}
$sql .= ' ORDER BY a.created_at DESC LIMIT 500';

$st = $pdo->prepare($sql);
$st->execute($params);

$cnt = $pdo->prepare(
    "SELECT COUNT(*) FROM public.course_enrollments WHERE assignment_id = :a AND status <> 'cancelled'"
);

$items = [];
foreach ($st->fetchAll() as $r) {
    $cnt->execute([':a' => (int)$r['id']]);
    $items[] = [
        'id' => (int)$r['id'],
        'courseVersionId' => (int)$r['course_version_id'],
        'courseId' => (int)$r['course_id'],
        'courseTitle' => (string)$r['course_title'],
        'targetType' => (string)$r['target_type'],
        'targetId' => (int)$r['target_id'],
        'startsAt' => $r['starts_at'],
        'deadlineAt' => $r['deadline_at'],
        'assignedBy' => (int)$r['assigned_by'],
        'comment' => $r['comment'],
        'includeChildren' => cs_bool($r['include_children']),
        'createdAt' => $r['created_at'],
        'cancelledAt' => $r['cancelled_at'],
        'enrollmentCount' => (int)$cnt->fetchColumn(0),
    ];
}

jsonOk(['items' => $items]);
