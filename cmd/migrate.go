package cmd

import (
	"fmt"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		fmt.Println("Running migrations...")
		err = db.AutoMigrate(
			&model.User{},
			&model.Identity{},
			&model.Contact{},
			&model.ContactGroup{},
			&model.Draft{},
			&model.UserSettings{},
			&model.Session{},
		)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		fmt.Println("Migrations completed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
