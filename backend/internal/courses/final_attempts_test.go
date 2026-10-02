package courses

import "testing"

// Q-08: «Курс не сдан» возможен только при включённом лимите и исчерпанных попытках.
func TestFinalAttemptsExhausted(t *testing.T) {
	cases := []struct {
		name      string
		limit     bool
		attempts  int
		completed int
		want      bool
	}{
		{"без лимита попытки не кончаются", false, 1, 50, false},
		{"лимит 3, использовано 2", true, 3, 2, false},
		{"лимит 3, использовано 3", true, 3, 3, true},
		{"лимит 1, одна попытка", true, 1, 1, true},
		{"лимит 0 трактуется как 1", true, 0, 1, true},
		{"нет попыток — не исчерпано", true, 2, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := finalAttemptsExhausted(tc.limit, tc.attempts, tc.completed); got != tc.want {
				t.Fatalf("finalAttemptsExhausted(%v,%d,%d) = %v, want %v", tc.limit, tc.attempts, tc.completed, got, tc.want)
			}
		})
	}
}
