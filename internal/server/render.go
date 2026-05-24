package server

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// TemplateRenderer stores pre-parsed templates for each route.
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// Render executes the pre-parsed template and injects request-level data.
func (t *TemplateRenderer) Render(c *echo.Context, w io.Writer, name string, data any) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Template not found: "+name)
	}

	currentPath := c.Request().URL.Path

	// Inject global view data into every map[string]interface{} payload.
	var viewData any = data
	if data == nil {
		viewData = map[string]any{"CurrentPath": currentPath}
	} else if m, ok := data.(map[string]any); ok {
		m["CurrentPath"] = currentPath
		viewData = m
	}

	// AJAX fragment templates do not use a layout wrapper.
	// Convention: "message.html" and any template prefixed with "partial_" are
	// executed directly by their own name, without a layout wrapper.
	baseName := name
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		baseName = name[idx+1:]
	}
	if baseName == "message.html" || strings.HasPrefix(baseName, "partial_") {
		// Fragments have no {{define}} block: execute the root template directly.
		return tmpl.Execute(w, viewData)
	}

	// Choose layout based on template name.
	layout := "layout/base.html"
	if strings.HasSuffix(name, "login.html") {
		layout = "layout/auth.html"
	}

	return tmpl.ExecuteTemplate(w, layout, viewData)
}

// loadTemplates parses all view templates from the embedded filesystem,
// wiring each page with the correct layout and any shared partials (partial_*.html).
func loadTemplates(embeddedFiles embed.FS) (*TemplateRenderer, error) {
	r := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}

	funcMap := templateFuncMap()

	layoutFile := "web/templates/layout/base.html"
	loginLayoutFile := "web/templates/layout/auth.html"

	// Collect partial templates (files whose name starts with "partial_").
	var partials []string
	fs.WalkDir(embeddedFiles, "web/templates", func(filePath string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "partial_") {
			partials = append(partials, filePath)
		}
		return nil
	})

	err := fs.WalkDir(embeddedFiles, "web/templates", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()

		// Skip layout files themselves and partials — they're included in page sets.
		switch {
		case filePath == layoutFile, filePath == loginLayoutFile:
			return nil
		case strings.HasPrefix(name, "partial_"):
			return nil
		// Skip non-HTML files
		case !strings.HasSuffix(name, ".html"):
			return nil
		}

		// Key: path relative to web/templates/ (e.g. "mailbox/index.html")
		tmplKey := strings.TrimPrefix(filePath, "web/templates/")

		// AJAX fragments are rendered without a layout — only the template file itself.
		isFragment := name == "message.html" || strings.HasPrefix(name, "partial_")

		var tmpl *template.Template
		var parseErr error

		if name == "login.html" {
			tmpl, parseErr = template.New(name).Funcs(funcMap).ParseFS(embeddedFiles,
				loginLayoutFile, filePath)
		} else if isFragment {
			// No layout wrapper — parse only the fragment file itself.
			tmpl, parseErr = template.New(name).Funcs(funcMap).ParseFS(embeddedFiles, filePath)
		} else {
			// Normal page: layout + partials + the page template itself.
			parseFiles := append([]string{layoutFile}, append(partials, filePath)...)
			tmpl, parseErr = template.New(name).Funcs(funcMap).ParseFS(embeddedFiles, parseFiles...)
		}

		if parseErr != nil {
			return fmt.Errorf("failed to parse template %s: %w", filePath, parseErr)
		}

		r.templates[tmplKey] = tmpl
		return nil
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}
