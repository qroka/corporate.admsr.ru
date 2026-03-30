import fs from 'node:fs/promises';
import path from 'node:path';
import sharp from 'sharp';

const ROOT = process.cwd();
const INPUT_DIR = path.join(ROOT, 'src', 'img', 'Events');
const OUTPUT_DIR_THUMB = path.join(ROOT, 'src', 'img', 'EventsWebp');
const OUTPUT_DIR_FULL = path.join(ROOT, 'src', 'img', 'EventsWebpFull');

const EXT_RE = /\.(jpe?g|png)$/i;

async function ensureDir(dir) {
  await fs.mkdir(dir, { recursive: true });
}

async function listFiles(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  return entries
    .filter((e) => e.isFile())
    .map((e) => e.name)
    .filter((name) => EXT_RE.test(name))
    .sort((a, b) => a.localeCompare(b, 'en'));
}

async function fileExists(p) {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
}

async function main() {
  await ensureDir(OUTPUT_DIR_THUMB);
  await ensureDir(OUTPUT_DIR_FULL);

  const files = await listFiles(INPUT_DIR);
  if (!files.length) {
    console.log('No input images found in', INPUT_DIR);
    return;
  }

  let convertedThumb = 0;
  let convertedFull = 0;
  for (let i = 0; i < files.length; i++) {
    const name = files[i];
    const inPath = path.join(INPUT_DIR, name);
    const outName = name.replace(EXT_RE, '.webp');
    const outThumbPath = path.join(OUTPUT_DIR_THUMB, outName);
    const outFullPath = path.join(OUTPUT_DIR_FULL, outName);

    if (!(await fileExists(outThumbPath))) {
      await sharp(inPath)
        .rotate()
        // Карточки / сетка
        .resize({ width: 960, withoutEnlargement: true })
        .webp({ quality: 76, effort: 5 })
        .toFile(outThumbPath);
      convertedThumb++;
    }

    if (!(await fileExists(outFullPath))) {
      await sharp(inPath)
        .rotate()
        // Модалка: выше качество и размер
        .resize({ width: 1920, withoutEnlargement: true })
        .webp({ quality: 100, effort: 5 })
        .toFile(outFullPath);
      convertedFull++;
    }
  }

  console.log(
    `Done. Generated ${convertedThumb} new thumb webp into ${OUTPUT_DIR_THUMB} and ${convertedFull} new full webp into ${OUTPUT_DIR_FULL}`,
  );
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
