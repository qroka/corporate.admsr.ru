import type { EditorToolbarItem } from '@nuxt/ui';

/** Панель инструментов для UEditor на страницах новостей ([Nuxt UI Editor](https://ui.nuxt.com/docs/components/editor)). */
export const newsEditorToolbarItems: EditorToolbarItem[][] = [
  [
    { kind: 'undo', icon: 'i-lucide-undo', tooltip: { text: 'Отменить' } },
    { kind: 'redo', icon: 'i-lucide-redo', tooltip: { text: 'Повторить' } },
  ],
  [
    {
      icon: 'i-lucide-heading',
      tooltip: { text: 'Заголовки' },
      content: { align: 'start' },
      items: [
        { kind: 'heading', level: 1, icon: 'i-lucide-heading-1', label: 'Заголовок 1' },
        { kind: 'heading', level: 2, icon: 'i-lucide-heading-2', label: 'Заголовок 2' },
        { kind: 'heading', level: 3, icon: 'i-lucide-heading-3', label: 'Заголовок 3' },
      ],
    },
    {
      icon: 'i-lucide-list',
      tooltip: { text: 'Списки' },
      content: { align: 'start' },
      items: [
        { kind: 'bulletList', icon: 'i-lucide-list', label: 'Маркированный' },
        { kind: 'orderedList', icon: 'i-lucide-list-ordered', label: 'Нумерованный' },
      ],
    },
    { kind: 'blockquote', icon: 'i-lucide-text-quote', tooltip: { text: 'Цитата' } },
    { kind: 'codeBlock', icon: 'i-lucide-square-code', tooltip: { text: 'Блок кода' } },
    {
      icon: 'i-lucide-align-left',
      tooltip: { text: 'Выравнивание' },
      content: { align: 'start' },
      items: [
        { kind: 'textAlign', align: 'left', icon: 'i-lucide-align-left', label: 'Слева' },
        { kind: 'textAlign', align: 'center', icon: 'i-lucide-align-center', label: 'По центру' },
        { kind: 'textAlign', align: 'right', icon: 'i-lucide-align-right', label: 'Справа' },
        { kind: 'textAlign', align: 'justify', icon: 'i-lucide-align-justify', label: 'По ширине' },
      ],
    },
  ],
  [
    { kind: 'mark', mark: 'bold', icon: 'i-lucide-bold', tooltip: { text: 'Жирный' } },
    { kind: 'mark', mark: 'italic', icon: 'i-lucide-italic', tooltip: { text: 'Курсив' } },
    { kind: 'mark', mark: 'underline', icon: 'i-lucide-underline', tooltip: { text: 'Подчёркнутый' } },
    { kind: 'mark', mark: 'strike', icon: 'i-lucide-strikethrough', tooltip: { text: 'Зачёркнутый' } },
    { kind: 'mark', mark: 'code', icon: 'i-lucide-code', tooltip: { text: 'Код' } },
  ],
  [
    { kind: 'link', icon: 'i-lucide-link', tooltip: { text: 'Ссылка' } },
  ],
];
