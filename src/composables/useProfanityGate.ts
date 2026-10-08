import { reactive } from 'vue';
import { apiSessionFetch, type ApiResult } from './useAuthSession';

/**
 * Фильтр мата в комментариях и на стене (ADR-053). Список слов и проверка — только
 * на сервере (backend/internal/profanity). Нашёл мат — отвечает 422 с задачей
 * data.profanity = { token, question, masked }; здесь открываем окно
 * ProfanityChallengeModal (одно на приложение, в App.vue), а с ответом повторяем тот
 * же запрос. Неверный ответ — сервер присылает новую задачу, окно остаётся открытым.
 * Решили — текст публикуется со звёздочками в найденных словах.
 */

export type ProfanityChallenge = {
  token: string;
  question: string;
  /** Как текст будет опубликован */
  masked: string;
};

/** Автор закрыл окно задачи — тост не нужен, текст остаётся в поле. */
export class ProfanityCancelledError extends Error {
  constructor() {
    super('Публикация отменена');
    this.name = 'ProfanityCancelledError';
  }
}

export function isProfanityCancelled(e: unknown): boolean {
  return e instanceof ProfanityCancelledError;
}

/** Состояние окна задачи (читает ProfanityChallengeModal). */
export const profanityState = reactive({
  open: false,
  question: '',
  masked: '',
  /** Пояснение сервера: «есть нецензурные выражения…» / «Неверный ответ…» */
  message: '',
  /** Прошлый ответ был неверным (или время вышло) — подсветить сообщение */
  retry: false,
  /** Ответ отправлен, ждём сервер */
  busy: false,
});

let resolveAnswer: ((answer: string | null) => void) | null = null;
/** Чей запрос сейчас показан в окне — закрывает окно только он. */
let owner = 0;
let seq = 0;

/** Окно: ответ или отмена. */
export function submitProfanityAnswer(answer: string) {
  resolveAnswer?.(answer);
}

export function cancelProfanityChallenge() {
  resolveAnswer?.(null);
}

function readChallenge(json: ApiResult<any> | null | undefined): ProfanityChallenge | null {
  const ch = json && !json.success ? json.data?.profanity : null;
  if (!ch?.token) return null;
  return { token: String(ch.token), question: String(ch.question ?? ''), masked: String(ch.masked ?? '') };
}

function askAnswer(me: number, ch: ProfanityChallenge, message: string, retry: boolean): Promise<string | null> {
  resolveAnswer?.(null); // окно уже ждало другой запрос — тот считаем отменённым
  owner = me;
  Object.assign(profanityState, { open: true, question: ch.question, masked: ch.masked, message, retry, busy: false });
  return new Promise((resolve) => {
    resolveAnswer = (answer) => {
      resolveAnswer = null;
      resolve(answer);
    };
  });
}

function closeChallenge(me: number) {
  if (owner !== me) return;
  owner = 0;
  profanityState.open = false;
  profanityState.busy = false;
}

/**
 * POST JSON с проверкой на мат: если сервер прислал задачу — спросить ответ и
 * повторить запрос. Возвращает итоговый ответ сервера (успех или другую ошибку);
 * закрыли окно — бросает ProfanityCancelledError.
 */
export async function postWithProfanityGate<T = any>(url: string, json: Record<string, unknown>): Promise<ApiResult<T>> {
  const me = ++seq;
  let res = await apiSessionFetch<T>(url, { method: 'POST', json });
  let ch = readChallenge(res);
  let retry = false;
  try {
    while (ch) {
      const answer = await askAnswer(me, ch, res.message || '', retry);
      if (answer === null) throw new ProfanityCancelledError();
      profanityState.busy = true;
      res = await apiSessionFetch<T>(url, {
        method: 'POST',
        json: { ...json, challengeToken: ch.token, challengeAnswer: answer },
      });
      ch = readChallenge(res);
      retry = true;
    }
  } finally {
    closeChallenge(me);
  }
  return res;
}
