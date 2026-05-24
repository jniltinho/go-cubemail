package handler

import (
	"go-cubemail/internal/config"
	"go-cubemail/internal/repository"
	"gorm.io/gorm"
)

// Handlers groups all domain-specific HTTP handler instances.
// It is created once at startup and passed to the route registration function.
type Handlers struct {
	Auth     *AuthHandler
	Mailbox  *MailboxHandler
	Message  *MessageHandler
	Compose  *ComposeHandler
	Contacts *ContactsHandler
	Settings *SettingsHandler
	Search   *SearchHandler
}

// New initialises all handler instances with the shared configuration and database connection.
func New(cfg *config.Config, db *gorm.DB) *Handlers {
	return &Handlers{
		Auth:     &AuthHandler{cfg: cfg},
		Mailbox:  &MailboxHandler{cfg: cfg},
		Message:  &MessageHandler{cfg: cfg},
		Compose:  &ComposeHandler{cfg: cfg},
		Contacts: &ContactsHandler{cfg: cfg, repo: repository.NewContactRepo(db), db: db},
		Settings: &SettingsHandler{cfg: cfg},
		Search:   &SearchHandler{cfg: cfg},
	}
}
