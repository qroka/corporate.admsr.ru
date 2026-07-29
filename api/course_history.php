<?php
/** POST /api/course_history.php — история завершений текущего пользователя. */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$uid = (int)$user['id'];

$st = $pdo->prepare(
    'SELECT cc.*, c.title AS course_title, c.category
     FROM public.course_completions cc
     JOIN public.course_courses c ON c.id = cc.course_id
     WHERE cc.user_id = :u
     ORDER BY cc.completed_at DESC'
);
$st->execute([':u' => $uid]);

$items = [];
foreach ($st->fetchAll() as $r) {
    $items[] = [
        'id' => (int)$r['id'],
        'enrollmentId' => (int)$r['enrollment_id'],
        'courseId' => (int)$r['course_id'],
        'courseTitle' => (string)$r['course_title'],
        'category' => $r['category'],
        'courseVersionId' => (int)$r['course_version_id'],
        'completionNumber' => (int)$r['completion_number'],
        'assignedAt' => $r['assigned_at'],
        'startedAt' => $r['started_at'],
        'completedAt' => $r['completed_at'],
        'totalActiveSeconds' => (int)$r['total_active_seconds'],
        'finalScore' => $r['final_score'] !== null ? (float)$r['final_score'] : null,
        'passed' => cs_bool($r['passed']),
    ];
}

jsonOk(['items' => $items]);
