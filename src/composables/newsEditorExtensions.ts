import TextAlign from '@tiptap/extension-text-align';
import { Emoji, gitHubEmojis } from '@tiptap/extension-emoji';

/** Расширения редактора новостей: выравнивание текста и эмодзи (меню по «:»). */
export const newsEditorExtensions = [
  TextAlign.configure({
    types: ['heading', 'paragraph'],
  }),
  Emoji.configure({
    emojis: gitHubEmojis,
  }),
];

/** Список для UEditorEmojiMenu (без флагов-регионов — короче меню). */
export const newsEditorEmojiMenuItems = gitHubEmojis.filter(
  (e) => !e.name.startsWith('regional_indicator_'),
);
