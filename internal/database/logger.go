package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// gormSlogLogger is a GORM logger implementation that writes to the
// application's structured slog logger.
type gormSlogLogger struct {
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
}

// NewGormSlogLogger creates a new GORM logger backed by slog.
func NewGormSlogLogger(level logger.LogLevel, slowThreshold time.Duration) logger.Interface {
	return &gormSlogLogger{
		LogLevel:      level,
		SlowThreshold: slowThreshold,
	}
}

// LogMode sets the log level for the logger (GORM calls this to change verbosity).
func (l *gormSlogLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *gormSlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		slog.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *gormSlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		slog.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *gormSlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		slog.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL queries. This is the main method used by GORM.
func (l *gormSlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Base attributes for every query log
	attrs := []any{
		slog.Duration("duration", elapsed),
		slog.String("sql", sql),
		slog.Int64("rows", rows),
	}

	// Log level decision
	switch {
	case err != nil && l.LogLevel >= logger.Error:
		attrs = append(attrs, slog.String("error", err.Error()))
		slog.ErrorContext(ctx, "database query error", attrs...)

	case elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		attrs = append(attrs, slog.Bool("slow", true))
		slog.WarnContext(ctx, "slow database query", attrs...)

	case l.LogLevel >= logger.Info:
		slog.InfoContext(ctx, "database query", attrs...)
	}
}
