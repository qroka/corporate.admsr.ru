<?php
/**
 * courses_common.php — общий модуль для эндпоинтов LMS «Курсы».
 * Подключается через require. Сам по себе не отдаёт данные.
 *
 * Зависит от tests_common.php ($pdo, jsonOk/jsonError, CORS, tf_*) и auth_context.php.
 */

if (defined('COURSES_COMMON_LOADED')) {
    return;
}
define('COURSES_COMMON_LOADED', true);

require_once __DIR__ . '/tests_common.php';
require_once __DIR__ . '/auth_context.php';

// ── Helpers ───────────────────────────────────────────────────────────────────

function cs_bool($v): bool
{
    return $v === true || $v === 't' || $v === '1' || $v === 1;
}

function cs_body(): array
{
    $b = json_decode(file_get_contents('php://input'), true);
    return is_array($b) ? $b : [];
}

function cs_audit(
    PDO $pdo,
    ?int $userId,
    string $action,
    string $entityType,
    ?int $entityId,
    array $payload = []
): void {
    $pdo->prepare(
        'INSERT INTO public.course_audit_logs
            (user_id, action, entity_type, entity_id, payload, ip_address, user_agent)
         VALUES (:u, :a, :et, :eid, :p::jsonb, :ip, :ua)'
    )->execute([
        ':u' => $userId,
        ':a' => $action,
        ':et' => $entityType,
        ':eid' => $entityId,
        ':p' => json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
        ':ip' => $_SERVER['REMOTE_ADDR'] ?? null,
        ':ua' => isset($_SERVER['HTTP_USER_AGENT'])
            ? mb_substr((string)$_SERVER['HTTP_USER_AGENT'], 0, 500)
            : null,
    ]);
}

/**
 * Очистка HTML: убирает script, on*-обработчики, javascript:-URL, опасные iframe;
 * оставляет базовую разметку для материалов курса.
 */
function cs_sanitize_html(?string $html): string
{
    if ($html === null || trim($html) === '') {
        return '';
    }

    $allowed = '<p><br><strong><b><em><i><ul><ol><li><a><h1><h2><h3><h4>'
        . '<img><table><thead><tbody><tr><td><th><blockquote><code><pre><span><div>';
    $clean = strip_tags($html, $allowed);

    if (!class_exists('DOMDocument')) {
        // Fallback без DOM: срезать on* и javascript: грубыми заменами
        $clean = preg_replace('/\son\w+\s*=\s*("[^"]*"|\'[^\']*\'|[^\s>]+)/iu', '', $clean) ?? $clean;
        $clean = preg_replace('/\s(href|src)\s*=\s*(["\'])\s*javascript:[^"\']*\2/iu', '', $clean) ?? $clean;
        $clean = preg_replace('/<\/?iframe\b[^>]*>/iu', '', $clean) ?? $clean;
        return $clean;
    }

    $doc = new DOMDocument('1.0', 'UTF-8');
    $prev = libxml_use_internal_errors(true);
    // Обёртка: фрагмент без html/body
    $wrapped = '<?xml encoding="UTF-8"><div id="cs-root">' . $clean . '</div>';
    $doc->loadHTML($wrapped, LIBXML_HTML_NOIMPLIED | LIBXML_HTML_NODEFDTD);
    libxml_clear_errors();
    libxml_use_internal_errors($prev);

    $allowedTags = [
        'p', 'br', 'strong', 'b', 'em', 'i', 'ul', 'ol', 'li', 'a',
        'h1', 'h2', 'h3', 'h4', 'img', 'table', 'thead', 'tbody', 'tr', 'td', 'th',
        'blockquote', 'code', 'pre', 'span', 'div',
    ];
    $allowedAttrs = [
        'a' => ['href', 'title', 'target', 'rel'],
        'img' => ['src', 'alt', 'title', 'width', 'height'],
        'td' => ['colspan', 'rowspan'],
        'th' => ['colspan', 'rowspan'],
        '*' => ['class'],
    ];

    $walk = function (DOMNode $node) use (&$walk, $allowedTags, $allowedAttrs): void {
        if ($node->nodeType === XML_ELEMENT_NODE) {
            /** @var DOMElement $node */
            $tag = strtolower($node->tagName);
            if ($tag === 'script' || $tag === 'iframe' || $tag === 'object' || $tag === 'embed') {
                $node->parentNode?->removeChild($node);
                return;
            }
            if (!in_array($tag, $allowedTags, true) && $tag !== 'div') {
                // неизвестный тег — разворачиваем детей
                while ($node->firstChild) {
                    $node->parentNode?->insertBefore($node->firstChild, $node);
                }
                $node->parentNode?->removeChild($node);
                return;
            }

            $keep = $allowedAttrs[$tag] ?? [];
            $keep = array_merge($keep, $allowedAttrs['*'] ?? []);
            $toRemove = [];
            foreach ($node->attributes ?? [] as $attr) {
                $name = strtolower($attr->name);
                $val = trim((string)$attr->value);
                if (str_starts_with($name, 'on')) {
                    $toRemove[] = $attr->name;
                    continue;
                }
                if (!in_array($name, $keep, true)) {
                    $toRemove[] = $attr->name;
                    continue;
                }
                if (($name === 'href' || $name === 'src') && preg_match('/^\s*javascript:/i', $val)) {
                    $toRemove[] = $attr->name;
                    continue;
                }
                if ($name === 'href' && preg_match('/^\s*data:/i', $val)) {
                    $toRemove[] = $attr->name;
                }
            }
            foreach ($toRemove as $n) {
                $node->removeAttribute($n);
            }
            if ($tag === 'a' && $node->hasAttribute('target')) {
                $node->setAttribute('rel', 'noopener noreferrer');
            }
        }

        $children = [];
        foreach ($node->childNodes ?? [] as $child) {
            $children[] = $child;
        }
        foreach ($children as $child) {
            $walk($child);
        }
    };

    $root = $doc->getElementById('cs-root');
    if (!$root) {
        return strip_tags($clean, $allowed);
    }
    $walk($root);

    $out = '';
    foreach ($root->childNodes as $child) {
        $out .= $doc->saveHTML($child);
    }
    return $out;
}

function cs_map_version_row(array $row): array
{
    return [
        'id' => (int)$row['id'],
        'courseId' => (int)$row['course_id'],
        'versionNumber' => (int)$row['version_number'],
        'status' => (string)$row['status'],
        'shortDescription' => (string)($row['short_description'] ?? ''),
        'fullDescription' => (string)($row['full_description'] ?? ''),
        'coverUrl' => $row['cover_url'] ?? null,
        'sequentialProgress' => cs_bool($row['sequential_progress'] ?? true),
        'completionRule' => (string)($row['completion_rule'] ?? 'all_required'),
        'defaultDeadlineDays' => $row['default_deadline_days'] !== null ? (int)$row['default_deadline_days'] : null,
        'finalPassingScore' => $row['final_passing_score'] !== null ? (float)$row['final_passing_score'] : null,
        'requireFinalTest' => cs_bool($row['require_final_test'] ?? false),
        'createdBy' => $row['created_by'] !== null ? (int)$row['created_by'] : null,
        'createdAt' => $row['created_at'] ?? null,
        'updatedAt' => $row['updated_at'] ?? null,
        'publishedAt' => $row['published_at'] ?? null,
        'archivedAt' => $row['archived_at'] ?? null,
    ];
}

function cs_map_course_row(array $row): array
{
    return [
        'id' => (int)$row['id'],
        'ownerId' => (int)$row['owner_id'],
        'title' => (string)$row['title'],
        'category' => $row['category'] ?? null,
        'currentVersionId' => $row['current_version_id'] !== null ? (int)$row['current_version_id'] : null,
        'createdAt' => $row['created_at'] ?? null,
        'updatedAt' => $row['updated_at'] ?? null,
        'deletedAt' => $row['deleted_at'] ?? null,
    ];
}

function cs_get_course(PDO $pdo, int $courseId, bool $includeDeleted = false): ?array
{
    $sql = 'SELECT * FROM public.course_courses WHERE id = :id';
    if (!$includeDeleted) {
        $sql .= ' AND deleted_at IS NULL';
    }
    $st = $pdo->prepare($sql);
    $st->execute([':id' => $courseId]);
    $row = $st->fetch();
    if (!$row) {
        return null;
    }
    $course = cs_map_course_row($row);
    $course['currentVersion'] = null;
    if ($course['currentVersionId']) {
        $v = cs_get_version($pdo, $course['currentVersionId']);
        if ($v) {
            $course['currentVersion'] = [
                'id' => $v['id'],
                'versionNumber' => $v['versionNumber'],
                'status' => $v['status'],
                'shortDescription' => $v['shortDescription'],
                'coverUrl' => $v['coverUrl'],
                'publishedAt' => $v['publishedAt'],
                'requireFinalTest' => $v['requireFinalTest'],
                'sequentialProgress' => $v['sequentialProgress'],
            ];
        }
    }
    return $course;
}

