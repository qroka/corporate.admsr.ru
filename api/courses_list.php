<?php
/** POST /api/courses_list.php — список курсов (админ). */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

auth_require_section(\, 'courses');

$st = $pdo->query(
    "SELECT c.*, v.status AS ver_status, v.short_description, v.published_at, v.version_number
     FROM public.course_courses c
     LEFT JOIN public.course_versions v ON v.id = c.current_version_id
     WHERE c.deleted_at IS NULL
     ORDER BY c.updated_at DESC"
);

$items = [];
foreach ($st->fetchAll() as $r) {
    $course = cs_map_course_row($r);
    $course['currentVersion'] = $r['current_version_id'] ? [
        'id' => (int)$r['current_version_id'],
        'versionNumber' => (int)($r['version_number'] ?? 0),
        'status' => (string)($r['ver_status'] ?? 'draft'),
        'shortDescription' => (string)($r['short_description'] ?? ''),
        'publishedAt' => $r['published_at'] ?? null,
    ] : null;
    $items[] = $course;
}

jsonOk(['items' => $items]);
