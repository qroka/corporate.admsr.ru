"""
Replace absence_journal.user_id values with users.json internalId.

Input:
  - public/data/users.json
  - public/data/absence_journal_202604061719.sql

Output:
  - Overwrites absence_journal_202604061719.sql (creates .bak рядом)
"""

from __future__ import annotations

import json
import os
import re
import shutil
from typing import Any


ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
USERS_JSON = os.path.join(ROOT, "public", "data", "users.json")
SQL_PATH = os.path.join(ROOT, "public", "data", "absence_journal_202604061719.sql")
BAK_PATH = SQL_PATH + ".bak"


def build_id_map(users_payload: Any) -> dict[str, str]:
    """
    Returns mapping: external user id (string) -> internalId (string).
    Accepts either:
      - list[dict]
      - dict with "data" key containing list[dict]
    """
    rows = None
    # Format 1: { data: [...] }
    if isinstance(users_payload, dict) and isinstance(users_payload.get("data"), list):
        rows = users_payload["data"]

    # Format 2: plain array of rows
    if rows is None and isinstance(users_payload, list) and users_payload and isinstance(users_payload[0], dict) and "internalId" in users_payload[0]:
        rows = users_payload

    # Format 3 (phpMyAdmin export): array of objects with {type:"table", name:"users", data:[...]}
    if rows is None and isinstance(users_payload, list):
        for item in users_payload:
            if not isinstance(item, dict):
                continue
            if item.get("type") == "table" and item.get("name") == "users" and isinstance(item.get("data"), list):
                rows = item["data"]
                break

    if rows is None:
        raise ValueError("Unexpected users.json shape: could not locate users rows")

    out: dict[str, str] = {}
    for u in rows:
        if not isinstance(u, dict):
            continue
        ext = u.get("id")
        internal = u.get("internalId")
        if ext is None or internal is None:
            continue
        out[str(ext)] = str(internal)
    if not out:
        raise ValueError("No id -> internalId mappings found in users.json")
    return out


def parse_insert_user_id_index(insert_header: str) -> int | None:
    """
    Given INSERT header text containing "(col1,col2,...) VALUES",
    returns index of 'user_id' in that list, or None if not found.
    """
    m = re.search(r"\(([^)]*?)\)\s*VALUES", insert_header, flags=re.IGNORECASE | re.DOTALL)
    if not m:
        return None
    cols_raw = m.group(1)
    cols = [c.strip().strip('"').strip() for c in cols_raw.split(",")]
    try:
        return cols.index("user_id")
    except ValueError:
        return None


def find_insert_blocks(sql: str) -> list[tuple[int, int, int]]:
    """
    Returns list of (start,end) ranges for VALUES payload of:
      INSERT INTO ... VALUES\n ... ;
    The range includes ONLY the part after VALUES\\n and before the terminating ;.
    Uses state machine so semicolons inside strings don't break blocks.
    """
    header_pat = re.compile(
        r"INSERT\s+INTO\s+(?:public\.)?`?absence_journal`?[\s\S]*?VALUES\s*\n",
        re.IGNORECASE,
    )
    ranges: list[tuple[int, int, int]] = []
    i = 0
    n = len(sql)
    while i < n:
        m = header_pat.search(sql, i)
        if not m:
            break
        user_id_index = parse_insert_user_id_index(m.group(0))
        if user_id_index is None:
            i = m.end()
            continue
        j = m.end()
        in_str = False
        while j < n:
            c = sql[j]
            if in_str:
                if c == "\\":
                    j += 2
                    continue
                if c == "'":
                    j += 1
                    if j < n and sql[j] == "'":  # ''
                        j += 1
                    else:
                        in_str = False
                    continue
                j += 1
            else:
                if c == "'":
                    in_str = True
                    j += 1
                elif c == ";":
                    ranges.append((m.end(), j, user_id_index))
                    j += 1
                    break
                else:
                    j += 1
        i = j
    return ranges