function cs_get_version(PDO $pdo, int $versionId): ?array
{
    $st = $pdo->prepare('SELECT * FROM public.course_versions WHERE id = :id');
    $st->execute([':id' => $versionId]);
    $row = $st->fetch();
    return $row ? cs_map_version_row($row) : null;
}

/**
 * @return array{0: array, 1: array} [user, course]
 */
function cs_require_course_admin(PDO $pdo, int $courseId): array
{
    $user = auth_require_admin($pdo);
    $course = cs_get_course($pdo, $courseId);
    if (!$course) {
        jsonError(404, 'Курс не найден');
    }
    return [$user, $course];
}

function cs_version_editable(array $version): bool
{
    return ($version['status'] ?? '') === 'draft';
}

function cs_assert_version_editable(array $version): void
{
    if (!cs_version_editable($version)) {
        jsonError(409, 'Версия недоступна для редактирования (только черновик)');
    }
}

/**
 * Базовая сводка test_form для ссылок курса.
 */
function cs_test_form_summary(PDO $pdo, int $formId): ?array
{
    $st = $pdo->prepare(
        "SELECT f.id, f.title, f.status, f.list_no, f.use_passing_score, f.passing_score,
                (SELECT COUNT(*) FROM public.test_questions q WHERE q.form_id = f.id) AS question_count
         FROM public.test_forms f
         WHERE f.id = :id"
    );
    $st->execute([':id' => $formId]);
    $r = $st->fetch();
    if (!$r) {
        return null;
    }
    return [
        'id' => (int)$r['id'],
        'title' => (string)$r['title'],
        'status' => (string)$r['status'],
        'listNo' => $r['list_no'] !== null ? (int)$r['list_no'] : null,
        'usePassingScore' => cs_bool($r['use_passing_score']),
        'passingScore' => (int)$r['passing_score'],
        'questionCount' => (int)$r['question_count'],
    ];
}

function cs_map_test_link(array $row, ?array $formSummary): array
{
    return [
        'id' => (int)$row['id'],
        'courseVersionId' => (int)$row['course_version_id'],
        'topicId' => $row['topic_id'] !== null ? (int)$row['topic_id'] : null,
        'testFormId' => (int)$row['test_form_id'],
        'type' => (string)$row['type'],
        'isRequired' => cs_bool($row['is_required']),
        'sortOrder' => (int)$row['sort_order'],
        'createdAt' => $row['created_at'] ?? null,
        'updatedAt' => $row['updated_at'] ?? null,
        'form' => $formSummary,
        'questionCount' => $formSummary['questionCount'] ?? 0,
    ];
}

/**
 * Полное дерево версии: topics → materials + topic test; final test.
 */
function cs_assemble_version(PDO $pdo, int $versionId, bool $withContent = true): ?array
{
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        return null;
    }

    $tSt = $pdo->prepare(
        'SELECT * FROM public.course_topics
         WHERE course_version_id = :v AND deleted_at IS NULL
         ORDER BY sort_order, id'
    );
    $tSt->execute([':v' => $versionId]);
    $topicsRaw = $tSt->fetchAll();

    $mSt = $pdo->prepare(
        'SELECT * FROM public.course_materials
         WHERE topic_id = :t AND deleted_at IS NULL
         ORDER BY sort_order, id'
    );

    $linkSt = $pdo->prepare(
        "SELECT l.*, f.title AS form_title, f.status AS form_status, f.list_no,
                f.use_passing_score, f.passing_score,
                (SELECT COUNT(*) FROM public.test_questions q WHERE q.form_id = l.test_form_id) AS question_count
         FROM public.course_test_links l
         JOIN public.test_forms f ON f.id = l.test_form_id
         WHERE l.course_version_id = :v"
    );
    $linkSt->execute([':v' => $versionId]);
    $links = $linkSt->fetchAll();

    $topicLinks = [];
    $finalLink = null;
    foreach ($links as $l) {
        $summary = [
            'id' => (int)$l['test_form_id'],
            'title' => (string)$l['form_title'],
            'status' => (string)$l['form_status'],
            'listNo' => $l['list_no'] !== null ? (int)$l['list_no'] : null,
            'usePassingScore' => cs_bool($l['use_passing_score']),
            'passingScore' => (int)$l['passing_score'],
            'questionCount' => (int)$l['question_count'],
        ];
        $mapped = cs_map_test_link($l, $summary);
        if ($l['type'] === 'final') {
            $finalLink = $mapped;
        } elseif ($l['topic_id'] !== null) {
            $topicLinks[(int)$l['topic_id']] = $mapped;
        }
    }

    $topics = [];
    foreach ($topicsRaw as $t) {
        $tid = (int)$t['id'];
        $materials = [];
        $mSt->execute([':t' => $tid]);
        foreach ($mSt->fetchAll() as $m) {
            $mat = [
                'id' => (int)$m['id'],
                'topicId' => $tid,
                'type' => (string)$m['type'],
                'title' => (string)$m['title'],
                'description' => (string)($m['description'] ?? ''),
                'fileUrl' => $m['file_url'] ?? null,
                'externalUrl' => $m['external_url'] ?? null,
                'mimeType' => $m['mime_type'] ?? null,
                'fileSize' => $m['file_size'] !== null ? (int)$m['file_size'] : null,
                'originalFilename' => $m['original_filename'] ?? null,
                'sortOrder' => (int)$m['sort_order'],
                'isRequired' => cs_bool($m['is_required']),
                'minimumActiveSeconds' => (int)($m['minimum_active_seconds'] ?? 0),
                'createdAt' => $m['created_at'] ?? null,
                'updatedAt' => $m['updated_at'] ?? null,
            ];
            if ($withContent) {
                $mat['contentHtml'] = (string)($m['content_html'] ?? '');
            }
            $materials[] = $mat;
        }

        $topics[] = [
            'id' => $tid,
            'courseVersionId' => $versionId,
            'title' => (string)$t['title'],
            'description' => (string)($t['description'] ?? ''),
            'sortOrder' => (int)$t['sort_order'],
            'isRequired' => cs_bool($t['is_required']),
            'minimumActiveSeconds' => (int)($t['minimum_active_seconds'] ?? 0),
            'completionRule' => (string)($t['completion_rule'] ?? 'all_required_materials'),
            'createdAt' => $t['created_at'] ?? null,
            'updatedAt' => $t['updated_at'] ?? null,
            'materials' => $materials,
            'topicTest' => $topicLinks[$tid] ?? null,
        ];
    }

    $version['topics'] = $topics;
    $version['finalTest'] = $finalLink;
    return $version;
}

/**
 * Проверка, что у формы есть вопросы с корректными ответами (для readiness).
 * option-типы: ≥1 is_correct; text-типы: correct_value IS NOT NULL.
 *
 * @return string[] ошибки
 */
