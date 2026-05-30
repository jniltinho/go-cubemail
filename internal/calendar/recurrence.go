package calendar

import (
	"strings"
	"time"

	rrulego "github.com/teambition/rrule-go"
	"go-cubemail/internal/model"
)

const maxOccurrences = 500

// Occurrence represents a single expanded instance of a recurring event.
type Occurrence struct {
	OriginalID uint
	StartAt    time.Time
	EndAt      time.Time
}

// ExpandRecurring returns all occurrences of a recurring event that overlap [rangeStart, rangeEnd).
// The slice is empty when the RRULE is blank or unparseable.
// Caps at maxOccurrences to prevent runaway expansion.
func ExpandRecurring(event *model.Event, rangeStart, rangeEnd time.Time) []Occurrence {
	if event.RRule == "" {
		return nil
	}
	rruleStr := buildRRuleString(event)
	set, err := rrulego.StrToRRuleSet(rruleStr)
	if err != nil {
		return nil
	}

	duration := event.EndAt.Sub(event.StartAt)
	if duration < 0 {
		duration = 0
	}

	// Fetch occurrences up to cap; filter to the requested range.
	starts := set.Between(rangeStart.Add(-duration), rangeEnd, true)
	var out []Occurrence
	for _, start := range starts {
		end := start.Add(duration)
		if end.Before(rangeStart) || !start.Before(rangeEnd) {
			continue
		}
		out = append(out, Occurrence{
			OriginalID: event.ID,
			StartAt:    start,
			EndAt:      end,
		})
		if len(out) >= maxOccurrences {
			break
		}
	}
	return out
}

// buildRRuleString prepends DTSTART to the RRULE string so rrule-go can parse it.
func buildRRuleString(event *model.Event) string {
	dtstart := "DTSTART:" + event.StartAt.UTC().Format("20060102T150405Z")
	rule := strings.TrimSpace(event.RRule)
	if !strings.HasPrefix(strings.ToUpper(rule), "RRULE:") {
		rule = "RRULE:" + rule
	}
	return dtstart + "\n" + rule
}
