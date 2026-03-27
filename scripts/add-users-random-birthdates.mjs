/**
 * Заполняет birthdate случайной датой для пользователей с плейсхолдером/некорректной датой.
 * Источник: public/data/users.json (экспорт PHPMyAdmin).
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');
const path = join(root, 'public', 'data', 'users.json');

function randomBirthdate() {
  const y = 1960 + Math.floor(Math.random() * 40); // 1960–1999
  const m = 1 + Math.floor(Math.random() * 12);
  const daysInMonth = new Date(y, m, 0).getDate();
  const d = 1 + Math.floor(Math.random() * daysInMonth);
  return `${y}-${String(m).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
}

function needsRandomBirthdate(value) {
  if (value == null || value === '') return true;
  const s = String(value).trim();
  if (s === '0000-00-00') return true;
  if (s === '1970-01-01') return true;
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (!m) return true;
  const year = Number(m[1]);
  const month = Number(m[2]);
  const day = Number(m[3]);
  if (year < 1940 || year > 2005) return true;
  const t = new Date(year, month - 1, day);
  if (t.getFullYear() !== year || t.getMonth() !== month - 1 || t.getDate() !== day) return true;
  return false;
}

const raw = readFileSync(path, 'utf8');
const data = JSON.parse(raw);

let tableEntry = null;
for (const item of data) {
  if (item && item.type === 'table' && item.name === 'users' && Array.isArray(item.data)) {
    tableEntry = item;
    break;
  }
}

if (!tableEntry) {
  console.error('Не найден блок table users');
  process.exit(1);
}

let updated = 0;
for (const row of tableEntry.data) {
  if (row && Object.prototype.hasOwnProperty.call(row, 'birthdate')) {
    if (needsRandomBirthdate(row.birthdate)) {
      row.birthdate = randomBirthdate();
      updated++;
    }
  }
}

writeFileSync(path, JSON.stringify(data), 'utf8');
console.log(`Готово: обновлено записей birthdate: ${updated}`);
