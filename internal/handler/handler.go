package handler

import (
	"go-cubemail/internal/config"
	"go-cubemail/internal/repository"
	"gorm.io/gorm"
)

// Handlers groups all domain-specific HTTP handler instances.
// It is created once at startup and passed to the route registration function.
type Handlers struct {
	Auth          *AuthHandler
	Mailbox       *MailboxHandler
	Message       *MessageHandler
	Compose       *ComposeHandler
	Contacts      *ContactsHandler
	Calendar      *CalendarHandler
	CalendarShare *CalendarShareHandler
	Event         *EventHandler
	Settings      *SettingsHandler
	Search        *SearchHandler
	CalDAV        *CalDAVHandler
}

// New initialises all handler instances with the shared configuration and database connection.
func New(cfg *config.Config, db *gorm.DB) *Handlers {
	calRepo       := repository.NewCalendarRepo(db)
	eventRepo     := repository.NewEventRepo(db)
	shareRepo     := repository.NewCalendarShareRepo(db)
	uLookup       := &userLookup{db: db}
	return &Handlers{
		Auth:     &AuthHandler{cfg: cfg},
		Mailbox:  &MailboxHandler{cfg: cfg},
		Message:  &MessageHandler{cfg: cfg},
		Compose:  &ComposeHandler{cfg: cfg},
		Contacts: &ContactsHandler{cfg: cfg, repo: repository.NewContactRepo(db), db: db},
		Calendar: &CalendarHandler{cfg: cfg, db: db, calRepo: calRepo, eventRepo: eventRepo},
		CalendarShare: &CalendarShareHandler{
			cfg: cfg, db: db, calRepo: calRepo, shareRepo: shareRepo, userRepo: uLookup,
		},
		Event:    &EventHandler{cfg: cfg, db: db, calRepo: calRepo, eventRepo: eventRepo},
		Settings: &SettingsHandler{cfg: cfg},
		Search:   &SearchHandler{cfg: cfg},
		CalDAV:   &CalDAVHandler{cfg: cfg, db: db, calRepo: calRepo, eventRepo: eventRepo},
	}
}
