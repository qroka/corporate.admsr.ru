/**
 * Правки Markdown в обычном <textarea> — как помощники Obsidian: обёртка выделения
 * (жирный, курсив…), префиксы строк (заголовки, списки, цитата), продолжение списка
 * по Enter, сдвиг вложенности по Tab, ссылка из вставленного адреса.
 *
 * Чистые функции: получают текст и выделение, возвращают одну замену диапазона и
 * новое выделение (или null — «ничего не делать, пусть сработает обычный ввод»).
 * Применяет замену компонент (MarkdownEditor.vue) — через ввод текста браузером,
 * чтобы работала отмена Ctrl+Z.
 */

export type MdEdit = {
  /** Заменить v.slice(start, end) … */
  start: number;
  end: number;
  /** … на text */
  text: string;
  /** и выделить в новом тексте */
  selStart: number;
  selEnd: number;
};

const URL_RE = /^(https?:\/\/|mailto:)\S+$/i;

export function isUrl(s: string): boolean {
  return URL_RE.test(s.trim());
}

/** Границы строк, которые задевает выделение [s, e). Выделение до начала следующей строки её не захватывает. */
function lineRange(v: string, s: number, e: number): [number, number] {
  const ee = e > s && v[e - 1] === '\n' ? e - 1 : e;
  const ls = v.lastIndexOf('\n', s - 1) + 1;
  let le = v.indexOf('\n', ee);
  if (le === -1) le = v.length;
  return [ls, le];
}

/** Заменить строки, задетые выделением, результатом fn. */
function mapLines(v: string, s: number, e: number, fn: (lines: string[]) => string[]): MdEdit {
  const [ls, le] = lineRange(v, s, e);
  const lines = v.slice(ls, le).split('\n');
  const text = fn(lines).join('\n');
  if (lines.length === 1 && s === e) {
    // Курсор в одной строке — сдвигаем его вместе с изменением начала строки.
    const pos = Math.min(ls + text.length, Math.max(ls, s + text.length - (le - ls)));
    return { start: ls, end: le, text, selStart: pos, selEnd: pos };
  }
  return { start: ls, end: le, text, selStart: ls, selEnd: ls + text.length };
}

// ── Обёртка выделения ──────────────────────────────────────────────────────────

/** **жирный**, _курсив_, ~~зачёркнутый~~, `код`: повторное нажатие снимает разметку. */
export function toggleWrap(v: string, s: number, e: number, marker: string, placeholder: string): MdEdit {
  const sel = v.slice(s, e);
  const n = marker.length;
  // Разметка снаружи выделения: **[текст]**
  if (v.slice(s - n, s) === marker && v.slice(e, e + n) === marker) {
    return { start: s - n, end: e + n, text: sel, selStart: s - n, selEnd: e - n };
  }
  // Разметка внутри выделения: [**текст**]
  if (sel.length >= 2 * n && sel.startsWith(marker) && sel.endsWith(marker)) {
    const inner = sel.slice(n, sel.length - n);
    return { start: s, end: e, text: inner, selStart: s, selEnd: s + inner.length };
  }
  const inner = sel || placeholder;
  return { start: s, end: e, text: marker + inner + marker, selStart: s + n, selEnd: s + n + inner.length };
}

// ── Префиксы строк ─────────────────────────────────────────────────────────────

const HEADING_RE = /^(#{1,6})\s+/;
const BULLET_RE = /^(\s*)[-*+]\s+/;
const NUMBER_RE = /^(\s*)\d+[.)]\s+/;
const QUOTE_RE = /^>\s?/;

/** Убрать маркер списка (любого) после отступа. */
function stripListMarker(line: string): [indent: string, rest: string] {
  const m = BULLET_RE.exec(line) || NUMBER_RE.exec(line);
  if (m) return [m[1], line.slice(m[0].length)];
  const indent = /^\s*/.exec(line)![0];
  return [indent, line.slice(indent.length)];
}

/** # / ## / ###: тот же уровень — снять, другой — заменить. */
export function toggleHeading(v: string, s: number, e: number, level: number): MdEdit {
  return mapLines(v, s, e, (lines) =>
    lines.map((line) => {
      const m = HEADING_RE.exec(line);
      const rest = m ? line.slice(m[0].length) : line;
      if (m && m[1].length === level) return rest;
      return `${'#'.repeat(level)} ${rest}`;
    }),
  );
}

/** «- » у каждой строки; если все уже пункты — снять. Нумерованные превращаются в маркированные. */
export function toggleBulletList(v: string, s: number, e: number): MdEdit {
  return mapLines(v, s, e, (lines) => {
    const filled = lines.filter((l) => l.trim());
    const all = filled.length > 0 && filled.every((l) => BULLET_RE.test(l));
    return lines.map((line) => {
      if (!line.trim()) return line;
      if (all) return line.replace(BULLET_RE, '$1');
      const [indent, rest] = stripListMarker(line);
      return `${indent}- ${rest}`;
    });
  });
}

