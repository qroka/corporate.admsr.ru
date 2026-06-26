<?php
/**
 * POST /api/tests_direct.php — направить форму в ОФО / лично.
 * Body: { userId, formId, mode: 'ofo'|'users', ids: number[], force?: bool }
 *  - если часть целей уже была направлена и force != true → { needConfirm: true, already: number[] }
 *  - иначе добавляет (source='directed') и возвращает обновлённую форму.
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);
$formId = (int)($body['formId'] ?? 0);
$mode = $body['mode'] ?? '';
$ids = array_values(array_unique(array_map('intval', $body['ids'] ?? [])));
$force = !empty($body['force']);

if ($formId <= 0) jsonError(400, 'Не передан formId');
if (!in_array($mode, ['ofo', 'users'], true)) jsonError(400, 'mode должен быть ofo или users');
if (!$ids) jsonError(400, 'Не выбраны получатели');

$st = $pdo->prepare('SELECT owner_id FROM public.test_forms WHERE id = :id');
$st->execute([':id' => $formId]);
$own = $st->fetchColumn(0);
if ($own === false) jsonError(404, 'Форма не найдена');
if ($own !== null && (int)$own !== $viewer) jsonError(403, 'Направлять может только создатель');

$table = $mode === 'ofo' ? 'public.test_audience_ofo' : 'public.test_audience_users';
$col = $mode === 'ofo' ? 'ofo_unit_id' : 'user_id';

// уже направленные
$ex = $pdo->prepare("SELECT $col FROM $table WHERE form_id = :f");
$ex->execute([':f' => $formId]);
$existing = array_map('intval', array_column($ex->fetchAll(), $col));
$already = array_values(array_intersect($ids, $existing));

if ($already && !$force) {
    jsonOk(['needConfirm' => true, 'already' => $already]);
}

$ins = $pdo->prepare("INSERT INTO $table (form_id, $col, source) VALUES (:f, :t, 'directed') ON CONFLICT (form_id, $col) DO NOTHING");
foreach ($ids as $t) $ins->execute([':f' => $formId, ':t' => $t]);

jsonOk(tf_loadForm($pdo, $formId, $viewer));
