<?php
/**
 * POST /api/course_assign.php
 * Body: {courseId|versionId, userIds?, ofoIds?, includeChildren?, startsAt?, deadlineAt?, deadlineDays?, comment?}
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');
$body = cs_body();
$versionId = cs_resolve_version_id($pdo, $body);
$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');
cs_require_course_admin($pdo, (int)$version['courseId']);

if ($version['status'] !== 'published') {
    jsonError(409, 'Назначать можно только опубликованную версию');
}

$userIds = array_values(array_unique(array_filter(array_map('intval', $body['userIds'] ?? []), fn($x) => $x > 0)));
$ofoIds = array_values(array_unique(array_filter(array_map('intval', $body['ofoIds'] ?? []), fn($x) => $x > 0)));
$includeChildren = cs_bool($body['includeChildren'] ?? false);

if (!$userIds && !$ofoIds) jsonError(400, 'Укажите userIds и/или ofoIds');

$startsAt = !empty($body['startsAt']) ? (string)$body['startsAt'] : null;
$deadlineAt = !empty($body['deadlineAt']) ? (string)$body['deadlineAt'] : null;
if ($deadlineAt === null && isset($body['deadlineDays'])) {
    $days = (int)$body['deadlineDays'];
} elseif ($deadlineAt === null && $version['defaultDeadlineDays'] !== null) {
    $days = (int)$version['defaultDeadlineDays'];
} else {
    $days = null;
}

$comment = isset($body['comment']) ? (string)$body['comment'] : null;
$uid = (int)$user['id'];
$createdAssignments = [];
$createdEnrollments = 0;
$skipped = 0;

$pdo->beginTransaction();
try {
    $aIns = $pdo->prepare(
        'INSERT INTO public.course_assignments (
            course_version_id, target_type, target_id, starts_at, deadline_at,
            assigned_by, comment, include_children
         ) VALUES (:v, :tt, :tid, :sa, :da, :by, :c, :ic) RETURNING id'
    );
    $eIns = $pdo->prepare(
        "INSERT INTO public.course_enrollments (
            assignment_id, course_version_id, user_id, status, starts_at, deadline_at
         ) VALUES (:a, :v, :u, 'not_started', :sa, :da) RETURNING id"
    );
    $exists = $pdo->prepare(
        "SELECT id FROM public.course_enrollments
         WHERE user_id = :u AND course_version_id = :v AND status NOT IN ('cancelled') LIMIT 1"
    );

    $resolveDeadline = static function (?string $sa) use ($deadlineAt, $days): ?string {
        if ($deadlineAt !== null) return $deadlineAt;
        if ($days === null || $days <= 0) return null;
        $base = $sa ? strtotime($sa) : time();
        return date('c', $base + $days * 86400);
    };

    // Assignments по пользователям
    foreach ($userIds as $targetUserId) {
        $da = $resolveDeadline($startsAt);
        $aIns->execute([
            ':v' => $versionId, ':tt' => 'user', ':tid' => $targetUserId,
            ':sa' => $startsAt, ':da' => $da, ':by' => $uid, ':c' => $comment, ':ic' => 'f',
        ]);
        $aid = (int)$aIns->fetchColumn(0);
        $createdAssignments[] = $aid;

        $exists->execute([':u' => $targetUserId, ':v' => $versionId]);
        if ($exists->fetch()) {
            $skipped++;
            continue;
        }
        $eIns->execute([
            ':a' => $aid, ':v' => $versionId, ':u' => $targetUserId,
            ':sa' => $startsAt, ':da' => $da,
        ]);
        $eid = (int)$eIns->fetchColumn(0);
        cs_ensure_topic_progress_rows($pdo, $eid, $versionId);
        $createdEnrollments++;
    }

    // Assignments по ОФО
    foreach ($ofoIds as $ofoId) {
        $da = $resolveDeadline($startsAt);
        $aIns->execute([
            ':v' => $versionId, ':tt' => 'ofo', ':tid' => $ofoId,
            ':sa' => $startsAt, ':da' => $da, ':by' => $uid, ':c' => $comment,
            ':ic' => $includeChildren ? 't' : 'f',
        ]);
        $aid = (int)$aIns->fetchColumn(0);
        $createdAssignments[] = $aid;

        $members = cs_resolve_ofo_users($pdo, [$ofoId], $includeChildren);
        foreach ($members as $mid) {
            if (in_array($mid, $userIds, true)) continue; // уже назначен лично
            $exists->execute([':u' => $mid, ':v' => $versionId]);
            if ($exists->fetch()) {
                $skipped++;
                continue;
            }
            $eIns->execute([
                ':a' => $aid, ':v' => $versionId, ':u' => $mid,
                ':sa' => $startsAt, ':da' => $da,
            ]);
            $eid = (int)$eIns->fetchColumn(0);
            cs_ensure_topic_progress_rows($pdo, $eid, $versionId);
            $createdEnrollments++;
        }
    }

    cs_audit($pdo, $uid, 'course.assign', 'course_version', $versionId, [
        'assignments' => $createdAssignments,
        'enrollments' => $createdEnrollments,
        'skipped' => $skipped,
    ]);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка назначения');
}

jsonOk([
    'assignmentIds' => $createdAssignments,
    'enrollmentsCreated' => $createdEnrollments,
    'skipped' => $skipped,
]);
