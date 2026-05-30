// Package config defines the application configuration structures and the Load function
// that populates them from Viper (config.toml + GORC_* environment variables).
package config

import "github.com/spf13/viper"

// Config is the root configuration object passed throughout the application.
type Config struct {
	Server     ServerConfig
	IMAP       IMAPConfig
	SMTP       SMTPConfig
	Database   DatabaseConfig
	Session    SessionConfig
	UI         UIConfig
	Upload     UploadConfig
	ActiveSync ActiveSyncConfig
}

// ActiveSyncConfig controls the Exchange ActiveSync (EAS) HTTP endpoint.
type ActiveSyncConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	Debug              bool   `mapstructure:"debug"`
	MaxPingIntervalSec int    `mapstructure:"max_ping_interval_sec"`
	MaxSyncWindowSize  int    `mapstructure:"max_sync_window_size"`
	ProtocolVersion    string `mapstructure:"protocol_version"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host      string
	Port      int
	Debug     bool
	// SecretKey is a 32-byte key used for AES-GCM password encryption in sessions.
	SecretKey string `mapstructure:"secret_key"`
	BaseURL   string `mapstructure:"base_url"`
	TLSCert   string `mapstructure:"tls_cert"`
	TLSKey    string `mapstructure:"tls_key"`
}

// IMAPConfig holds IMAP connection settings.
type IMAPConfig struct {
	Host       string
	Port       int
	TLS        bool
	TimeoutSec int `mapstructure:"timeout_sec"`
	// ShowHostInput allows users to enter a custom IMAP host on the login form.
	ShowHostInput bool `mapstructure:"show_host_input"`
}

// SMTPConfig holds SMTP submission settings.
type SMTPConfig struct {
	Host       string
	Port       int
	StartTLS   bool `mapstructure:"starttls"`
	TimeoutSec int  `mapstructure:"timeout_sec"`
}

// DatabaseConfig selects the database driver and connection string.
// Supported drivers: "sqlite" (default), "mariadb", "mysql", "postgres", "postgresql".
//
// The Debug field controls GORM query logging:
//   - true  → logs all SQL queries (useful for development)
//   - false → only logs errors and slow queries (recommended for production)
type DatabaseConfig struct {
	Driver string
	DSN    string
	Debug  bool `mapstructure:"debug"`
}

// SessionConfig controls the HTTP session cookie behaviour.
type SessionConfig struct {
	Name     string
	MaxAge   int  `mapstructure:"max_age"`
	Secure   bool
	HTTPOnly bool `mapstructure:"http_only"`
}

// UIConfig contains default UI preferences applied to all users.
type UIConfig struct {
	Theme          string
	RowsPerPage    int    `mapstructure:"rows_per_page"`
	Timezone       string
	DateFormat     string `mapstructure:"date_format"`
	DatetimeFormat string `mapstructure:"datetime_format"`
	ComposeHTML    bool   `mapstructure:"compose_html"`
}

// UploadConfig configures attachment upload limits and temporary storage.
type UploadConfig struct {
	MaxSizeMB int    `mapstructure:"max_size_mb"`
	TempDir   string `mapstructure:"temp_dir"`
}

// Load reads all configuration keys from Viper and returns a populated Config.
// Call this after Viper has been initialised (e.g., inside a cobra RunE function).
func Load() *Config {
	cfg := &Config{}
	cfg.Server.Host = viper.GetString("server.host")
	cfg.Server.Port = viper.GetInt("server.port")
	cfg.Server.Debug = viper.GetBool("server.debug")
	cfg.Server.SecretKey = viper.GetString("server.secret_key")
	cfg.Server.BaseURL = viper.GetString("server.base_url")
	cfg.Server.TLSCert = viper.GetString("server.tls_cert")
	cfg.Server.TLSKey = viper.GetString("server.tls_key")

	cfg.IMAP.Host = viper.GetString("imap.host")
	cfg.IMAP.Port = viper.GetInt("imap.port")
	cfg.IMAP.TLS = viper.GetBool("imap.tls")
	cfg.IMAP.TimeoutSec = viper.GetInt("imap.timeout_sec")
	cfg.IMAP.ShowHostInput = viper.GetBool("imap.show_host_input")

	cfg.SMTP.Host = viper.GetString("smtp.host")
	cfg.SMTP.Port = viper.GetInt("smtp.port")
	cfg.SMTP.StartTLS = viper.GetBool("smtp.starttls")
	cfg.SMTP.TimeoutSec = viper.GetInt("smtp.timeout_sec")

	cfg.Database.Driver = viper.GetString("database.driver")
	cfg.Database.DSN = viper.GetString("database.dsn")
	cfg.Database.Debug = viper.GetBool("database.debug")

	cfg.Session.Name = viper.GetString("session.name")
	cfg.Session.MaxAge = viper.GetInt("session.max_age")
	cfg.Session.Secure = viper.GetBool("session.secure")
	cfg.Session.HTTPOnly = viper.GetBool("session.http_only")

	cfg.UI.Theme = viper.GetString("ui.theme")
	cfg.UI.RowsPerPage = viper.GetInt("ui.rows_per_page")
	cfg.UI.Timezone = viper.GetString("ui.timezone")
	cfg.UI.DateFormat = viper.GetString("ui.date_format")
	cfg.UI.DatetimeFormat = viper.GetString("ui.datetime_format")
	cfg.UI.ComposeHTML = viper.GetBool("ui.compose_html")

	cfg.Upload.MaxSizeMB = viper.GetInt("upload.max_size_mb")
	cfg.Upload.TempDir = viper.GetString("upload.temp_dir")

	cfg.ActiveSync.Enabled = viper.GetBool("activesync.enabled")
	cfg.ActiveSync.Debug = viper.GetBool("activesync.debug")
	cfg.ActiveSync.MaxPingIntervalSec = viper.GetInt("activesync.max_ping_interval_sec")
	cfg.ActiveSync.MaxSyncWindowSize = viper.GetInt("activesync.max_sync_window_size")
	cfg.ActiveSync.ProtocolVersion = viper.GetString("activesync.protocol_version")
	if cfg.ActiveSync.ProtocolVersion == "" {
		cfg.ActiveSync.ProtocolVersion = "16.1"
	}

	return cfg
}
