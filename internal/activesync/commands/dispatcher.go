package commands

import (
	"fmt"
	"strings"

	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	"go-cubemail/internal/repository"
	"gorm.io/gorm"
)

// Dispatcher routes EAS commands to handler functions.
type Dispatcher struct {
	cfg         *config.Config
	db          *gorm.DB
	store       *state.Store
	calRepo     *repository.CalendarRepo
	eventRepo   *repository.EventRepo
	contactRepo *repository.ContactRepo
	provision   *ProvisionHandler
	folder      *FolderSyncHandler
	ping        *PingHandler
	sync        *SyncHandler
	estimate    *GetItemEstimateHandler
	sendmail    *SendMailHandler
	meeting     *MeetingResponseHandler
	search      *SearchHandler
	settings    *SettingsHandler
	itemOps     *ItemOperationsHandler
	moveItems   *MoveItemsHandler
}

// NewDispatcher constructs a command dispatcher with phase 0–5 handlers.
func NewDispatcher(cfg *config.Config, db *gorm.DB, store *state.Store, calRepo *repository.CalendarRepo) *Dispatcher {
	folders := NewFolderBuilder(cfg, store)
	mail := NewMailSyncEngine(cfg, store)
	eventRepo := repository.NewEventRepo(db)
	contactRepo := repository.NewContactRepo(db)
	calendar := NewCalendarSyncEngine(store, calRepo, eventRepo)
	contacts := NewContactsSyncEngine(store, contactRepo)
	tasks := NewTasksSyncEngine(store, eventRepo)
	return &Dispatcher{
		cfg:         cfg,
		db:          db,
		store:       store,
		calRepo:     calRepo,
		eventRepo:   eventRepo,
		contactRepo: contactRepo,
		provision:   &ProvisionHandler{},
		folder:      &FolderSyncHandler{store: store, folders: folders, calRepo: calRepo},
		ping:        NewPingHandler(cfg, store, mail, calendar, contacts),
		sync:        NewSyncHandler(store, mail, calendar, contacts, tasks),
		estimate:    NewGetItemEstimateHandler(mail, calendar, contacts, tasks),
		sendmail:    NewSendMailHandler(cfg),
		meeting:     NewMeetingResponseHandler(cfg, eventRepo),
		search:      NewSearchHandler(cfg, store),
		settings:    NewSettingsHandler(db),
		itemOps:     NewItemOperationsHandler(cfg, store),
		moveItems:   NewMoveItemsHandler(cfg, store),
	}
}

// Dispatch executes the named EAS command and returns WBXML response bytes.
func (d *Dispatcher) Dispatch(ctx *Context, cmd string, body []byte) ([]byte, error) {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "Provision":
		return d.provision.Handle(ctx, body)
	case "FolderSync":
		return d.folder.Handle(ctx, body)
	case "Ping":
		return d.ping.Handle(ctx, body)
	case "Sync":
		return d.sync.Handle(ctx, body)
	case "GetItemEstimate":
		return d.estimate.Handle(ctx, body)
	case "SendMail":
		return d.sendmail.Handle(ctx, body)
	case "MeetingResponse":
		return d.meeting.Handle(ctx, body)
	case "Search":
		return d.search.Handle(ctx, body)
	case "Settings":
		return d.settings.Handle(ctx, body)
	case "ItemOperations":
		return d.itemOps.Handle(ctx, body)
	case "MoveItems":
		return d.moveItems.Handle(ctx, body)
	default:
		return nil, fmt.Errorf("unsupported EAS command: %q", cmd)
	}
}
