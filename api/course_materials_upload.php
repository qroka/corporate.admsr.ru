<?php
/**
 * POST /api/course_materials_upload.php — multipart upload.
 * Fields: file, topicId, title?, type?, materialId? (update existing), isRequired?, minimumActiveSeconds?
 */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_section($pdo, 'courses');

$topicId = (int)($_POST['topicId'] ?? 0);
$materialId = (int)($_POST['materialId'] ?? 0);
if ($topicId <= 0 && $materialId <= 0) jsonError(400, 'Нужен topicId или materialId');

if ($materialId > 0) {
    $mRow = cs_material_version_row($pdo, $materialId);
    if (!$mRow) jsonError(404, 'Материал не найден');
    $topicId = (int)$mRow['topic_id'];
    $topic = cs_topic_version_row($pdo, $topicId);
} else {
    $topic = cs_topic_version_row($pdo, $topicId);
}
if (!$topic) jsonError(404, 'Тема не найдена');
cs_require_course_admin($pdo, (int)$topic['course_id']);
if (!in_array((string)$topic['version_status'], ['draft', 'published'], true)) {
    jsonError(409, 'Версия недоступна для редактирования (только черновик/опубликовано)');
}

if (empty($_FILES['file']) || !is_uploaded_file($_FILES['file']['tmp_name'])) {
    jsonError(400, 'Файл не загружен');
}

$file = $_FILES['file'];
if (($file['error'] ?? UPLOAD_ERR_OK) !== UPLOAD_ERR_OK) {
    jsonError(400, 'Ошибка загрузки файла');
}

$maxBytes = 50 * 1024 * 1024;
if ((int)$file['size'] > $maxBytes) {
    jsonError(400, 'Файл больше 50 МБ');
}

$origName = (string)($file['name'] ?? 'file');
$ext = strtolower(pathinfo($origName, PATHINFO_EXTENSION));
$allowed = ['pdf','png','jpg','jpeg','gif','webp','mp4','webm','doc','docx','xls','xlsx','ppt','pptx','txt','zip'];
$forbidden = ['php','phar','phtml','sh','exe','bat','cmd','js','html','htm','shtml','cgi'];

if ($ext === '' || in_array($ext, $forbidden, true) || !in_array($ext, $allowed, true)) {
    jsonError(400, 'Недопустимое расширение файла');
}

$finfo = new finfo(FILEINFO_MIME_TYPE);
$mime = $finfo->file($file['tmp_name']) ?: 'application/octet-stream';
// мягкая проверка MIME — блокируем явный text/html / php
if (preg_match('#^(text/html|application/x-httpd-php|application/x-php|text/x-php)#i', $mime)) {
    jsonError(400, 'Недопустимый тип файла');
}

$courseId = (int)$topic['course_id'];
[$absDir, $relPrefix] = cs_course_upload_dir($courseId);
$rand = bin2hex(random_bytes(16));
$storedName = $rand . '.' . $ext;
$absPath = $absDir . DIRECTORY_SEPARATOR . $storedName;
if (!move_uploaded_file($file['tmp_name'], $absPath)) {
    jsonError(500, 'Не удалось сохранить файл');
}

$relKey = $relPrefix . '/' . $storedName;
// URL для клиента через course_file.php
$fileUrl = '/api/course_file.php?path=' . rawurlencode($relKey);

// Тип вложения: PDF/картинки/видео — это тоже «файл»
$type = (string)($_POST['type'] ?? '');
if ($type === '' || in_array($type, ['pdf', 'image', 'video'], true)) {
    $type = 'file';
}
$allowedTypes = ['rich_text', 'file', 'link'];
if (!in_array($type, $allowedTypes, true)) $type = 'file';

$title = trim((string)($_POST['title'] ?? ''));
if ($title === '') $title = $origName;

if ($materialId > 0) {
    $pdo->prepare(
        'UPDATE public.course_materials SET
            type = :type, title = :title, file_url = :fu, mime_type = :mime,
            file_size = :fs, original_filename = :ofn, updated_at = now()
         WHERE id = :id'
    )->execute([
        ':type' => $type,
        ':title' => $title,
        ':fu' => $relKey,
        ':mime' => $mime,
        ':fs' => (int)$file['size'],
        ':ofn' => $origName,
        ':id' => $materialId,
    ]);
} else {
    $max = $pdo->prepare('SELECT COALESCE(MAX(sort_order), -1) + 1 FROM public.course_materials WHERE topic_id = :t AND deleted_at IS NULL');
    $max->execute([':t' => $topicId]);
    $ord = (int)$max->fetchColumn(0);
    $ins = $pdo->prepare(
        'INSERT INTO public.course_materials (
            topic_id, type, title, description, file_url, mime_type, file_size,
            original_filename, sort_order, is_required, minimum_active_seconds
         ) VALUES (
            :tid, :type, :title, :desc, :fu, :mime, :fs, :ofn, :ord, :req, :min
         ) RETURNING id'
    );
    $ins->execute([
        ':tid' => $topicId,
        ':type' => $type,
        ':title' => $title,
        ':desc' => (string)($_POST['description'] ?? ''),
        ':fu' => $relKey,
        ':mime' => $mime,
        ':fs' => (int)$file['size'],
        ':ofn' => $origName,
        ':ord' => $ord,
        ':req' => cs_bool($_POST['isRequired'] ?? true) ? 't' : 'f',
        ':min' => max(0, (int)($_POST['minimumActiveSeconds'] ?? 0)),
    ]);
    $materialId = (int)$ins->fetchColumn(0);
}

cs_audit($pdo, (int)$user['id'], 'course.material.upload', 'course_material', $materialId, [
    'fileUrl' => $relKey,
    'size' => (int)$file['size'],
]);

jsonOk([
    'materialId' => $materialId,
    'fileUrl' => $fileUrl,
    'storageKey' => $relKey,
    'mimeType' => $mime,
    'fileSize' => (int)$file['size'],
    'originalFilename' => $origName,
    'type' => $type,
]);
