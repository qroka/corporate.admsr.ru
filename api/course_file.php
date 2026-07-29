<?php
/**
 * GET /api/course_file.php?materialId=N  — отдать файл материала.
 * Доступ: admin или enrolled пользователь на версию курса.
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'GET') jsonError(405, 'Метод не поддерживается');

$user = auth_require_user($pdo);
$materialId = (int)($_GET['materialId'] ?? 0);
$pathKey = trim((string)($_GET['path'] ?? ''));

$fileUrl = null;
$mime = 'application/octet-stream';
$origName = 'file';
$courseId = 0;
$versionId = 0;

if ($materialId > 0) {
    $m = cs_material_version_row($pdo, $materialId);
    if (!$m || empty($m['file_url'])) jsonError(404, 'Файл не найден');
    $fileUrl = (string)$m['file_url'];
    $mime = $m['mime_type'] ?: $mime;
    $origName = $m['original_filename'] ?: $origName;
    $courseId = (int)$m['course_id'];
    $versionId = (int)$m['course_version_id'];
} elseif ($pathKey !== '') {
    // path=courses/{courseId}/...
    if (!preg_match('#^courses/(\d+)/[a-zA-Z0-9._-]+$#', $pathKey, $mm)) {
        jsonError(400, 'Некорректный path');
    }
    $courseId = (int)$mm[1];
    $fileUrl = $pathKey;
    // найдём материал по file_url для mime/auth
    $st = $pdo->prepare(
        'SELECT m.*, t.course_version_id, v.course_id
         FROM public.course_materials m
         JOIN public.course_topics t ON t.id = m.topic_id
         JOIN public.course_versions v ON v.id = t.course_version_id
         WHERE m.file_url = :fu AND m.deleted_at IS NULL LIMIT 1'
    );
    $st->execute([':fu' => $pathKey]);
    $m = $st->fetch();
    if ($m) {
        $mime = $m['mime_type'] ?: $mime;
        $origName = $m['original_filename'] ?: $origName;
        $versionId = (int)$m['course_version_id'];
        $courseId = (int)$m['course_id'];
        $materialId = (int)$m['id'];
    }
} else {
    jsonError(400, 'Укажите materialId или path');
}

$allowed = auth_is_admin($user);
if (!$allowed && $versionId > 0) {
    $chk = $pdo->prepare(
        "SELECT 1 FROM public.course_enrollments
         WHERE user_id = :u AND course_version_id = :v
           AND status NOT IN ('cancelled')
         LIMIT 1"
    );
    $chk->execute([':u' => (int)$user['id'], ':v' => $versionId]);
    $allowed = (bool)$chk->fetchColumn(0);
}
if (!$allowed) jsonError(403, 'Нет доступа к файлу');

// Абсолютный путь
$abs = null;
if (str_starts_with($fileUrl, 'courses/')) {
    $parts = explode('/', $fileUrl, 3);
    // courses/{id}/{name}
    if (count($parts) >= 3) {
        $abs = cs_uploads_root() . DIRECTORY_SEPARATOR . $parts[1] . DIRECTORY_SEPARATOR . $parts[2];
    }
} elseif (is_file($fileUrl)) {
    $abs = $fileUrl;
}
if (!$abs || !is_file($abs)) {
    jsonError(404, 'Файл отсутствует на диске');
}

// Сбросить JSON Content-Type от tests_common
header_remove('Content-Type');
header('Content-Type: ' . $mime);
header('Content-Length: ' . filesize($abs));
header('Content-Disposition: inline; filename="' . rawurlencode($origName) . '"');
header('X-Content-Type-Options: nosniff');
readfile($abs);
exit;
