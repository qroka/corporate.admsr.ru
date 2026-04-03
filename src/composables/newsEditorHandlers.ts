import type { Editor } from '@tiptap/vue-3';
import type { EditorHandler } from '@nuxt/ui/dist/runtime/types/editor';

function isImageExtensionAvailable(editor: Editor): boolean {
  return editor.extensionManager.extensions.some((ext) => ext.name === 'image');
}

async function uploadNewsEditorImage(file: File): Promise<string> {
  const formData = new FormData();
  formData.append('image', file);
  const res = await fetch('/api/Upload/upload.php', { method: 'POST', body: formData });
  const json = await res.json();
  if (!json?.success) throw new Error(json?.message || 'Ошибка загрузки изображения');
  const src = json.data?.image;
  if (typeof src !== 'string' || !src.trim()) throw new Error('Сервер не вернул путь к файлу');
  return src.trim();
}

/** Как у Nuxt UI `createImageHandler`, но вместо prompt — загрузка на `/api/Upload/upload.php`. */
export const newsEditorHandlers: Record<string, EditorHandler> = {
  image: {
    canExecute: (editor: Editor) => editor.can().setImage({ src: '' }),
    execute(editor: Editor, cmd?: { src?: string }) {
      const chain = editor.chain().focus();
      if (cmd?.src) {
        return chain.setImage({ src: cmd.src });
      }
      const input = document.createElement('input');
      input.type = 'file';
      input.accept = 'image/jpeg,image/png,image/webp,image/gif';
      input.onchange = () => {
        const file = input.files?.[0];
        input.remove();
        if (!file) return;
        void (async () => {
          try {
            const src = await uploadNewsEditorImage(file);
            editor.chain().focus().setImage({ src }).run();
          } catch (e) {
            console.error(e);
          }
        })();
      };
      input.click();
      return chain;
    },
    isActive: (editor: Editor) => editor.isActive('image'),
    isDisabled: (editor: Editor) => !isImageExtensionAvailable(editor),
  },
};
