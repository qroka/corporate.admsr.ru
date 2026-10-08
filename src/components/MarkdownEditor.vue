<script setup lang="ts">
/**
 * Поле Markdown с помощниками в духе Obsidian: панель форматирования, горячие
 * клавиши, продолжение списков по Enter, вложенность по Tab, ссылка из
 * вставленного адреса, счётчик слов. Логика правок — markdownEditing.ts.
 *
 * Правки вставляются через ввод текста браузером (execCommand('insertText')) —
 * так работает отмена Ctrl+Z. Клавиши — по физической клавише (e.code), как у
 * глобального поиска: работают и на русской раскладке. Ctrl+K занят поиском
 * портала, поэтому ссылка — Ctrl+Shift+K.
 */
import { computed, ref } from 'vue';
import {
  continueList,
  countWords,
  indentLines,
  insertCodeBlock,
  insertLink,
  isListLine,
  pasteAsLink,
  toggleBulletList,
  toggleHeading,
  toggleNumberedList,
  toggleQuote,
  toggleWrap,
  type MdEdit,
} from '../composables/markdownEditing';
import { plural } from '../pages/Courses/courseDuration';

const model = defineModel<string>({ default: '' });

const props = withDefaults(
  defineProps<{ placeholder?: string; maxlength?: number; rows?: number; ariaLabel?: string }>(),
  { placeholder: '', maxlength: undefined, rows: 20, ariaLabel: 'Текст в Markdown' },
);

const field = ref<{ textareaRef?: HTMLTextAreaElement } | null>(null);

function textarea(): HTMLTextAreaElement | null {
  return field.value?.textareaRef ?? null;
}

function apply(edit: MdEdit | null) {
  const el = textarea();
  if (!el || !edit) return;
  el.focus();
  el.setSelectionRange(edit.start, edit.end);
  let done = false;
  if (edit.text) done = document.execCommand('insertText', false, edit.text);
  else if (edit.start !== edit.end) done = document.execCommand('delete');
  else done = true;
  if (!done) {
    // Браузер без execCommand — без истории отмены, но правка применится.
    el.setRangeText(edit.text, edit.start, edit.end, 'end');
    el.dispatchEvent(new Event('input', { bubbles: true }));
  }
  el.setSelectionRange(edit.selStart, edit.selEnd);
}

type Action = (v: string, s: number, e: number) => MdEdit | null;

function run(action: Action) {
  const el = textarea();
  if (!el) return;
  apply(action(el.value, el.selectionStart, el.selectionEnd));
}

const actions = {
  h1: (v: string, s: number, e: number) => toggleHeading(v, s, e, 1),
  h2: (v: string, s: number, e: number) => toggleHeading(v, s, e, 2),
  h3: (v: string, s: number, e: number) => toggleHeading(v, s, e, 3),
  bold: (v: string, s: number, e: number) => toggleWrap(v, s, e, '**', 'жирный текст'),
  italic: (v: string, s: number, e: number) => toggleWrap(v, s, e, '_', 'курсив'),
  strike: (v: string, s: number, e: number) => toggleWrap(v, s, e, '~~', 'зачёркнутый текст'),
  code: (v: string, s: number, e: number) => toggleWrap(v, s, e, '`', 'код'),
  bullet: toggleBulletList,
  numbered: toggleNumberedList,
  quote: toggleQuote,
  codeBlock: insertCodeBlock,
  link: insertLink,
} satisfies Record<string, Action>;

type ActionKey = keyof typeof actions;

const toolbar: { key: ActionKey; icon: string; label: string; shortcut?: string }[][] = [
  [
    { key: 'h1', icon: 'i-lucide-heading-1', label: 'Заголовок 1' },
    { key: 'h2', icon: 'i-lucide-heading-2', label: 'Заголовок 2' },
    { key: 'h3', icon: 'i-lucide-heading-3', label: 'Заголовок 3' },
  ],
  [
    { key: 'bold', icon: 'i-lucide-bold', label: 'Жирный', shortcut: 'Ctrl+B' },
    { key: 'italic', icon: 'i-lucide-italic', label: 'Курсив', shortcut: 'Ctrl+I' },
    { key: 'strike', icon: 'i-lucide-strikethrough', label: 'Зачёркнутый', shortcut: 'Ctrl+Shift+X' },
    { key: 'code', icon: 'i-lucide-code', label: 'Код в строке', shortcut: 'Ctrl+E' },
  ],
  [
    { key: 'bullet', icon: 'i-lucide-list', label: 'Маркированный список', shortcut: 'Ctrl+Shift+8' },
    { key: 'numbered', icon: 'i-lucide-list-ordered', label: 'Нумерованный список', shortcut: 'Ctrl+Shift+7' },
    { key: 'quote', icon: 'i-lucide-text-quote', label: 'Цитата', shortcut: 'Ctrl+Shift+9' },
    { key: 'codeBlock', icon: 'i-lucide-square-code', label: 'Блок кода' },
  ],
  [{ key: 'link', icon: 'i-lucide-link', label: 'Ссылка', shortcut: 'Ctrl+Shift+K' }],
];

