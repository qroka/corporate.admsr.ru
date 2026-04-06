<?php
$pdo = new PDO('pgsql:host=localhost;port=5432;dbname=corporate_portal', 'postgres', 'postgres');
$stmt = $pdo->query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'");
print_r($stmt->fetchAll(PDO::FETCH_COLUMN));
