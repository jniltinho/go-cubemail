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

		// The DAV tables must exist, and the DAV columns must be populated,
		// before AutoMigrate tries to build the unique indexes over them.
		if err := db.AutoMigrate(&model.AddressBook{}, &model.DAVChange{}); err != nil {
			return fmt.Errorf("dav table migration failed: %w", err)
		}
		if err := database.PrepareDAVSchema(db); err != nil {
			return fmt.Errorf("dav schema preparation failed: %w", err)
		}

		err = db.AutoMigrate(
			&model.User{},
			&model.Identity{},
			&model.Contact{},            // UID field added for CardDAV
			&model.ContactGroup{},
			&model.Draft{},
			&model.UserSettings{},    // OOFEnabled, OOFMessage, OOFMessageHTML
			&model.Session{},
			&model.Calendar{},
			&model.CalendarShare{},          // calendar sharing
			&model.CalendarSubscription{},   // remote .ics subscriptions
			&model.Event{},           // IsTask
			&model.EventAttendee{},
			&model.PushSubscription{},   // web push subscriptions
			&model.AddressBook{},        // CardDAV collections
			&model.DAVChange{},          // RFC 6578 changelog
			&state.EasDevice{},
			&state.EasFolderState{},
			&state.ImapFolderMapping{},
		)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		if err := database.FinishDAVMigration(db); err != nil {
			return fmt.Errorf("dav backfill failed: %w", err)
		}

		fmt.Println("Migrations completed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
