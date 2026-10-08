package handlers

import (
	"strings"
	"testing"
)

// V16 (Q-03): проверка тела записи личного календаря до обращения к БД.
func TestCalendarEntryInput(t *testing.T) {
	ok := map[string]any{"source": "meeting", "dateKey": "2026-10-02", "title": "  Планёрка  ", "timeStart": "09:30", "timeEnd": "10:00", "location": " 301 "}
	e, problem := calendarEntryInput(ok)
	if problem != "" || e.Source != "meeting" || e.Title != "Планёрка" || e.DateKey != "2026-10-02" || e.TimeStart != "09:30" || e.TimeEnd != "10:00" || e.Location != "301" {
		t.Fatalf("корректное тело отвергнуто или не обрезано: %+v %q", e, problem)
	}
	// V19: цвет — '#rrggbb', приводится к нижнему регистру.
	ok["color"] = "#3B82F6"
	if e, problem := calendarEntryInput(ok); problem != "" || e.Color != "#3b82f6" {
		t.Fatalf("цвет: %+v %q", e, problem)
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
		{"цвет не hex", func(m map[string]any) { m["color"] = "red" }},
		{"цвет короткий", func(m map[string]any) { m["color"] = "#fff" }},
		{"цвет со стилем", func(m map[string]any) { m["color"] = "#ffffff;background:url(x)" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{"source": "personal", "dateKey": "2026-10-02", "title": "x"}
			tc.mut(m)
			if _, problem := calendarEntryInput(m); problem == "" {
				t.Fatalf("ожидалась ошибка валидации")
			}
		})
	}

	// Необязательные поля не обязательны.
	if _, problem := calendarEntryInput(map[string]any{"source": "personal", "dateKey": "2026-10-02", "title": "x"}); problem != "" {
		t.Fatalf("тело без времени и места отвергнуто: %s", problem)
	}
}
