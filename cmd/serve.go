package cmd

// serveCmd starts the HTTP server. It opens the configured database, initialises
// the handler and session layers, and hands off to server.Start.
// Port and debug-mode flags can override the config file values.

import (
	"log/slog"
	"os"

	"go-cubemail/internal/config"
	"go-cubemail/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))

		cfg := config.Load()

		var db *gorm.DB
		var err error

		switch cfg.Database.Driver {
		case "mariadb":
			db, err = gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{})
		default:
			db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
		}
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}

		server.AppVersion = Version
		return server.Start(cfg, db, globalFS)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("port", "", "HTTP port (overrides config)")
	serveCmd.Flags().Bool("debug", false, "Echo debug mode")
	viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("server.debug", serveCmd.Flags().Lookup("debug"))
}
