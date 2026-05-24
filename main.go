// go-cubemail is a self-hosted webmail application backed by IMAP and SMTP.
// The Vue 3 SPA is embedded at build time from web/dist and served directly by the binary.
package main

import (
	"embed"

	"go-cubemail/cmd"
)

//go:embed all:web/dist
var embeddedFiles embed.FS

// main is the binary entry point; it delegates all CLI parsing to the cmd package.
func main() {
	cmd.Execute(embeddedFiles)
}
