package courses

import (
	"testing"
	"time"
)

func TestEnrollmentDeadline(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	days := 14
	zero := 0
	fixed := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		a    Assignment
		want *time.Time
	}{
		{"без срока", Assignment{}, nil},
		{"общий срок — как у всех, даже прошедший", Assignment{DeadlineAt: &fixed}, &fixed},
		{"N дней — от момента выдачи курса", Assignment{DeadlineDays: &days, DeadlineAt: &fixed}, ptr(now.Add(14 * 24 * time.Hour))},
		{"N дней — от даты начала, если она позже", Assignment{DeadlineDays: &days, StartsAt: &later}, ptr(later.Add(14 * 24 * time.Hour))},
		{"0 дней — без относительного срока", Assignment{DeadlineDays: &zero}, nil},
	}
	for _, c := range cases {
		got := EnrollmentDeadline(now, c.a)
		switch {
		case c.want == nil && got != nil:
			t.Errorf("%s: ждали nil, получили %v", c.name, *got)
		case c.want != nil && (got == nil || !got.Equal(*c.want)):
			t.Errorf("%s: ждали %v, получили %v", c.name, *c.want, got)
		}
	}
}

func ptr(t time.Time) *time.Time { return &t }
