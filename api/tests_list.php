<?php
/**
 * POST /api/tests_list.php — списки форm для пользователя.
 * Body: { userId }
 * → {
 *     drafts:    TestForm[]  — мои черновики,
 *     published: TestForm[]  — «Все формы»: только публичные (приватные не видны никому),
 *     mine:      TestForm[]  — «Мои формы»: все опубликованные мной (публичные + приватные),
 *     forMe:     TestForm[]  — направленные мне (лично или в мой ОФО), не мои
 *   }
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);

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
$s = $pdo->prepare("SELECT * FROM public.test_forms WHERE status = 'draft' AND owner_id = :u ORDER BY updated_at DESC");
$s->execute([':u' => $viewer]);
foreach ($s->fetchAll() as $row) $drafts[] = tf_assembleForm($pdo, $row, $viewer);

// «Все формы»: только публичные (приватные не видны никому, даже создателю — он их видит в «Мои формы»)
$published = [];
$s = $pdo->query("SELECT * FROM public.test_forms WHERE status = 'published' AND visibility = 'public' ORDER BY list_no");
foreach ($s->fetchAll() as $row) $published[] = tf_assembleForm($pdo, $row, $viewer);

// «Мои формы»: все опубликованные текущим пользователем (публичные + приватные)
$mine = [];
$s = $pdo->prepare("SELECT * FROM public.test_forms WHERE status = 'published' AND owner_id = :u ORDER BY list_no");
$s->execute([':u' => $viewer]);
foreach ($s->fetchAll() as $row) $mine[] = tf_assembleForm($pdo, $row, $viewer);

// Тесты для меня: направленные мне лично или в мой ОФО (и не мои)
$forMe = [];
$s = $pdo->prepare(
    "SELECT f.* FROM public.test_forms f
     WHERE f.status = 'published'
       AND f.owner_id IS DISTINCT FROM :u1
       AND (
         EXISTS (SELECT 1 FROM public.test_audience_users au WHERE au.form_id = f.id AND au.user_id = :u2)
         OR EXISTS (SELECT 1 FROM public.test_audience_ofo ao WHERE ao.form_id = f.id AND ao.ofo_unit_id = :ofo)
       )
     ORDER BY f.list_no"
);
$s->execute([':u1' => $viewer, ':u2' => $viewer, ':ofo' => $myOfo]);
foreach ($s->fetchAll() as $row) $forMe[] = tf_assembleForm($pdo, $row, $viewer);

jsonOk(['drafts' => $drafts, 'published' => $published, 'mine' => $mine, 'forMe' => $forMe]);
