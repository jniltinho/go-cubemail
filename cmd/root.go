// Package cmd defines the Cobra CLI commands for go-cubemail (init, serve, migrate, version).
// Configuration is loaded from config.toml via Viper and can be overridden with GORC_* env vars.
package cmd

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	// Version is injected at build time via -ldflags "-X go-cubemail/cmd.Version=x.y.z".
	Version   = "dev"
	// BuildDate is injected at build time.
	BuildDate = "unknown"
	// GitCommit is injected at build time.
	GitCommit = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "go-cubemail",
	Short: "Go CubeMail — webmail client via IMAP/SMTP",
	Long: `Go CubeMail is a modern, self-hosted webmail client written in Go with a Vue 3 frontend.

Available commands:
  init      Create a default configuration file
  serve     Start the web server
  migrate   Run database migrations
  version   Show version information`,
}

var globalFS embed.FS

// Execute sets the embedded filesystem and runs the root Cobra command.
// It is called from main() with the compiled-in web/dist filesystem.
func Execute(fs embed.FS) {
	globalFS = fs
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config.toml (default: ./config.toml)")
}

// initConfig loads configuration from disk and environment variables via Viper.
// Search order: --config flag → ./config.toml → /etc/go-cubemail/config.toml.
// Environment variables prefixed with GORC_ override any file values.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/go-cubemail/")
	}

	viper.SetEnvPrefix("GORC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		}
	}
}
