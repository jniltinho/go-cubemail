package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	mailimap "go-cubemail/internal/imap"
)

// SearchHandler implements the MS-ASCMD Search command (mailbox full-text search).
//
// Returns matching messages as Search.Result entries (LongId + Properties) plus Total count.
// GAL/contact search is not implemented yet.
type SearchHandler struct {
	cfg   *config.Config
	store *state.Store
}

// NewSearchHandler creates a SearchHandler.
func NewSearchHandler(cfg *config.Config, store *state.Store) *SearchHandler {
	return &SearchHandler{cfg: cfg, store: store}
}

// Handle executes a mailbox search and returns matching message summaries.
//
// Each Search.Result includes LongId (collectionId+serverId), CollectionId, and Properties
// with Email headers and optional AirSyncBase.Body preview. Range defaults to 0-99.
func (h *SearchHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req easSearchRequest
	if len(body) > 0 {
		_ = wbxml.Unmarshal(body, &req)
	}

	query := strings.TrimSpace(req.Store.Query.FreeText)
	collectionID := strings.TrimSpace(req.Store.CollectionID)
	mailbox := "INBOX"
	if strings.HasPrefix(collectionID, "mail/") {
		guid := strings.TrimPrefix(collectionID, "mail/")
		if path, err := h.store.FolderPathByGUID(ctx.UserID, guid); err == nil {
			mailbox = path
		}
	}

	start, end := parseSearchRange(req.Store.Range)
	opts := defaultMailFetchOptions()
	if len(req.Store.Options.BodyPreference) > 0 {
		opts = mailFetchOptionsFromBodyPreference(req.Store.Options.BodyPreference)
	}

	results, total, err := h.searchMailbox(ctx, collectionID, mailbox, query, start, end, opts)
	if err != nil {
		return wbxml.Marshal(easSearchResponse{Status: eas.SyncStatusObjectNotFound})
	}

	rangeEnd := end
	if total > 0 && len(results) > 0 {
		rangeEnd = start + int32(len(results)) - 1
	}
	if total == 0 {
		start = 0
		rangeEnd = 0
	}

	return wbxml.Marshal(easSearchResponse{
		Status: eas.StatusSuccess,
		Response: &easSearchResponseBody{
			Store: easSearchStoreResult{
				Status:  eas.StatusSuccess,
				Total:   total,
				Range:   fmt.Sprintf("%d-%d", start, rangeEnd),
				Results: results,
			},
		},
	})
}

// parseSearchRange parses MS-ASCMD Search Range strings such as "0-99" into inclusive bounds.
// Defaults to 0-99 when the input is empty or malformed.
func parseSearchRange(raw string) (start, end int32) {
	start, end = 0, 99
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return start, end
	}
	parts := strings.SplitN(raw, "-", 2)
	if len(parts) != 2 {
		return start, end
	}
	if v, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32); err == nil {
		start = int32(v)
	}
	if v, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32); err == nil {
		end = int32(v)
	}
	if end < start {
		end = start + 99
	}
	return start, end
}

// searchMailbox runs IMAP SEARCH and builds Search.Result entries for the requested UID slice.
func (h *SearchHandler) searchMailbox(ctx *Context, collectionID, mailbox, query string, start, end int32, opts mailFetchOptions) ([]easSearchResult, int32, error) {
	client, err := imapConnect(h.cfg, ctx)
	if err != nil {
		return nil, 0, err
	}
	defer client.Close()

	if err := client.SelectMailbox(mailbox); err != nil {
		return nil, 0, err
	}

	criteria := &mailimap.SearchCriteria{}
	if query != "" {
		criteria.Subject = query
		criteria.From = query
		criteria.Body = query
	}
	uids, err := client.Search(criteria)
	if err != nil {
		return nil, 0, err
	}
	total := int32(len(uids))
	if total == 0 {
		return nil, 0, nil
	}

	if collectionID == "" {
		if guid, err := h.store.FolderGUID(ctx.UserID, mailbox); err == nil {
			collectionID = "mail/" + guid
		}
	}

	max := end - start + 1
	if max <= 0 {
		max = 100
	}

	var results []easSearchResult
	for i := int(start); i < len(uids) && int32(len(results)) < max; i++ {
		serverID := serverIDForUID(uids[i])
		props, err := fetchMailProperties(h.cfg, h.store, ctx, collectionID, serverID, opts)
		if err != nil {
			continue
		}
		propBytes, err := mailSearchPropertiesBytes(props)
		if err != nil {
			continue
		}
		longID := collectionID + "+" + serverID
		results = append(results, easSearchResult{
			LongID:       longID,
			CollectionID: collectionID,
			Properties: &wbxml.RawElement{
				Page:  wbxml.PageSearch,
				Bytes: propBytes,
			},
		})
	}
	return results, total, nil
}

type easSearchRequest struct {
	XMLName struct{} `wbxml:"Search.Search"`
	Store   struct {
		Name         string `wbxml:"Search.Name,omitempty"`
		CollectionID string `wbxml:"Search.CollectionId,omitempty"`
		Range        string `wbxml:"Search.Range,omitempty"`
		Query        struct {
			FreeText string `wbxml:"Search.FreeText,omitempty"`
		} `wbxml:"Search.Query,omitempty"`
		Options struct {
			BodyPreference []eas.BodyPreference `wbxml:"AirSyncBase.BodyPreference,omitempty"`
		} `wbxml:"Search.Options,omitempty"`
	} `wbxml:"Search.Store"`
}

type easSearchResponse struct {
	XMLName  struct{}               `wbxml:"Search.Search"`
	Status   int32                  `wbxml:"Search.Status"`
	Response *easSearchResponseBody `wbxml:"Search.Response,omitempty"`
}

type easSearchResponseBody struct {
	Store easSearchStoreResult `wbxml:"Search.Store"`
}

type easSearchStoreResult struct {
	Status  int32             `wbxml:"Search.Status"`
	Total   int32             `wbxml:"Search.Total"`
	Range   string            `wbxml:"Search.Range,omitempty"`
	Results []easSearchResult `wbxml:"Search.Result,omitempty"`
}

type easSearchResult struct {
	LongID       string            `wbxml:"Search.LongId,omitempty"`
	CollectionID string            `wbxml:"AirSyncBase.CollectionId,omitempty"`
	Properties   *wbxml.RawElement `wbxml:"Search.Properties,omitempty"`
}
