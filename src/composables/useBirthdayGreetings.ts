import { ref } from 'vue';
import { apiSessionFetch, getAuthUser } from './useAuthSession';
import { WALL_POST_MAX_LENGTH } from './useProfileWall';
import { postWithProfanityGate } from './useProfanityGate';

/**
 * Поздравление с днём рождения — запись на стене именинника
 * (/api/profile_wall.php, action=greet; wall_posts.birthday_year, V17).
 * Поздравить можно в день рождения и BIRTHDAY_GREET_WINDOW_DAYS дней после,
 * один раз в год. Окно повторяет birthdayGreetWindowDays в
 * backend/internal/handlers/birthday_greetings.go — менять парами; здесь оно
 * нужно только для показа кнопки, решает сервер.
 */
export const BIRTHDAY_GREET_WINDOW_DAYS = 3;

export { WALL_POST_MAX_LENGTH as BIRTHDAY_GREETING_MAX_LENGTH };

/** `{name}` — имя и отчество именинника. */
export const BIRTHDAY_GREETING_TEMPLATES = [
  '{name}, с днём рождения! Здоровья, сил и хорошего настроения.',
  '{name}, поздравляю с днём рождения! Пусть работа приносит радость, а коллеги — поддержку.',
  'С днём рождения, {name}! Успехов во всех делах и ярких событий в новом году жизни.',
  '{name}, от всей души поздравляю! Счастья, тепла и исполнения задуманного.',
  'С праздником, {name}! Пусть этот год будет спокойным, удачным и щедрым на хорошие новости.',
] as const;

export type BirthdayGreetTarget = {
  userId: number;
  /** ФИО «Фамилия Имя Отчество», как в xlsx */
  name: string;
  avatar?: string;
  /** Год дня рождения, с которым поздравляем */
  year: number;
};

/** Что показать рядом с именинником. */
export type BirthdayGreetState = 'greet' | 'greeted' | 'none';

type MyGreeting = { postId: number; userId: number; year: number };

const myGreetings = ref<MyGreeting[]>([]);
let loadPromise: Promise<void> | null = null;
let loadedFor = 0;

const target = ref<BirthdayGreetTarget | null>(null);
const slideoverOpen = ref(false);

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

/**
 * Год, с которым можно поздравить, если день рождения выпал на `date`
 * (конкретная дата в календаре), или null — вне окна.
 */
export function greetYearForDate(date: Date): number | null {
  const days = Math.round((startOfDay(new Date()).getTime() - startOfDay(date).getTime()) / 86_400_000);
  return days >= 0 && days <= BIRTHDAY_GREET_WINDOW_DAYS ? date.getFullYear() : null;
}

/** «Филяков Ставр Владимирович» → «Ставр Владимирович» (для обращения в шаблонах). */
export function greetingAddressName(fio: string): string {
  const parts = fio.trim().split(/\s+/).filter(Boolean);
  return parts.length >= 2 ? parts.slice(1).join(' ') : fio.trim();
}

export function useBirthdayGreetings() {
  function currentUserId(): number {
    return Number(getAuthUser()?.id) || 0;
  }

  async function load(force = false) {
    const me = currentUserId();
    if (!me) {
      myGreetings.value = [];
      return;
    }
    if (!force && loadedFor === me) return;
    if (loadPromise) return loadPromise;
    loadPromise = (async () => {
      try {
        const json = await apiSessionFetch<MyGreeting[]>('/api/profile_wall.php?action=my_greetings');
        if (json?.success && Array.isArray(json.data)) {
          myGreetings.value = json.data.map((g) => ({
            postId: Number(g.postId) || 0,
            userId: Number(g.userId) || 0,
            year: Number(g.year) || 0,
          }));
          loadedFor = me;
        }
      } catch {
        // без списка кнопка «Поздравить» останется — повтор сервер отклонит
      } finally {
        loadPromise = null;
      }
    })();
    return loadPromise;
  }

  function ensureLoaded() {
    void load();
  }

  /**
   * greet — можно поздравить; greeted — уже поздравили с этим днём рождения;
   * none — не с кем (ФИО без учётной записи), это вы сами или вне окна.
   */
  function greetState(userId: number | null | undefined, birthdayDate: Date): BirthdayGreetState {
    if (!userId || userId === currentUserId()) return 'none';
    const year = greetYearForDate(birthdayDate);
    if (year == null) return 'none';
    return myGreetings.value.some((g) => g.userId === userId && g.year === year) ? 'greeted' : 'greet';
  }

  function openGreeting(person: { userId: number; name: string; avatar?: string }, birthdayDate: Date) {
    const year = greetYearForDate(birthdayDate);
    if (year == null) return;
    target.value = { ...person, year };
    slideoverOpen.value = true;
  }

  /** Публикует поздравление; при отказе бросает Error с текстом для тоста. */
  async function sendGreeting(content: string): Promise<void> {
    const t = target.value;
    if (!t) throw new Error('Не выбран именинник');
    // Свой текст с матом — сначала задача (useProfanityGate).
    const json = await postWithProfanityGate<{ id?: number; birthdayYear?: number }>('/api/profile_wall.php', {
      action: 'greet',
      userId: t.userId,
      content: content.trim(),
    });
    if (!json?.success || !json.data?.id) {
      // 409 «уже поздравили» — подтянуть список, чтобы кнопка сменилась на «Вы поздравили»
      void load(true);
      throw new Error(json?.message || 'Не удалось опубликовать поздравление');
    }
    myGreetings.value = [
      ...myGreetings.value,
      { postId: Number(json.data.id), userId: t.userId, year: Number(json.data.birthdayYear) || t.year },
    ];
  }

  return {
    target,
    slideoverOpen,
    ensureLoaded,
    reload: () => load(true),
    greetState,
    openGreeting,
    sendGreeting,
  };
}
