/**
 * Export rows from dist/data/gallery.json into SQL seed for public.gallery.
 *
 * Output: dist/data/gallery_seed.sql
 * Columns: id, name, description, date
 */
import { readFileSync, writeFileSync } from 'node:fs';

function escSqlString(s) {
  return String(s).replace(/'/g, "''");
}

const inputPath = 'dist/data/gallery.json';
const outputPath = 'dist/data/gallery_seed.sql';

const data = JSON.parse(readFileSync(inputPath, 'utf8'));

const vals = [];
for (const a of data?.albums ?? []) {
  const id = String(a?.id ?? '').trim();
  if (!id) continue;

  const name = String(a?.title ?? '').trim();
  const description = String(a?.description ?? '').trim();
  const date = String(a?.date ?? '').trim();

  vals.push(
    `(${Number(id)}, '${escSqlString(name)}', '${escSqlString(description)}', '${escSqlString(date)}')`,
  );
}

const out =
  'INSERT INTO public.gallery (id, "name", description, "date") VALUES\n' +
  vals.join(',\n') +
  ';\n';

writeFileSync(outputPath, out, 'utf8');
console.log(`OK: wrote ${vals.length} rows -> ${outputPath}`);

