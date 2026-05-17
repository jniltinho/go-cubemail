package config

import "github.com/spf13/viper"

type Config struct {
	Server   ServerConfig
	IMAP     IMAPConfig
	SMTP     SMTPConfig
	Database DatabaseConfig
	Session  SessionConfig
	UI       UIConfig
	Upload   UploadConfig
}

type ServerConfig struct {
	Host      string
	Port      int
	Debug     bool
	SecretKey string `mapstructure:"secret_key"`
	BaseURL   string `mapstructure:"base_url"`
	TLSCert   string `mapstructure:"tls_cert"`
	TLSKey    string `mapstructure:"tls_key"`
}

type IMAPConfig struct {
	Host          string
	Port          int
	TLS           bool
	TimeoutSec    int  `mapstructure:"timeout_sec"`
	ShowHostInput bool `mapstructure:"show_host_input"`
}

type SMTPConfig struct {
	Host       string
	Port       int
	StartTLS   bool `mapstructure:"starttls"`
	TimeoutSec int  `mapstructure:"timeout_sec"`
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type SessionConfig struct {
	Name     string
	MaxAge   int  `mapstructure:"max_age"`
	Secure   bool
	HTTPOnly bool `mapstructure:"http_only"`
}

type UIConfig struct {
	Theme          string
	RowsPerPage    int    `mapstructure:"rows_per_page"`
	Timezone       string
	DateFormat     string `mapstructure:"date_format"`
	DatetimeFormat string `mapstructure:"datetime_format"`
	ComposeHTML    bool   `mapstructure:"compose_html"`
}

type UploadConfig struct {
	MaxSizeMB int    `mapstructure:"max_size_mb"`
	TempDir   string `mapstructure:"temp_dir"`
}

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

	return cfg
}