function cs_form_answer_errors(PDO $pdo, int $formId, string $label): array
{
    $errors = [];
    $qs = $pdo->prepare('SELECT id, type, correct_value FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
    $qs->execute([':f' => $formId]);
    $questions = $qs->fetchAll();
    if (!$questions) {
        $errors[] = "{$label}: нет вопросов";
        return $errors;
    }

    $optTypes = ['single', 'multiple', 'dropdown'];
    $valueTypes = ['text', 'textarea', 'yesno', 'number', 'scale', 'date'];
    $optCnt = $pdo->prepare(
        'SELECT COUNT(*) FROM public.test_options WHERE question_id = :q AND is_correct IS TRUE'
    );

    foreach ($questions as $i => $q) {
        $n = $i + 1;
        $type = (string)$q['type'];
        if (in_array($type, $optTypes, true)) {
            $optCnt->execute([':q' => (int)$q['id']]);
            if ((int)$optCnt->fetchColumn(0) < 1) {
                $errors[] = "{$label}: вопрос #{$n} без правильного варианта";
            }
        } elseif (in_array($type, $valueTypes, true)) {
            if ($q['correct_value'] === null || $q['correct_value'] === '') {
                $errors[] = "{$label}: вопрос #{$n} без правильного ответа";
            }
        }
    }
    return $errors;
}

function cs_readiness(PDO $pdo, int $versionId): array
{
    $errors = [];
    $warnings = [];
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        return ['ready' => false, 'errors' => ['Версия не найдена'], 'warnings' => []];
    }

    $assembled = cs_assemble_version($pdo, $versionId, false);
    $topics = $assembled['topics'] ?? [];
    if (!$topics) {
        $errors[] = 'Нет тем в версии курса';
    }

    foreach ($topics as $topic) {
        $title = $topic['title'] ?: ('Тема #' . $topic['id']);
        if (!$topic['isRequired']) {
            if (!$topic['materials'] && !$topic['topicTest']) {
                $warnings[] = "Тема «{$title}» необязательна и пуста";
            }
            continue;
        }

        $requiredMats = array_filter($topic['materials'], fn($m) => $m['isRequired']);
        if (!$requiredMats) {
            // обязательная тема без обязательных материалов — ошибка, если нет материалов вовсе
            if (!$topic['materials']) {
                $errors[] = "Обязательная тема «{$title}» без материалов";
            } else {
                $warnings[] = "Тема «{$title}»: нет обязательных материалов";
            }
        }

        $tt = $topic['topicTest'];
        if ($tt && $tt['isRequired']) {
            $label = "Промежуточный тест темы «{$title}»";
            $errors = array_merge($errors, cs_form_answer_errors($pdo, (int)$tt['testFormId'], $label));
        } elseif ($tt && !$tt['isRequired']) {
            $warnings[] = "Тема «{$title}»: тест необязателен";
        }
    }

    if ($version['requireFinalTest']) {
        $ft = $assembled['finalTest'] ?? null;
        if (!$ft) {
            $errors[] = 'Требуется итоговый тест, но он не создан';
        } else {
            $errors = array_merge($errors, cs_form_answer_errors($pdo, (int)$ft['testFormId'], 'Итоговый тест'));
            $form = $ft['form'] ?? null;
            if ($form && !empty($form['usePassingScore'])) {
                if (($form['passingScore'] ?? 0) <= 0) {
                    $errors[] = 'Итоговый тест: не задан проходной балл';
                }
            } elseif ($version['finalPassingScore'] !== null && (float)$version['finalPassingScore'] > 0) {
                // версия задаёт порог, а форма без use_passing_score — предупреждение
                $warnings[] = 'На версии задан finalPassingScore, но у итогового теста выключен usePassingScore';
            }
        }
    } elseif (!empty($assembled['finalTest'])) {
        $warnings[] = 'Итоговый тест есть, но requireFinalTest выключен';
    }

    return [
        'ready' => count($errors) === 0,
        'errors' => $errors,
        'warnings' => $warnings,
    ];
}

function cs_create_course(PDO $pdo, array $user, array $data): array
{
    $title = trim((string)($data['title'] ?? ''));
    if ($title === '') {
        jsonError(400, 'Не указано название курса');
    }

    $uid = (int)$user['id'];
    $pdo->beginTransaction();
    try {
        $cIns = $pdo->prepare(
            'INSERT INTO public.course_courses (owner_id, title, category)
             VALUES (:o, :t, :cat) RETURNING id'
        );
        $cIns->execute([
            ':o' => $uid,
            ':t' => $title,
            ':cat' => isset($data['category']) ? (string)$data['category'] : null,
        ]);
        $courseId = (int)$cIns->fetchColumn(0);

        $vIns = $pdo->prepare(
            'INSERT INTO public.course_versions (
                course_id, version_number, status, short_description, full_description,
                cover_url, sequential_progress, completion_rule, default_deadline_days,
                final_passing_score, require_final_test, created_by
             ) VALUES (
                :cid, 1, \'draft\', :sd, :fd, :cover, :seq, :cr, :ddl,
                :fps, :rft, :by
             ) RETURNING id'
        );
        $vIns->execute([
            ':cid' => $courseId,
            ':sd' => cs_sanitize_html((string)($data['shortDescription'] ?? $data['short_description'] ?? '')),
            ':fd' => cs_sanitize_html((string)($data['fullDescription'] ?? $data['full_description'] ?? '')),
            ':cover' => $data['coverUrl'] ?? $data['cover_url'] ?? null,
            ':seq' => cs_bool($data['sequentialProgress'] ?? $data['sequential_progress'] ?? true) ? 't' : 'f',
            ':cr' => (string)($data['completionRule'] ?? $data['completion_rule'] ?? 'all_required'),
            ':ddl' => isset($data['defaultDeadlineDays'])
                ? (int)$data['defaultDeadlineDays']
                : (isset($data['default_deadline_days']) ? (int)$data['default_deadline_days'] : null),
            ':fps' => isset($data['finalPassingScore'])
                ? $data['finalPassingScore']
                : ($data['final_passing_score'] ?? null),
            ':rft' => cs_bool($data['requireFinalTest'] ?? $data['require_final_test'] ?? false) ? 't' : 'f',
            ':by' => $uid,
        ]);
        $versionId = (int)$vIns->fetchColumn(0);

        $pdo->prepare(
            'UPDATE public.course_courses
             SET current_version_id = :v, updated_at = now()
             WHERE id = :id'
        )->execute([':v' => $versionId, ':id' => $courseId]);

        cs_audit($pdo, $uid, 'course.create', 'course', $courseId, [
            'versionId' => $versionId,
            'title' => $title,
        ]);

        $pdo->commit();
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        throw $e;
    }

    $assembled = cs_assemble_version($pdo, $versionId, true);
    $course = cs_get_course($pdo, $courseId);
    return [
        'course' => $course,
        'version' => $assembled,
    ];
}

/**
 * Глубокое копирование курса в новый черновик (версия 1) с копией тестов.
 */
function cs_duplicate_course(PDO $pdo, int $courseId, array $user): array
{
    $srcCourse = cs_get_course($pdo, $courseId);
    if (!$srcCourse || !$srcCourse['currentVersionId']) {
        jsonError(404, 'Курс или текущая версия не найдены');
    }
    $srcVersionId = (int)$srcCourse['currentVersionId'];
    $src = cs_assemble_version($pdo, $srcVersionId, true);
    if (!$src) {
        jsonError(404, 'Версия не найдена');
    }

    $uid = (int)$user['id'];
    $newTitle = 'Копия: ' . $srcCourse['title'];

    $pdo->beginTransaction();
    try {
        $cIns = $pdo->prepare(
            'INSERT INTO public.course_courses (owner_id, title, category)
             VALUES (:o, :t, :cat) RETURNING id'
        );
        $cIns->execute([
            ':o' => $uid,
            ':t' => $newTitle,
            ':cat' => $srcCourse['category'],
        ]);
        $newCourseId = (int)$cIns->fetchColumn(0);

        $vIns = $pdo->prepare(
            'INSERT INTO public.course_versions (
                course_id, version_number, status, short_description, full_description,
                cover_url, sequential_progress, completion_rule, default_deadline_days,
                final_passing_score, require_final_test, created_by
             ) VALUES (
                :cid, 1, \'draft\', :sd, :fd, :cover, :seq, :cr, :ddl, :fps, :rft, :by
             ) RETURNING id'
        );
        $vIns->execute([
            ':cid' => $newCourseId,
            ':sd' => $src['shortDescription'],
            ':fd' => $src['fullDescription'],
            ':cover' => $src['coverUrl'],
            ':seq' => $src['sequentialProgress'] ? 't' : 'f',
            ':cr' => $src['completionRule'],
            ':ddl' => $src['defaultDeadlineDays'],
            ':fps' => $src['finalPassingScore'],
            ':rft' => $src['requireFinalTest'] ? 't' : 'f',
            ':by' => $uid,
        ]);
        $newVersionId = (int)$vIns->fetchColumn(0);

        $pdo->prepare(
            'UPDATE public.course_courses SET current_version_id = :v, updated_at = now() WHERE id = :id'
        )->execute([':v' => $newVersionId, ':id' => $newCourseId]);

        $topicIns = $pdo->prepare(
            'INSERT INTO public.course_topics (
                course_version_id, title, description, sort_order, is_required,
                minimum_active_seconds, completion_rule
             ) VALUES (:v, :title, :desc, :ord, :req, :min, :cr) RETURNING id'
        );
        $matIns = $pdo->prepare(
            'INSERT INTO public.course_materials (
                topic_id, type, title, description, content_html, file_url, external_url,
                mime_type, file_size, original_filename, sort_order, is_required, minimum_active_seconds
             ) VALUES (
                :tid, :type, :title, :desc, :html, :fu, :eu, :mime, :fs, :ofn, :ord, :req, :min
             )'
        );
        $linkIns = $pdo->prepare(
            'INSERT INTO public.course_test_links (
                course_version_id, topic_id, test_form_id, type, is_required, sort_order
             ) VALUES (:v, :tid, :fid, :type, :req, :ord)'
        );

        foreach ($src['topics'] as $topic) {
            $topicIns->execute([
                ':v' => $newVersionId,
                ':title' => $topic['title'],
                ':desc' => $topic['description'],
                ':ord' => $topic['sortOrder'],
                ':req' => $topic['isRequired'] ? 't' : 'f',
                ':min' => $topic['minimumActiveSeconds'],
                ':cr' => $topic['completionRule'],
            ]);
            $newTopicId = (int)$topicIns->fetchColumn(0);

            foreach ($topic['materials'] as $m) {
                $matIns->execute([
                    ':tid' => $newTopicId,
                    ':type' => $m['type'],
                    ':title' => $m['title'],
                    ':desc' => $m['description'],
                    ':html' => $m['contentHtml'] ?? '',
                    ':fu' => $m['fileUrl'],
                    ':eu' => $m['externalUrl'],
                    ':mime' => $m['mimeType'],
                    ':fs' => $m['fileSize'],
                    ':ofn' => $m['originalFilename'],
                    ':ord' => $m['sortOrder'],
                    ':req' => $m['isRequired'] ? 't' : 'f',
                    ':min' => $m['minimumActiveSeconds'],
                ]);
            }

            if (!empty($topic['topicTest'])) {
                $newFormId = cs_clone_test_form($pdo, (int)$topic['topicTest']['testFormId'], $uid);
                $linkIns->execute([
                    ':v' => $newVersionId,
                    ':tid' => $newTopicId,
                    ':fid' => $newFormId,
                    ':type' => 'topic',
                    ':req' => $topic['topicTest']['isRequired'] ? 't' : 'f',
                    ':ord' => $topic['topicTest']['sortOrder'],
                ]);
            }
        }

        if (!empty($src['finalTest'])) {
            $newFormId = cs_clone_test_form($pdo, (int)$src['finalTest']['testFormId'], $uid);
            $linkIns->execute([
                ':v' => $newVersionId,
                ':tid' => null,
                ':fid' => $newFormId,
                ':type' => 'final',
                ':req' => $src['finalTest']['isRequired'] ? 't' : 'f',
                ':ord' => $src['finalTest']['sortOrder'],
            ]);
        }

        cs_audit($pdo, $uid, 'course.duplicate', 'course', $newCourseId, [
            'sourceCourseId' => $courseId,
            'sourceVersionId' => $srcVersionId,
            'versionId' => $newVersionId,
        ]);

        $pdo->commit();
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        throw $e;
    }

    return [
        'course' => cs_get_course($pdo, $newCourseId),
        'version' => cs_assemble_version($pdo, $newVersionId, true),
    ];
}

