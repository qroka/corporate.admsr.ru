<?php
/** POST /api/course_admin_participant.php — Body: {enrollmentId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_admin($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
if ($enrollmentId <= 0) jsonError(400, 'Не передан enrollmentId');

$enr = cs_enrollment_row($pdo, $enrollmentId);
if (!$enr) jsonError(404, 'Запись не найдена');

$version = cs_get_version($pdo, (int)$enr['course_version_id']);
$course = $version ? cs_get_course($pdo, (int)$version['courseId'], true) : null;
$progress = cs_enrollment_progress($pdo, $enrollmentId);
$assembled = cs_assemble_version($pdo, (int)$enr['course_version_id'], false);

$uSt = $pdo->prepare('SELECT id, firstname, surname, lastname, login, role, ofo, email, phone FROM public.user_info WHERE id = :id');
$uSt->execute([':id' => (int)$enr['user_id']]);
$u = $uSt->fetch() ?: [];

// Topic progress
$tp = $pdo->prepare(
    'SELECT p.*, t.title FROM public.course_topic_progress p
     JOIN public.course_topics t ON t.id = p.topic_id
     WHERE p.enrollment_id = :e ORDER BY t.sort_order, t.id'
);
$tp->execute([':e' => $enrollmentId]);
$topics = [];
foreach ($tp->fetchAll() as $p) {
    $topics[] = [
        'topicId' => (int)$p['topic_id'],
        'title' => (string)$p['title'],
        'status' => (string)$p['status'],
        'activeSeconds' => (int)$p['active_seconds'],
        'openedAt' => $p['opened_at'],
        'completedAt' => $p['completed_at'],
    ];
}

// Material progress
$mp = $pdo->prepare(
    'SELECT mp.*, m.title, m.topic_id FROM public.course_material_progress mp
     JOIN public.course_materials m ON m.id = mp.material_id
     WHERE mp.enrollment_id = :e'
);
$mp->execute([':e' => $enrollmentId]);
$materials = [];
foreach ($mp->fetchAll() as $p) {
    $materials[] = [
        'materialId' => (int)$p['material_id'],
        'topicId' => (int)$p['topic_id'],
        'title' => (string)$p['title'],
        'status' => (string)$p['status'],
        'activeSeconds' => (int)$p['active_seconds'],
        'openedAt' => $p['opened_at'],
        'completedAt' => $p['completed_at'],
    ];
}

// Test attempts
$att = $pdo->prepare(
    "SELECT tal.course_test_link_id, l.type, l.topic_id, a.id AS attempt_id, a.score, a.passed, a.status, a.started_at, a.finished_at
     FROM public.course_test_attempt_links tal
     JOIN public.course_test_links l ON l.id = tal.course_test_link_id
     JOIN public.test_attempts a ON a.id = tal.test_attempt_id
     WHERE tal.enrollment_id = :e
     ORDER BY a.started_at"
);
$att->execute([':e' => $enrollmentId]);
$attempts = [];
foreach ($att->fetchAll() as $r) {
    $attempts[] = [
        'courseTestLinkId' => (int)$r['course_test_link_id'],
        'type' => (string)$r['type'],
        'topicId' => $r['topic_id'] !== null ? (int)$r['topic_id'] : null,
        'attemptId' => (int)$r['attempt_id'],
        'score' => $r['score'] !== null ? (float)$r['score'] : null,
        'passed' => $r['passed'] !== null ? cs_bool($r['passed']) : null,
        'status' => (string)$r['status'],
        'startedAt' => $r['started_at'],
        'finishedAt' => $r['finished_at'],
    ];
}

$comp = $pdo->prepare('SELECT * FROM public.course_completions WHERE enrollment_id = :e ORDER BY id DESC LIMIT 1');
$comp->execute([':e' => $enrollmentId]);
$completion = $comp->fetch() ?: null;

jsonOk([
    'enrollment' => cs_map_enrollment($enr, $progress, $course),
    'version' => $assembled,
    'user' => [
        'id' => (int)($u['id'] ?? 0),
        'fio' => cs_user_fio($u),
        'login' => (string)($u['login'] ?? ''),
        'role' => (string)($u['role'] ?? ''),
        'ofo' => $u['ofo'] ?? null,
        'email' => $u['email'] ?? null,
        'phone' => $u['phone'] ?? null,
    ],
    'topics' => $topics,
    'materials' => $materials,
    'attempts' => $attempts,
    'completion' => $completion ? [
        'id' => (int)$completion['id'],
        'completionNumber' => (int)$completion['completion_number'],
        'completedAt' => $completion['completed_at'],
        'totalActiveSeconds' => (int)$completion['total_active_seconds'],
        'finalScore' => $completion['final_score'] !== null ? (float)$completion['final_score'] : null,
        'passed' => cs_bool($completion['passed']),
        'resultSnapshot' => json_decode($completion['result_snapshot'] ?? '{}', true),
    ] : null,
]);
