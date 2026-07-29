<?php
/** POST /api/courses_readiness.php — Body: {courseId|versionId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_admin($pdo);
$body = cs_body();
$versionId = cs_resolve_version_id($pdo, $body);
$version = cs_get_version($pdo, $versionId);
if (!$version) jsonError(404, 'Версия не найдена');

jsonOk(cs_readiness($pdo, $versionId));
