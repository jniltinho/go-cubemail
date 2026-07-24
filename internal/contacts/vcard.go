// Package contacts provides vCard serialisation for the address book.
//
// The guiding rule is the one that makes CardDAV interoperate: the blob a
// client sends is stored verbatim and handed back verbatim. The flat
// model.Contact fields are only an index for the web UI and ActiveSync, so
// anything the flat model cannot express — ADR, PHOTO, BDAY, additional
// EMAIL/TEL entries, X-* extensions — must survive untouched. ApplyToVCard
// therefore patches the properties the UI can edit and leaves the rest alone,
// rather than regenerating the card from scratch.
package contacts

import (
	"errors"
	"strings"

	"go-cubemail/internal/model"
)

// ErrEmptyCard is returned when a payload carries no usable contact data.
var ErrEmptyCard = errors.New("contacts: vCard has no name or e-mail")

// property is one unfolded vCard line split into its parts.
type property struct {
	Group string // optional "item1" prefix
	Name  string // upper-cased property name
	Raw   string // full left-hand side, including parameters
	Value string // raw (still escaped) value
}

// Unfold joins RFC 6350 continuation lines: any line starting with a space or
// tab belongs to the previous one. Skipping this step is why folded cards from
// Apple Contacts — very common in NOTE and ADR — parse as garbage.
func Unfold(raw string) []string {
	normalised := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")
	var out []string
	for _, line := range strings.Split(normalised, "\n") {
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(out) > 0 {
			out[len(out)-1] += line[1:]
			continue
		}
		out = append(out, line)
	}
	return out
}

// parseLine splits an unfolded vCard line into group, name, parameters and value.
func parseLine(line string) (property, bool) {
	left, value, ok := cutValue(line)
	if !ok {
		return property{}, false
	}
	name := left
	if i := strings.IndexByte(name, ';'); i >= 0 {
		name = name[:i]
	}
	var group string
	if i := strings.IndexByte(name, '.'); i >= 0 {
		group, name = name[:i], name[i+1:]
	}
	return property{
		Group: group,
		Name:  strings.ToUpper(strings.TrimSpace(name)),
		Raw:   left,
		Value: value,
	}, true
}

// cutValue splits a content line at the first colon that is not inside a
// quoted parameter value (e.g. TEL;TYPE="work:home":+55...).
func cutValue(line string) (left, value string, ok bool) {
	inQuotes := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuotes = !inQuotes
		case ':':
			if !inQuotes {
				return line[:i], line[i+1:], true
			}
		}
	}
	return "", "", false
}

// hasParam reports whether the left-hand side declares the given parameter value,
// e.g. hasParam(`EMAIL;TYPE=INTERNET,PREF`, "TYPE", "PREF").
func hasParam(raw, key, want string) bool {
	parts := strings.Split(raw, ";")
	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(p, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		for _, item := range strings.Split(v, ",") {
			if strings.EqualFold(strings.Trim(strings.TrimSpace(item), `"`), want) {
				return true
			}
		}
	}
	return false
}

