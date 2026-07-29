<?php
/**
 * POST /api/course_assign_preview.php
 * Body: {userIds?: number[], ofoIds?: number[], includeChildren?: bool}
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_admin($pdo);
$body = cs_body();

$userIds = array_values(array_unique(array_filter(array_map('intval', $body['userIds'] ?? []), fn($x) => $x > 0)));
$ofoIds = array_values(array_unique(array_filter(array_map('intval', $body['ofoIds'] ?? []), fn($x) => $x > 0)));
$includeChildren = cs_bool($body['includeChildren'] ?? false);

$fromOfo = $ofoIds ? cs_resolve_ofo_users($pdo, $ofoIds, $includeChildren) : [];
$allIds = array_values(array_unique(array_merge($userIds, $fromOfo)));

$recipients = [];
if ($allIds) {
    $ph = [];
    $params = [];
    foreach ($allIds as $i => $id) {
        $ph[] = ':u' . $i;
        $params[':u' . $i] = $id;
    }
    $st = $pdo->prepare(
        'SELECT id, firstname, surname, lastname, role, ofo, status
         FROM public.user_info WHERE id IN (' . implode(',', $ph) . ') ORDER BY surname, firstname'
    );
    $st->execute($params);
    foreach ($st->fetchAll() as $u) {
        if (!($u['status'] === true || $u['status'] === 't' || $u['status'] === '1' || $u['status'] === 1)) {
            continue;
        }
        $recipients[] = [
            'id' => (int)$u['id'],
            'fio' => cs_user_fio($u),
            'role' => (string)($u['role'] ?? ''),
            'ofo' => $u['ofo'],
        ];
    }
}

jsonOk([
    'count' => count($recipients),
    'recipients' => $recipients,
    'fromUsers' => count($userIds),
    'fromOfo' => count($fromOfo),
]);
