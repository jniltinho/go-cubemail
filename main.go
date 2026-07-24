// go-cubemail is a self-hosted webmail application backed by IMAP and SMTP.
// The Vue 3 SPA is embedded at build time from web/dist and served directly by the binary.
//
// @title           Go-Cubemail API
// @version         1.0
// @description     Self-hosted webmail application API backed by IMAP and SMTP.
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http https
//
// @securityDefinitions.apikey CookieAuth
// @in header
// @name Cookie
// @description The session cookie 'gorc_session' is required for authenticated endpoints.
package main

import (
	"embed"
	_ "go-cubemail/docs"

	// Embed the IANA time zone database (~450 KB). iCalendar events carry a
	// TZID and the binary is shipped standalone, so it cannot rely on
	// /usr/share/zoneinfo being present — it is absent in scratch and
	// distroless containers, and on stripped-down hosts.
	_ "time/tzdata"

	"go-cubemail/cmd"
)

//go:embed all:web/dist
//go:embed all:web/files
var embeddedFiles embed.FS

// main is the binary entry point; it delegates all CLI parsing to the cmd package.
func main() {
	cmd.Execute(embeddedFiles)
}
