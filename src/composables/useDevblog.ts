import { apiSessionFetch, apiSessionUpload } from './useAuthSession';
import { seedNewsReactions } from './useNewsReactions';

/**
 * Девблог (/api/devblog.php, V20, ADR-052): администраторы пишут один общий
 * черновик в Markdown, публикуют его новостью категории «Девблог». Всем — окошко
 * при первом заходе после публикации и уведомление в колокольчике.
 */

export const DEVBLOG_DEFAULT_COVER = '/devblog-cover.svg'; // = devblogDefaultCover в devblog.go
/** Категория новости-девблога (= devblogCategory в devblog.go). */
export const DEVBLOG_CATEGORY = 'Девблог';
/** Подпись автора девблога в ленте и колокольчике — не конкретный администратор. */
export const DEVBLOG_AUTHOR = 'Разработчики портала';

export function isDevblogCategory(category: string | null | undefined): boolean {
  return String(category ?? '').trim().toLowerCase() === DEVBLOG_CATEGORY.toLowerCase();
}
export const DEVBLOG_TITLE_MAX = 200; // = devblogTitleMax
export const DEVBLOG_BODY_MAX = 50000; // = devblogBodyMax
/** Версия выпуска «X.Y.Z» (= devblogVersionRe в devblog.go). */
export const DEVBLOG_VERSION_RE = /^\d{1,4}\.\d{1,4}\.\d{1,4}$/;

export type DevblogDraft = {
  title: string;
  body: string;
  imagePath: string;
  /** Версия выпуска «1.0.1»; пусто — ещё не задавали */
  releaseVersion: string;
  /** Предложение сервера: последняя опубликованная + 0.0.1 (первый выпуск — 1.0.0) */
  suggestedVersion: string;
  /** Номер правки черновика — для проверки «изменил другой» */
  version: number;
  updatedAt: string | null;
  updatedBy: { id: number; name: string } | null;
};

export type DevblogPending = {
  newsId: number;
  title: string;
  html: string;
  imagePath: string;
  date: string;
  publishedAt: string;
  /** Версия выпуска */
  version: string;
};

/** Ответ сохранения: conflict — черновик за это время сохранил другой администратор. */
export type DevblogSaveResult =
  | { ok: true; draft: DevblogDraft }
  | { ok: false; conflict: true; draft: DevblogDraft; message: string };

const API = '/api/devblog.php';

function normalizeDraft(raw: any): DevblogDraft {
  return {
    title: String(raw?.title ?? ''),
    body: String(raw?.body ?? ''),
    imagePath: String(raw?.imagePath ?? ''),
    releaseVersion: String(raw?.releaseVersion ?? ''),
    suggestedVersion: String(raw?.suggestedVersion ?? '1.0.0'),
    version: Number(raw?.version) || 0,
    updatedAt: raw?.updatedAt ? String(raw.updatedAt) : null,
    updatedBy: raw?.updatedBy ? { id: Number(raw.updatedBy.id) || 0, name: String(raw.updatedBy.name ?? '') } : null,
  };
}

/** 409 от сервера — в data лежит актуальный черновик. */
function isConflict(json: any): boolean {
  return json && json.success === false && json.data && typeof json.data.version !== 'undefined';
}

export async function loadDevblogDraft(): Promise<DevblogDraft> {
  const json = await apiSessionFetch(`${API}?action=draft`);
  if (!json?.success) throw new Error(json?.message || 'Не удалось загрузить черновик');
  return normalizeDraft(json.data);
}

export async function saveDevblogDraft(
  fields: { title: string; body: string; imagePath: string; releaseVersion: string },
  version: number,
  force = false,
): Promise<DevblogSaveResult> {
  const json = await apiSessionFetch(API, { method: 'POST', json: { action: 'save', ...fields, version, force } });
  if (json?.success) return { ok: true, draft: normalizeDraft(json.data) };
  if (isConflict(json)) {
    return { ok: false, conflict: true, draft: normalizeDraft(json.data), message: json.message || 'Черновик изменил другой администратор' };
  }
  throw new Error(json?.message || 'Не удалось сохранить черновик');
}

export async function publishDevblog(
  fields: { title: string; body: string; imagePath: string; releaseVersion: string; html: string },
  version: number,
  force = false,
): Promise<
  | { ok: true; newsId: number; notified: number; draft: DevblogDraft }
  | { ok: false; conflict: true; draft: DevblogDraft; message: string }
> {
  const json = await apiSessionFetch<any>(API, { method: 'POST', json: { action: 'publish', ...fields, version, force } });
  if (json?.success) {
    return {
      ok: true,
      newsId: Number(json.data?.newsId) || 0,
      notified: Number(json.data?.notified) || 0,
      draft: normalizeDraft(json.data?.draft),
    };
  }
  if (isConflict(json)) {
    return { ok: false, conflict: true, draft: normalizeDraft(json.data), message: json.message || 'Черновик изменил другой администратор' };
  }
  throw new Error(json?.message || 'Не удалось опубликовать');
}

/** Обложка — через общий загрузчик картинок портала; возвращает путь /img/… */
export async function uploadDevblogCover(file: File): Promise<string> {
  const fd = new FormData();
  fd.append('image', file);
  const json = await apiSessionUpload<{ image?: string; image_full?: string }>('/api/Upload/upload.php', fd);
  const path = json?.data?.image_full || json?.data?.image;
  if (!json?.success || !path) throw new Error(json?.message || 'Не удалось загрузить обложку');
  return String(path);
}

/** Последний девблог, который сотрудник ещё не закрыл, или null. */
export async function loadPendingDevblog(): Promise<DevblogPending | null> {
  const json = await apiSessionFetch<any>(API);
  if (!json?.success || !json.data) return null;
  const d = json.data;
  seedNewsReactions(String(d.newsId), d.reactions);
  return {
    newsId: Number(d.newsId) || 0,
    title: String(d.title ?? ''),
    html: String(d.html ?? ''),
    imagePath: String(d.imagePath ?? ''),
    date: String(d.date ?? ''),
    publishedAt: String(d.publishedAt ?? ''),
    version: String(d.version ?? ''),
  };
}

export async function dismissDevblog(newsId: number): Promise<void> {
  const json = await apiSessionFetch(API, { method: 'POST', json: { action: 'dismiss', newsId } });
  if (!json?.success) throw new Error(json?.message || 'Не удалось закрыть');
}
