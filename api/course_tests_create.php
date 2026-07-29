<?php
/** POST /api/course_tests_create.php — Body: {topicId} | {versionId|courseId, type:'final'} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_admin($pdo);
$body = cs_body();
$topicId = (int)($body['topicId'] ?? 0);
$type = (string)($body['type'] ?? ($topicId > 0 ? 'topic' : 'final'));

try {
    if ($type === 'final' || ($topicId <= 0 && isset($body['versionId']))) {
        $versionId = cs_resolve_version_id($pdo, $body);
        $version = cs_get_version($pdo, $versionId);
        if (!$version) jsonError(404, 'Версия не найдена');
        cs_require_course_admin($pdo, (int)$version['courseId']);
        $link = cs_create_final_test($pdo, $versionId, $user, $body['form'] ?? $body);
    } else {
        if ($topicId <= 0) jsonError(400, 'Укажите topicId или type=final');
        $topic = cs_topic_version_row($pdo, $topicId);
        if (!$topic) jsonError(404, 'Тема не найдена');
        cs_require_course_admin($pdo, (int)$topic['course_id']);
        $link = cs_create_topic_test($pdo, $topicId, $user, $body['form'] ?? $body);
    }
} catch (Throwable $e) {
    if (!headers_sent()) jsonError(500, 'Ошибка создания теста');
    throw $e;
}

jsonOk(['link' => $link]);
