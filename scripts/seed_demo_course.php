#!/usr/bin/env php
<?php
/**
 * Создаёт демо-курс «Охрана труда: основы» с темами, материалами, тестами,
 * публикует и назначает владельцу (админу).
 *
 * Запуск: php scripts/seed_demo_course.php
 */
declare(strict_types=1);

$root = dirname(__DIR__);

if (PHP_SAPI === 'cli') {
    $_SERVER['REQUEST_METHOD'] = $_SERVER['REQUEST_METHOD'] ?? 'CLI';
}

require_once $root . '/api/courses_common.php';

/** @var PDO $pdo */

$admin = $pdo->query(
    "SELECT id, login, role, firstname, surname, lastname
     FROM public.user_info
     WHERE lower(coalesce(role,'')) LIKE '%admin%'
        OR lower(coalesce(role,'')) IN ('hr', 'editor')
     ORDER BY id
     LIMIT 1"
)->fetch();

if (!$admin) {
    $admin = $pdo->query('SELECT id, login, role, firstname, surname, lastname FROM public.user_info ORDER BY id LIMIT 1')->fetch();
}
if (!$admin) {
    fwrite(STDERR, "Нет пользователей в user_info\n");
    exit(1);
}

$user = [
    'id' => (int)$admin['id'],
    'login' => (string)$admin['login'],
    'role' => (string)($admin['role'] ?? 'admin'),
];

echo "Owner: #{$user['id']} {$user['login']} ({$user['role']})\n";

$created = cs_create_course($pdo, $user, [
    'title' => 'Охрана труда: основы',
    'category' => 'Безопасность',
    'shortDescription' => 'Короткий вводный курс по охране труда для сотрудников.',
    'fullDescription' => '<p>Курс знакомит с базовыми правилами охраны труда, действиями при ЧС и первой помощью.</p>',
    'sequentialProgress' => true,
    'requireFinalTest' => true,
    'defaultDeadlineDays' => 14,
    'finalPassingScore' => 70,
]);

$courseId = (int)$created['course']['id'];
$versionId = (int)$created['version']['id'];
echo "Course #{$courseId}, version #{$versionId}\n";

function seed_topic(PDO $pdo, int $versionId, string $title, string $description, int $ord): array
{
    $ins = $pdo->prepare(
        'INSERT INTO public.course_topics (
            course_version_id, title, description, sort_order, is_required, minimum_active_seconds
         ) VALUES (:v, :t, :d, :ord, true, 0) RETURNING *'
    );
    $ins->execute([':v' => $versionId, ':t' => $title, ':d' => $description, ':ord' => $ord]);
    return $ins->fetch();
}

function seed_material(PDO $pdo, int $topicId, string $title, string $html, int $ord): void
{
    $ins = $pdo->prepare(
        'INSERT INTO public.course_materials (
            topic_id, type, title, description, content_html, sort_order, is_required, minimum_active_seconds
         ) VALUES (:tid, \'rich_text\', :title, \'\', :html, :ord, true, 0)'
    );
    $ins->execute([
        ':tid' => $topicId,
        ':title' => $title,
        ':html' => cs_sanitize_html($html),
        ':ord' => $ord,
    ]);
}

$t1 = seed_topic($pdo, $versionId, 'Введение в охрану труда', 'Зачем нужна ОТ и основные обязанности работника.', 0);
$t2 = seed_topic($pdo, $versionId, 'Действия при чрезвычайных ситуациях', 'Эвакуация, оповещение, поведение при пожаре.', 1);
$t3 = seed_topic($pdo, $versionId, 'Первая помощь', 'Базовые приёмы до приезда медиков.', 2);

seed_material($pdo, (int)$t1['id'], 'Что такое охрана труда', <<<'HTML'
<p>Охрана труда — система сохранения жизни и здоровья работников в процессе трудовой деятельности.</p>
<ul>
  <li>соблюдайте инструкции;</li>
  <li>используйте СИЗ;</li>
  <li>сообщайте о нарушениях руководителю.</li>
</ul>
HTML, 0);

seed_material($pdo, (int)$t1['id'], 'Права и обязанности', <<<'HTML'
<p>Работник обязан проходить инструктажи и медосмотры, а работодатель — обеспечивать безопасные условия.</p>
HTML, 1);

seed_material($pdo, (int)$t2['id'], 'План эвакуации', <<<'HTML'
<p>При сигнале тревоги:</p>
<ol>
  <li>спокойно выйдите по маршруту эвакуации;</li>
  <li>не пользуйтесь лифтом;</li>
  <li>соберитесь в месте сбора.</li>
</ol>
HTML, 0);

seed_material($pdo, (int)$t2['id'], 'Полезные ссылки', <<<'HTML'
<p>Изучите схему эвакуации на этаже и номера экстренных служб на стенде.</p>
HTML, 1);

