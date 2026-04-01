/**
 * Reads JSONL formData.json (PHP-serialized formData in each row),
 * outputs UTF-8 CSV with BOM for Excel: ID, Data (datetime), one column per question.
 * Run: node scripts/formdata-to-excel.mjs [input.json] [output.csv]
 */
import { readFileSync, writeFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { unserialize } from 'php-serialize';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');
const inputPath = process.argv[2] ?? join(root, 'formData.json');
const outputPath = process.argv[3] ?? join(root, 'formData-export.csv');

function parseJsonLine(line) {
  const trimmed = line.trim();
  if (!trimmed) return null;
  const withoutTrailingComma = trimmed.replace(/,\s*$/, '');
  return JSON.parse(withoutTrailingComma);
}

function escapeCsvCell(value) {
  if (value === null || value === undefined) return '';
  const s = String(value);
  if (/[",\r\n;]/.test(s)) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

/** Skip PHP artifact keys like i:0 (duplicate trailing value) */
function isQuestionKey(k) {
  if (k === 'formId') return false;
  if (/^\d+$/.test(k)) return false;
  return true;
}

function collectEntries(data) {
  if (data === null || typeof data !== 'object') return [];
  const out = [];
  if (Array.isArray(data)) {
    for (let i = 0; i < data.length; i++) {
      const v = data[i];
      if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
        out.push(...collectEntries(v));
      } else {
        out.push([String(i), v]);
      }
    }
    return out;
  }
  for (const [k, v] of Object.entries(data)) {
    if (!isQuestionKey(k)) continue;
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
      out.push(...collectEntries(v));
    } else {
      out.push([k, v]);
    }
  }
  return out;
}

function flattenAnswers(data) {
  const map = new Map();
  const entries = collectEntries(data);
  for (const [key, val] of entries) {
    if (typeof val === 'object' && val !== null) {
      map.set(key, JSON.stringify(val));
    } else {
      map.set(key, val === undefined ? '' : val);
    }
  }
  return map;
}

const raw = readFileSync(inputPath, 'utf8');
const lines = raw.split(/\r?\n/).filter((l) => l.trim());

const parsedRows = [];
const questionKeys = new Set();

for (const line of lines) {
  let row;
  try {
    row = parseJsonLine(line);
  } catch (e) {
    console.warn('Skip invalid JSON line:', e.message);
    continue;
  }
  if (!row || !row.formData) continue;

  let data;
  try {
    data = unserialize(row.formData);
  } catch (e) {
    console.warn('Skip id', row.id, 'unserialize:', e.message);
    continue;
  }

  const answers = flattenAnswers(data);
  for (const k of answers.keys()) {
    questionKeys.add(k);
  }

  const ts = row.time != null ? Number(row.time) : NaN;
  const dataCol =
    Number.isFinite(ts) ? new Date(ts * 1000).toISOString().replace('T', ' ').slice(0, 19) : '';

  parsedRows.push({
    id: row.id ?? '',
    dataCol,
    answers,
  });
}

const sortedQuestions = [...questionKeys].sort((a, b) =>
  a.localeCompare(b, 'ru', { numeric: true })
);

const sep = ';';
const header = ['ID', 'Data', ...sortedQuestions];
const bom = '\uFEFF';
const linesOut = [header.map(escapeCsvCell).join(sep)];

for (const { id, dataCol, answers } of parsedRows) {
  const cells = [
    id,
    dataCol,
    ...sortedQuestions.map((q) => answers.get(q) ?? ''),
  ];
  linesOut.push(cells.map(escapeCsvCell).join(sep));
}

writeFileSync(outputPath, bom + linesOut.join('\r\n'), 'utf8');
console.log(`Wrote ${parsedRows.length} rows, ${sortedQuestions.length} question columns -> ${outputPath}`);
