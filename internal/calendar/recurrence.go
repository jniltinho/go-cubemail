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

// ExpandRecurring returns all occurrences of a recurring event that overlap
// [rangeStart, rangeEnd). The slice is empty when the RRULE is blank or
// unparseable. EXDATE (cancelled instances) and RDATE (extra instances) from
// the master's ICalContent are honored, so a client that cancels or adds a
// single occurrence sees the correct set. Caps at maxOccurrences to prevent
// runaway expansion.
func ExpandRecurring(event *model.Event, rangeStart, rangeEnd time.Time) []Occurrence {
	if event.RRule == "" {
		return nil
	}
	set, err := rrulego.StrToRRuleSet(buildRRuleString(event))
	if err != nil {
		return nil
	}

	// Honor EXDATE / RDATE from the raw VEVENT (rrule-go excludes/includes them
	// during Between). Occurrences are generated in UTC (DTSTART is written as
	// ...Z), so exception dates are normalized to UTC to match exactly.
	for _, ex := range parseICalDates(event.ICalContent, "EXDATE") {
		set.ExDate(ex)
	}
	for _, rd := range parseICalDates(event.ICalContent, "RDATE") {
		set.RDate(rd)
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

// parseICalDates extracts the DATE-TIME / DATE values of an iCalendar property
// (EXDATE or RDATE) from raw VEVENT text, returning them as UTC instants. It
// reuses ical.go's CutProperty/parseICalTime (which already handle VALUE=DATE,
// trailing Z, and all-day date-only forms). Handles folded lines and
// comma-separated value lists; unparseable values are skipped.
func parseICalDates(ical, prop string) []time.Time {
	if ical == "" {
		return nil
	}
	prop = strings.ToUpper(prop)
	var out []time.Time
	for _, line := range unfoldICalLines(ical) {
		propRaw, valList, ok := CutProperty(line)
		if !ok {
			continue
		}
		name := propRaw
		if i := strings.IndexByte(name, ';'); i >= 0 {
			name = name[:i]
		}
		if strings.ToUpper(strings.TrimSpace(name)) != prop {
			continue
		}
		for _, v := range strings.Split(valList, ",") {
			// parseICalTime returns UTC; its bool is "is all-day", and a failed
			// parse yields the zero time, so guard on IsZero.
			if t, _ := parseICalTime(propRaw, strings.TrimSpace(v)); !t.IsZero() {
				out = append(out, t.UTC())
			}
		}
	}
	return out
}

// unfoldICalLines splits raw iCalendar text into logical lines, joining folded
// continuation lines (those beginning with a space or tab, per RFC 5545 §3.1).
func unfoldICalLines(ical string) []string {
	raw := strings.Split(strings.ReplaceAll(ical, "\r\n", "\n"), "\n")
	var lines []string
	for _, l := range raw {
		if (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += l[1:]
			continue
		}
		lines = append(lines, l)
	}
	return lines
}
