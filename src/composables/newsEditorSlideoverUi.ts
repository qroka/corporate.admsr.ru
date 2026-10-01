/**
 * Отступы у области ввода TipTap (`ui.base` → класс на contenteditable ProseMirror).
 * Тема редактора задаёт `sm:px-8`; для слайдовера нужен компактный `p-3`.
 * `min-h-44` растягивает поле на всю рамку: иначе кликабельна только первая строка.
 */
export const newsEditorSlideoverUi = {
  base: '!p-3 sm:!px-3 min-h-44',
};
