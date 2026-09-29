/**
 * Как показывать материал сотруднику. Тип в БД у загруженного файла всегда
 * `file` (Go приводит pdf/image/video к нему при загрузке), поэтому вид
 * определяем по MIME, а если его нет или он общий — по расширению имени файла.
 */
export type MaterialViewKind = 'text' | 'pdf' | 'video' | 'image' | 'file' | 'link';

const EXT_KIND: Record<string, MaterialViewKind> = {
  pdf: 'pdf',
  mp4: 'video',
  webm: 'video',
  png: 'image',
  jpg: 'image',
  jpeg: 'image',
  gif: 'image',
  webp: 'image',
};

function extOf(name: unknown): string {
  const m = /\.([a-z0-9]+)$/i.exec(String(name || ''));
  return m ? m[1].toLowerCase() : '';
}

export function materialViewKind(m: any): MaterialViewKind {
  const type = String(m?.type || '');
  if (type === 'rich_text') return 'text';
  if (type === 'link') return 'link';
  const mime = String(m?.mimeType || '').toLowerCase();
  if (mime === 'application/pdf') return 'pdf';
  if (mime.startsWith('video/')) return 'video';
  if (mime.startsWith('image/')) return 'image';
  const byExt = EXT_KIND[extOf(m?.originalFilename)];
  if (byExt) return byExt;
  if (type === 'pdf' || type === 'video' || type === 'image') return type;
  return 'file';
}

export function materialKindIcon(kind: MaterialViewKind): string {
  switch (kind) {
    case 'text':
      return 'i-lucide-file-text';
    case 'pdf':
      return 'i-lucide-file-type';
    case 'video':
      return 'i-lucide-circle-play';
    case 'image':
      return 'i-lucide-image';
    case 'link':
      return 'i-lucide-link';
    default:
      return 'i-lucide-paperclip';
  }
}

export function materialKindLabel(kind: MaterialViewKind, m?: any): string {
  switch (kind) {
    case 'text':
      return 'Текст';
    case 'pdf':
      return 'PDF';
    case 'video':
      return 'Видео';
    case 'image':
      return 'Изображение';
    case 'link':
      return 'Ссылка';
    default: {
      const ext = extOf(m?.originalFilename);
      return ext ? `Файл ${ext.toUpperCase()}` : 'Файл';
    }
  }
}

export function formatFileSize(bytes: unknown): string {
  const n = Number(bytes);
  if (!Number.isFinite(n) || n <= 0) return '';
  if (n < 1024 * 1024) return `${Math.max(1, Math.round(n / 1024))} КБ`;
  return `${(n / 1024 / 1024).toFixed(1).replace('.', ',')} МБ`;
}
