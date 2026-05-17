package cmd

import (
	"log/slog"
	"os"

	"go-cubemail/internal/config"
	"go-cubemail/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))

		cfg := config.Load()
		return server.Start(cfg, globalFS)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("port", "", "HTTP port (overrides config)")
	serveCmd.Flags().Bool("debug", false, "Echo debug mode")
	viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("server.debug", serveCmd.Flags().Lookup("debug"))
}
