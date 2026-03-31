<?php
error_reporting(0);
ini_set('display_errors', '0');

header('Content-Type: application/json; charset=utf-8');

$allowedOrigins = ['http://localhost:5173', 'http://localhost:5174', 'http://127.0.0.1:5173'];
$origin = $_SERVER['HTTP_ORIGIN'] ?? '';
header('Access-Control-Allow-Origin: ' . (in_array($origin, $allowedOrigins, true) ? $origin : $allowedOrigins[0]));
header('Access-Control-Allow-Methods: POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Access-Control-Max-Age: 86400');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }

define('PROJECT_PUBLIC', '/var/www/corporate.admsr.ru/');
define('FULL_DIR',  PROJECT_PUBLIC . 'img/FullPic/');
define('SMALL_DIR', PROJECT_PUBLIC . 'img/SmallPic/');

function jsonOk(mixed $data, string $msg = 'OK'): never
{
    echo json_encode(['success' => true, 'message' => $msg, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function jsonError(int $code, string $msg): never
{
    http_response_code($code);
    echo json_encode(['success' => false, 'message' => $msg, 'data' => null], JSON_UNESCAPED_UNICODE);
    exit;
}

function detectMime(string $path): string
{
    $f = fopen($path, 'rb');
    $bytes = fread($f, 12);
    fclose($f);

    if (substr($bytes, 0, 3) === "\xFF\xD8\xFF") return 'image/jpeg';
    if (substr($bytes, 0, 8) === "\x89PNG\r\n\x1A\n") return 'image/png';
    if (substr($bytes, 0, 4) === 'RIFF' && substr($bytes, 8, 4) === 'WEBP') return 'image/webp';
    if (substr($bytes, 0, 6) === 'GIF87a' || substr($bytes, 0, 6) === 'GIF89a') return 'image/gif';
    return 'application/octet-stream';
}

/**
 * Конвертирует изображение в WebP и сохраняет в $destPath.
 * $maxWidth = 0 — без изменения размера.
 */
function toWebP(string $srcPath, string $destPath, string $mime, int $maxWidth = 0, int $quality = 82): void
{
    $src = match ($mime) {
        'image/jpeg' => imagecreatefromjpeg($srcPath),
        'image/png'  => imagecreatefrompng($srcPath),
        'image/webp' => imagecreatefromwebp($srcPath),
        default      => throw new RuntimeException("Неподдерживаемый формат: $mime"),
    };
    if (!$src) throw new RuntimeException('Не удалось прочитать изображение');

    // Поправка ориентации EXIF для JPEG
    if ($mime === 'image/jpeg' && function_exists('exif_read_data')) {
        $exif = @exif_read_data($srcPath);
        $src = match ((int)($exif['Orientation'] ?? 1)) {
            3 => imagerotate($src, 180, 0),
            6 => imagerotate($src, -90, 0),
            8 => imagerotate($src, 90, 0),
            default => $src,
        };
    }

    if ($maxWidth > 0 && imagesx($src) > $maxWidth) {
        $src = imagescale($src, $maxWidth);
        if (!$src) throw new RuntimeException('Не удалось масштабировать');
    }

    if (!is_dir(dirname($destPath))) mkdir(dirname($destPath), 0755, true);

    if (!imagewebp($src, $destPath, $quality)) {
        throw new RuntimeException('Не удалось сохранить WebP');
    }
    imagedestroy($src);
}

try {
    if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');
    if (empty($_FILES['image'])) jsonError(400, 'Файл не передан');

    $file = $_FILES['image'];
    if ($file['error'] !== UPLOAD_ERR_OK) jsonError(400, 'Ошибка загрузки файла (код ' . $file['error'] . ')');
    if ($file['size'] > 20 * 1024 * 1024) jsonError(400, 'Файл превышает 20 МБ');

    $mime = detectMime($file['tmp_name']);
    $ext = match ($mime) {
        'image/jpeg' => 'jpg',
        'image/png'  => 'png',
        'image/webp' => 'webp',
        'image/gif'  => 'gif',
        default      => jsonError(400, 'Допустимы только JPEG, PNG, WebP или GIF'),
    };

    $baseName = bin2hex(random_bytes(12));
    $origName = $baseName . '.' . $ext;
    $webpName = $baseName . '.webp';

    // 1. Сохраняем оригинал в FullPic/
    if (!is_dir(FULL_DIR)) mkdir(FULL_DIR, 0755, true);
    if (!move_uploaded_file($file['tmp_name'], FULL_DIR . $origName)) {
        jsonError(500, 'Не удалось сохранить файл');
    }

    // 2. Конвертируем в WebP и кладём в SmallPic/ (960px, quality 76)
    try {
        toWebP(FULL_DIR . $origName, SMALL_DIR . $webpName, $mime, 960, 76);
    } catch (Throwable) {
        // Конвертация не критична — продолжаем без WebP
    }

    $fullUrl  = '/img/FullPic/'  . $origName;
    $smallUrl = is_file(SMALL_DIR . $webpName) ? '/img/SmallPic/' . $webpName : $fullUrl;

    jsonOk([
        'image'      => $smallUrl,
        'image_full' => $fullUrl,
    ], 'Изображение загружено');

} catch (Throwable $e) {
    jsonError(500, 'Ошибка обработки изображения: ' . $e->getMessage());
}