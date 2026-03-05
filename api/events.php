<?php

// Простой PHP backend для работы с событиями через JSON-файл.
// Эндпоинт: /api/events.php
//
// Поддерживаемые запросы:
// - GET /api/events.php                — список всех мероприятий
// - GET /api/events.php?id=0           — одно мероприятие по индексу
// - POST /api/events.php               — создать мероприятие (JSON в body)
// - PUT /api/events.php?id=0           — обновить мероприятие по индексу
// - DELETE /api/events.php?id=0        — удалить мероприятие по индексу
//
// ВНИМАНИЕ: это упрощённый вариант для локальной разработки.

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(204);
    exit;
}

$dataFile = __DIR__ . '/../src/data/events.json';

function loadEvents(string $file): array
{
    if (!file_exists($file)) {
        return [];
    }

    $json = file_get_contents($file);
    if ($json === false || $json === '') {
        return [];
    }

    $data = json_decode($json, true);
    return is_array($data) ? $data : [];
}

function saveEvents(string $file, array $events): bool
{
    $json = json_encode($events, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
    if ($json === false) {
        return false;
    }

    $fp = fopen($file, 'c+');
    if ($fp === false) {
        return false;
    }

    // простая блокировка на время записи
    if (flock($fp, LOCK_EX)) {
        ftruncate($fp, 0);
        fwrite($fp, $json);
        fflush($fp);
        flock($fp, LOCK_UN);
    } else {
        fclose($fp);
        return false;
    }

    fclose($fp);
    return true;
}

$method = $_SERVER['REQUEST_METHOD'];
$id = isset($_GET['id']) ? (int) $_GET['id'] : null;

switch ($method) {
    case 'GET':
        $events = loadEvents($dataFile);
        if ($id === null) {
            echo json_encode($events, JSON_UNESCAPED_UNICODE);
        } else {
            if ($id < 0 || $id >= count($events)) {
                http_response_code(404);
                echo json_encode(['error' => 'Event not found'], JSON_UNESCAPED_UNICODE);
                exit;
            }
            echo json_encode($events[$id], JSON_UNESCAPED_UNICODE);
        }
        break;

    case 'POST':
        $input = json_decode(file_get_contents('php://input'), true);
        if (!is_array($input)) {
            http_response_code(400);
            echo json_encode(['error' => 'Invalid JSON body'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $events = loadEvents($dataFile);
        $events[] = $input;

        if (!saveEvents($dataFile, $events)) {
            http_response_code(500);
            echo json_encode(['error' => 'Failed to save events'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        http_response_code(201);
        echo json_encode(['id' => count($events) - 1, 'event' => $input], JSON_UNESCAPED_UNICODE);
        break;

    case 'PUT':
        if ($id === null) {
            http_response_code(400);
            echo json_encode(['error' => 'Missing id parameter'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $events = loadEvents($dataFile);
        if ($id < 0 || $id >= count($events)) {
            http_response_code(404);
            echo json_encode(['error' => 'Event not found'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $input = json_decode(file_get_contents('php://input'), true);
        if (!is_array($input)) {
            http_response_code(400);
            echo json_encode(['error' => 'Invalid JSON body'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $events[$id] = $input;
        if (!saveEvents($dataFile, $events)) {
            http_response_code(500);
            echo json_encode(['error' => 'Failed to save events'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        echo json_encode(['id' => $id, 'event' => $input], JSON_UNESCAPED_UNICODE);
        break;

    case 'DELETE':
        if ($id === null) {
            http_response_code(400);
            echo json_encode(['error' => 'Missing id parameter'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        $events = loadEvents($dataFile);
        if ($id < 0 || $id >= count($events)) {
            http_response_code(404);
            echo json_encode(['error' => 'Event not found'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        array_splice($events, $id, 1);
        if (!saveEvents($dataFile, $events)) {
            http_response_code(500);
            echo json_encode(['error' => 'Failed to save events'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        echo json_encode(['success' => true], JSON_UNESCAPED_UNICODE);
        break;

    default:
        http_response_code(405);
        echo json_encode(['error' => 'Method not allowed'], JSON_UNESCAPED_UNICODE);
        break;
}

