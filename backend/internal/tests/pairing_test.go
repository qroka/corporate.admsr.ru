package tests

import "testing"

func i64(v int64) *int64 { return &v }

func TestParseMappingAcceptsObjectAndJSONString(t *testing.T) {
	fromObject := ParseMapping(map[string]any{"1": "10", "2": float64(20), "x": "5", "3": "y"})
	if len(fromObject) != 2 || fromObject[1] != 10 || fromObject[2] != 20 {
		t.Fatalf("объект: %v", fromObject)
	}
	fromString := ParseMapping(`{"1":"10","2":"20"}`)
	if len(fromString) != 2 || fromString[2] != 20 {
		t.Fatalf("строка: %v", fromString)
	}
	if len(ParseMapping("")) != 0 || len(ParseMapping(nil)) != 0 || len(ParseMapping("не json")) != 0 {
		t.Fatal("пустой и мусорный ответ должен давать пустой словарь")
	}
}

func TestMappingJSONIsStable(t *testing.T) {
	got := MappingJSON(map[int64]int64{3: 30, 1: 10, 2: 20})
	if got != `{"1":"10","2":"20","3":"30"}` {
		t.Fatalf("получили %s", got)
	}
	if again := ParseMapping(got); again[2] != 20 || len(again) != 3 {
		t.Fatalf("круговой разбор: %v", again)
	}
}

func TestScorePairingCountsEachPair(t *testing.T) {
	items := []PairItem{
		{ID: 1, Text: "Токен", TargetID: i64(10)},
		{ID: 2, Text: "Контекст", TargetID: i64(20)},
		{ID: 3, Text: "Без ключа", TargetID: nil}, // опрос / ключ не задан — не оценивается
		{ID: 4, Text: "Промпт", TargetID: i64(10)}, // классификация: варианты повторяются
	}
	right, total := ScorePairing(items, map[int64]int64{1: 10, 2: 99, 4: 10, 3: 5})
	if total != 3 || right != 2 {
		t.Fatalf("ждали 2 из 3, получили %d из %d", right, total)
	}
	if right, total := ScorePairing(items, nil); right != 0 || total != 3 {
		t.Fatalf("без ответа: %d из %d", right, total)
	}
}

func TestPairingTexts(t *testing.T) {
	items := []PairItem{{ID: 1, Text: "Токен", TargetID: i64(10)}, {ID: 2, Text: "Контекст", TargetID: i64(20)}}
	targets := []PairTarget{{ID: 10, Text: "Фрагмент текста"}, {ID: 20, Text: "Сведения для ответа"}}
	user, correct := PairingTexts(items, targets, map[int64]int64{1: 20})
	if user != "Токен → Сведения для ответа\nКонтекст → —" {
		t.Fatalf("ответ: %q", user)
	}
	if correct != "Токен → Фрагмент текста\nКонтекст → Сведения для ответа" {
		t.Fatalf("ключ: %q", correct)
	}
}