seed_material($pdo, (int)$t3['id'], 'Алгоритм первой помощи', <<<'HTML'
<p>Оцените безопасность места происшествия, вызовите помощь, окажите помощь пострадавшему в рамках своей подготовки.</p>
HTML, 0);

echo "Topics & materials created\n";

cs_create_topic_test($pdo, (int)$t1['id'], $user, [
    'title' => 'Проверка: введение в ОТ',
    'usePassingScore' => true,
    'passingScore' => 60,
    'questions' => [
        [
            'type' => 'single',
            'title' => 'Что обязан делать работник?',
            'required' => true,
            'correct' => 'a',
            'options' => [
                ['id' => 'a', 'text' => 'Соблюдать инструкции по охране труда'],
                ['id' => 'b', 'text' => 'Игнорировать инструктажи'],
                ['id' => 'c', 'text' => 'Отключать СИЗ по желанию'],
            ],
        ],
        [
            'type' => 'single',
            'title' => 'Кому сообщить об опасной ситуации?',
            'required' => true,
            'correct' => 'a',
            'options' => [
                ['id' => 'a', 'text' => 'Непосредственному руководителю'],
                ['id' => 'b', 'text' => 'Никому'],
            ],
        ],
    ],
]);

cs_create_final_test($pdo, $versionId, $user, [
    'title' => 'Итоговый тест: охрана труда',
    'usePassingScore' => true,
    'passingScore' => 70,
    'questions' => [
        [
            'type' => 'single',
            'title' => 'При пожаре лифтом:',
            'required' => true,
            'correct' => 'b',
            'options' => [
                ['id' => 'a', 'text' => 'Можно пользоваться'],
                ['id' => 'b', 'text' => 'Пользоваться нельзя'],
            ],
        ],
        [
            'type' => 'single',
            'title' => 'Первое действие при ЧС:',
            'required' => true,
            'correct' => 'a',
            'options' => [
                ['id' => 'a', 'text' => 'Оценить безопасность и следовать плану эвакуации'],
                ['id' => 'b', 'text' => 'Остаться на рабочем месте'],
            ],
        ],
        [
            'type' => 'single',
            'title' => 'СИЗ нужны, чтобы:',
            'required' => true,
            'correct' => 'a',
            'options' => [
                ['id' => 'a', 'text' => 'Снизить риск травм и воздействия вредных факторов'],
                ['id' => 'b', 'text' => 'Украсить рабочее место'],
            ],
        ],
    ],
]);

echo "Tests created\n";

$published = cs_publish_version($pdo, $versionId, $user);
echo "Published: status=" . ($published['status'] ?? '?') . "\n";

$deadline = date('c', time() + 14 * 86400);
$pdo->beginTransaction();
try {
    $aIns = $pdo->prepare(
        "INSERT INTO public.course_assignments (
            course_version_id, target_type, target_id, starts_at, deadline_at,
            assigned_by, comment, include_children
         ) VALUES (:v, 'user', :tid, null, :da, :by, :c, false) RETURNING id"
    );
    $aIns->execute([
        ':v' => $versionId,
        ':tid' => $user['id'],
        ':da' => $deadline,
        ':by' => $user['id'],
        ':c' => 'Демо-назначение',
    ]);
    $aid = (int)$aIns->fetchColumn(0);

    $exists = $pdo->prepare(
        "SELECT id FROM public.course_enrollments
         WHERE user_id = :u AND course_version_id = :v AND status NOT IN ('cancelled') LIMIT 1"
    );
    $exists->execute([':u' => $user['id'], ':v' => $versionId]);
    if (!$exists->fetch()) {
        $eIns = $pdo->prepare(
            "INSERT INTO public.course_enrollments (
                assignment_id, course_version_id, user_id, status, starts_at, deadline_at
             ) VALUES (:a, :v, :u, 'not_started', null, :da) RETURNING id"
        );
        $eIns->execute([
            ':a' => $aid,
            ':v' => $versionId,
            ':u' => $user['id'],
            ':da' => $deadline,
        ]);
        $eid = (int)$eIns->fetchColumn(0);
        echo "Enrollment #{$eid} for user #{$user['id']}\n";
    } else {
        echo "Enrollment already exists\n";
    }
    $pdo->commit();
} catch (Throwable $e) {
    if ($pdo->inTransaction()) {
        $pdo->rollBack();
    }
    fwrite(STDERR, "Assign error: {$e->getMessage()}\n");
    exit(1);
}

echo "\nOK: курс «Охрана труда: основы» создан (id={$courseId}), опубликован и назначен.\n";
echo "Откройте /courses или /admin/courses/{$courseId}\n";
