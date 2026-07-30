<?php
/**
 * POST /api/courses_update.php
 * Обновить title/category курса и/или поля draft-версии.
 * Если версия published — отказ 409 (нужна новая версия для правок контента).
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section(\, 'courses');
$body = cs_body();
$courseId = (int)($body['courseId'] ?? 0);
if ($courseId <= 0) jsonError(400, 'Не передан courseId');

[, $course] = cs_require_course_admin($pdo, $courseId);
$versionId = (int)($body['versionId'] ?? $course['currentVersionId'] ?? 0);
$version = $versionId > 0 ? cs_get_version($pdo, $versionId) : null;
if (!$version || (int)$version['courseId'] !== $courseId) {
    jsonError(404, 'Версия не найдена');
}

$metaOnly = array_intersect_key($body, array_flip(['title', 'category']));
$contentKeys = [
    'shortDescription', 'fullDescription', 'coverUrl', 'sequentialProgress',
    'completionRule', 'defaultDeadlineDays', 'finalPassingScore', 'requireFinalTest',
];
$wantsContent = false;
foreach ($contentKeys as $k) {
    if (array_key_exists($k, $body)) {
        $wantsContent = true;
        break;
    }
}

if ($wantsContent && !cs_version_editable($version)) {
    jsonError(409, 'Архивированная версия недоступна для правки контента.');
}

$pdo->beginTransaction();
try {
    if (isset($body['title']) || array_key_exists('category', $body)) {
        $pdo->prepare(
            'UPDATE public.course_courses
             SET title = COALESCE(:t, title),
                 category = CASE WHEN :cat_set THEN :cat ELSE category END,
                 updated_at = now()
             WHERE id = :id'
        )->execute([
            ':t' => isset($body['title']) ? trim((string)$body['title']) : null,
            ':cat_set' => array_key_exists('category', $body) ? 1 : 0,
            ':cat' => array_key_exists('category', $body) ? ($body['category'] !== null ? (string)$body['category'] : null) : null,
            ':id' => $courseId,
        ]);
    }

    if ($wantsContent) {
        $sets = [];
        $params = [':id' => $versionId];
        if (array_key_exists('shortDescription', $body)) {
            $sets[] = 'short_description = :sd';
            $params[':sd'] = cs_sanitize_html((string)$body['shortDescription']);
        }
        if (array_key_exists('fullDescription', $body)) {
            $sets[] = 'full_description = :fd';
            $params[':fd'] = cs_sanitize_html((string)$body['fullDescription']);
        }
        if (array_key_exists('coverUrl', $body)) {
            $sets[] = 'cover_url = :cover';
            $params[':cover'] = $body['coverUrl'];
        }
        if (array_key_exists('sequentialProgress', $body)) {
            $sets[] = 'sequential_progress = :seq';
            $params[':seq'] = cs_bool($body['sequentialProgress']) ? 't' : 'f';
        }
        if (array_key_exists('completionRule', $body)) {
            $sets[] = 'completion_rule = :cr';
            $params[':cr'] = (string)$body['completionRule'];
        }
        if (array_key_exists('defaultDeadlineDays', $body)) {
            $sets[] = 'default_deadline_days = :ddl';
            $params[':ddl'] = $body['defaultDeadlineDays'] !== null ? (int)$body['defaultDeadlineDays'] : null;
        }
        if (array_key_exists('finalPassingScore', $body)) {
            $sets[] = 'final_passing_score = :fps';
            $params[':fps'] = $body['finalPassingScore'];
        }
        if (array_key_exists('requireFinalTest', $body)) {
            $sets[] = 'require_final_test = :rft';
            $params[':rft'] = cs_bool($body['requireFinalTest']) ? 't' : 'f';
        }
        if ($sets) {
            $pdo->prepare(
                'UPDATE public.course_versions SET ' . implode(', ', $sets) . ', updated_at = now() WHERE id = :id'
            )->execute($params);
        }
    }

    cs_audit($pdo, (int)$user['id'], 'course.update', 'course', $courseId, [
        'versionId' => $versionId,
        'meta' => $metaOnly,
        'content' => $wantsContent,
    ]);
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) $pdo->rollBack();
    jsonError(500, 'Ошибка обновления');
}

jsonOk([
    'course' => cs_get_course($pdo, $courseId),
    'version' => cs_assemble_version($pdo, $versionId, true),
]);
