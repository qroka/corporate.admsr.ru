import TextAlign from '@tiptap/extension-text-align';
import { gitHubEmojis } from '@tiptap/extension-emoji';

/**
 * Расширения редактора стены.
 * TipTap Emoji extension не подключаем — у него свой Suggestion на «:»,
 * который конфликтует с UEditorEmojiMenu (см. Nuxt UI docs).
 */
export const wallEditorExtensions = [
  TextAlign.configure({
    types: ['heading', 'paragraph'],
  }),
];

export const newsEditorEmojiMenuItems = gitHubEmojis.filter(
  (e) => !e.name.startsWith('regional_indicator_'),
);
