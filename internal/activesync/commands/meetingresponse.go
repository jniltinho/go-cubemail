package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	calpkg "go-cubemail/internal/calendar"
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	smtppkg "go-cubemail/internal/smtp"
	"gorm.io/gorm"
)

// MeetingResponseHandler implements the MS-ASCMD MeetingResponse command.
//
// When a mobile user accepts, declines, or tentatively accepts a meeting invitation,
// the handler updates the matching attendee PartStat on the SQL event record,
// regenerates ICalContent, and sends an iMIP METHOD:REPLY email to the organizer.
type MeetingResponseHandler struct {
	cfg       *config.Config
	eventRepo *repository.EventRepo
}

// NewMeetingResponseHandler creates a MeetingResponseHandler.
func NewMeetingResponseHandler(cfg *config.Config, eventRepo *repository.EventRepo) *MeetingResponseHandler {
	return &MeetingResponseHandler{cfg: cfg, eventRepo: eventRepo}
}

// Handle processes a meeting accept/decline/tentative response from a mobile client.
//
// Event lookup uses CalendarId (decimal event.ID ServerId) first, then RequestId
// (iCalendar UID). When the attendee email does not match ctx.Username, the handler
// returns success without modifying the event.
func (h *MeetingResponseHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req easMeetingResponseRequest
	if len(body) > 0 {
		if err := wbxml.Unmarshal(body, &req); err != nil {
			return wbxml.Marshal(easMeetingResponseReply{Status: eas.SyncStatusProtocolError})
		}
	}

	partStat := userResponseToPartStat(req.Request.UserResponse)
	if partStat == "" {
		return wbxml.Marshal(easMeetingResponseReply{Status: eas.SyncStatusProtocolError})
	}

	event, err := h.findEvent(ctx, req.Request.CalendarID, req.Request.RequestID)
	if err != nil {
		return wbxml.Marshal(easMeetingResponseReply{
			Status: eas.StatusSuccess,
			Result: &easMeetingResponseResult{
				RequestID: req.Request.RequestID,
				Status:    eas.SyncStatusObjectNotFound,
			},
		})
	}

	updated := false
	userEmail := strings.ToLower(strings.TrimSpace(ctx.Username))
	for i := range event.Attendees {
		if strings.EqualFold(event.Attendees[i].Email, userEmail) {
			event.Attendees[i].PartStat = partStat
			updated = true
			break
		}
	}
	if !updated {
		return wbxml.Marshal(easMeetingResponseReply{
			Status: eas.StatusSuccess,
			Result: &easMeetingResponseResult{
				RequestID:  req.Request.RequestID,
				CalendarID: req.Request.CalendarID,
				Status:     eas.StatusSuccess,
			},
		})
	}

	event.ICalContent = calpkg.BuildICalContent(event, event.Attendees)
	if err := h.eventRepo.Update(event); err != nil {
		return wbxml.Marshal(easMeetingResponseReply{Status: eas.SyncStatusServerError})
	}

	// Send iMIP METHOD:REPLY to organizer (best-effort; failures do not block the response).
	if event.OrganizerEmail != "" {
		go h.sendIMIPReply(ctx, event, partStat)
	}

	return wbxml.Marshal(easMeetingResponseReply{
		Status: eas.StatusSuccess,
		Result: &easMeetingResponseResult{
			RequestID:  req.Request.RequestID,
			CalendarID: req.Request.CalendarID,
			Status:     eas.StatusSuccess,
		},
	})
}

