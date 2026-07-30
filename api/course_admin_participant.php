<?php
/** POST /api/course_admin_participant.php — Body: {enrollmentId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_section(\, 'courses');
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

$ofoId = null;
$ofoName = null;
$rawOfo = $u['ofo'] ?? null;
if ($rawOfo !== null && $rawOfo !== '' && $rawOfo !== '-1') {
    $ofoId = (int)$rawOfo;
    if ($ofoId > 0) {
        $oSt = $pdo->prepare('SELECT name FROM public.ofo_unit WHERE id = :id');
        $oSt->execute([':id' => $ofoId]);
        $ofoName = $oSt->fetchColumn(0) ?: null;
    }
}

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

// Тесты версии + лучшая попытка по каждому
$linksSt = $pdo->prepare(
    "SELECT l.id, l.type, l.topic_id, l.is_required,
            f.title AS form_title, t.title AS topic_title, t.sort_order
     FROM public.course_test_links l
     JOIN public.test_forms f ON f.id = l.test_form_id
     LEFT JOIN public.course_topics t ON t.id = l.topic_id AND t.deleted_at IS NULL
     WHERE l.course_version_id = :v
     ORDER BY CASE WHEN l.type = 'final' THEN 1 ELSE 0 END, t.sort_order NULLS LAST, t.id, l.id"
);
$linksSt->execute([':v' => (int)$enr['course_version_id']]);
$tests = [];
foreach ($linksSt->fetchAll() as $link) {
    $linkId = (int)$link['id'];
    $bestSt = $pdo->prepare(
        "SELECT a.id AS attempt_id, a.score, a.passed, a.status, a.started_at, a.finished_at
         FROM public.course_test_attempt_links tal
         JOIN public.test_attempts a ON a.id = tal.test_attempt_id
         WHERE tal.enrollment_id = :e AND tal.course_test_link_id = :l
         ORDER BY
           CASE WHEN a.passed IS TRUE THEN 0 WHEN a.status = 'finished' THEN 1 ELSE 2 END,
           a.score DESC NULLS LAST,
           a.finished_at DESC NULLS LAST,
           a.id DESC
         LIMIT 1"
    );
    $bestSt->execute([':e' => $enrollmentId, ':l' => $linkId]);
    $best = $bestSt->fetch() ?: null;

    $cntSt = $pdo->prepare(
        'SELECT COUNT(*) FROM public.course_test_attempt_links WHERE enrollment_id = :e AND course_test_link_id = :l'
    );
    $cntSt->execute([':e' => $enrollmentId, ':l' => $linkId]);
    $attemptsCount = (int)$cntSt->fetchColumn(0);

    $type = (string)$link['type'];
    $topicTitle = $link['topic_title'] !== null ? (string)$link['topic_title'] : null;
    $formTitle = trim((string)($link['form_title'] ?? ''));
    if ($formTitle === '') {
        $formTitle = $type === 'final' ? 'Итоговый тест' : ($topicTitle ? "Тест: {$topicTitle}" : 'Тест');
    }

    $tests[] = [
        'courseTestLinkId' => $linkId,
        'type' => $type,
        'topicId' => $link['topic_id'] !== null ? (int)$link['topic_id'] : null,
        'topicTitle' => $topicTitle,
        'title' => $formTitle,
        'isRequired' => cs_bool($link['is_required'] ?? true),
        'attemptsCount' => $attemptsCount,
        'attemptId' => $best ? (int)$best['attempt_id'] : null,
        'score' => $best && $best['score'] !== null ? (float)$best['score'] : null,
        'passed' => $best && $best['passed'] !== null ? cs_bool($best['passed']) : null,
        'status' => $best ? (string)$best['status'] : 'not_started',
        'startedAt' => $best['started_at'] ?? null,
        'finishedAt' => $best['finished_at'] ?? null,
    ];
}

// Test attempts (сырой список)
$att = $pdo->prepare(
    "SELECT tal.course_test_link_id, l.type, l.topic_id, f.title AS form_title, t.title AS topic_title,
            a.id AS attempt_id, a.score, a.passed, a.status, a.started_at, a.finished_at
     FROM public.course_test_attempt_links tal
     JOIN public.course_test_links l ON l.id = tal.course_test_link_id
     JOIN public.test_forms f ON f.id = l.test_form_id
     LEFT JOIN public.course_topics t ON t.id = l.topic_id
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
        'topicTitle' => $r['topic_title'] !== null ? (string)$r['topic_title'] : null,
        'title' => (string)($r['form_title'] ?? ''),
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
        'ofoId' => $ofoId,
        'ofoName' => $ofoName,
        'email' => $u['email'] ?? null,
        'phone' => $u['phone'] ?? null,
    ],
    'topics' => $topics,
    'materials' => $materials,
    'tests' => $tests,
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
