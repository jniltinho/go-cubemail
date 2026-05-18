package server

import (
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strings"
)

// templateFuncMap returns the custom template functions used across all templates.
func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		// Arithmetic
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int { return a % b },
		"float64": func(i any) float64 {
			switch v := i.(type) {
			case int:
				return float64(v)
			case int64:
				return float64(v)
			case float64:
				return v
			default:
				return 0
			}
		},

		// Formatting
		"formatSize": func(size int64) string {
			switch {
			case size >= 1073741824:
				return fmt.Sprintf("%.1f GB", float64(size)/1073741824)
			case size >= 1048576:
				return fmt.Sprintf("%.1f MB", float64(size)/1048576)
			case size >= 1024:
				return fmt.Sprintf("%.1f KB", float64(size)/1024)
			default:
				return fmt.Sprintf("%d B", size)
			}
		},
		"calcPercentage": func(count, total any) string {
			var c, t float64
			switch v := count.(type) {
			case int:
				c = float64(v)
			case int64:
				c = float64(v)
			case float64:
				c = v
			}
			switch v := total.(type) {
			case int:
				t = float64(v)
			case int64:
				t = float64(v)
			case float64:
				t = v
			}
			if t == 0 {
				return "0"
			}
			return fmt.Sprintf("%.0f", math.Min((c/t)*100.0, 100))
		},

		// String utilities
		"upper":   strings.ToUpper,
		"lower":   strings.ToLower,
		"title":   strings.Title, //nolint:staticcheck
		"trim":    strings.TrimSpace,
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"split": strings.Split,
		"join":  func(sep string, parts []string) string { return strings.Join(parts, sep) },

		"commaToLines": func(s string) template.HTML {
			parts := strings.Split(s, ",")
			var trimmed []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					trimmed = append(trimmed, template.HTMLEscapeString(p))
				}
			}
			return template.HTML(strings.Join(trimmed, "<br>"))
		},
		"commaToNewlines": func(s string) string {
			parts := strings.Split(s, ",")
			var trimmed []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					trimmed = append(trimmed, p)
				}
			}
			return strings.Join(trimmed, "\n")
		},

		// HTML safety
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeAttr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},

		// Sequence
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i + 1
			}
			return s
		},
		"last": func(i, total int) bool {
			return i == total-1
		},

		// Boolean helpers
		"urlEncode": url.PathEscape,

		"not": func(b bool) bool { return !b },
		"and": func(a, b bool) bool { return a && b },
		"or":  func(a, b bool) bool { return a || b },

		// App version (can be overridden via ldflags)
		"appVersion": func() string { return AppVersion },
	}
}
