import { avatarUrlFromFilename, DEFAULT_PROFILE_AVATAR_FILENAME } from '../constants/profileAvatars';

/**
 * Единый формат имени сотрудника на всём портале.
 * Поля — как в user_info: surname = фамилия, firstname = имя, lastname = отчество.
 */
type NameParts = {
  surname?: string | null;
  firstname?: string | null;
  lastname?: string | null;
};

function clean(v: unknown): string {
  return String(v ?? '').trim();
}

/** «Фамилия Имя Отчество» — списки, таблицы, журнал, результаты. */
export function userFullName(u: NameParts | null | undefined): string {
  if (!u) return '';
  return [u.surname, u.firstname, u.lastname].map(clean).filter(Boolean).join(' ');
}

/** «Фамилия Имя» — шапка, профиль, где мало места. */
export function userShortName(u: NameParts | null | undefined): string {
  if (!u) return '';
  return [u.surname, u.firstname].map(clean).filter(Boolean).join(' ');
}

/** Свой снимок (загружен сотрудником) — заполняет рамку целиком; стандартные аватары — картинки-стикеры с полями. */
export function isUploadedAvatar(src: string | null | undefined): boolean {
  return String(src ?? '').includes('/img/FullPic/avatars/uploads/');
}

/** Аватар сотрудника; без своего — тот же аватар по умолчанию, что в шапке и профиле. */
export function userAvatarSrc(u: { avatar_url?: string | null } | null | undefined): string {
  const src = String(u?.avatar_url ?? '').trim();
  return src || avatarUrlFromFilename(DEFAULT_PROFILE_AVATAR_FILENAME);
}
