package handlers

import (
	"strings"
	"testing"
)

// V18: текст комментария — простой, обрезается, не пустой и не длиннее лимита.
func TestNormalizeCommentContent(t *testing.T) {
	if s, msg := normalizeCommentContent("  привет\r\nмир  "); msg != "" || s != "привет\nмир" {
		t.Fatalf("корректный текст: %q %q", s, msg)
	}
	if _, msg := normalizeCommentContent("   "); msg == "" {
		t.Fatal("пустой комментарий принят")
	}
	if _, msg := normalizeCommentContent(strings.Repeat("я", newsCommentMaxRunes)); msg != "" {
		t.Fatalf("комментарий ровно по лимиту отвергнут: %s", msg)
	}
	if _, msg := normalizeCommentContent(strings.Repeat("я", newsCommentMaxRunes+1)); msg == "" {
		t.Fatal("длинный комментарий принят")
	}
}
