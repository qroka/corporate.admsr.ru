"""Разбор Тест.docx из архива курса в вопросы модуля тестов.

Правильный ответ в документах выделен красным цветом шрифта (EE0000).
Вопросы с вариантами — тип single. Таблицы «соотнесите» / «подберите» становятся вопросом
«Соответствие» (match), таблицы «распределите» / «классифицируйте» — «Классификацией» (classify):
  {'type': 'match'|'classify', 'items': [текст слева], 'options': [текст справа], 'key': [индекс варианта для каждого элемента]}
"""
import json
import os
import re
import sys

import docx
from docx.oxml.ns import qn
from docx.table import Table
from docx.text.paragraph import Paragraph

ROOT = sys.argv[1]
OUT = sys.argv[2]

OPT_RE = re.compile(r'^([АБВГДЕЖ])[.)]\s*(.+)$')
Q_RE = re.compile(r'^(\d+)[.)]?\s+(.+)$')
KEY_RE = re.compile(r'^(?:[А-Я]\d+\s*)+$')
SKIP = ('Выберите один правильный ответ', 'Проверка автоматического блока', 'Выберите один ответ. Подтверждайте')


def is_red(run):
    c = run.font.color
    return bool(c is not None and c.type is not None and str(c.rgb) not in ('000000', 'None'))


def para_lines(p):
    """[(red, text)] по строкам абзаца (разрывы строк w:br)."""
    lines, cur, red = [], '', False
    for r in p.runs:
        for ch in r._r.iterchildren():
            if ch.tag == qn('w:br'):
                lines.append((red, cur))
                cur, red = '', False
            elif ch.tag == qn('w:t'):
                t = ch.text or ''
                cur += t
                if t.strip() and is_red(r):
                    red = True
    lines.append((red, cur))
    return [(rd, t.strip()) for rd, t in lines if t.strip()]


def is_bold(p):
    runs = [r for r in p.runs if r.text.strip()]
    return bool(runs) and all(r.bold for r in runs)


def clean(t):
    return re.sub(r'\s+', ' ', t).strip()


def parse_key(text):
    return {m.group(1): [int(d) for d in m.group(2)] for m in re.finditer(r'([А-Я])(\d+)', text)}


def parse(path):
    d = docx.Document(path)
    questions, context = [], []
    q = None
    last_table = None
    last_bold = ''
    title = None
    for el in d.element.body.iterchildren():
        if el.tag == qn('w:tbl'):
            last_table = [[clean(c.text) for c in row.cells] for row in Table(el, d).rows]
            q = None
            continue
        if el.tag != qn('w:p'):
            continue
        p = Paragraph(el, d)
        text = clean(p.text)
        if not text:
            continue
        lines = para_lines(p)

        # Ключ к таблице: «А4 Б3 В2 …» или «А12 Б35 В46».
        if last_table is not None and KEY_RE.match(text.replace(' ', ' ')):
            questions.append(expand_table(last_table, parse_key(text), last_bold, context))
            last_table, context, last_bold = None, [], ''
            continue

        # Абзац с вариантами (каждый вариант — строка) или один вариант.
        opts = [(rd, OPT_RE.match(t)) for rd, t in lines]
        if q is not None and all(m for _, m in opts):
            for rd, m in opts:
                q['options'].append({'text': clean(m.group(2)), 'correct': rd})
            continue

        if any(text.startswith(s) for s in SKIP):
            continue

        m = Q_RE.match(text)
        if m and is_bold(p):
            q = {'title': clean(m.group(2)), 'hint': '', 'options': []}
            questions.append(q)
            context = []
            continue

        # Шапка сценария («1 Проверка цифр и дат») без вопроса — заменяется текстом ситуации.
        if q is not None and not q['options'] and not is_bold(p):
            q['title'] = text if '?' not in q['title'] else q['title'] + ' ' + text
            continue

        # Длинное жирное утверждение — вопрос-подтверждение ознакомления.
        if is_bold(p) and len(text) > 150:
            q = {'title': text, 'hint': 'Подтверждение ознакомления', 'options': []}
            questions.append(q)
            continue

        if title is None and is_bold(p) and not questions:
            title = text
        q = None if (q is not None and q['options']) else q
        if is_bold(p):
            last_bold = text  # заголовок задания: «Соотнесите понятия с определениями:»
        else:
            context.append(text)
    return title, questions


def expand_table(rows, key, heading, context):
    """Таблица задания → один вопрос: match (понятие → определение) или classify (утверждение → категория)."""
    head = [h.lower() for h in rows[0]]
    body = rows[1:]
    title = heading.rstrip(':').strip()
    hint = ' '.join(c for c in context if not c.startswith('Выберите'))
    if 'понятие' in head or 'задача' in head:
        left = {r[0]: r[1] for r in body if r[0]}
        right = {int(r[2]): r[3] for r in body if r[2].isdigit()}
        nums = sorted(right)
        letters = [k for k in left if k in key]
        return {
            'type': 'match', 'title': title, 'hint': hint,
            'items': [left[k] for k in letters],
            'options': [right[n] for n in nums],
            'key': [nums.index(key[k][0]) for k in letters],
        }
    cats = {r[0]: r[1] for r in body if r[0]}
    items = {int(r[3]): r[4] for r in body if r[3].isdigit()}
    letters = list(cats)
    owner = {n: letters.index(letter) for letter, nums_ in key.items() for n in nums_}
    order = sorted(items)
    return {
        'type': 'classify', 'title': title, 'hint': hint,
        'items': [items[n] for n in order],
        'options': [cats[c] for c in letters],
        'key': [owner[n] for n in order],
    }


result = {}
for folder in sorted(os.listdir(ROOT)):
    f = os.path.join(ROOT, folder, 'Тест.docx')
    if not os.path.exists(f):
        continue
    title, qs = parse(f)
    def broken(q):
        if q.get('type') in ('match', 'classify'):
            return len(q['items']) != len(q['key']) or not q['items'] or any(k >= len(q['options']) for k in q['key'])
        return sum(o['correct'] for o in q['options']) != 1 or len(q['options']) < 2

    bad = [i + 1 for i, q in enumerate(qs) if broken(q)]
    print(folder, 'вопросов:', len(qs), 'проблемных:', bad)
    result[folder] = {'title': title, 'questions': qs}
json.dump(result, open(OUT, 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