// sendIMIPReply constructs a METHOD:REPLY iCalendar and sends it to the event organizer via SMTP.
// Called in a goroutine; all errors are silently discarded.
func (h *MeetingResponseHandler) sendIMIPReply(ctx *Context, event *model.Event, partStat string) {
	if h.cfg == nil || event.OrganizerEmail == "" {
		return
	}
	attendeeName := ctx.Username
	attendeeEmail := ctx.Username
	for _, a := range event.Attendees {
		if strings.EqualFold(a.Email, ctx.Username) {
			attendeeEmail = a.Email
			if a.Name != "" {
				attendeeName = a.Name
			}
			break
		}
	}

	ics := fmt.Sprintf(
		"BEGIN:VCALENDAR\r\nMETHOD:REPLY\r\nVERSION:2.0\r\nPRODID:-//go-cubemail//Calendar//EN\r\n"+
			"BEGIN:VEVENT\r\nUID:%s\r\nSUMMARY:%s\r\nDTSTAMP:%s\r\n"+
			"ATTENDEE;PARTSTAT=%s;CN=%s:mailto:%s\r\n"+
			"END:VEVENT\r\nEND:VCALENDAR\r\n",
		event.UID,
		calpkg.FoldLine(event.Summary),
		easTime(event.UpdatedAt, false),
		partStat,
		attendeeName,
		attendeeEmail,
	)

	subject := fmt.Sprintf("%s: %s", partStatLabel(partStat), event.Summary)
	smtpCfg := smtppkg.Config{
		Host:       h.cfg.SMTP.Host,
		Port:       h.cfg.SMTP.Port,
		StartTLS:   h.cfg.SMTP.StartTLS,
		TimeoutSec: h.cfg.SMTP.TimeoutSec,
	}
	_, _ = smtppkg.Send(smtpCfg, ctx.Username, ctx.Password, &smtppkg.Message{
		From:    attendeeEmail,
		To:      []string{event.OrganizerEmail},
		Subject: subject,
		Attachments: []smtppkg.Attachment{
			{Filename: "meeting-reply.ics", ContentType: "text/calendar", Data: []byte(ics)},
		},
	})
}

// partStatLabel returns a human-readable prefix for an iMIP REPLY subject line.
func partStatLabel(partStat string) string {
	switch strings.ToUpper(partStat) {
	case "ACCEPTED":
		return "Accepted"
	case "DECLINED":
		return "Declined"
	case "TENTATIVE":
		return "Tentative"
	default:
		return "Response"
	}
}

// findEvent resolves a calendar event from EAS CalendarId (numeric ServerId) or RequestId (UID).
func (h *MeetingResponseHandler) findEvent(ctx *Context, calendarID, requestID string) (*model.Event, error) {
	if id, ok := parseServerID(calendarID); ok {
		ev, err := h.eventRepo.Get(ctx.UserID, id)
		if err == nil {
			return ev, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if requestID != "" {
		return h.eventRepo.GetByUID(ctx.UserID, requestID)
	}
	return nil, gorm.ErrRecordNotFound
}

// userResponseToPartStat maps MS-ASCMD UserResponse values to iCalendar PARTSTAT strings.
func userResponseToPartStat(userResponse int32) string {
	switch userResponse {
	case 1:
		return "ACCEPTED"
	case 2:
		return "DECLINED"
	case 3:
		return "TENTATIVE"
	default:
		return ""
	}
}

type easMeetingResponseRequest struct {
	XMLName struct{} `wbxml:"MeetingResponse.MeetingResponse"`
	Request struct {
		UserResponse int32  `wbxml:"MeetingResponse.UserResponse"`
		CollectionID string `wbxml:"MeetingResponse.CollectionId,omitempty"`
		CalendarID   string `wbxml:"MeetingResponse.CalendarId,omitempty"`
		RequestID    string `wbxml:"MeetingResponse.RequestId,omitempty"`
	} `wbxml:"MeetingResponse.Request"`
}

type easMeetingResponseReply struct {
	XMLName struct{}                  `wbxml:"MeetingResponse.MeetingResponse"`
	Status  int32                     `wbxml:"MeetingResponse.Status,omitempty"`
	Result  *easMeetingResponseResult `wbxml:"MeetingResponse.Result,omitempty"`
}

type easMeetingResponseResult struct {
	RequestID  string `wbxml:"MeetingResponse.RequestId,omitempty"`
	CalendarID string `wbxml:"MeetingResponse.CalendarId,omitempty"`
	Status     int32  `wbxml:"MeetingResponse.Status"`
}
