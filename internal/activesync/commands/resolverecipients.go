package commands

import (
	"strings"

	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/repository"
)

// ResolveRecipientsHandler implements MS-ASCMD ResolveRecipients.
//
// Clients (typically iOS Mail in meeting-invitation flows) send a list of
// email addresses or display names; the server returns matching contact details
// (DisplayName + EmailAddress) from the user's address book. GAL search is not
// implemented — only the local contacts table is searched.
type ResolveRecipientsHandler struct {
	contactRepo *repository.ContactRepo
}

// NewResolveRecipientsHandler creates a ResolveRecipientsHandler.
func NewResolveRecipientsHandler(contactRepo *repository.ContactRepo) *ResolveRecipientsHandler {
	return &ResolveRecipientsHandler{contactRepo: contactRepo}
}

// Handle processes a ResolveRecipients request and returns contact matches.
//
// For each requested email/name, the handler searches the contact list and returns
// the best match. Unknown recipients receive Status=4 (AmbiguousRecipientFull).
// On protocol error the outer status is 5 (ProtocolError).
func (h *ResolveRecipientsHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req easResolveRecipientsRequest
	if len(body) > 0 {
		if err := wbxml.Unmarshal(body, &req); err != nil {
			return wbxml.Marshal(easResolveRecipientsResponse{Status: 5})
		}
	}
	if len(req.To) == 0 {
		return wbxml.Marshal(easResolveRecipientsResponse{Status: 5})
	}

	contacts, _ := h.contactRepo.List(ctx.UserID)

	responses := make([]easResolveResponse, 0, len(req.To))
	for _, to := range req.To {
		query := strings.ToLower(strings.TrimSpace(to))
		var matched []easResolveRecipient
		for _, c := range contacts {
			fullName := strings.TrimSpace(c.FirstName + " " + c.LastName)
			if strings.Contains(strings.ToLower(c.Email), query) ||
				strings.Contains(strings.ToLower(fullName), query) {
				matched = append(matched, easResolveRecipient{
					Type:         1, // type 1 = SMTP
					DisplayName:  fullName,
					EmailAddress: c.Email,
				})
			}
		}

		status := int32(4) // AmbiguousRecipientFull (not found)
		if len(matched) == 1 {
			status = 1 // success + unique match
		} else if len(matched) > 1 {
			status = 2 // ambiguous — multiple matches
		}

		if len(matched) > 20 {
			matched = matched[:20]
		}

		responses = append(responses, easResolveResponse{
			To:           to,
			Status:       status,
			RecipientCount: int32(len(matched)),
			Recipients:   matched,
		})
	}

	return wbxml.Marshal(easResolveRecipientsResponse{
		Status:    1,
		Responses: responses,
	})
}

type easResolveRecipientsRequest struct {
	XMLName struct{} `wbxml:"ResolveRecipients.ResolveRecipients"`
	To      []string `wbxml:"ResolveRecipients.To"`
}

type easResolveRecipientsResponse struct {
	XMLName   struct{}              `wbxml:"ResolveRecipients.ResolveRecipients"`
	Status    int32                 `wbxml:"ResolveRecipients.Status"`
	Responses []easResolveResponse  `wbxml:"ResolveRecipients.Response,omitempty"`
}

type easResolveResponse struct {
	To             string                 `wbxml:"ResolveRecipients.To"`
	Status         int32                  `wbxml:"ResolveRecipients.Status"`
	RecipientCount int32                  `wbxml:"ResolveRecipients.RecipientCount,omitempty"`
	Recipients     []easResolveRecipient  `wbxml:"ResolveRecipients.Recipient,omitempty"`
}

type easResolveRecipient struct {
	Type         int32  `wbxml:"ResolveRecipients.Type"`
	DisplayName  string `wbxml:"ResolveRecipients.DisplayName,omitempty"`
	EmailAddress string `wbxml:"ResolveRecipients.EmailAddress"`
}
