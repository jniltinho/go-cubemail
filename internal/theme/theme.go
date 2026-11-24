// Package theme holds the ordered registry of UI themes. It is the single
// list: the configuration check, the settings validation and the public
// GET /api/v1/themes endpoint all read it, and the frontend tests parse this
// file to compare it with the theme stylesheets.
package theme

// Theme describes one selectable UI theme.
type Theme struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Scheme string `json:"scheme"` // "light" | "dark"
}

// registry lists the themes in display order; the first entry is the default
// of last resort. Keep one entry per line in this exact literal form: a
// frontend test reads the ids and schemes from this file with a regex.
var registry = []Theme{
	{ID: "larry", Label: "Larry", Scheme: "light"},
	{ID: "outlook", Label: "Outlook", Scheme: "light"},
	{ID: "outlook-dark", Label: "Outlook (dark)", Scheme: "dark"},
	{ID: "apple-mail", Label: "Apple Mail", Scheme: "light"},
	{ID: "apple-mail-dark", Label: "Apple Mail (dark)", Scheme: "dark"},
	{ID: "breeze", Label: "Breeze", Scheme: "light"},
	{ID: "breeze-dark", Label: "Breeze (dark)", Scheme: "dark"},
	{ID: "graphite", Label: "Graphite (dark)", Scheme: "dark"},
	{ID: "high-contrast", Label: "High contrast (dark)", Scheme: "dark"},
}

// List returns a copy of the registry in display order.
func List() []Theme {
	return append([]Theme(nil), registry...)
}

// ByID returns the theme with the given id. For an unknown id it returns the
// first registry entry and false.
func ByID(id string) (Theme, bool) {
	for _, t := range registry {
		if t.ID == id {
			return t, true
		}
	}
	return registry[0], false
}

// Valid reports whether id is a registered theme id.
func Valid(id string) bool {
	_, ok := ByID(id)
	return ok
}