/**
 * Копия test_form + questions/options внутри текущей транзакции (без tf_persistForm —
 * у того свой begin/commit и нельзя вкладывать).
 */
function cs_clone_test_form(PDO $pdo, int $formId, int $ownerId): int
{
    $st = $pdo->prepare('SELECT * FROM public.test_forms WHERE id = :id');
    $st->execute([':id' => $formId]);
    $f = $st->fetch();
    if (!$f) {
        throw new RuntimeException('Тест для копирования не найден');
    }

    $ins = $pdo->prepare(
        "INSERT INTO public.test_forms (
            status, owner_id, kind, visibility, title, description, completion_message,
            shuffle, shuffle_options, show_progress, free_navigation, anonymous,
            allow_change_answer, live_results, allow_revote, notify_creator,
            use_passing_score, passing_score, show_correct_answers, restrict_by_ofo,
            use_time_limit, time_limit_sec, limit_attempts, attempts,
            use_start, starts_at, use_end, ends_at, show_result,
            access_by_link, link_access
         ) VALUES (
            'draft', :owner, :kind, 'private', :title, :desc, :cm,
            :sh, :sho, :sp, :fn, :anon,
            :aca, :lr, :ar, :nc,
            :ups, :ps, :sca, :rbo,
            :utl, :tls, :la, :att,
            :us, :sa, :ue, :ea, :sr,
            false, :lac
         ) RETURNING id"
    );
    $b = static fn($v) => cs_bool($v) ? 't' : 'f';
    $ins->execute([
        ':owner' => $ownerId,
        ':kind' => $f['kind'] ?: 'test',
        ':title' => $f['title'],
        ':desc' => $f['description'],
        ':cm' => $f['completion_message'],
        ':sh' => $b($f['shuffle']),
        ':sho' => $b($f['shuffle_options']),
        ':sp' => $b($f['show_progress']),
        ':fn' => $b($f['free_navigation']),
        ':anon' => $b($f['anonymous']),
        ':aca' => $b($f['allow_change_answer']),
        ':lr' => $b($f['live_results']),
        ':ar' => $b($f['allow_revote']),
        ':nc' => $b($f['notify_creator']),
        ':ups' => $b($f['use_passing_score']),
        ':ps' => (int)$f['passing_score'],
        ':sca' => $b($f['show_correct_answers']),
        ':rbo' => $b($f['restrict_by_ofo']),
        ':utl' => $b($f['use_time_limit']),
        ':tls' => $f['time_limit_sec'],
        ':la' => $b($f['limit_attempts']),
        ':att' => (int)$f['attempts'],
        ':us' => $b($f['use_start']),
        ':sa' => $f['starts_at'],
        ':ue' => $b($f['use_end']),
        ':ea' => $f['ends_at'],
        ':sr' => $f['show_result'],
        ':lac' => $f['link_access'] ?? 'any',
    ]);
    $newId = (int)$ins->fetchColumn(0);

    $qSt = $pdo->prepare('SELECT * FROM public.test_questions WHERE form_id = :f ORDER BY position, id');
    $qSt->execute([':f' => $formId]);
    $qIns = $pdo->prepare(
        'INSERT INTO public.test_questions (
            form_id, position, type, title, hint, required,
            scale_min, scale_max, scale_min_label, scale_max_label, correct_value
         ) VALUES (
            :f, :pos, :type, :title, :hint, :req, :smin, :smax, :slmin, :slmax, :cv
         ) RETURNING id'
    );
    $oSt = $pdo->prepare('SELECT * FROM public.test_options WHERE question_id = :q ORDER BY position, id');
    $oIns = $pdo->prepare(
        'INSERT INTO public.test_options (question_id, position, text, is_correct)
         VALUES (:q, :pos, :text, :ic)'
    );

    foreach ($qSt->fetchAll() as $q) {
        $qIns->execute([
            ':f' => $newId,
            ':pos' => (int)$q['position'],
            ':type' => $q['type'],
            ':title' => $q['title'],
            ':hint' => $q['hint'],
            ':req' => $b($q['required']),
            ':smin' => $q['scale_min'],
            ':smax' => $q['scale_max'],
            ':slmin' => $q['scale_min_label'],
            ':slmax' => $q['scale_max_label'],
            ':cv' => $q['correct_value'],
        ]);
        $newQid = (int)$qIns->fetchColumn(0);
        $oSt->execute([':q' => (int)$q['id']]);
        foreach ($oSt->fetchAll() as $o) {
            $oIns->execute([
                ':q' => $newQid,
                ':pos' => (int)$o['position'],
                ':text' => $o['text'],
                ':ic' => $b($o['is_correct']),
            ]);
        }
    }

    return $newId;
}

function cs_create_topic_test(PDO $pdo, int $topicId, array $user, array $testData = []): array
{
    $st = $pdo->prepare(
        'SELECT t.*, v.status AS version_status, v.id AS version_id
         FROM public.course_topics t
         JOIN public.course_versions v ON v.id = t.course_version_id
         WHERE t.id = :id AND t.deleted_at IS NULL'
    );
    $st->execute([':id' => $topicId]);
    $topic = $st->fetch();
    if (!$topic) {
        jsonError(404, 'Тема не найдена');
    }
    if ($topic['version_status'] !== 'draft') {
        jsonError(409, 'Версия недоступна для редактирования (только черновик)');
    }

    $exist = $pdo->prepare(
        "SELECT id FROM public.course_test_links WHERE topic_id = :t AND type = 'topic' LIMIT 1"
    );
    $exist->execute([':t' => $topicId]);
    if ($exist->fetch()) {
        jsonError(409, 'У темы уже есть промежуточный тест');
    }

    $uid = (int)$user['id'];
    $payload = array_merge([
        'kind' => 'test',
        'visibility' => 'private',
        'title' => (string)($testData['title'] ?? ('Тест: ' . $topic['title'])),
        'questions' => $testData['questions'] ?? [],
    ], $testData);
    $payload['kind'] = 'test';
    $payload['visibility'] = 'private';
    unset($payload['id']);

    // tf_persistForm ведёт свою транзакцию — вызываем вне внешней
    $formId = tf_persistForm($pdo, $payload, $uid);

    $isRequired = cs_bool($testData['isRequired'] ?? $testData['is_required'] ?? true);
    $sortOrder = (int)($testData['sortOrder'] ?? $testData['sort_order'] ?? 0);

    $ins = $pdo->prepare(
        "INSERT INTO public.course_test_links (
            course_version_id, topic_id, test_form_id, type, is_required, sort_order
         ) VALUES (:v, :tid, :fid, 'topic', :req, :ord)
         RETURNING *"
    );
    $ins->execute([
        ':v' => (int)$topic['version_id'],
        ':tid' => $topicId,
        ':fid' => $formId,
        ':req' => $isRequired ? 't' : 'f',
        ':ord' => $sortOrder,
    ]);
    $link = $ins->fetch();

    cs_audit($pdo, $uid, 'course.topic_test.create', 'course_test_link', (int)$link['id'], [
        'topicId' => $topicId,
        'testFormId' => $formId,
    ]);

    return cs_map_test_link($link, cs_test_form_summary($pdo, $formId));
}

