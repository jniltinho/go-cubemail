package calendar

import (
	"testing"
	"time"

	"go-cubemail/internal/model"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return tm
}

func starts(occs []Occurrence) []time.Time {
	out := make([]time.Time, len(occs))
	for i, o := range occs {
		out[i] = o.StartAt.UTC()
	}
	return out
}

func TestExpandRecurring(t *testing.T) {
	base := &model.Event{
		ID:      1,
		StartAt: mustTime(t, "2026-01-05T10:00:00Z"), // a Monday
		EndAt:   mustTime(t, "2026-01-05T11:00:00Z"),
		RRule:   "FREQ=WEEKLY;COUNT=4", // Jan 5, 12, 19, 26
	}
	rangeStart := mustTime(t, "2026-01-01T00:00:00Z")
	rangeEnd := mustTime(t, "2026-02-01T00:00:00Z")

	t.Run("plain weekly expands to all occurrences", func(t *testing.T) {
		occs := ExpandRecurring(base, rangeStart, rangeEnd)
		if len(occs) != 4 {
			t.Fatalf("got %d occurrences, want 4: %v", len(occs), starts(occs))
		}
		if !occs[0].StartAt.Equal(base.StartAt) {
			t.Errorf("first occurrence = %v, want %v", occs[0].StartAt, base.StartAt)
		}
		if d := occs[0].EndAt.Sub(occs[0].StartAt); d != time.Hour {
			t.Errorf("duration = %v, want 1h", d)
		}
	})

	t.Run("EXDATE cancels a single occurrence", func(t *testing.T) {
		ev := *base
		ev.ICalContent = "BEGIN:VEVENT\r\n" +
			"EXDATE:20260112T100000Z\r\n" + // cancel the 2nd instance
			"END:VEVENT\r\n"
		occs := ExpandRecurring(&ev, rangeStart, rangeEnd)
		if len(occs) != 3 {
			t.Fatalf("got %d, want 3 after EXDATE: %v", len(occs), starts(occs))
		}
		for _, o := range occs {
			if o.StartAt.UTC().Equal(mustTime(t, "2026-01-12T10:00:00Z")) {
				t.Errorf("cancelled instance 2026-01-12 still present")
			}
		}
	})

	t.Run("multiple EXDATEs in one comma-separated line", func(t *testing.T) {
		ev := *base
		ev.ICalContent = "EXDATE:20260112T100000Z,20260119T100000Z\r\n"
		occs := ExpandRecurring(&ev, rangeStart, rangeEnd)
		if len(occs) != 2 {
			t.Fatalf("got %d, want 2 after two EXDATEs: %v", len(occs), starts(occs))
		}
	})

	t.Run("RDATE adds an extra occurrence", func(t *testing.T) {
		ev := *base
		ev.ICalContent = "RDATE:20260108T100000Z\r\n" // add a Thursday
		occs := ExpandRecurring(&ev, rangeStart, rangeEnd)
		if len(occs) != 5 {
			t.Fatalf("got %d, want 5 after RDATE: %v", len(occs), starts(occs))
		}
		found := false
		for _, o := range occs {
			if o.StartAt.UTC().Equal(mustTime(t, "2026-01-08T10:00:00Z")) {
				found = true
			}
		}
		if !found {
			t.Errorf("RDATE occurrence 2026-01-08 missing")
		}
	})

	t.Run("all-day EXDATE (VALUE=DATE)", func(t *testing.T) {
		ev := model.Event{
			ID:          2,
			StartAt:     mustTime(t, "2026-03-02T00:00:00Z"),
			EndAt:       mustTime(t, "2026-03-03T00:00:00Z"),
			IsAllDay:    true,
			RRule:       "FREQ=DAILY;COUNT=3", // Mar 2, 3, 4
			ICalContent: "EXDATE;VALUE=DATE:20260303\r\n",
		}
		occs := ExpandRecurring(&ev, mustTime(t, "2026-03-01T00:00:00Z"), mustTime(t, "2026-03-10T00:00:00Z"))
		if len(occs) != 2 {
			t.Fatalf("got %d, want 2 after all-day EXDATE: %v", len(occs), starts(occs))
		}
	})

	t.Run("blank rrule yields nothing", func(t *testing.T) {
		ev := *base
		ev.RRule = ""
		if occs := ExpandRecurring(&ev, rangeStart, rangeEnd); occs != nil {
			t.Errorf("blank RRULE should return nil, got %v", starts(occs))
		}
	})

	t.Run("folded EXDATE line is unfolded", func(t *testing.T) {
		ev := *base
		// RFC 5545 line folding: continuation starts with a space.
		ev.ICalContent = "EXDATE:20260112T100000Z,\r\n 20260119T100000Z\r\n"
		occs := ExpandRecurring(&ev, rangeStart, rangeEnd)
		if len(occs) != 2 {
			t.Fatalf("got %d, want 2 after folded EXDATE: %v", len(occs), starts(occs))
		}
	})
}