/** Горячие клавиши: [Shift?, e.code] → действие. */
const SHORTCUTS: Record<string, ActionKey> = {
  'KeyB': 'bold',
  'KeyI': 'italic',
  'KeyE': 'code',
  'shift+KeyX': 'strike',
  'shift+KeyK': 'link',
  'shift+Digit8': 'bullet',
  'shift+Digit7': 'numbered',
  'shift+Digit9': 'quote',
};

function onKeydown(e: KeyboardEvent) {
  const el = e.target as HTMLTextAreaElement;
  const mod = e.ctrlKey || e.metaKey;
  if (mod && !e.altKey) {
    const key = SHORTCUTS[`${e.shiftKey ? 'shift+' : ''}${e.code}`];
    if (key) {
      e.preventDefault();
      run(actions[key]);
    }
    return;
  }
  if (e.key === 'Enter' && !e.shiftKey && !e.altKey && !e.isComposing) {
    const edit = continueList(el.value, el.selectionStart, el.selectionEnd);
    if (edit) {
      e.preventDefault();
      apply(edit);
    }
    return;
  }
  // Tab сдвигает пункты списка и несколько строк; в обычном тексте — уходит фокус (доступность).
  if (e.key === 'Tab' && !e.altKey) {
    const multiline = el.value.slice(el.selectionStart, el.selectionEnd).includes('\n');
    if (multiline || isListLine(el.value, el.selectionStart)) {
      e.preventDefault();
      apply(indentLines(el.value, el.selectionStart, el.selectionEnd, e.shiftKey));
    }
  }
}

function onPaste(e: ClipboardEvent) {
  const el = e.target as HTMLTextAreaElement;
  const pasted = e.clipboardData?.getData('text/plain') ?? '';
  const edit = pasteAsLink(el.value, el.selectionStart, el.selectionEnd, pasted);
  if (edit) {
    e.preventDefault();
    apply(edit);
  }
}

const stats = computed(() => {
  const words = countWords(model.value);
  return `${words} ${plural(words, ['слово', 'слова', 'слов'])} · ${model.value.length}${props.maxlength ? ` / ${props.maxlength}` : ''} симв.`;
});
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <div class="flex flex-wrap items-center gap-0.5 rounded-md border border-default bg-elevated/40 p-1" role="toolbar" aria-label="Форматирование Markdown">
      <template v-for="(group, gi) in toolbar" :key="gi">
        <USeparator v-if="gi > 0" orientation="vertical" class="mx-1 h-5" />
        <UTooltip
          v-for="b in group"
          :key="b.key"
          :text="b.label"
          :kbds="b.shortcut ? b.shortcut.split('+').map((k) => (k === 'Ctrl' ? 'ctrl' : k === 'Shift' ? 'shift' : k)) : undefined"
        >
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            size="xs"
            square
            :icon="b.icon"
            :aria-label="b.shortcut ? `${b.label} (${b.shortcut})` : b.label"
            @mousedown.prevent
            @click="run(actions[b.key])"
          />
        </UTooltip>
      </template>
    </div>

    <UTextarea
      ref="field"
      v-model="model"
      :rows="rows"
      :maxlength="maxlength"
      :placeholder="placeholder"
      :aria-label="ariaLabel"
      class="w-full"
      :ui="{ base: 'font-mono text-sm leading-6 min-h-[50vh]' }"
      @keydown="onKeydown"
      @paste="onPaste"
    />

    <p class="text-xs text-dimmed tabular-nums text-right">{{ stats }}</p>
  </div>
</template>
