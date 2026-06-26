<?php
/**
 * POST /api/tests_save.php — сохранить/обновить ЧЕРНОВИК.
 * Body: { userId, form: TestForm }
 * → сохранённая TestForm (status остаётся как есть; новый = draft).
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);
$form = $body['form'] ?? null;
if (!is_array($form)) jsonError(400, 'Не передана форма');

try {
    $id = tf_persistForm($pdo, $form, $viewer);
} catch (RuntimeException $e) {
    jsonError(403, $e->getMessage());
} catch (Throwable $e) {
    jsonError(500, 'Ошибка сохранения');
}

jsonOk(tf_loadForm($pdo, $id, $viewer));