// Parse extracts the indexed fields from a vCard. The caller keeps the original
// blob; this only fills the columns used for listing and searching.
func Parse(raw string) (*model.Contact, error) {
	c := &model.Contact{}
	var formattedName string
	var emailSet, phoneSet, prefEmail, prefPhone bool

	for _, line := range Unfold(raw) {
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "BEGIN:") || strings.HasPrefix(upper, "END:") {
			continue
		}
		p, ok := parseLine(line)
		if !ok {
			continue
		}
		val := Unescape(strings.TrimSpace(p.Value))
		if val == "" {
			continue
		}
		switch p.Name {
		case "FN":
			formattedName = val
		case "N":
			// N:Family;Given;Additional;Prefix;Suffix
			parts := strings.Split(val, ";")
			if len(parts) > 0 {
				c.LastName = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				c.FirstName = strings.TrimSpace(parts[1])
			}
		case "EMAIL":
			pref := hasParam(p.Raw, "TYPE", "PREF") || hasParam(p.Raw, "PREF", "1")
			if !emailSet || (pref && !prefEmail) {
				c.Email = val
				emailSet = true
				prefEmail = prefEmail || pref
			}
		case "TEL":
			pref := hasParam(p.Raw, "TYPE", "PREF") || hasParam(p.Raw, "PREF", "1")
			if !phoneSet || (pref && !prefPhone) {
				c.Phone = val
				phoneSet = true
				prefPhone = prefPhone || pref
			}
		case "ORG":
			c.Company = strings.TrimSpace(strings.SplitN(val, ";", 2)[0])
		case "TITLE":
			c.Title = val
		case "NOTE":
			c.Notes = val
		case "UID":
			c.UID = strings.TrimPrefix(val, "urn:uuid:")
		}
	}

	// A card may legally carry FN without N — split the display name so the UI
	// still shows something sensible.
	if c.FirstName == "" && c.LastName == "" && formattedName != "" {
		first, last := SplitName(formattedName)
		c.FirstName, c.LastName = first, last
	}
	if c.FirstName == "" && c.LastName == "" && c.Email == "" {
		return nil, ErrEmptyCard
	}
	return c, nil
}

// SplitName divides a display name into first and last parts at the last space.
func SplitName(full string) (first, last string) {
	full = strings.TrimSpace(full)
	if i := strings.LastIndex(full, " "); i > 0 {
		return strings.TrimSpace(full[:i]), strings.TrimSpace(full[i+1:])
	}
	return full, ""
}

// DisplayName renders the contact's FN value.
func DisplayName(c model.Contact) string {
	return strings.TrimSpace(strings.TrimSpace(c.FirstName) + " " + strings.TrimSpace(c.LastName))
}

// UID returns a stable vCard UID for a contact, deriving one from the row ID
// when the record predates CardDAV support.
func UID(c model.Contact) string {
	if c.UID != "" {
		return c.UID
	}
	return "contact-" + itoa(c.ID) + "@go-cubemail"
}

// Build renders a contact as a fresh vCard 3.0 document. It is only used for
// records that have no stored blob yet; existing blobs go through ApplyToVCard.
func Build(c model.Contact) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\r\n")
	b.WriteString("VERSION:3.0\r\n")
	writeProp(&b, "UID", UID(c))
	writeProp(&b, "FN", Escape(DisplayName(c)))
	writeProp(&b, "N", Escape(c.LastName)+";"+Escape(c.FirstName)+";;;")
	if c.Email != "" {
		writeProp(&b, "EMAIL;TYPE=INTERNET", Escape(c.Email))
	}
	if c.Phone != "" {
		writeProp(&b, "TEL;TYPE=CELL", Escape(c.Phone))
	}
	if c.Company != "" {
		writeProp(&b, "ORG", Escape(c.Company))
	}
	if c.Title != "" {
		writeProp(&b, "TITLE", Escape(c.Title))
	}
	if c.Notes != "" {
		writeProp(&b, "NOTE", Escape(c.Notes))
	}
	if !c.UpdatedAt.IsZero() {
		writeProp(&b, "REV", c.UpdatedAt.UTC().Format("20060102T150405Z"))
	}
	b.WriteString("END:VCARD\r\n")
	return b.String()
}

// editableProps are the properties the web UI owns. Everything else in a stored
// card is preserved untouched by ApplyToVCard.
var editableProps = map[string]bool{
	"FN": true, "N": true, "EMAIL": true, "TEL": true,
	"ORG": true, "TITLE": true, "NOTE": true, "REV": true,
}

