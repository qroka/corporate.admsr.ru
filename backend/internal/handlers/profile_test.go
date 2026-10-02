package handlers

import "testing"

// ADR-041: ФИО, телефон и почта на портале не редактируются, ОФО и должность —
// один раз (пока не заданы). Сервер обязан молча игнорировать остальное.
func TestPlanWorkFields(t *testing.T) {
	str := func(p *string) string {
		if p == nil {
			return "<нет>"
		}
		return *p
	}
	cases := []struct {
		name             string
		curOFO, curRole  string
		body             map[string]any
		wantOFO, wantRol string
	}{
		{"первый вход: ОФО и должность принимаются", "-1", "", map[string]any{"ofo": "101", "role": "Специалист"}, "101", "Специалист"},
		{"ОФО пустое", "", "", map[string]any{"ofo": "5", "role": "Х"}, "5", "Х"},
		{"ОФО уже задано — не меняется", "100", "Начальник", map[string]any{"ofo": "101", "role": "Специалист"}, "<нет>", "<нет>"},
		{"ОФО задано, должность пуста — должность можно указать", "100", "", map[string]any{"ofo": "101", "role": "Специалист"}, "<нет>", "Специалист"},
		{"мусор вместо ОФО не записывается", "-1", "", map[string]any{"ofo": "-1", "role": ""}, "<нет>", ""},
		{"поля не присланы — ничего не трогаем", "-1", "", map[string]any{}, "<нет>", "<нет>"},
		{"ФИО, телефон и почта сюда не попадают", "100", "Х", map[string]any{"firstname": "Хакер", "phone": "1", "email": "a@b"}, "<нет>", "<нет>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ofo, role := planWorkFields(tc.curOFO, tc.curRole, tc.body)
			if str(ofo) != tc.wantOFO || str(role) != tc.wantRol {
				t.Fatalf("ofo=%s role=%s, ожидалось ofo=%s role=%s", str(ofo), str(role), tc.wantOFO, tc.wantRol)
			}
		})
	}
}
