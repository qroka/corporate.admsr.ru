<?php
/** POST /api/course_materials_delete.php — Body: {materialId} */
require_once __DIR__ . '/courses_common.php';
if ($_SERVER['REQUEST_METHOD'] !== 'POST') jsonError(405, 'Метод не поддерживается');

$user = auth_require_admin($pdo);
$body = cs_body();
$materialId = (int)($body['materialId'] ?? 0);
if ($materialId <= 0) jsonError(400, 'Не передан materialId');

$m = cs_material_version_row($pdo, $materialId);
if (!$m) jsonError(404, 'Материал не найден');
cs_require_course_admin($pdo, (int)$m['course_id']);
if ($m['version_status'] !== 'draft') {
    jsonError(409, 'Версия недоступна для редактирования (только черновик)');
}

$pdo->prepare(
    'UPDATE public.course_materials SET deleted_at = now(), updated_at = now() WHERE id = :id'
)->execute([':id' => $materialId]);

cs_audit($pdo, (int)$user['id'], 'course.material.delete', 'course_material', $materialId, []);
jsonOk(['materialId' => $materialId]);
