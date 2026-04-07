"""
Конвертер MySQL дампа → PostgreSQL SQL
Исправление: корректно обрабатывает многострочные строки (реальные \n внутри VARCHAR/TEXT).
"""

import re, sys

SRC = r'c:\Users\KolomietsVM\Desktop\corporate.admsr.ru\public\data\absence_journal.sql'
DST = r'c:\Users\KolomietsVM\Desktop\corporate.admsr.ru\public\data\absence_journal_pg.sql'

COLUMNS = 'id, user_id, fio, ofo, pos, start_datetime, end_datetime, reason, created_at'
COL_COUNT = 9   # ожидаем ровно 9 значений в каждой строке
BATCH = 300     # строк на один INSERT


# ─── State-machine парсер SQL VALUES ─────────────────────────────────────────

def parse_values_block(block: str) -> list[str]:
    """
    Разбирает блок между VALUES и ;.
    Корректно обрабатывает:
      - многострочные строки (реальные \\n внутри '')
      - escape-последовательности MySQL (\\', \\n, \\r, \\\\, …)
      - '' как экранированная кавычка внутри строки
      - '(' и ')' внутри строковых литералов
    Возвращает список строк вида '(val1, val2, ...)'
    """
    rows: list[str] = []
    n = len(block)
    i = 0

    while i < n:
        # Ищем начало строки-значений '('
        while i < n and block[i] != '(':
            i += 1
        if i >= n:
            break

        row_start = i
        depth = 0
        in_str = False
        j = i

        while j < n:
            c = block[j]

            if in_str:
                if c == '\\':           # MySQL escape: \' \n \r \\ итп
                    j += 2              # пропускаем символ после слеша
                elif c == "'":
                    j += 1
                    if j < n and block[j] == "'":   # '' — экранированная кавычка
                        j += 1
                    else:
                        in_str = False  # конец строки
                else:
                    j += 1
            else:
                if c == "'":
                    in_str = True
                    j += 1
                elif c == '(':
                    depth += 1
                    j += 1
                elif c == ')':
                    depth -= 1
                    j += 1
                    if depth == 0:
                        rows.append(block[i:j])
                        i = j
                        break
                else:
                    j += 1
        else:
            # Незакрытая строка — пропускаем
            i = j

    return rows


# ─── Конвертация строки для PostgreSQL ───────────────────────────────────────

def convert_row_for_pg(row: str) -> str | None:
    """
    Обрабатывает одну строку-значений:
    - Если строковое поле содержит реальные \\n/\\r → оборачивает в E'...'
    - Возвращает None если строка содержит неверное кол-во столбцов
    """
    # Быстрая проверка: подсчитываем кол-во значений через мини-парсер
    values = split_values(row)
    if values is None or len(values) != COL_COUNT:
        print(f'  SKIP (col count={len(values) if values else "?"}): {row[:60]}…', file=sys.stderr)
        return None

    # Проверяем, есть ли поля с реальными переносами строк
    if '\n' not in row and '\r' not in row:
        return row   # быстрый путь — ничего не надо конвертировать

    # Оборачиваем каждое строковое поле в E'...' если содержит \n или \r
    result = convert_strings_to_estring(row)
    return result


def split_values(row: str) -> list[str] | None:
    """Разбивает '(v1, v2, ...)' на список значений."""
    if not (row.startswith('(') and row.endswith(')')):
        return None
    inner = row[1:-1]
    vals = []
    i = 0
    n = len(inner)
    buf = []
    in_str = False

    while i < n:
        c = inner[i]
        if in_str:
            buf.append(c)
            if c == '\\':
                i += 1
                if i < n:
                    buf.append(inner[i])
                    i += 1
            elif c == "'":
                i += 1
                if i < n and inner[i] == "'":
                    buf.append(inner[i])
                    i += 1
                else:
                    in_str = False
            else:
                i += 1
        else:
            if c == "'":
                in_str = True
                buf.append(c)
                i += 1
            elif c == ',':
                vals.append(''.join(buf).strip())
                buf = []
                i += 1
            else:
                buf.append(c)
                i += 1

    if buf or not vals:
        vals.append(''.join(buf).strip())

    return vals


