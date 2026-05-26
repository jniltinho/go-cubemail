package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	initOutput string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default configuration file in the current directory",
	Long: `Create a configuration file with default values in the current working directory.

By default, the generated file will have a Unix timestamp in the name
(e.g. config_1750945822.toml) to avoid accidentally overwriting existing files.

If a file with the same name already exists, the command will fail.

This is the recommended way to bootstrap a new installation.

Examples:
  go-cubemail init
  go-cubemail init -o config.toml
  go-cubemail init -o my-server-config.toml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine current directory: %w", err)
		}

		// Determine output filename
		outputName := initOutput
		if outputName == "" {
			// Default behavior: generate filename with Unix timestamp
			ts := time.Now().Unix()
			outputName = fmt.Sprintf("config_%d.toml", ts)
		}

		targetFile := filepath.Join(cwd, outputName)

		// Check if file already exists (no --force option anymore)
		if _, err := os.Stat(targetFile); err == nil {
			return fmt.Errorf("file already exists: %s\nUse -o to specify a different filename", targetFile)
		}

		// Load the default configuration template that was embedded at build time
		// (embedded in main.go and passed via cmd.Execute, same pattern as web/dist)
		defaultConfig, err := fs.ReadFile(globalFS, "web/files/config.default.toml")
		if err != nil {
			return fmt.Errorf("failed to load default configuration template: %w", err)
		}

		// Write the default configuration
		if err := os.WriteFile(targetFile, defaultConfig, 0644); err != nil {
			return fmt.Errorf("failed to write configuration file: %w", err)
		}

		fmt.Printf("✓ Created default configuration file: %s\n", targetFile)
		fmt.Println("  Please review and edit the file, especially the [server], [imap], and [smtp] sections.")
		fmt.Println("  Then run: go-cubemail serve")
		return nil
	},
}

func init() {
	initCmd.Flags().StringVarP(&initOutput, "output", "o", "", "Custom name for the generated configuration file (default: config_<unix-timestamp>.toml)")

	rootCmd.AddCommand(initCmd)
}
