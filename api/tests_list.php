<?php
/**
 * POST /api/tests_list.php — списки форм модуля «Тесты».
 * Body: { userId }
 * → { drafts: TestForm[] (мои черновики), published: TestForm[] (все опубликованные) }
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);

$drafts = [];
$s = $pdo->prepare("SELECT * FROM public.test_forms WHERE status = 'draft' AND owner_id = :u ORDER BY updated_at DESC");
$s->execute([':u' => $viewer]);
foreach ($s->fetchAll() as $row) $drafts[] = tf_assembleForm($pdo, $row, $viewer);

$published = [];
$s = $pdo->query("SELECT * FROM public.test_forms WHERE status = 'published' ORDER BY list_no");
foreach ($s->fetchAll() as $row) $published[] = tf_assembleForm($pdo, $row, $viewer);

jsonOk(['drafts' => $drafts, 'published' => $published]);