def convert_strings_to_estring(row: str) -> str:
    """
    Находит строковые литералы внутри VALUES-строки и конвертирует те,
    что содержат реальные переносы строк, в PostgreSQL E'...' формат.
    """
    result = []
    i = 0
    n = len(row)

    while i < n:
        c = row[i]
        if c == "'":
            # Собираем строковый литерал
            j = i + 1
            content = []
            has_real_newline = False

            while j < n:
                sc = row[j]
                if sc == '\\':
                    j += 1
                    if j < n:
                        content.append('\\')
                        content.append(row[j])
                        j += 1
                elif sc == "'":
                    j += 1
                    if j < n and row[j] == "'":
                        content.append("''")
                        j += 1
                    else:
                        break
                else:
                    if sc == '\n':
                        has_real_newline = True
                        content.append('\\n')   # реальный \n → escape \n
                    elif sc == '\r':
                        has_real_newline = True
                        # пропускаем \r если за ним идёт \n (уже обработаем)
                        if j + 1 < n and row[j + 1] == '\n':
                            j += 1
                            continue
                        content.append('\\r')
                    else:
                        content.append(sc)
                    j += 1

            str_content = ''.join(content)

            if has_real_newline:
                result.append("E'")
                result.append(str_content)
                result.append("'")
            else:
                result.append("'")
                result.append(str_content)
                result.append("'")

            i = j
        else:
            result.append(c)
            i += 1

    return ''.join(result)


# ─── Поиск блоков VALUES с state-machine (без regex) ─────────────────────────

def find_values_blocks(content: str) -> list[str]:
    """
    Находит все блоки данных INSERT INTO `absence_journal` ... VALUES\n ... ;
    используя state-machine вместо re.DOTALL, чтобы корректно пропускать ';'
    внутри строковых литералов с реальными переносами строк.
    """
    blocks: list[str] = []
    i = 0
    n = len(content)
    header_pat = re.compile(
        r'INSERT\s+INTO\s+`absence_journal`[^V]+VALUES\s*\n',
        re.IGNORECASE
    )

    while i < n:
        m = header_pat.search(content, i)
        if not m:
            break

        j = m.end()  # начало данных VALUES
        in_str = False
        buf: list[str] = []

        while j < n:
            c = content[j]

            if in_str:
                buf.append(c)
                if c == '\\':       # MySQL escape: \' \n \r \\ …
                    j += 1
                    if j < n:
                        buf.append(content[j])
                        j += 1
                elif c == "'":
                    j += 1
                    if j < n and content[j] == "'":  # '' — экранированная кавычка
                        buf.append(content[j])
                        j += 1
                    else:
                        in_str = False
                else:
                    j += 1
            else:
                if c == "'":
                    in_str = True
                    buf.append(c)
                    j += 1
                elif c == ';':      # настоящий конец блока VALUES
                    blocks.append(''.join(buf))
                    j += 1
                    break
                else:
                    buf.append(c)
                    j += 1

        i = j

    return blocks


# ─── Основная конвертация ─────────────────────────────────────────────────────