function cs_create_final_test(PDO $pdo, int $versionId, array $user, array $testData = []): array
{
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        jsonError(404, 'Версия не найдена');
    }
    cs_assert_version_editable($version);

    $exist = $pdo->prepare(
        "SELECT id FROM public.course_test_links WHERE course_version_id = :v AND type = 'final' LIMIT 1"
    );
    $exist->execute([':v' => $versionId]);
    if ($exist->fetch()) {
        jsonError(409, 'Итоговый тест уже создан');
    }

    $uid = (int)$user['id'];
    $payload = array_merge([
        'kind' => 'test',
        'visibility' => 'private',
        'title' => (string)($testData['title'] ?? 'Итоговый тест'),
        'questions' => $testData['questions'] ?? [],
    ], $testData);
    $payload['kind'] = 'test';
    $payload['visibility'] = 'private';
    unset($payload['id']);

    $formId = tf_persistForm($pdo, $payload, $uid);

    $isRequired = cs_bool($testData['isRequired'] ?? $testData['is_required'] ?? true);
    $sortOrder = (int)($testData['sortOrder'] ?? $testData['sort_order'] ?? 0);

    $ins = $pdo->prepare(
        "INSERT INTO public.course_test_links (
            course_version_id, topic_id, test_form_id, type, is_required, sort_order
         ) VALUES (:v, NULL, :fid, 'final', :req, :ord)
         RETURNING *"
    );
    $ins->execute([
        ':v' => $versionId,
        ':fid' => $formId,
        ':req' => $isRequired ? 't' : 'f',
        ':ord' => $sortOrder,
    ]);
    $link = $ins->fetch();

    // При создании итогового — включаем флаг на версии, если ещё выключен
    if (!$version['requireFinalTest']) {
        $pdo->prepare(
            'UPDATE public.course_versions SET require_final_test = true, updated_at = now() WHERE id = :id'
        )->execute([':id' => $versionId]);
    }

    cs_audit($pdo, $uid, 'course.final_test.create', 'course_test_link', (int)$link['id'], [
        'versionId' => $versionId,
        'testFormId' => $formId,
    ]);

    return cs_map_test_link($link, cs_test_form_summary($pdo, $formId));
}

/**
 * Публикация версии + связанных test_forms (list_no как в tests_publish.php).
 */
function cs_publish_version(PDO $pdo, int $versionId, array $user): array
{
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        jsonError(404, 'Версия не найдена');
    }
    if ($version['status'] === 'published') {
        jsonError(409, 'Версия уже опубликована');
    }
    if ($version['status'] === 'archived') {
        jsonError(409, 'Архивная версия не может быть опубликована');
    }

    $ready = cs_readiness($pdo, $versionId);
    if (!$ready['ready']) {
        jsonError(422, 'Версия не готова к публикации: ' . implode('; ', $ready['errors']));
    }

    $uid = (int)$user['id'];
    $pdo->beginTransaction();
    try {
        $forms = $pdo->prepare(
            'SELECT DISTINCT test_form_id FROM public.course_test_links WHERE course_version_id = :v'
        );
        $forms->execute([':v' => $versionId]);
        $formIds = array_map(static fn($r) => (int)$r['test_form_id'], $forms->fetchAll());

        $pubForm = $pdo->prepare(
            "UPDATE public.test_forms
             SET status = 'published',
                 published_at = COALESCE(published_at, now()),
                 list_no = COALESCE(list_no, nextval('public.test_forms_list_no_seq')),
                 updated_at = now()
             WHERE id = :id"
        );
        foreach ($formIds as $fid) {
            $pubForm->execute([':id' => $fid]);
        }

        $pdo->prepare(
            "UPDATE public.course_versions
             SET status = 'published', published_at = COALESCE(published_at, now()), updated_at = now()
             WHERE id = :id"
        )->execute([':id' => $versionId]);

        $pdo->prepare(
            'UPDATE public.course_courses
             SET current_version_id = :v, updated_at = now()
             WHERE id = :cid'
        )->execute([':v' => $versionId, ':cid' => $version['courseId']]);

        cs_audit($pdo, $uid, 'course.version.publish', 'course_version', $versionId, [
            'courseId' => $version['courseId'],
            'formIds' => $formIds,
            'warnings' => $ready['warnings'],
        ]);

        $pdo->commit();
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        throw $e;
    }

    return cs_assemble_version($pdo, $versionId, true) ?? [];
}

function cs_enrollment_row(PDO $pdo, int $enrollmentId): ?array
{
    $st = $pdo->prepare('SELECT * FROM public.course_enrollments WHERE id = :id');
    $st->execute([':id' => $enrollmentId]);
    $r = $st->fetch();
    return $r ?: null;
}

function cs_ensure_topic_progress_rows(PDO $pdo, int $enrollmentId, int $versionId): void
{
    $enr = cs_enrollment_row($pdo, $enrollmentId);
    if (!$enr) {
        return;
    }
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        return;
    }

    $topics = $pdo->prepare(
        'SELECT id FROM public.course_topics
         WHERE course_version_id = :v AND deleted_at IS NULL
         ORDER BY sort_order, id'
    );
    $topics->execute([':v' => $versionId]);
    $ids = array_map(static fn($r) => (int)$r['id'], $topics->fetchAll());
    if (!$ids) {
        return;
    }

    $ins = $pdo->prepare(
        "INSERT INTO public.course_topic_progress (enrollment_id, topic_id, status)
         VALUES (:e, :t, :s)
         ON CONFLICT (enrollment_id, topic_id) DO NOTHING"
    );

    $sequential = $version['sequentialProgress'];
    foreach ($ids as $i => $tid) {
        $status = (!$sequential || $i === 0) ? 'available' : 'locked';
        $ins->execute([':e' => $enrollmentId, ':t' => $tid, ':s' => $status]);
    }
}

/**
 * Пересчёт locked/available по sequential_progress.
 * completed / in_progress не понижаем до locked без нужды; completed не трогаем.
 */
function cs_recalculate_locks(PDO $pdo, int $enrollmentId): void
{
    $enr = cs_enrollment_row($pdo, $enrollmentId);
    if (!$enr) {
        return;
    }
    $versionId = (int)$enr['course_version_id'];
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        return;
    }

    cs_ensure_topic_progress_rows($pdo, $enrollmentId, $versionId);

    $topics = $pdo->prepare(
        'SELECT t.id, COALESCE(p.status, \'locked\') AS status, p.id AS progress_id
         FROM public.course_topics t
         LEFT JOIN public.course_topic_progress p
           ON p.topic_id = t.id AND p.enrollment_id = :e
         WHERE t.course_version_id = :v AND t.deleted_at IS NULL
         ORDER BY t.sort_order, t.id'
    );
    $topics->execute([':e' => $enrollmentId, ':v' => $versionId]);
    $rows = $topics->fetchAll();

    $upd = $pdo->prepare(
        'UPDATE public.course_topic_progress
         SET status = :s, updated_at = now()
         WHERE enrollment_id = :e AND topic_id = :t'
    );

    $sequential = $version['sequentialProgress'];
    $prevCompleted = true;

    foreach ($rows as $r) {
        $tid = (int)$r['id'];
        $status = (string)$r['status'];

        if ($status === 'completed') {
            $prevCompleted = true;
            continue;
        }

        if (!$sequential || $prevCompleted) {
            if ($status === 'locked') {
                $upd->execute([':s' => 'available', ':e' => $enrollmentId, ':t' => $tid]);
            }
            $prevCompleted = false;
        } else {
            if ($status !== 'locked') {
                // незавершённую тему блокируем снова, если предыдущая не пройдена
                $upd->execute([':s' => 'locked', ':e' => $enrollmentId, ':t' => $tid]);
            }
            $prevCompleted = false;
        }
    }
}

/**
 * Прошёл ли обязательный тест ссылки (passed=true либо completed без порога).
 */
function cs_test_link_passed(PDO $pdo, int $enrollmentId, int $courseTestLinkId): bool
{
    $st = $pdo->prepare(
        "SELECT a.passed, a.status, f.use_passing_score
         FROM public.course_test_attempt_links tal
         JOIN public.test_attempts a ON a.id = tal.test_attempt_id
         JOIN public.course_test_links l ON l.id = tal.course_test_link_id
         JOIN public.test_forms f ON f.id = l.test_form_id
         WHERE tal.enrollment_id = :e AND tal.course_test_link_id = :l
           AND a.status = 'completed'
         ORDER BY a.finished_at DESC NULLS LAST, a.id DESC
         LIMIT 1"
    );
    $st->execute([':e' => $enrollmentId, ':l' => $courseTestLinkId]);
    $r = $st->fetch();
    if (!$r) {
        return false;
    }
    if (cs_bool($r['use_passing_score'])) {
        return cs_bool($r['passed']);
    }
    return true;
}

