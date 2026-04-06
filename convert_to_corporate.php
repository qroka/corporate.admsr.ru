<?php
$sql = file_get_contents('public/data/ofo.sql');

// Извлекаем блок INSERT
preg_match('/INSERT INTO `ofo` \([^)]+\) VALUES\s*(.*?);/s', $sql, $matches);
$values = $matches[1] ?? '';

if ($values) {
    $values = str_replace('\"', '"', $values); // Очищаем возможные экранирования двойных кавычек
    $out = "-- PostgreSQL совместимый дамп для таблицы corporate\n\n";
    $out .= "DROP TABLE IF EXISTS corporate CASCADE;\n";
    $out .= "CREATE TABLE corporate (\n";
    $out .= "  id INTEGER PRIMARY KEY,\n";
    $out .= "  title VARCHAR(1000) NOT NULL,\n";
    $out .= "  parent INTEGER NOT NULL,\n";
    $out .= "  type VARCHAR(255) NOT NULL,\n";
    $out .= "  sort_order INTEGER NOT NULL\n";
    $out .= ");\n\n";
    
    $out .= "INSERT INTO corporate (id, title, parent, type, sort_order) VALUES\n";
    $out .= $values . ";\n";
    
    file_put_contents('public/data/corporate.sql', $out);
    echo "Done";
} else {
    echo "Values not found";
}
