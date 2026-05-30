package handler

import (
	"go-cubemail/internal/config"
	"go-cubemail/internal/repository"
	"gorm.io/gorm"
)

// Handlers groups all domain-specific HTTP handler instances.
// It is created once at startup and passed to the route registration function.
type Handlers struct {
	Auth                *AuthHandler
	Mailbox             *MailboxHandler
	Message             *MessageHandler
	Compose             *ComposeHandler
	Contacts            *ContactsHandler
	Calendar            *CalendarHandler
	CalendarShare       *CalendarShareHandler
	CalendarSubscription *CalendarSubscriptionHandler
	Event               *EventHandler
	Settings            *SettingsHandler
	Search              *SearchHandler
	CalDAV              *CalDAVHandler
	CardDAV             *CardDAVHandler
	Push                *PushHandler
	SSE                 *SSEHandler
}

// New initialises all handler instances with the shared configuration and database connection.
func New(cfg *config.Config, db *gorm.DB) *Handlers {
	calRepo       := repository.NewCalendarRepo(db)
	eventRepo     := repository.NewEventRepo(db)
	shareRepo     := repository.NewCalendarShareRepo(db)
	subRepo       := repository.NewCalendarSubscriptionRepo(db)
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
		Settings: &SettingsHandler{
			cfg: cfg, db: db,
			settingsRepo: repository.NewSettingsRepo(db),
			identityRepo: repository.NewIdentityRepo(db),
		},
		Search:   &SearchHandler{cfg: cfg},
		CalDAV:   &CalDAVHandler{cfg: cfg, db: db, calRepo: calRepo, eventRepo: eventRepo},
		CardDAV:  &CardDAVHandler{cfg: cfg, db: db, contactRepo: repository.NewContactRepo(db)},
		Push:     &PushHandler{cfg: cfg, db: db, pushRepo: repository.NewPushSubscriptionRepo(db)},
		SSE:      &SSEHandler{cfg: cfg, db: db, pushRepo: repository.NewPushSubscriptionRepo(db)},
		CalendarSubscription: &CalendarSubscriptionHandler{
			cfg: cfg, db: db, calRepo: calRepo, subRepo: subRepo, evtRepo: eventRepo,
		},
	}
}
