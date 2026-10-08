package handlers

import (
	"testing"
	"time"
)

// V17: окно поздравления — день рождения и три дня после, с переходом через Новый год.
func TestBirthdayGreetingYear(t *testing.T) {
	loc := time.FixedZone("UTC+5", 5*3600)
	day := func(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, loc) }

	cases := []struct {
		name       string
		month, day int
		now        time.Time
		wantYear   int
		wantOK     bool
	}{
		{"в сам день утром", 10, 15, day(2026, 10, 15, 0), 2026, true},
		{"в сам день вечером", 10, 15, day(2026, 10, 15, 23), 2026, true},
		{"третий день после", 10, 15, day(2026, 10, 18, 12), 2026, true},
		{"четвёртый день после", 10, 15, day(2026, 10, 19, 0), 0, false},
		{"накануне", 10, 15, day(2026, 10, 14, 23), 0, false},
		{"через Новый год", 12, 31, day(2027, 1, 2, 9), 2026, true},
		{"через Новый год, поздно", 12, 30, day(2027, 1, 3, 9), 0, false},
		{"29 февраля в невисокосный год — 1 марта", 2, 29, day(2027, 3, 1, 9), 2027, true},
		{"плохой месяц", 13, 1, day(2026, 1, 1, 9), 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			y, ok := birthdayGreetingYear(tc.month, tc.day, tc.now)
			if ok != tc.wantOK || y != tc.wantYear {
				t.Fatalf("birthdayGreetingYear(%d, %d, %s) = %d, %v; ожидалось %d, %v",
					tc.month, tc.day, tc.now.Format(time.RFC3339), y, ok, tc.wantYear, tc.wantOK)
			}
		})
	}
}
