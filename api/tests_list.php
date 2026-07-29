<?php
/**
 * POST /api/tests_list.php — списки форм для пользователя.
 * Body: { userId }
 * → {
 *     drafts:    TestForm[]  — мои черновики,
 *     published: TestForm[]  — «Все формы»: только публичные (приватные не видны никому),
 *     mine:      TestForm[]  — «Мои формы»: все опубликованные мной (публичные + приватные),
 *     forMe:     TestForm[]  — направленные мне (лично или в мой ОФО), не мои
 *   }
 * Формы, привязанные к курсам (course_test_links), исключаются из всех списков.
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);

$notInCourse = '';
try {
    $pdo->query('SELECT 1 FROM public.course_test_links LIMIT 0');
    $notInCourse = "AND NOT EXISTS (
        SELECT 1 FROM public.course_test_links ctl WHERE ctl.test_form_id = f.id
    )";
} catch (Throwable $e) {
    // V4 ещё не применена — списки тестов работают без фильтра курсов
}

// ОФО текущего пользователя
$myOfo = null;
if ($viewer > 0) {
    $os = $pdo->prepare('SELECT ofo FROM public.user_info WHERE id = :u');
    $os->execute([':u' => $viewer]);
    $raw = $os->fetchColumn(0);
    if ($raw !== false && preg_match('/^[0-9]+$/', (string)$raw)) $myOfo = (int)$raw;
}

// Черновики
$drafts = [];
$s = $pdo->prepare(
    "SELECT f.* FROM public.test_forms f
     WHERE f.status = 'draft' AND f.owner_id = :u
       $notInCourse
     ORDER BY f.updated_at DESC"
);
$s->execute([':u' => $viewer]);
foreach ($s->fetchAll() as $row) $drafts[] = tf_assembleForm($pdo, $row, $viewer);

// «Все формы»: только публичные
$published = [];
$s = $pdo->query(
    "SELECT f.* FROM public.test_forms f
     WHERE f.status = 'published' AND f.visibility = 'public'
       $notInCourse
     ORDER BY f.list_no"
);
foreach ($s->fetchAll() as $row) $published[] = tf_assembleForm($pdo, $row, $viewer);

// «Мои формы»
$mine = [];
$s = $pdo->prepare(
    "SELECT f.* FROM public.test_forms f
     WHERE f.status = 'published' AND f.owner_id = :u
       $notInCourse
     ORDER BY f.list_no"
);
$s->execute([':u' => $viewer]);
foreach ($s->fetchAll() as $row) $mine[] = tf_assembleForm($pdo, $row, $viewer);

// Тесты для меня
$forMe = [];
$s = $pdo->prepare(
    "SELECT f.* FROM public.test_forms f
     WHERE f.status = 'published'
       $notInCourse
       AND (
         EXISTS (SELECT 1 FROM public.test_audience_users au WHERE au.form_id = f.id AND au.user_id = :u)
         OR EXISTS (SELECT 1 FROM public.test_audience_ofo ao WHERE ao.form_id = f.id AND ao.ofo_unit_id = :ofo)
       )
     ORDER BY f.list_no"
);
$s->execute([':u' => $viewer, ':ofo' => $myOfo]);
foreach ($s->fetchAll() as $row) $forMe[] = tf_assembleForm($pdo, $row, $viewer);

jsonOk(['drafts' => $drafts, 'published' => $published, 'mine' => $mine, 'forMe' => $forMe]);
