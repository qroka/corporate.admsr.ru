<?php
/**
 * POST /api/tests_delete.php — удалить ЧЕРНОВИК навсегда (только status = draft).
 * Опубликованную форму сначала нужно «Убрать» (unpublish) в черновики.
 * Body: { userId, formId }
 * → { deleted: true }
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);
$formId = (int)($body['formId'] ?? 0);
if ($formId <= 0) jsonError(400, 'Не передан formId');

$st = $pdo->prepare('SELECT owner_id, status FROM public.test_forms WHERE id = :id');
$st->execute([':id' => $formId]);
$row = $st->fetch();
if (!$row) jsonError(404, 'Форма не найдена');
if ($row['owner_id'] !== null && (int)$row['owner_id'] !== $viewer) jsonError(403, 'Удалить может только создатель');
if ($row['status'] !== 'draft') jsonError(409, 'Сначала уберите форму из публикации');

$pdo->prepare('DELETE FROM public.test_forms WHERE id = :id')->execute([':id' => $formId]);

jsonOk(['deleted' => true]);
