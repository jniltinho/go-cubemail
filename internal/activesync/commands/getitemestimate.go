package commands

import (
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
)

// GetItemEstimateHandler implements the MS-ASCMD GetItemEstimate command.
type GetItemEstimateHandler struct {
	mail     *MailSyncEngine
	calendar *CalendarSyncEngine
	contacts *ContactsSyncEngine
	tasks    *TasksSyncEngine
}

// NewGetItemEstimateHandler creates a GetItemEstimateHandler.
func NewGetItemEstimateHandler(mail *MailSyncEngine, calendar *CalendarSyncEngine, contacts *ContactsSyncEngine, tasks *TasksSyncEngine) *GetItemEstimateHandler {
	return &GetItemEstimateHandler{
		mail:     mail,
		calendar: calendar,
		contacts: contacts,
		tasks:    tasks,
	}
}

// Handle returns approximate item counts per requested collection.
func (h *GetItemEstimateHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req struct {
		XMLName     struct{} `wbxml:"GetItemEstimate.GetItemEstimate"`
		Collections struct {
			Collection []struct {
				CollectionID string `wbxml:"GetItemEstimate.CollectionId"`
			} `wbxml:"GetItemEstimate.Collection"`
		} `wbxml:"GetItemEstimate.Collections"`
	}
	if len(body) > 0 {
		_ = wbxml.Unmarshal(body, &req)
	}

	collections := make([]easGetItemEstimateCollection, 0, len(req.Collections.Collection))
	for _, col := range req.Collections.Collection {
		if col.CollectionID == "" {
			continue
		}
		est, status := h.estimate(ctx, col.CollectionID)
		collections = append(collections, easGetItemEstimateCollection{
			CollectionID: col.CollectionID,
			Estimate:     est,
			Status:       status,
		})
	}

	resp := easGetItemEstimateResponse{
		Status: eas.StatusSuccess,
	}
	if len(collections) > 0 {
		resp.Collections = &easGetItemEstimateCollections{Collection: collections}
	}
	return wbxml.Marshal(resp)
}

// estimate returns the item count and status code for one collection ID.
func (h *GetItemEstimateHandler) estimate(ctx *Context, collectionID string) (int32, int32) {
	switch {
	case strings.HasPrefix(collectionID, "mail/"):
		n, err := h.mail.EstimateCount(ctx, collectionID)
		if err != nil {
			return 0, eas.SyncStatusObjectNotFound
		}
		return n, eas.StatusSuccess
	case strings.HasPrefix(collectionID, "vevent/"):
		n, err := h.calendar.EstimateCount(ctx, collectionID)
		if err != nil {
			return 0, eas.SyncStatusObjectNotFound
		}
		return n, eas.StatusSuccess
	case strings.HasPrefix(collectionID, "vcard/"):
		n, err := h.contacts.EstimateCount(ctx, collectionID)
		if err != nil {
			return 0, eas.SyncStatusObjectNotFound
		}
		return n, eas.StatusSuccess
	case strings.HasPrefix(collectionID, "vtodo/"):
		n, err := h.tasks.EstimateCount(ctx)
		if err != nil {
			return 0, eas.SyncStatusObjectNotFound
		}
		return n, eas.StatusSuccess
	default:
		return 0, eas.StatusSuccess
	}
}

// easGetItemEstimateResponse is the WBXML root for a GetItemEstimate command reply.
type easGetItemEstimateResponse struct {
	XMLName     struct{}                       `wbxml:"GetItemEstimate.GetItemEstimate"`
	Status      int32                          `wbxml:"GetItemEstimate.Status"`
	Collections *easGetItemEstimateCollections `wbxml:"GetItemEstimate.Collections,omitempty"`
}

// easGetItemEstimateCollections wraps one or more per-collection estimate entries.
type easGetItemEstimateCollections struct {
	Collection []easGetItemEstimateCollection `wbxml:"GetItemEstimate.Collection"`
}

// easGetItemEstimateCollection holds the estimated item count for a single collection.
type easGetItemEstimateCollection struct {
	CollectionID string `wbxml:"GetItemEstimate.CollectionId"`
	Estimate     int32  `wbxml:"GetItemEstimate.Estimate"`
	Status       int32  `wbxml:"GetItemEstimate.Status"`
}