def main() -> None:
    print(f'Читаю: {SRC}')
    with open(SRC, 'r', encoding='utf-8') as f:
        content = f.read().replace('\r\n', '\n')

    # Найти все блоки VALUES через state-machine (корректно обрабатывает ; в строках)
    all_raw_rows: list[str] = []
    for block in find_values_blocks(content):
        block_rows = parse_values_block(block)
        all_raw_rows.extend(block_rows)

    print(f'Всего строк (raw): {len(all_raw_rows)}')

    # Конвертируем и фильтруем
    valid_rows: list[str] = []
    skipped = 0
    for raw in all_raw_rows:
        converted = convert_row_for_pg(raw)
        if converted is not None:
            valid_rows.append(converted)
        else:
            skipped += 1

    print(f'Валидных строк: {len(valid_rows)}, пропущено: {skipped}')

    # Определяем max id для sequence
    max_id = 0
    for row in valid_rows:
        m2 = re.match(r'\((\d+),', row)
        if m2:
            max_id = max(max_id, int(m2.group(1)))
    print(f'Максимальный id: {max_id}')

    # Пишем PostgreSQL файл
    with open(DST, 'w', encoding='utf-8', newline='\n') as out:
        def w(s: str = '') -> None:
            out.write(s + '\n')

        w('-- PostgreSQL: Журнал отсутствия сотрудников')
        w('-- Конвертировано из MySQL дампа (phpMyAdmin)')
        w('-- Применить:')
        w('--   psql -U myuser -d corporate_portal -f public/data/absence_journal_pg.sql')
        w(f'-- Строк данных: {len(valid_rows)}, пропущено (некорректных): {skipped}')
        w(f'-- Максимальный id: {max_id}')
        w()
        w('BEGIN;')
        w()

        # Схема
        w('CREATE TABLE IF NOT EXISTS public.absence_journal (')
        w('    id              SERIAL         PRIMARY KEY,')
        w('    user_id         INTEGER        NOT NULL,')
        w('    fio             VARCHAR(255)   NOT NULL,')
        w('    ofo             INTEGER        NOT NULL,')
        w('    pos             INTEGER        NOT NULL,')
        w('    start_datetime  TIMESTAMP      NOT NULL,')
        w('    end_datetime    TIMESTAMP      DEFAULT NULL,')
        w('    reason          TEXT           DEFAULT NULL,')
        w('    created_at      TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP')
        w(');')
        w()
        w("COMMENT ON TABLE  public.absence_journal                IS 'Журнал отсутствия сотрудников';")
        w("COMMENT ON COLUMN public.absence_journal.user_id        IS 'ID пользователя';")
        w("COMMENT ON COLUMN public.absence_journal.fio            IS 'ФИО пользователя';")
        w("COMMENT ON COLUMN public.absence_journal.ofo            IS 'ID ОФО';")
        w("COMMENT ON COLUMN public.absence_journal.pos            IS 'ID должности';")
        w("COMMENT ON COLUMN public.absence_journal.start_datetime IS 'Дата и время начала отсутствия';")
        w("COMMENT ON COLUMN public.absence_journal.end_datetime   IS 'Дата и время конца (NULL = незавершённое)';")
        w("COMMENT ON COLUMN public.absence_journal.reason         IS 'Причина отсутствия';")
        w("COMMENT ON COLUMN public.absence_journal.created_at     IS 'Дата создания записи';")
        w()
        w('CREATE INDEX IF NOT EXISTS idx_absence_user_id        ON public.absence_journal (user_id);')
        w('CREATE INDEX IF NOT EXISTS idx_absence_ofo            ON public.absence_journal (ofo);')
        w('CREATE INDEX IF NOT EXISTS idx_absence_pos            ON public.absence_journal (pos);')
        w('CREATE INDEX IF NOT EXISTS idx_absence_start_datetime ON public.absence_journal (start_datetime DESC);')
        w('CREATE INDEX IF NOT EXISTS idx_absence_end_datetime   ON public.absence_journal (end_datetime);')
        w('CREATE INDEX IF NOT EXISTS idx_absence_active         ON public.absence_journal (user_id) WHERE end_datetime IS NULL;')
        w()

        # Данные батчами
        col_header = (
            f'INSERT INTO public.absence_journal ({COLUMNS}) VALUES\n'
        )
        total = len(valid_rows)
        for i, row in enumerate(valid_rows):
            pos_in_batch = i % BATCH
            is_batch_end = (pos_in_batch == BATCH - 1) or (i == total - 1)

            if pos_in_batch == 0:
                out.write(col_header)

            if is_batch_end:
                out.write(row + ';\n\n')
            else:
                out.write(row + ',\n')

        # Sequence reset
        w('-- Сбрасываем sequence после вставки данных')
        w('SELECT setval(')
        w("    pg_get_serial_sequence('public.absence_journal', 'id'),")
        w('    COALESCE((SELECT MAX(id) FROM public.absence_journal), 1)')
        w(');')
        w()
        w('COMMIT;')

    print(f'Готово: {DST}')


if __name__ == '__main__':
    main()
