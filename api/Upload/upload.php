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

define('PROJECT_PUBLIC', '/var/www/corporate.admsr.ru/public/');
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
    if (!in_array($mime, ['image/jpeg', 'image/png', 'image/webp'], true)) {
        jsonError(400, 'Допустимы только JPEG, PNG или WebP');
    }

    $baseName  = bin2hex(random_bytes(12));
    $webpName  = $baseName . '.webp';
    $tmpPath   = $file['tmp_name'];

    if (!is_dir(FULL_DIR))  mkdir(FULL_DIR,  0755, true);
    if (!is_dir(SMALL_DIR)) mkdir(SMALL_DIR, 0755, true);

    // 1. FullPic — WebP 1920px, quality 100
    toWebP($tmpPath, FULL_DIR . $webpName, $mime, 1920, 100);

    // 2. SmallPic — WebP 960px, quality 76
    toWebP($tmpPath, SMALL_DIR . $webpName, $mime, 960, 76);

    $fullUrl  = '/img/FullPic/'  . $webpName;
    $smallUrl = '/img/SmallPic/' . $webpName;

    jsonOk([
        'image'      => $smallUrl,
        'image_full' => $fullUrl,
    ], 'Изображение загружено');

} catch (Throwable $e) {
    jsonError(500, 'Ошибка обработки изображения: ' . $e->getMessage());
}