/** «1. », «2. »… по непустым строкам; если все уже нумерованные — снять. */
export function toggleNumberedList(v: string, s: number, e: number): MdEdit {
  return mapLines(v, s, e, (lines) => {
    const filled = lines.filter((l) => l.trim());
    const all = filled.length > 0 && filled.every((l) => NUMBER_RE.test(l));
    let n = 0;
    return lines.map((line) => {
      if (!line.trim()) return line;
      if (all) return line.replace(NUMBER_RE, '$1');
      const [indent, rest] = stripListMarker(line);
      n += 1;
      return `${indent}${n}. ${rest}`;
    });
  });
}

/** «> » у каждой строки; если все уже цитата — снять. */
export function toggleQuote(v: string, s: number, e: number): MdEdit {
  return mapLines(v, s, e, (lines) => {
    const all = lines.every((l) => QUOTE_RE.test(l) || !l.trim());
    return lines.map((line) => (all ? line.replace(QUOTE_RE, '') : `> ${line}`));
  });
}

/** Блок кода: обернуть строки в ``` или вставить пустой блок с курсором внутри. */
export function insertCodeBlock(v: string, s: number, e: number): MdEdit {
  if (s === e) {
    const before = s > 0 && v[s - 1] !== '\n' ? '\n' : '';
    const text = `${before}\`\`\`\n\n\`\`\``;
    const pos = s + before.length + 4;
    return { start: s, end: e, text, selStart: pos, selEnd: pos };
  }
  const [ls, le] = lineRange(v, s, e);
  const block = v.slice(ls, le);
  const text = `\`\`\`\n${block}\n\`\`\``;
  return { start: ls, end: le, text, selStart: ls + 4, selEnd: ls + 4 + block.length };
}

/** Ссылка: выделен адрес — [|](адрес); выделен текст — [текст](|https://|). */
export function insertLink(v: string, s: number, e: number): MdEdit {
  const sel = v.slice(s, e);
  if (sel && isUrl(sel)) {
    return { start: s, end: e, text: `[](${sel.trim()})`, selStart: s + 1, selEnd: s + 1 };
  }
  const label = sel || 'текст ссылки';
  const url = 'https://';
  const text = `[${label}](${url})`;
  const urlStart = s + label.length + 3;
  return sel
    ? { start: s, end: e, text, selStart: urlStart, selEnd: urlStart + url.length }
    : { start: s, end: e, text, selStart: s + 1, selEnd: s + 1 + label.length };
}

/** Вставили адрес поверх выделенного текста — сделать ссылку. Иначе — обычная вставка (null). */
export function pasteAsLink(v: string, s: number, e: number, pasted: string): MdEdit | null {
  const sel = v.slice(s, e);
  if (!sel || sel.includes('\n') || isUrl(sel) || !isUrl(pasted)) return null;
  const text = `[${sel}](${pasted.trim()})`;
  return { start: s, end: e, text, selStart: s + text.length, selEnd: s + text.length };
}

// ── Enter и Tab в списках ──────────────────────────────────────────────────────

const LIST_LINE_RE = /^(\s*)(?:([-*+])|(\d+)([.)]))\s+/;

/** Строка курсора — пункт списка или цитата? */
export function isListLine(v: string, pos: number): boolean {
  const [ls, le] = lineRange(v, pos, pos);
  const line = v.slice(ls, le);
  return LIST_LINE_RE.test(line) || QUOTE_RE.test(line);
}

/**
 * Enter в пункте списка или цитате: новый пункт с тем же маркером (номер + 1);
 * Enter на пустом пункте — убрать маркер (выйти из списка). Не список — null.
 */
export function continueList(v: string, s: number, e: number): MdEdit | null {
  if (s !== e) return null;
  const [ls, le] = lineRange(v, s, s);
  const line = v.slice(ls, le);
  let prefixLen: number;
  let next: string;
  const m = LIST_LINE_RE.exec(line);
  if (m) {
    prefixLen = m[0].length;
    next = m[2] ? `${m[1]}${m[2]} ` : `${m[1]}${Number(m[3]) + 1}${m[4]} `;
  } else {
    const q = QUOTE_RE.exec(line);
    if (!q) return null;
    prefixLen = q[0].length;
    next = '> ';
  }
  if (s - ls < prefixLen) return null; // курсор внутри маркера — обычный Enter
  if (!line.slice(prefixLen).trim()) {
    return { start: ls, end: le, text: '', selStart: ls, selEnd: ls };
  }
  const text = `\n${next}`;
  return { start: s, end: s, text, selStart: s + text.length, selEnd: s + text.length };
}

/** Tab / Shift+Tab: сдвинуть строки на 2 пробела вправо / влево. */
export function indentLines(v: string, s: number, e: number, outdent: boolean): MdEdit {
  return mapLines(v, s, e, (lines) =>
    lines.map((line) => (outdent ? line.replace(/^ {1,2}/, '') : `  ${line}`)),
  );
}

// ── Счётчик ────────────────────────────────────────────────────────────────────

export function countWords(v: string): number {
  return v.trim() ? v.trim().split(/\s+/).length : 0;
}
