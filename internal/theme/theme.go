// Package theme holds the ordered registry of UI themes. It is the single
// list the server uses: the configuration check, the settings validation and
// the public GET /api/v1/themes endpoint all read it. The list itself is in
// registry_gen.go, generated from the theme stylesheets of the frontend
// (frontend/scripts/themes.mjs), so a theme is declared in one place.
package theme

// Theme describes one selectable UI theme.
type Theme struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Scheme string `json:"scheme"` // "light" | "dark"
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
