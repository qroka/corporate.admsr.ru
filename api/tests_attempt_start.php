<?php
/**
 * POST /api/tests_attempt_start.php
 * Body: {formId, enrollmentId?, courseTestLinkId?}
 * → {attemptId, form} (без correct для не-админа / не-preview)
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$body = cs_body();
$formId = (int)($body['formId'] ?? 0);
$enrollmentId = (int)($body['enrollmentId'] ?? 0);
$linkId = (int)($body['courseTestLinkId'] ?? 0);
$preview = cs_bool($body['preview'] ?? false);

if ($formId <= 0 && $linkId <= 0) jsonError(400, 'Укажите formId или courseTestLinkId');

$link = null;
if ($linkId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE id = :id');
    $st->execute([':id' => $linkId]);
    $link = $st->fetch();
    if (!$link) jsonError(404, 'Связь теста не найдена');
    $formId = (int)$link['test_form_id'];
} elseif ($formId > 0) {
    $st = $pdo->prepare('SELECT * FROM public.course_test_links WHERE test_form_id = :f LIMIT 1');
    $st->execute([':f' => $formId]);
    $link = $st->fetch() ?: null;
}

$st = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
$st->execute([':id' => $formId]);
$formRow = $st->fetch();
if (!$formRow) jsonError(404, 'Форма не найдена');

$uid = (int)$user['id'];
$enr = null;

if ($link) {
    if ($enrollmentId <= 0) jsonError(400, 'Для теста курса нужен enrollmentId');
    $enr = cs_require_enrollment_access($pdo, $enrollmentId, $user, false);
    if ((int)$enr['course_version_id'] !== (int)$link['course_version_id']) {
        jsonError(403, 'Enrollment не соответствует тесту');
    }
    if (!in_array($enr['status'], ['in_progress', 'not_started', 'overdue'], true)) {
        jsonError(409, 'Курс недоступен для прохождения теста');
    }
} else {
    // обычный тест — базовая проверка доступа как в submit
    if ($formRow['status'] !== 'published' && !auth_is_admin($user) && (int)$formRow['owner_id'] !== $uid) {
        jsonError(403, 'Нет доступа');
    }
}

// Вернуть существующую in_progress попытку
$ex = $pdo->prepare(
    "SELECT id FROM public.test_attempts
     WHERE form_id = :f AND user_id = :u AND status = 'in_progress'
     ORDER BY id DESC LIMIT 1"
);
$ex->execute([':f' => $formId, ':u' => $uid]);
$existingId = $ex->fetchColumn(0);

if (!$existingId) {
    if (!($formRow['kind'] === 'poll' && tf_bool($formRow['allow_revote']))) {
        $allowedAttempts = tf_bool($formRow['limit_attempts']) ? max(1, (int)$formRow['attempts']) : 999;
        $cnt = $pdo->prepare("SELECT COUNT(*) FROM public.test_attempts WHERE form_id = :f AND user_id = :u AND status = 'completed'");
        $cnt->execute([':f' => $formId, ':u' => $uid]);
        if ((int)$cnt->fetchColumn(0) >= $allowedAttempts) {
            jsonError(409, 'Исчерпан лимит попыток');
        }
    }
}

$pdo->beginTransaction();
try {
    if ($existingId) {
        $attemptId = (int)$existingId;
    } else {
        $ins = $pdo->prepare(
            "INSERT INTO public.test_attempts
                (form_id, user_id, status, current_page, started_at, ip, user_agent, via_link)
             VALUES (:f, :u, 'in_progress', 0, now(), :ip, :ua, false)
             RETURNING id"
        );
        $ins->execute([
            ':f' => $formId,
            ':u' => $uid,
            ':ip' => $_SERVER['REMOTE_ADDR'] ?? null,
            ':ua' => $_SERVER['HTTP_USER_AGENT'] ?? null,
        ]);
        $attemptId = (int)$ins->fetchColumn(0);
    }

    if ($link && $enr) {
        $pdo->prepare(
            'INSERT INTO public.course_test_attempt_links (enrollment_id, course_test_link_id, test_attempt_id)
             VALUES (:e, :l, :a)
             ON CONFLICT (test_attempt_id) DO NOTHING'
        )->execute([
            ':e' => $enrollmentId,
            ':l' => (int)$link['id'],
            ':a' => $attemptId,
        ]);

        if ($enr['status'] === 'not_started') {
            $pdo->prepare(
                "UPDATE public.course_enrollments
                 SET status = 'in_progress', started_at = COALESCE(started_at, now()),
                     last_activity_at = now(), updated_at = now()
                 WHERE id = :id"
            )->execute([':id' => $enrollmentId]);
        }
    }

    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка создания попытки');
}

$form = tf_loadForm($pdo, $formId, $uid);
if (!$form) jsonError(404, 'Форма не найдена');
$showCorrect = $preview && auth_is_admin($user);
if (!$showCorrect) {
    $form = tf_strip_correct($form);
}

jsonOk([
    'attemptId' => $attemptId,
    'form' => $form,
    'courseTestLinkId' => $link ? (int)$link['id'] : null,
    'enrollmentId' => $enrollmentId > 0 ? $enrollmentId : null,
]);
