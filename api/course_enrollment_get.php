<?php
/** POST /api/course_enrollment_get.php — Body: {enrollmentId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
if ($enrollmentId <= 0) jsonError(400, 'Не передан enrollmentId');

$enr = cs_require_enrollment_access($pdo, $enrollmentId, $user);
cs_recalculate_locks($pdo, $enrollmentId);

$version = cs_get_version($pdo, (int)$enr['course_version_id']);
$course = $version ? cs_get_course($pdo, (int)$version['courseId']) : null;
$assembled = cs_assemble_version($pdo, (int)$enr['course_version_id'], false);
$progress = cs_enrollment_progress($pdo, $enrollmentId);

// Topic statuses for UI
$tp = $pdo->prepare('SELECT topic_id, status, active_seconds, last_material_id FROM public.course_topic_progress WHERE enrollment_id = :e');
$tp->execute([':e' => $enrollmentId]);
$topicStatus = [];
foreach ($tp->fetchAll() as $p) {
    $topicStatus[(int)$p['topic_id']] = [
        'status' => (string)$p['status'],
        'activeSeconds' => (int)$p['active_seconds'],
        'lastMaterialId' => $p['last_material_id'] !== null ? (int)$p['last_material_id'] : null,
    ];
}
$mp = $pdo->prepare('SELECT material_id, status, active_seconds FROM public.course_material_progress WHERE enrollment_id = :e');
$mp->execute([':e' => $enrollmentId]);
$matStatus = [];
foreach ($mp->fetchAll() as $p) {
    $matStatus[(int)$p['material_id']] = [
        'status' => (string)$p['status'],
        'activeSeconds' => (int)$p['active_seconds'],
    ];
}

if ($assembled) {
    foreach ($assembled['topics'] as &$t) {
        $t['progress'] = $topicStatus[$t['id']] ?? ['status' => 'locked', 'activeSeconds' => 0, 'lastMaterialId' => null];
        foreach ($t['materials'] as &$m) {
            $m['progress'] = $matStatus[$m['id']] ?? ['status' => 'not_started', 'activeSeconds' => 0];
        }
        unset($m);
    }
    unset($t);
}

jsonOk([
    'enrollment' => cs_map_enrollment($enr, $progress, $course),
    'version' => $assembled,
    'nextAction' => $progress['nextAction'],
]);
