package main

import (
	"embed"

	"go-cubemail/cmd"
)

//go:embed web
var embeddedFiles embed.FS

func main() {
	cmd.Execute(embeddedFiles)
}
