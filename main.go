package main

import (
	"embed"

	"go-cubemail/cmd"
)

//go:embed all:web/dist
var embeddedFiles embed.FS

func main() {
	cmd.Execute(embeddedFiles)
}