function cs_check_topic_complete(PDO $pdo, int $enrollmentId, int $topicId): bool
{
    $st = $pdo->prepare(
        'SELECT t.*, p.active_seconds AS progress_active
         FROM public.course_topics t
         LEFT JOIN public.course_topic_progress p
           ON p.topic_id = t.id AND p.enrollment_id = :e
         WHERE t.id = :t AND t.deleted_at IS NULL'
    );
    $st->execute([':e' => $enrollmentId, ':t' => $topicId]);
    $topic = $st->fetch();
    if (!$topic) {
        return false;
    }

    // обязательные материалы
    $mats = $pdo->prepare(
        "SELECT m.id, m.is_required, m.minimum_active_seconds,
                COALESCE(mp.status, 'not_started') AS mp_status,
                COALESCE(mp.active_seconds, 0) AS mp_active
         FROM public.course_materials m
         LEFT JOIN public.course_material_progress mp
           ON mp.material_id = m.id AND mp.enrollment_id = :e
         WHERE m.topic_id = :t AND m.deleted_at IS NULL"
    );
    $mats->execute([':e' => $enrollmentId, ':t' => $topicId]);
    foreach ($mats->fetchAll() as $m) {
        if (!cs_bool($m['is_required'])) {
            continue;
        }
        if ($m['mp_status'] !== 'completed') {
            return false;
        }
        if ((int)$m['minimum_active_seconds'] > 0
            && (int)$m['mp_active'] < (int)$m['minimum_active_seconds']) {
            return false;
        }
    }

    // минимальное время по теме
    $minTopic = (int)($topic['minimum_active_seconds'] ?? 0);
    if ($minTopic > 0 && (int)($topic['progress_active'] ?? 0) < $minTopic) {
        return false;
    }

    // обязательный промежуточный тест
    $link = $pdo->prepare(
        "SELECT id, is_required FROM public.course_test_links
         WHERE topic_id = :t AND type = 'topic' LIMIT 1"
    );
    $link->execute([':t' => $topicId]);
    $tl = $link->fetch();
    if ($tl && cs_bool($tl['is_required'])) {
        if (!cs_test_link_passed($pdo, $enrollmentId, (int)$tl['id'])) {
            return false;
        }
    }

    return true;
}

function cs_next_action(PDO $pdo, int $enrollmentId): array
{
    $enr = cs_enrollment_row($pdo, $enrollmentId);
    if (!$enr) {
        return ['type' => 'unknown', 'label' => 'Запись не найдена'];
    }
    if (in_array($enr['status'], ['completed', 'cancelled'], true)) {
        return ['type' => 'done', 'label' => 'Курс завершён'];
    }
    if ($enr['status'] === 'failed') {
        return ['type' => 'failed', 'label' => 'Курс не сдан'];
    }
    if ($enr['status'] === 'overdue') {
        return ['type' => 'overdue', 'label' => 'Просрочен дедлайн'];
    }

    $versionId = (int)$enr['course_version_id'];
    cs_recalculate_locks($pdo, $enrollmentId);
    $assembled = cs_assemble_version($pdo, $versionId, false);
    if (!$assembled) {
        return ['type' => 'unknown', 'label' => 'Версия не найдена'];
    }

    $progSt = $pdo->prepare(
        'SELECT topic_id, status, last_material_id FROM public.course_topic_progress WHERE enrollment_id = :e'
    );
    $progSt->execute([':e' => $enrollmentId]);
    $progress = [];
    foreach ($progSt->fetchAll() as $p) {
        $progress[(int)$p['topic_id']] = $p;
    }

    $matProg = $pdo->prepare(
        "SELECT material_id, status FROM public.course_material_progress WHERE enrollment_id = :e"
    );
    $matProg->execute([':e' => $enrollmentId]);
    $mp = [];
    foreach ($matProg->fetchAll() as $r) {
        $mp[(int)$r['material_id']] = $r['status'];
    }

    foreach ($assembled['topics'] as $topic) {
        $tid = (int)$topic['id'];
        $ps = $progress[$tid]['status'] ?? 'locked';
        if ($ps === 'completed') {
            continue;
        }
        if ($ps === 'locked') {
            return [
                'type' => 'locked',
                'topicId' => $tid,
                'label' => 'Тема «' . $topic['title'] . '» ещё недоступна',
            ];
        }

        // незавершённый обязательный материал
        foreach ($topic['materials'] as $m) {
            if (!$m['isRequired']) {
                continue;
            }
            $st = $mp[(int)$m['id']] ?? 'not_started';
            if ($st !== 'completed') {
                return [
                    'type' => 'material',
                    'topicId' => $tid,
                    'materialId' => (int)$m['id'],
                    'label' => 'Изучить: ' . $m['title'],
                ];
            }
        }

        // промежуточный тест
        if (!empty($topic['topicTest']) && $topic['topicTest']['isRequired']) {
            $lid = (int)$topic['topicTest']['id'];
            if (!cs_test_link_passed($pdo, $enrollmentId, $lid)) {
                return [
                    'type' => 'topic_test',
                    'topicId' => $tid,
                    'courseTestLinkId' => $lid,
                    'label' => 'Пройти тест темы «' . $topic['title'] . '»',
                ];
            }
        }

        // время / завершение темы ещё не отмечено
        if (!cs_check_topic_complete($pdo, $enrollmentId, $tid)) {
            return [
                'type' => 'topic',
                'topicId' => $tid,
                'label' => 'Завершите тему «' . $topic['title'] . '»',
            ];
        }

        return [
            'type' => 'complete_topic',
            'topicId' => $tid,
            'label' => 'Отметить тему «' . $topic['title'] . '» завершённой',
        ];
    }

    // итоговый тест
    $ft = $assembled['finalTest'] ?? null;
    if ($assembled['requireFinalTest'] && $ft) {
        if (!cs_test_link_passed($pdo, $enrollmentId, (int)$ft['id'])) {
            return [
                'type' => 'final_test',
                'courseTestLinkId' => (int)$ft['id'],
                'label' => 'Пройти итоговый тест',
            ];
        }
    }

    return ['type' => 'complete_course', 'label' => 'Завершить курс'];
}

function cs_enrollment_progress(PDO $pdo, int $enrollmentId): array
{
    $enr = cs_enrollment_row($pdo, $enrollmentId);
    if (!$enr) {
        return [
            'percent' => 0,
            'topicsCompleted' => 0,
            'topicsTotal' => 0,
            'nextAction' => ['type' => 'unknown', 'label' => 'Запись не найдена'],
        ];
    }

    $versionId = (int)$enr['course_version_id'];
    cs_ensure_topic_progress_rows($pdo, $enrollmentId, $versionId);

    $st = $pdo->prepare(
        "SELECT
            COUNT(*) FILTER (WHERE t.is_required IS TRUE) AS total_req,
            COUNT(*) FILTER (
                WHERE t.is_required IS TRUE AND p.status = 'completed'
            ) AS done_req
         FROM public.course_topics t
         LEFT JOIN public.course_topic_progress p
           ON p.topic_id = t.id AND p.enrollment_id = :e
         WHERE t.course_version_id = :v AND t.deleted_at IS NULL"
    );
    $st->execute([':e' => $enrollmentId, ':v' => $versionId]);
    $row = $st->fetch() ?: ['total_req' => 0, 'done_req' => 0];
    $total = (int)$row['total_req'];
    $done = (int)$row['done_req'];

    // финальный тест как +1 шаг, если обязателен
    $version = cs_get_version($pdo, $versionId);
    $stepsTotal = $total;
    $stepsDone = $done;
    if ($version && $version['requireFinalTest']) {
        $stepsTotal++;
        $ft = $pdo->prepare(
            "SELECT id FROM public.course_test_links
             WHERE course_version_id = :v AND type = 'final' LIMIT 1"
        );
        $ft->execute([':v' => $versionId]);
        $fl = $ft->fetch();
        if ($fl && cs_test_link_passed($pdo, $enrollmentId, (int)$fl['id'])) {
            $stepsDone++;
        }
    }

    $percent = $stepsTotal > 0 ? (int)round($stepsDone / $stepsTotal * 100) : 0;
    if ($enr['status'] === 'completed') {
        $percent = 100;
    }

    return [
        'percent' => $percent,
        'topicsCompleted' => $done,
        'topicsTotal' => $total,
        'nextAction' => cs_next_action($pdo, $enrollmentId),
    ];
}

/**
 * Если все обязательные темы + итоговый тест готовы — создать course_completions.
 */
