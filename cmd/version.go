package cmd

// versionCmd prints the binary version, git commit, and build date to stdout.

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("go-cubemail %s (commit: %s, built: %s)\n", Version, GitCommit, BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
