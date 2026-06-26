<?php
/**
 * POST /api/tests_publish.php — опубликовать форму (новую или из черновика).
 * Body: { userId, form: TestForm }
 * list_no выдаётся при первой публикации (закрепляется); при повторной — сохраняется.
 * → опубликованная TestForm.
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body);
$form = $body['form'] ?? null;
if (!is_array($form)) jsonError(400, 'Не передана форма');

try {
    $id = tf_persistForm($pdo, $form, $viewer);
    $pdo->prepare(
        "UPDATE public.test_forms
         SET status = 'published',
             published_at = COALESCE(published_at, now()),
             list_no = COALESCE(list_no, nextval('public.test_forms_list_no_seq')),
             updated_at = now()
         WHERE id = :id"
    )->execute([':id' => $id]);
} catch (RuntimeException $e) {
    jsonError(403, $e->getMessage());
} catch (Throwable $e) {
    jsonError(500, 'Ошибка публикации');
}

jsonOk(tf_loadForm($pdo, $id, $viewer));
