<?php
$sql = file_get_contents('public/data/office_seats.sql');

// Извлекаем блок INSERT
preg_match('/INSERT INTO `office_seats` \([^)]+\) VALUES\s*(.*?);/s', $sql, $matches);
$values = $matches[1] ?? '';

if ($values) {
    $values = str_replace('\"', '"', $values); // Очищаем возможные экранирования двойных кавычек
    $out = "-- PostgreSQL совместимый дамп для таблицы office_seats\n\n";
    $out .= "DROP TABLE IF EXISTS office_seats CASCADE;\n";
    $out .= "CREATE TABLE office_seats (\n";
    $out .= "  id INTEGER PRIMARY KEY,\n";
    $out .= "  title TEXT NOT NULL,\n";
    $out .= "  ofo INTEGER NOT NULL,\n";
    $out .= "  insurance INTEGER NOT NULL,\n";
    $out .= "  rating INTEGER NOT NULL\n";
    $out .= ");\n\n";
    
    $out .= "INSERT INTO office_seats (id, title, ofo, insurance, rating) VALUES\n";
    $out .= $values . ";\n";
    
    file_put_contents('public/data/corporate_office_seats.sql', $out);
    echo "Done";
} else {
    echo "Values not found";
}
