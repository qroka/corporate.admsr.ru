<?php
/**
 * POST /api/tests_by_token.php — форма для прохождения по ссылке (публичный, без авторизации).
 * Body: { token, userId? }
 * → форма (TestForm) c полем linkAccess; либо ошибка, если ссылка недействительна/закрыта.
 */
require __DIR__ . '/tests_common.php';

$body = tf_body();
$viewer = tf_viewer($body); // может быть 0 (гость)
$token = trim((string)($body['token'] ?? $_GET['token'] ?? ''));
if ($token === '') jsonError(400, 'Не передан token');

$st = $pdo->prepare("SELECT * FROM public.test_forms WHERE access_token = :t AND status = 'published' AND access_by_link = true");
$st->execute([':t' => $token]);
$row = $st->fetch();
if (!$row) jsonError(404, 'Ссылка недействительна');

// Окно дат
$today = date('Y-m-d');
if (tf_bool($row['use_start']) && $row['starts_at'] !== null && $row['starts_at'] > $today) {
    jsonError(409, 'Форма ещё не началась (с ' . $row['starts_at'] . ')');
}
if (tf_bool($row['use_end']) && $row['ends_at'] !== null && $row['ends_at'] < $today) {
    jsonError(409, 'Приём ответов завершён (' . $row['ends_at'] . ')');
}

jsonOk(tf_assembleForm($pdo, $row, $viewer));
