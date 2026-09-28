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
