package handlers

import (
	"strings"
	"testing"
)

// V16 (Q-03): проверка тела записи личного календаря до обращения к БД.
func TestCalendarEntryInput(t *testing.T) {
	ok := map[string]any{"source": "meeting", "dateKey": "2026-10-02", "title": "  Планёрка  ", "timeStart": "09:30", "timeEnd": "10:00", "location": " 301 "}
	source, title, dateKey, ts, te, loc, problem := calendarEntryInput(ok)
	if problem != "" || source != "meeting" || title != "Планёрка" || dateKey != "2026-10-02" || ts != "09:30" || te != "10:00" || loc != "301" {
		t.Fatalf("корректное тело отвергнуто или не обрезано: %q %q %q %q %q %q %q", source, title, dateKey, ts, te, loc, problem)
	}

	cases := []struct {
		name string
		mut  func(m map[string]any)
	}{
		{"неизвестный тип", func(m map[string]any) { m["source"] = "birthday" }},
		{"пустое название", func(m map[string]any) { m["title"] = "   " }},
		{"длинное название", func(m map[string]any) { m["title"] = strings.Repeat("я", calendarTitleMaxRunes+1) }},
		{"плохая дата", func(m map[string]any) { m["dateKey"] = "2026-13-40" }},
		{"дата не в формате", func(m map[string]any) { m["dateKey"] = "02.10.2026" }},
		{"плохое время", func(m map[string]any) { m["timeStart"] = "25:00" }},
		{"плохое время конца", func(m map[string]any) { m["timeEnd"] = "9-30" }},
		{"длинное место", func(m map[string]any) { m["location"] = strings.Repeat("м", calendarLocationMaxRunes+1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{"source": "personal", "dateKey": "2026-10-02", "title": "x"}
			tc.mut(m)
			if _, _, _, _, _, _, problem := calendarEntryInput(m); problem == "" {
				t.Fatalf("ожидалась ошибка валидации")
			}
		})
	}

	// Необязательные поля не обязательны.
	if _, _, _, _, _, _, problem := calendarEntryInput(map[string]any{"source": "personal", "dateKey": "2026-10-02", "title": "x"}); problem != "" {
		t.Fatalf("тело без времени и места отвергнуто: %s", problem)
	}
}
