import type { EditorToolbarItem } from '@nuxt/ui';
import { newsEditorToolbarItems } from './newsEditorToolbar';

/** Панель инструментов стены: форматирование + фото и смайлики в тулбаре. */
export const wallEditorToolbarItems: EditorToolbarItem[][] = [
  ...newsEditorToolbarItems.slice(0, -1),
  [
    ...newsEditorToolbarItems[newsEditorToolbarItems.length - 1],
    { kind: 'image', icon: 'i-lucide-image', tooltip: { text: 'Фото' } },
    { kind: 'emoji', icon: 'i-lucide-smile', tooltip: { text: 'Смайлик' } },
  ],
];
