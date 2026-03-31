/**
 * Export rows from dist/data/gallery_albums.json into SQL seed for public.gallery_base.
 *
 * Output: dist/data/gallery_base_seed.sql
 * Columns: id, album_id, image_full_url, image_small_url
 */
import { readFileSync, writeFileSync } from 'node:fs';

function escSqlString(s) {
  return String(s).replace(/'/g, "''");
}

const inputPath = 'dist/data/gallery_albums.json';
const outputPath = 'dist/data/gallery_base_seed.sql';

const data = JSON.parse(readFileSync(inputPath, 'utf8'));

const vals = [];
for (const a of data?.albums ?? []) {
  const albumId = String(a?.albumId ?? '').trim();
  for (const f of a?.files ?? []) {
    const id = String(f?.id ?? '').trim();
    const full = f?.urls?.fullWebp;
    const small = f?.urls?.smallWebp;
    if (!id || !albumId || !full || !small) continue;

    vals.push(
      `(${Number(id)}, ${Number(albumId)}, '${escSqlString(full)}', '${escSqlString(small)}')`,
    );
  }
}

const out =
  'INSERT INTO public.gallery_base (id, album_id, image_full_url, image_small_url) VALUES\n' +
  vals.join(',\n') +
  ';\n';

writeFileSync(outputPath, out, 'utf8');
console.log(`OK: wrote ${vals.length} rows -> ${outputPath}`);

