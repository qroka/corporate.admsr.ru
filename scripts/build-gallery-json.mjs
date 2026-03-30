/**
 * Сборка статических JSON для фотогалереи:
 * - gallery.json — метаданные альбомов из photo_cats
 * - gallery_albums.json — файлы из files, только row_type === "gal", сгруппировано по row_id
 *
 * Источники: dist/data/photo_cats (1).json, dist/data/files.json
 * Результат: public/data/ и dist/data/
 */
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');

function resolveFirst(paths) {
  for (const p of paths) {
    if (existsSync(p)) return p;
  }
  throw new Error(`Не найден ни один из файлов:\n${paths.join('\n')}`);
}

const PHOTO_CATS = resolveFirst([
  join(root, 'public/data/photo_cats.json'),
  join(root, 'dist/data/photo_cats.json'),
  join(root, 'dist/data/photo_cats (1).json'),
]);

const FILES = resolveFirst([join(root, 'public/data/files.json'), join(root, 'dist/data/files.json')]);

function extractTable(json, tableName) {
  if (!Array.isArray(json)) return [];
  for (const entry of json) {
    if (entry?.type === 'table' && entry.name === tableName && Array.isArray(entry.data)) {
      return entry.data;
    }
  }
  return [];
}

function stripHtml(html) {
  return String(html ?? '')
    .replace(/<[^>]*>/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

function tsToDate(ts) {
  const n = Number(ts);
  if (!Number.isFinite(n) || n <= 0) return '';
  return new Date(n * 1000).toISOString().slice(0, 10);
}

function main() {
  const catsRaw = JSON.parse(readFileSync(PHOTO_CATS, 'utf8'));
  const filesRaw = JSON.parse(readFileSync(FILES, 'utf8'));

  const photoCats = extractTable(catsRaw, 'photo_cats');
  const fileRows = extractTable(filesRaw, 'files');

  const galOnly = fileRows.filter((r) => String(r.row_type ?? '').toLowerCase() === 'gal');
  galOnly.sort((a, b) => Number(a.id) - Number(b.id));

  /** @type {Record<string, typeof galOnly>} */
  const byAlbum = {};
  for (const row of galOnly) {
    const rid = String(row.row_id ?? '').trim();
    if (!rid) continue;
    if (!byAlbum[rid]) byAlbum[rid] = [];
    byAlbum[rid].push(row);
  }

  const gallery = {
    version: 2,
    source: 'photo_cats',
    generated: new Date().toISOString(),
    albums: photoCats.map((cat) => {
      const html = String(cat.html ?? '');
      const desc = stripHtml(html);
      return {
        id: String(cat.id ?? '').trim(),
        title: String(cat.title ?? '').trim(),
        description: desc || undefined,
        count: cat.count != null ? String(cat.count) : undefined,
        date: tsToDate(cat.timestamp),
        timestamp: cat.timestamp != null ? String(cat.timestamp) : undefined,
        hided: cat.hided != null ? String(cat.hided) : '0',
        html: html || undefined,
      };
    }),
    albumFallback: {
      titleTemplate: 'Альбом {albumId}',
      description: 'Фотографии с мероприятия. Нажмите на фото, чтобы открыть в большом размере.',
    },
    photoMeta: {
      every: 0,
      titleTemplate: 'Фото {n}',
      descriptionTemplate: 'Кадр {n} из альбома {albumId}.',
    },
  };

  const galleryAlbums = {
    version: 1,
    source: 'files',
    rowType: 'gal',
    generated: new Date().toISOString(),
    /** Снимки WebP в public: /img/FullPic/{filename}.webp и /img/SmallPic/{filename}.webp (basename без расширения из поля filename) */
    albums: Object.keys(byAlbum)
      .sort((a, b) => Number(a) - Number(b))
      .map((albumId) => ({
        albumId,
        files: byAlbum[albumId].map((f) => ({
          id: String(f.id ?? ''),
          filename: String(f.filename ?? ''),
          extension: String(f.extension ?? ''),
          filesize: f.filesize != null ? String(f.filesize) : undefined,
          original: f.original != null ? String(f.original) : undefined,
          timestamp: f.timestamp != null ? String(f.timestamp) : undefined,
          description: f.description != null ? String(f.description) : undefined,
          urls: {
            fullWebp: `/img/FullPic/${String(f.filename ?? '').replace(/\.[^/.]+$/, '')}.webp`,
            smallWebp: `/img/SmallPic/${String(f.filename ?? '').replace(/\.[^/.]+$/, '')}.webp`,
          },
        })),
      })),
    stats: {
      totalFiles: galOnly.length,
      albumCount: Object.keys(byAlbum).length,
    },
  };

  const outPaths = [join(root, 'public/data'), join(root, 'dist/data')];
  for (const dir of outPaths) {
    try {
      mkdirSync(dir, { recursive: true });
    } catch {
      /* exists */
    }
    writeFileSync(join(dir, 'gallery.json'), JSON.stringify(gallery, null, 2), 'utf8');
    writeFileSync(join(dir, 'gallery_albums.json'), JSON.stringify(galleryAlbums, null, 2), 'utf8');
  }

  console.log(
    `OK: gallery.json (${gallery.albums.length} альбомов), gallery_albums.json (${galleryAlbums.stats.totalFiles} файлов gal, ${galleryAlbums.stats.albumCount} альбомов с фото).`,
  );
}

main();
