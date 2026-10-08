package handlers

import (
	"strings"
	"testing"
)

// V20: поля черновика девблога проверяются до обращения к БД.
func TestDevblogInput(t *testing.T) {
	f, problem := devblogInput(map[string]any{
		"title": "  Обновление 1.2  ", "body": "# Что нового\r\n- пункт", "imagePath": " /img/FullPic/a.jpg ",
		"releaseVersion": " 1.0.1 ",
	})
	if problem != "" || f.Title != "Обновление 1.2" || f.Body != "# Что нового\n- пункт" || f.Image != "/img/FullPic/a.jpg" || f.Release != "1.0.1" {
		t.Fatalf("корректное тело: %+v %q", f, problem)
	}
	bad := []map[string]any{
		{"title": strings.Repeat("я", devblogTitleMax+1)},
		{"body": strings.Repeat("я", devblogBodyMax+1)},
		{"imagePath": "https://evil.example/x.png"},
		{"imagePath": "javascript:alert(1)"},
		{"releaseVersion": "v1.0"},
		{"releaseVersion": "1.0"},
		{"releaseVersion": "1.0.0<script>"},
	}
	for _, b := range bad {
		if _, problem := devblogInput(b); problem == "" {
			t.Fatalf("принято некорректное тело: %v", b)
		}
	}
}

// V21: предложенная версия — последняя + 0.0.1, первый выпуск — 1.0.0.
func TestNextDevblogVersion(t *testing.T) {
	for in, want := range map[string]string{"": "1.0.0", "1.0.0": "1.0.1", "1.2.9": "1.2.10", " 2.0.3 ": "2.0.4", "мусор": "1.0.0"} {
		if got := nextDevblogVersion(in); got != want {
			t.Fatalf("nextDevblogVersion(%q) = %q, ждали %q", in, got, want)
		}
	}
}