// ApplyToVCard rewrites the UI-editable properties of an existing vCard while
// keeping every other line — including grouped properties, ADR, PHOTO, BDAY and
// X-* extensions — exactly as the client sent them.
//
// Only the first occurrence of a repeated editable property (EMAIL, TEL) is
// updated; the remaining ones are preserved, so a contact with three phone
// numbers does not lose two of them because the UI only shows one.
func ApplyToVCard(raw string, c model.Contact) string {
	if strings.TrimSpace(raw) == "" {
		return Build(c)
	}
	lines := Unfold(raw)

	// A blank replacement means the field was cleared in the UI and the whole
	// property should go away, so emptiness is tracked per property rather than
	// inferred from the rendered value (a nameless N renders as ";;;;", not "").
	replacements := map[string]string{
		"FN":    Escape(DisplayName(c)),
		"N":     Escape(c.LastName) + ";" + Escape(c.FirstName) + ";;;",
		"EMAIL": Escape(c.Email),
		"TEL":   Escape(c.Phone),
		"ORG":   Escape(c.Company),
		"TITLE": Escape(c.Title),
		"NOTE":  Escape(c.Notes),
	}
	blank := map[string]bool{
		"FN":    DisplayName(c) == "",
		"N":     c.FirstName == "" && c.LastName == "",
		"EMAIL": c.Email == "",
		"TEL":   c.Phone == "",
		"ORG":   c.Company == "",
		"TITLE": c.Title == "",
		"NOTE":  c.Notes == "",
	}
	if !c.UpdatedAt.IsZero() {
		replacements["REV"] = c.UpdatedAt.UTC().Format("20060102T150405Z")
	} else {
		blank["REV"] = true
	}

	applied := make(map[string]bool, len(replacements))
	var b strings.Builder
	var endLine string

	for _, line := range lines {
		p, ok := parseLine(line)
		if !ok {
			b.WriteString(FoldLine(line))
			continue
		}
		if p.Name == "END" {
			endLine = line
			continue
		}
		if !editableProps[p.Name] || applied[p.Name] {
			b.WriteString(FoldLine(line))
			continue
		}
		applied[p.Name] = true
		if blank[p.Name] {
			// The field was cleared in the UI: drop the property entirely.
			continue
		}
		b.WriteString(FoldLine(p.Raw + ":" + replacements[p.Name]))
	}

	// Append properties the original card did not carry.
	for _, name := range []string{"FN", "N", "EMAIL", "TEL", "ORG", "TITLE", "NOTE", "REV"} {
		if applied[name] || blank[name] {
			continue
		}
		val := replacements[name]
		left := name
		switch name {
		case "EMAIL":
			left = "EMAIL;TYPE=INTERNET"
		case "TEL":
			left = "TEL;TYPE=CELL"
		}
		b.WriteString(FoldLine(left + ":" + val))
	}

	if endLine == "" {
		endLine = "END:VCARD"
	}
	b.WriteString(FoldLine(endLine))
	return b.String()
}

// writeProp appends one folded content line.
func writeProp(b *strings.Builder, name, value string) {
	b.WriteString(FoldLine(name + ":" + value))
}

// FoldLine appends CRLF and folds the content line at 75 octets as RFC 6350
// requires, continuing with a single leading space.
func FoldLine(line string) string {
	const limit = 75
	if len(line) <= limit {
		return line + "\r\n"
	}
	var b strings.Builder
	// Fold on octet boundaries but never in the middle of a UTF-8 sequence.
	start := 0
	width := limit
	for start < len(line) {
		end := start + width
		if end > len(line) {
			end = len(line)
		} else {
			for end > start && !utf8Boundary(line[end]) {
				end--
			}
		}
		if start > 0 {
			b.WriteString(" ")
		}
		b.WriteString(line[start:end])
		b.WriteString("\r\n")
		start = end
		width = limit - 1 // subsequent lines carry the leading space
	}
	return b.String()
}

// utf8Boundary reports whether b starts a UTF-8 code point.
func utf8Boundary(b byte) bool { return b&0xC0 != 0x80 }

// Escape encodes the characters that are special in a vCard value.
func Escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// Unescape reverses Escape.
func Unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'N':
			b.WriteByte('\n')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// itoa avoids pulling strconv in for a single conversion.
func itoa(v uint) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
