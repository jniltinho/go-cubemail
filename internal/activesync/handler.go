// Package activesync implements a Microsoft Exchange ActiveSync (EAS) HTTP server.
package activesync

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/commands"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"gorm.io/gorm"
)

// Handler serves OPTIONS and POST on /Microsoft-Server-ActiveSync.
type Handler struct {
	cfg      *config.Config
	db       *gorm.DB
	store    *state.Store
	dispatch *commands.Dispatcher
	protocol string
	cmdList  string
	versions string
}

// NewHandler constructs an ActiveSync HTTP handler.
func NewHandler(cfg *config.Config, db *gorm.DB) *Handler {
	store := state.NewStore(db)
	calRepo := repository.NewCalendarRepo(db)
	return &Handler{
		cfg:      cfg,
		db:       db,
		store:    store,
		dispatch: commands.NewDispatcher(cfg, db, store, calRepo),
		protocol: cfg.ActiveSync.ProtocolVersion,
		cmdList:  "Sync,SendMail,FolderSync,GetItemEstimate,MeetingResponse,Search,Settings,Ping,ItemOperations,Provision,ResolveRecipients,MoveItems",
		versions: "2.5,12.0,12.1,14.0,14.1,16.0,16.1",
	}
}

// setResponseHeaders writes standard EAS response headers on every OPTIONS/POST reply.
func (h *Handler) setResponseHeaders(c *echo.Context) {
	c.Response().Header().Set("MS-Server-ActiveSync", h.protocol)
	c.Response().Header().Set("MS-ASProtocolCommands", h.cmdList)
	c.Response().Header().Set("MS-ASProtocolVersions", h.versions)
	c.Response().Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")
}

// Options handles OPTIONS /Microsoft-Server-ActiveSync (capability probe).
func (h *Handler) Options(c *echo.Context) error {
	h.setResponseHeaders(c)
	c.Response().Header().Set("Allow", "OPTIONS, POST")
	return c.NoContent(http.StatusOK)
}

// Handle handles POST /Microsoft-Server-ActiveSync and dispatches by Cmd query param.
func (h *Handler) Handle(c *echo.Context) error {
	if !h.cfg.ActiveSync.Enabled {
		return c.NoContent(http.StatusNotFound)
	}

	deviceID := c.QueryParam("DeviceId")
	if deviceID == "" {
		return c.NoContent(http.StatusInternalServerError)
	}

	userID, err := h.userID(c)
	if err != nil {
		c.Response().Header().Set("WWW-Authenticate", `Basic realm="go-cubemail"`)
		return c.NoContent(http.StatusUnauthorized)
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	cmd := strings.TrimSpace(c.QueryParam("Cmd"))
	if cmd == "" && len(body) > 0 {
		cmd = detectCommand(body)
	}

	ctx := &commands.Context{
		UserID:          userID,
		Username:        c.Get("eas_user").(string),
		Password:        c.Get("eas_password").(string),
		DeviceID:        deviceID,
		DeviceType:      c.QueryParam("DeviceType"),
		ProtocolVersion: c.Request().Header.Get("MS-ASProtocolVersion"),
		Query:           c.QueryParams(),
	}

	out, err := h.dispatch.Dispatch(ctx, cmd, body)
	if err != nil {
		if h.cfg.ActiveSync.Debug {
			c.Logger().Error("EAS dispatch error", "cmd", cmd, "error", err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	h.setResponseHeaders(c)
	if len(out) == 0 {
		return c.NoContent(http.StatusOK)
	}
	_, err = c.Response().Write(out)
	return err
}

// userID resolves the authenticated IMAP username to a model.User ID (FirstOrCreate).
func (h *Handler) userID(c *echo.Context) (uint, error) {
	sUser := c.Get("eas_user").(string)
	var user model.User
	err := h.db.Where("imap_user = ?", sUser).
		FirstOrCreate(&user, model.User{ImapUser: sUser}).Error
	return user.ID, err
}

// detectCommand infers the EAS command name from the WBXML root element when Cmd query param is absent.
func detectCommand(body []byte) string {
	type root struct {
		XMLName struct {
			Local string
		}
	}
	var r root
	if err := wbxml.Unmarshal(body, &r); err != nil {
		return ""
	}
	switch strings.ToLower(r.XMLName.Local) {
	case "provision":
		return "Provision"
	case "foldersync":
		return "FolderSync"
	case "ping":
		return "Ping"
	case "sync":
		return "Sync"
	case "sendmail":
		return "SendMail"
	case "meetingresponse":
		return "MeetingResponse"
	case "search":
		return "Search"
	case "settings":
		return "Settings"
	case "itemoperations":
		return "ItemOperations"
	case "moveitems":
		return "MoveItems"
	default:
		return r.XMLName.Local
	}
}

// WriteWBXML encodes a value to ActiveSync WBXML bytes.
func WriteWBXML(v any) ([]byte, error) {
	return wbxml.Marshal(v)
}

// ReadWBXML decodes ActiveSync WBXML bytes into v.
func ReadWBXML(data []byte, v any) error {
	return wbxml.Unmarshal(data, v)
}

// DebugXML returns a best-effort token trace for logging.
func DebugXML(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var buf bytes.Buffer
	dec := wbxml.NewDecoder(bytes.NewReader(data))
	if _, err := dec.ReadHeader(); err != nil {
		return fmt.Sprintf("<wbxml %d bytes>", len(data))
	}
	for {
		tok, err := dec.NextToken()
		if err != nil {
			break
		}
		buf.WriteString(tok.Kind.String())
		buf.WriteByte(' ')
	}
	return strings.TrimSpace(buf.String())
}

// NowUTC returns the current UTC time.
func NowUTC() time.Time { return time.Now().UTC() }
