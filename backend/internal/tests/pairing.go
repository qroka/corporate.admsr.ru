package tests

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Вопросы «Соответствие» (match) и «Классификация» (classify): каждому элементу
// слева выбирается один вариант справа. Хранение — V13__test_question_pairing.sql,
// ответ — словарь {itemId: targetId}, в БД — JSON в test_answers.text_value.

// IsPairing — тип вопроса, у которого ответ — словарь «элемент → вариант».
func IsPairing(qtype string) bool { return qtype == "match" || qtype == "classify" }

type PairItem struct {
	ID       int64
	Text     string
	TargetID *int64 // правильный вариант справа; nil — не задан (опрос / черновик)
}

type PairTarget struct {
	ID   int64
	Text string
}

// LoadPairing читает элементы и варианты вопроса в порядке position.
func LoadPairing(ctx context.Context, pool *pgxpool.Pool, qid int64) ([]PairItem, []PairTarget, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, role, text, target_option_id FROM public.test_options
		WHERE question_id = $1 AND role IN ('item', 'target') ORDER BY position, id`, qid)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []PairItem
	var targets []PairTarget
	for rows.Next() {
		var id int64
		var role, text string
		var target *int64
		if err := rows.Scan(&id, &role, &text, &target); err != nil {
			return nil, nil, err
		}
		if role == "item" {
			items = append(items, PairItem{ID: id, Text: text, TargetID: target})
		} else {
			targets = append(targets, PairTarget{ID: id, Text: text})
		}
	}
	return items, targets, rows.Err()
}

// ParseMapping разбирает ответ клиента: объект {"itemId": "targetId"} либо
// его JSON-строка (так ответ лежит в БД). Мусорные пары пропускаются.
func ParseMapping(v any) map[int64]int64 {
	out := map[int64]int64{}
	var raw map[string]any
	switch x := v.(type) {
	case map[string]any:
		raw = x
	case string:
		if strings.TrimSpace(x) == "" {
			return out
		}
		if err := json.Unmarshal([]byte(x), &raw); err != nil {
			return out
		}
	default:
		return out
	}
	for k, val := range raw {
		item, err := strconv.ParseInt(strings.TrimSpace(k), 10, 64)
		if err != nil {
			continue
		}
		target, err := toInt64(val)
		if err != nil {
			continue
		}
		out[item] = target
	}
	return out
}

// MappingJSON — стабильная JSON-запись ответа (ключи по возрастанию).
func MappingJSON(m map[int64]int64) string {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(strconv.FormatInt(k, 10)))
		b.WriteByte(':')
		b.WriteString(strconv.Quote(strconv.FormatInt(m[k], 10)))
	}
	b.WriteByte('}')
	return b.String()
}

// ScorePairing — сколько соответствий верно из тех, у которых правильный ответ задан.
// Каждое соответствие весит как отдельный вопрос: за «7 из 7» и «5 из 7» баллы разные.
func ScorePairing(items []PairItem, m map[int64]int64) (correct, total int) {
	for _, it := range items {
		if it.TargetID == nil {
			continue
		}
		total++
		if t, ok := m[it.ID]; ok && t == *it.TargetID {
			correct++
		}
	}
	return correct, total
}

// PairingTexts — ответ сотрудника и правильный ответ построчно «элемент → вариант»
// для разбора попытки.
func PairingTexts(items []PairItem, targets []PairTarget, m map[int64]int64) (user, correct string) {
	label := map[int64]string{}
	for _, t := range targets {
		label[t.ID] = t.Text
	}
	var u, c []string
	for _, it := range items {
		ans := "—"
		if t, ok := m[it.ID]; ok {
			if l := label[t]; l != "" {
				ans = l
			}
		}
		u = append(u, it.Text+" → "+ans)
		if it.TargetID != nil {
			c = append(c, it.Text+" → "+label[*it.TargetID])
		}
	}
	return strings.Join(u, "\n"), strings.Join(c, "\n")
}
