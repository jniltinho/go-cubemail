package cmd

// migrateCmd runs GORM AutoMigrate for all database models.
// It is safe to run repeatedly; GORM only adds missing columns or tables.

import (
	"fmt"

	"go-cubemail/internal/config"
	"go-cubemail/internal/database"
	"go-cubemail/internal/model"
	"go-cubemail/internal/activesync/state"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		db, err := database.Open(cfg)
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
			&model.UserSettings{},    // OOFEnabled, OOFMessage, OOFMessageHTML
			&model.Session{},
			&model.Calendar{},
			&model.CalendarShare{},   // calendar sharing
			&model.Event{},           // IsTask
			&model.EventAttendee{},
			&state.EasDevice{},
			&state.EasFolderState{},
			&state.ImapFolderMapping{},
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
