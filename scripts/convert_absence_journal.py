"""
Конвертер MySQL дампа → PostgreSQL INSERT
==========================================
Читает:  public/data/absence_journal.sql   (MySQL / phpMyAdmin dump)
Пишет:   public/data/absence_journal_data_pg.sql  (PostgreSQL INSERT)

Запуск из корня проекта:
    python scripts/convert_absence_journal.py

После запуска применить в PostgreSQL:
    psql -U myuser -d corporate_portal -f public/data/absence_journal_pg.sql
    psql -U myuser -d corporate_portal -f public/data/absence_journal_data_pg.sql
"""

import re
import os

BASE_DIR  = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
INPUT     = os.path.join(BASE_DIR, "public", "data", "absence_journal.sql")
OUTPUT    = os.path.join(BASE_DIR, "public", "data", "absence_journal_data_pg.sql")

# Столбцы таблицы (порядок такой же как в дампе)
COLUMNS = "id, user_id, fio, ofo, pos, start_datetime, end_datetime, reason, created_at"


def convert(src: str, dst: str) -> None:
    with open(src, "r", encoding="utf-8") as f:
        content = f.read()

    # ── Извлекаем блок VALUES из MySQL INSERT ──────────────────────────────────
    # MySQL dump: INSERT INTO `table` (`c1`, ...) VALUES\n(row1),\n(row2),...;
    insert_match = re.search(
        r"INSERT INTO `absence_journal`[^(]+VALUES\s*\n(.*?);",
        content,
        re.DOTALL,
    )
    if not insert_match:
        raise RuntimeError("INSERT INTO ... VALUES не найден в исходном файле")

    rows_raw = insert_match.group(1).strip()

    # ── Нормализуем строки ────────────────────────────────────────────────────
    # Убираем \r (Windows line endings из mysqldump)
    rows_raw = rows_raw.replace("\r\n", "\n").replace("\r", "\n")

    # В MySQL-дампе каждая строка значений оканчивается на ),
    # последняя — просто )  (без запятой, перед ;)
    # Нам нужно разбить на отдельные строки-значения.
    # Используем регексп, чтобы не ломать запятые внутри строк.
    row_pattern = re.compile(r"\((?:[^)(]|\((?:[^)(])*\))*\)")
    rows = row_pattern.findall(rows_raw)

    if not rows:
        raise RuntimeError("Не удалось распарсить строки VALUES")

    # ── Пост-обработка строк ─────────────────────────────────────────────────
    # В MySQL-дампе \r\n внутри строковых литералов — это escape-последовательности.
    # В PostgreSQL стандартные строки не интерпретируют \r\n, поэтому:
    #   'текст\r\nтекст'  →  E'текст\r\nтекст'
    # Попутно экранируем $-знаки (PG воспринимает $N как параметры).
    def pg_row(row: str) -> str:
        # Если строка содержит \ внутри одиночных кавычек — нужен E-literal
        if "\\r" in row or "\\n" in row or "\\t" in row:
            # Заменяем только внутри одиночных кавычек
            def replace_escapes(m: re.Match) -> str:
                inner = m.group(1)
                inner = inner.replace("\\r\\n", "\r\n")
                inner = inner.replace("\\r", "\r")
                inner = inner.replace("\\n", "\n")
                inner = inner.replace("\\t", "\t")
                # Повторно экранируем для E-string PostgreSQL
                inner = inner.replace("\\", "\\\\")
                inner = inner.replace("'", "''")
                inner = inner.replace("\r\n", "\\r\\n")
                inner = inner.replace("\r", "\\r")
                inner = inner.replace("\n", "\\n")
                inner = inner.replace("\t", "\\t")
                return f"E'{inner}'"

            row = re.sub(r"'((?:[^'\\]|\\.)*)'", replace_escapes, row)
        return row

    processed_rows = [pg_row(r) for r in rows]

    # ── Находим максимальный id для сброса sequence ───────────────────────────
    max_id = 0
    for row in rows:
        id_match = re.match(r"\((\d+),", row)
        if id_match:
            max_id = max(max_id, int(id_match.group(1)))

    # ── Пишем результат ──────────────────────────────────────────────────────
    with open(dst, "w", encoding="utf-8", newline="\n") as out:
        out.write("-- Автоматически сгенерировано скриптом scripts/convert_absence_journal.py\n")
        out.write(f"-- Исходник: {os.path.basename(src)}\n")
        out.write(f"-- Строк данных: {len(processed_rows)}\n\n")
        out.write("BEGIN;\n\n")
        out.write(
            f"INSERT INTO public.absence_journal ({COLUMNS}) VALUES\n"
        )

        chunk_size = 500  # INSERT батчами по 500 строк — быстрее и безопаснее
        total = len(processed_rows)
        for i, row in enumerate(processed_rows):
            is_last_in_chunk = ((i + 1) % chunk_size == 0) or (i == total - 1)
            is_last_overall  = i == total - 1

            if is_last_in_chunk and not is_last_overall:
                out.write(f"{row};\n\n")
                out.write(
                    f"INSERT INTO public.absence_journal ({COLUMNS}) VALUES\n"
                )
            elif is_last_overall:
                out.write(f"{row};\n\n")
            else:
                out.write(f"{row},\n")

        # Сбрасываем sequence на max(id) + 1
        out.write(
            f"-- Сбрасываем sequence после массовой вставки\n"
            f"SELECT setval(\n"
            f"    pg_get_serial_sequence('public.absence_journal', 'id'),\n"
            f"    COALESCE((SELECT MAX(id) FROM public.absence_journal), 1)\n"
            f");\n\n"
        )
        out.write("COMMIT;\n")

    print(f"✓ Конвертировано {len(processed_rows)} строк")
    print(f"✓ Максимальный id: {max_id}")
    print(f"✓ Результат: {dst}")
    print()
    print("Применить в PostgreSQL:")
    print(f"  psql -U myuser -d corporate_portal -f public/data/absence_journal_pg.sql")
    print(f"  psql -U myuser -d corporate_portal -f public/data/absence_journal_data_pg.sql")


if __name__ == "__main__":
    convert(INPUT, OUTPUT)
