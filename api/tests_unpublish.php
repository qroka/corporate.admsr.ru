<?php
/**
 * POST /api/tests_unpublish.php — снять форму с публикации (→ в черновики, list_no сохраняется).
 * Body: { userId, formId }
 * → обновлённая TestForm (status = draft).
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);
$formId = (int)($body['formId'] ?? 0);
if ($formId <= 0) jsonError(400, 'Не передан formId');

$st = $pdo->prepare('SELECT owner_id FROM public.test_forms WHERE id = :id');
$st->execute([':id' => $formId]);
$own = $st->fetchColumn(0);
if ($own === false) jsonError(404, 'Форма не найдена');
if ($own !== null && (int)$own !== $viewer) jsonError(403, 'Снять с публикации может только создатель');

$pdo->prepare("UPDATE public.test_forms SET status = 'draft', updated_at = now() WHERE id = :id")->execute([':id' => $formId]);

jsonOk(tf_loadForm($pdo, $formId, $viewer));