function cs_try_complete_enrollment(PDO $pdo, int $enrollmentId): ?array
{
    $enr = cs_enrollment_row($pdo, $enrollmentId);
    if (!$enr) {
        return null;
    }
    if (in_array($enr['status'], ['completed', 'cancelled'], true)) {
        $ex = $pdo->prepare('SELECT * FROM public.course_completions WHERE enrollment_id = :e ORDER BY id DESC LIMIT 1');
        $ex->execute([':e' => $enrollmentId]);
        $row = $ex->fetch();
        return $row ?: null;
    }

    $versionId = (int)$enr['course_version_id'];
    $version = cs_get_version($pdo, $versionId);
    if (!$version) {
        return null;
    }

    // все обязательные темы
    $topics = $pdo->prepare(
        'SELECT id FROM public.course_topics
         WHERE course_version_id = :v AND deleted_at IS NULL AND is_required IS TRUE
         ORDER BY sort_order, id'
    );
    $topics->execute([':v' => $versionId]);
    foreach ($topics->fetchAll() as $t) {
        $tid = (int)$t['id'];
        if (!cs_check_topic_complete($pdo, $enrollmentId, $tid)) {
            return null;
        }
        // убедимся, что прогресс отмечен completed
        $pdo->prepare(
            "UPDATE public.course_topic_progress
             SET status = 'completed',
                 completed_at = COALESCE(completed_at, now()),
                 updated_at = now()
             WHERE enrollment_id = :e AND topic_id = :t AND status <> 'completed'"
        )->execute([':e' => $enrollmentId, ':t' => $tid]);
    }

    $finalScore = null;
    $finalPassed = true;
    if ($version['requireFinalTest']) {
        $ft = $pdo->prepare(
            "SELECT id FROM public.course_test_links
             WHERE course_version_id = :v AND type = 'final' LIMIT 1"
        );
        $ft->execute([':v' => $versionId]);
        $fl = $ft->fetch();
        if (!$fl || !cs_test_link_passed($pdo, $enrollmentId, (int)$fl['id'])) {
            return null;
        }
        $scoreSt = $pdo->prepare(
            "SELECT a.score, a.passed
             FROM public.course_test_attempt_links tal
             JOIN public.test_attempts a ON a.id = tal.test_attempt_id
             WHERE tal.enrollment_id = :e AND tal.course_test_link_id = :l
               AND a.status = 'completed'
             ORDER BY a.finished_at DESC NULLS LAST, a.id DESC
             LIMIT 1"
        );
        $scoreSt->execute([':e' => $enrollmentId, ':l' => (int)$fl['id']]);
        $sr = $scoreSt->fetch();
        if ($sr) {
            $finalScore = $sr['score'] !== null ? (float)$sr['score'] : null;
            if ($sr['passed'] !== null) {
                $finalPassed = cs_bool($sr['passed']);
            }
        }
    }

    $snapshot = cs_build_completion_snapshot($pdo, $enrollmentId, $enr, $version, $finalScore);

    $totalActive = (int)($snapshot['totalActiveSeconds'] ?? 0);
    $userId = (int)$enr['user_id'];
    $courseId = (int)$version['courseId'];

    $pdo->beginTransaction();
    try {
        $cnt = $pdo->prepare(
            'SELECT COALESCE(MAX(completion_number), 0) + 1
             FROM public.course_completions
             WHERE user_id = :u AND course_id = :c'
        );
        $cnt->execute([':u' => $userId, ':c' => $courseId]);
        $completionNumber = (int)$cnt->fetchColumn(0);

        $ins = $pdo->prepare(
            'INSERT INTO public.course_completions (
                enrollment_id, user_id, course_id, course_version_id, completion_number,
                assigned_at, started_at, completed_at, total_active_seconds,
                final_score, passed, result_snapshot
             ) VALUES (
                :e, :u, :c, :v, :n, :aa, :sa, now(), :tas, :fs, :passed, :snap::jsonb
             ) RETURNING *'
        );
        $ins->execute([
            ':e' => $enrollmentId,
            ':u' => $userId,
            ':c' => $courseId,
            ':v' => $versionId,
            ':n' => $completionNumber,
            ':aa' => $enr['assigned_at'],
            ':sa' => $enr['started_at'],
            ':tas' => $totalActive,
            ':fs' => $finalScore,
            ':passed' => $finalPassed ? 't' : 'f',
            ':snap' => json_encode($snapshot, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
        ]);
        $completion = $ins->fetch();

        $pdo->prepare(
            "UPDATE public.course_enrollments
             SET status = 'completed',
                 completed_at = now(),
                 final_score = :fs,
                 updated_at = now()
             WHERE id = :id"
        )->execute([':fs' => $finalScore, ':id' => $enrollmentId]);

        cs_audit($pdo, $userId, 'course.enrollment.complete', 'course_enrollment', $enrollmentId, [
            'completionId' => (int)$completion['id'],
            'finalScore' => $finalScore,
        ]);

        $pdo->commit();
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        throw $e;
    }

    return $completion;
}

function cs_build_completion_snapshot(
    PDO $pdo,
    int $enrollmentId,
    array $enr,
    array $version,
    ?float $finalScore
): array {
    $course = cs_get_course($pdo, (int)$version['courseId'], true);
    $userId = (int)$enr['user_id'];

    $uSt = $pdo->prepare(
        'SELECT id, firstname, surname, lastname, role, ofo FROM public.user_info WHERE id = :id'
    );
    $uSt->execute([':id' => $userId]);
    $u = $uSt->fetch() ?: [];

    $fio = trim(implode(' ', array_filter([
        $u['surname'] ?? '',
        $u['firstname'] ?? '',
        $u['lastname'] ?? '',
    ])));

    $ofoName = null;
    $ofoId = null;
    $rawOfo = $u['ofo'] ?? null;
    if ($rawOfo !== null && $rawOfo !== '' && $rawOfo !== '-1' && is_numeric($rawOfo)) {
        $ofoId = (int)$rawOfo;
        $oSt = $pdo->prepare('SELECT name FROM public.ofo_unit WHERE id = :id');
        $oSt->execute([':id' => $ofoId]);
        $ofoName = $oSt->fetchColumn(0) ?: null;
    }

    $topicsOut = [];
    $totalActive = 0;

    $tSt = $pdo->prepare(
        'SELECT t.id, t.title, COALESCE(p.active_seconds, 0) AS active_seconds
         FROM public.course_topics t
         LEFT JOIN public.course_topic_progress p
           ON p.topic_id = t.id AND p.enrollment_id = :e
         WHERE t.course_version_id = :v AND t.deleted_at IS NULL
         ORDER BY t.sort_order, t.id'
    );
    $tSt->execute([':e' => $enrollmentId, ':v' => (int)$version['id']]);

    $linkSt = $pdo->prepare(
        "SELECT id FROM public.course_test_links WHERE topic_id = :t AND type = 'topic' LIMIT 1"
    );
    $attSt = $pdo->prepare(
        "SELECT a.score, a.passed
         FROM public.course_test_attempt_links tal
         JOIN public.test_attempts a ON a.id = tal.test_attempt_id
         WHERE tal.enrollment_id = :e AND tal.course_test_link_id = :l
           AND a.status = 'completed'
         ORDER BY a.finished_at DESC NULLS LAST, a.id DESC"
    );
    $cntAtt = $pdo->prepare(
        "SELECT COUNT(*)
         FROM public.course_test_attempt_links tal
         JOIN public.test_attempts a ON a.id = tal.test_attempt_id
         WHERE tal.enrollment_id = :e AND tal.course_test_link_id = :l
           AND a.status = 'completed'"
    );

    foreach ($tSt->fetchAll() as $t) {
        $tid = (int)$t['id'];
        $active = (int)$t['active_seconds'];
        $totalActive += $active;
        $topicScore = null;
        $topicAttempts = 0;
        $linkSt->execute([':t' => $tid]);
        $tl = $linkSt->fetch();
        if ($tl) {
            $lid = (int)$tl['id'];
            $cntAtt->execute([':e' => $enrollmentId, ':l' => $lid]);
            $topicAttempts = (int)$cntAtt->fetchColumn(0);
            $attSt->execute([':e' => $enrollmentId, ':l' => $lid]);
            $best = $attSt->fetch();
            if ($best && $best['score'] !== null) {
                $topicScore = (float)$best['score'];
            }
        }
        $topicsOut[] = [
            'topicId' => $tid,
            'title' => (string)$t['title'],
            'activeSeconds' => $active,
            'testScore' => $topicScore,
            'testAttempts' => $topicAttempts,
        ];
    }

    $finalAttempts = 0;
    $ft = $pdo->prepare(
        "SELECT id FROM public.course_test_links
         WHERE course_version_id = :v AND type = 'final' LIMIT 1"
    );
    $ft->execute([':v' => (int)$version['id']]);
    $fl = $ft->fetch();
    if ($fl) {
        $cntAtt->execute([':e' => $enrollmentId, ':l' => (int)$fl['id']]);
        $finalAttempts = (int)$cntAtt->fetchColumn(0);
    }

    return [
        'courseTitle' => $course['title'] ?? '',
        'versionNumber' => $version['versionNumber'],
        'userFio' => $fio,
        'role' => (string)($u['role'] ?? ''),
        'ofoId' => $ofoId,
        'ofoName' => $ofoName,
        'topics' => $topicsOut,
        'finalScore' => $finalScore,
        'finalAttempts' => $finalAttempts,
        'totalActiveSeconds' => $totalActive,
    ];
}