def parse_rows(values_block: str) -> list[tuple[int, int]]:
    """
    Returns list of (start,end) slices for each row tuple "(...)" inside VALUES block.
    """
    rows: list[tuple[int, int]] = []
    i = 0
    n = len(values_block)
    while i < n:
        while i < n and values_block[i] != "(":
            i += 1
        if i >= n:
            break
        depth = 0
        in_str = False
        j = i
        while j < n:
            c = values_block[j]
            if in_str:
                if c == "\\":
                    j += 2
                    continue
                if c == "'":
                    j += 1
                    if j < n and values_block[j] == "'":
                        j += 1
                    else:
                        in_str = False
                    continue
                j += 1
            else:
                if c == "'":
                    in_str = True
                    j += 1
                elif c == "(":
                    depth += 1
                    j += 1
                elif c == ")":
                    depth -= 1
                    j += 1
                    if depth == 0:
                        rows.append((i, j))
                        i = j
                        break
                else:
                    j += 1
        else:
            break
    return rows


def split_values(row_tuple: str) -> list[str]:
    assert row_tuple.startswith("(") and row_tuple.endswith(")")
    inner = row_tuple[1:-1]
    vals: list[str] = []
    buf: list[str] = []
    i = 0
    n = len(inner)
    in_str = False
    while i < n:
        c = inner[i]
        if in_str:
            buf.append(c)
            if c == "\\":
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
            elif c == ",":
                vals.append("".join(buf).strip())
                buf = []
                i += 1
            else:
                buf.append(c)
                i += 1
    vals.append("".join(buf).strip())
    return vals


def replace_user_id_in_row(row_tuple: str, id_map: dict[str, str], user_id_index: int) -> tuple[str, bool]:
    """
    Replaces the value at user_id_index if present in map.
    """
    vals = split_values(row_tuple)
    if user_id_index < 0 or user_id_index >= len(vals):
        return row_tuple, False
    user_id = vals[user_id_index]
    if not user_id or user_id.upper() == "NULL":
        return row_tuple, False
    # keep only integer-like (no quotes)
    if user_id.startswith("'") and user_id.endswith("'"):
        key = user_id[1:-1]
    else:
        key = user_id
    key = key.strip()
    if key not in id_map:
        return row_tuple, False
    new_val = id_map[key]
    # keep quoting style (normally none)
    vals[user_id_index] = new_val if not (user_id.startswith("'") and user_id.endswith("'")) else f"'{new_val}'"
    return "(" + ", ".join(vals) + ")", True


def main() -> None:
    with open(USERS_JSON, "r", encoding="utf-8") as f:
        users_payload = json.load(f)
    id_map = build_id_map(users_payload)

    with open(SQL_PATH, "r", encoding="utf-8") as f:
        sql = f.read()

    ranges = find_insert_blocks(sql)
    if not ranges:
        raise RuntimeError("No INSERT INTO absence_journal ... VALUES blocks found in SQL file.")

    total_rows = 0
    replaced = 0
    missing_users: set[str] = set()

    out_parts: list[str] = []
    cursor = 0
    for (start, end, user_id_index) in ranges:
        out_parts.append(sql[cursor:start])
        block = sql[start:end]

        row_ranges = parse_rows(block)
        if not row_ranges:
            out_parts.append(block)
            cursor = end
            continue

        b_parts: list[str] = []
        b_cursor = 0
        for (rs, re_) in row_ranges:
            b_parts.append(block[b_cursor:rs])
            row_tuple = block[rs:re_]
            total_rows += 1

            new_tuple, did = replace_user_id_in_row(row_tuple, id_map, user_id_index)
            if did:
                replaced += 1
            else:
                # gather missing if looks like numeric id
                vals = split_values(row_tuple)
                if user_id_index < len(vals):
                    uid = vals[user_id_index].strip().strip("'")
                    if uid and uid.upper() != "NULL" and uid.isdigit() and uid not in id_map:
                        missing_users.add(uid)
            b_parts.append(new_tuple)
            b_cursor = re_
        b_parts.append(block[b_cursor:])
        out_parts.append("".join(b_parts))
        cursor = end

    out_parts.append(sql[cursor:])
    new_sql = "".join(out_parts)

    # backup + write
    if not os.path.exists(BAK_PATH):
        shutil.copyfile(SQL_PATH, BAK_PATH)
    with open(SQL_PATH, "w", encoding="utf-8", newline="\n") as f:
        f.write(new_sql)

    print(f"Mappings loaded: {len(id_map)}")
    print(f"INSERT blocks: {len(ranges)}")
    print(f"Rows scanned: {total_rows}")
    print(f"Rows replaced: {replaced}")
    if missing_users:
        sample = ", ".join(sorted(list(missing_users))[:30])
        print(f"Missing user_id in users.json (sample up to 30): {sample}")


if __name__ == "__main__":
    main()

