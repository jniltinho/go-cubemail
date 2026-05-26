package database

import (
	"fmt"
	"time"

	"go-cubemail/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open establishes a connection to the database according to the
// configured driver in config.toml (or GORC_* environment variables).
//
// Supported drivers:
//   - "sqlite"      (default)
//   - "mariadb"
//   - "mysql"
//   - "postgres"
//   - "postgresql"
//
// The database.debug option in config controls query logging:
//   - true  → logs all SQL statements (recommended only in development)
//   - false → logs only errors + slow queries (recommended for production)
//
// Returns a *gorm.DB instance ready for use, or an error if the
// connection or driver is invalid.
func Open(cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Determine GORM log level based on database.debug
	logLevel := logger.Error
	if cfg.Database.Debug {
		logLevel = logger.Info
	}

	// Use our custom slog-backed GORM logger for consistency with the app
	gormLogger := NewGormSlogLogger(logLevel, time.Second)

	gormConfig := &gorm.Config{
		Logger: gormLogger,
	}

	switch cfg.Database.Driver {
	case "mariadb", "mysql":
		db, err = gorm.Open(mysql.Open(cfg.Database.DSN), gormConfig)
	case "postgres", "postgresql":
		db, err = gorm.Open(postgres.Open(cfg.Database.DSN), gormConfig)
	default:
		// Default to SQLite for development and unknown drivers
		db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), gormConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database (driver=%s): %w", cfg.Database.Driver, err)
	}

	return db, nil
}