function cs_mark_overdue(PDO $pdo): void
{
    $pdo->exec(
        "UPDATE public.course_enrollments
         SET status = 'overdue', updated_at = now()
         WHERE deadline_at IS NOT NULL
           AND deadline_at < now()
           AND status IN ('not_started', 'in_progress')"
    );
}

/** База загрузок: /var/lib/... или локальный fallback. */
function cs_uploads_root(): string
{
    static $root = null;
    if ($root !== null) {
        return $root;
    }
    $preferred = '/var/lib/corporate-app/uploads/courses';
    if (@is_dir($preferred) || @mkdir($preferred, 0755, true)) {
        if (is_writable($preferred)) {
            return $root = $preferred;
        }
    }
    $fallback = realpath(__DIR__ . '/..') . DIRECTORY_SEPARATOR . 'uploads' . DIRECTORY_SEPARATOR . 'courses';
    if (!is_dir($fallback)) {
        @mkdir($fallback, 0755, true);
    }
    return $root = $fallback;
}

/**
 * Каталог для courseId; возвращает [absDir, relativeKeyPrefix]
 * relativeKeyPrefix: "courses/{courseId}"
 */
function cs_course_upload_dir(int $courseId): array
{
    $rel = 'courses/' . $courseId;
    $abs = cs_uploads_root() . DIRECTORY_SEPARATOR . $courseId;
    if (!is_dir($abs)) {
        mkdir($abs, 0755, true);
    }
    return [$abs, $rel];
}

function cs_resolve_version_id(PDO $pdo, array $body): int
{
    $versionId = (int)($body['versionId'] ?? 0);
    if ($versionId > 0) {
        return $versionId;
    }
    $courseId = (int)($body['courseId'] ?? 0);
    if ($courseId <= 0) {
        jsonError(400, 'Укажите courseId или versionId');
    }
    $course = cs_get_course($pdo, $courseId);
    if (!$course || !$course['currentVersionId']) {
        jsonError(404, 'Курс или версия не найдены');
    }
    return (int)$course['currentVersionId'];
}

function cs_map_enrollment(array $r, ?array $progress = null, ?array $course = null): array
{
    $out = [
        'id' => (int)$r['id'],
        'assignmentId' => $r['assignment_id'] !== null ? (int)$r['assignment_id'] : null,
        'courseVersionId' => (int)$r['course_version_id'],
        'userId' => (int)$r['user_id'],
        'status' => (string)$r['status'],
        'assignedAt' => $r['assigned_at'] ?? null,
        'startsAt' => $r['starts_at'] ?? null,
        'deadlineAt' => $r['deadline_at'] ?? null,
        'startedAt' => $r['started_at'] ?? null,
        'lastActivityAt' => $r['last_activity_at'] ?? null,
        'completedAt' => $r['completed_at'] ?? null,
        'finalScore' => $r['final_score'] !== null ? (float)$r['final_score'] : null,
        'createdAt' => $r['created_at'] ?? null,
        'updatedAt' => $r['updated_at'] ?? null,
    ];
    if ($progress !== null) {
        $out['progress'] = $progress;
    }
    if ($course !== null) {
        $out['course'] = $course;
    }
    return $out;
}

/**
 * Enrollment доступен: владелец или админ.
 * @return array enrollment row
 */
function cs_require_enrollment_access(PDO $pdo, int $enrollmentId, array $user, bool $adminOk = true): array
{
    $enr = cs_enrollment_row($pdo, $enrollmentId);
    if (!$enr) {
        jsonError(404, 'Запись на курс не найдена');
    }
    $uid = (int)$user['id'];
    if ((int)$enr['user_id'] === $uid) {
        return $enr;
    }
    if ($adminOk && auth_is_admin($user)) {
        return $enr;
    }
    jsonError(403, 'Нет доступа к этой записи');
}

function cs_topic_version_row(PDO $pdo, int $topicId): ?array
{
    $st = $pdo->prepare(
        'SELECT t.*, v.status AS version_status, v.id AS version_id, v.course_id
         FROM public.course_topics t
         JOIN public.course_versions v ON v.id = t.course_version_id
         WHERE t.id = :id AND t.deleted_at IS NULL'
    );
    $st->execute([':id' => $topicId]);
    $r = $st->fetch();
    return $r ?: null;
}

function cs_material_version_row(PDO $pdo, int $materialId): ?array
{
    $st = $pdo->prepare(
        'SELECT m.*, t.course_version_id, v.status AS version_status, v.course_id, t.id AS topic_id_chk
         FROM public.course_materials m
         JOIN public.course_topics t ON t.id = m.topic_id
         JOIN public.course_versions v ON v.id = t.course_version_id
         WHERE m.id = :id AND m.deleted_at IS NULL AND t.deleted_at IS NULL'
    );
    $st->execute([':id' => $materialId]);
    $r = $st->fetch();
    return $r ?: null;
}

function cs_user_fio(array $u): string
{
    return trim(implode(' ', array_filter([
        $u['surname'] ?? '',
        $u['firstname'] ?? '',
        $u['lastname'] ?? '',
    ])));
}

/**
 * Потомки ОФО по parent_id (ofo_unit.parent_id есть в рабочей схеме / ofo_tree.php).
 * Если колонки нет — возвращаем входные id.
 *
 * @param int[] $ofoIds
 * @return int[]
 */
function cs_ofo_descendants(PDO $pdo, array $ofoIds): array
{
    $ids = array_values(array_unique(array_filter(array_map('intval', $ofoIds), static fn($x) => $x > 0)));
    if (!$ids) {
        return [];
    }

    // Проверяем наличие parent_id (без path в текущей схеме)
    static $hasParent = null;
    if ($hasParent === null) {
        $chk = $pdo->query(
            "SELECT 1 FROM information_schema.columns
             WHERE table_schema = 'public' AND table_name = 'ofo_unit' AND column_name = 'parent_id'
             LIMIT 1"
        );
        $hasParent = (bool)$chk->fetchColumn(0);
    }
    if (!$hasParent) {
        // Нет иерархии в схеме — возвращаем как есть
        return $ids;
    }

    $placeholders = [];
    $params = [];
    foreach ($ids as $i => $id) {
        $k = ':id' . $i;
        $placeholders[] = $k;
        $params[$k] = $id;
    }
    $in = implode(', ', $placeholders);

    $sql = "WITH RECURSIVE tree AS (
                SELECT id FROM public.ofo_unit WHERE id IN ($in)
                UNION ALL
                SELECT u.id
                FROM public.ofo_unit u
                INNER JOIN tree t ON u.parent_id = t.id
            )
            SELECT DISTINCT id FROM tree";
    $st = $pdo->prepare($sql);
    $st->execute($params);
    return array_map(static fn($r) => (int)$r['id'], $st->fetchAll());
}

/**
 * Пользователи с user_info.ofo ∈ ofoIds (+ потомки при includeChildren), status=true, ofo задан.
 *
 * @param int[] $ofoIds
 * @return int[]
 */
function cs_resolve_ofo_users(PDO $pdo, array $ofoIds, bool $includeChildren): array
{
    $ids = array_values(array_unique(array_filter(array_map('intval', $ofoIds), static fn($x) => $x > 0)));
    if (!$ids) {
        return [];
    }
    if ($includeChildren) {
        $ids = cs_ofo_descendants($pdo, $ids);
    }
    if (!$ids) {
        return [];
    }

    $placeholders = [];
    $params = [];
    foreach ($ids as $i => $id) {
        $k = ':o' . $i;
        $placeholders[] = $k;
        $params[$k] = (string)$id; // user_info.ofo хранится строкой
    }
    $in = implode(', ', $placeholders);

    $sql = "SELECT id FROM public.user_info
            WHERE status IS TRUE
              AND ofo IS NOT NULL
              AND ofo <> ''
              AND ofo <> '-1'
              AND ofo IN ($in)";
    $st = $pdo->prepare($sql);
    $st->execute($params);
    return array_map(static fn($r) => (int)$r['id'], $st->fetchAll());
}

// Защита от прямого вызова модуля
if (realpath(__FILE__) === realpath($_SERVER['SCRIPT_FILENAME'] ?? '')) {
    jsonError(404, 'Not found');
}
