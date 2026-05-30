package commands

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	mailimap "go-cubemail/internal/imap"
)

// ItemOperationsHandler implements the MS-ASCMD ItemOperations command (Fetch).
//
// Phase 5 MVP: mail Fetch with full message body via IMAP RFC822 download.
// Attachment fetch, Move, and EmptyFolderContents are not implemented yet.
type ItemOperationsHandler struct {
	cfg   *config.Config
	store *state.Store
}

// NewItemOperationsHandler creates an ItemOperationsHandler wired to SMTP/IMAP config and folder state.
func NewItemOperationsHandler(cfg *config.Config, store *state.Store) *ItemOperationsHandler {
	return &ItemOperationsHandler{cfg: cfg, store: store}
}

// Handle processes ItemOperations Fetch and EmptyFolderContents requests.
//
// Fetch: each entry is handled independently; ObjectNotFound (8) per missing item.
// EmptyFolderContents: expunges all messages from the specified mail collection.
func (h *ItemOperationsHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req easItemOperationsRequest
	if len(body) > 0 {
		if err := wbxml.Unmarshal(body, &req); err != nil {
			return wbxml.Marshal(easItemOperationsResponse{Status: eas.SyncStatusProtocolError})
		}
	}

	if len(req.Fetch) == 0 && len(req.EmptyFolderContents) == 0 {
		return wbxml.Marshal(easItemOperationsResponse{Status: eas.SyncStatusProtocolError})
	}

	resp := easItemOperationsResponse{
		Status:   eas.StatusSuccess,
		Response: &easItemOpsResponseBody{},
	}

	for _, fetch := range req.Fetch {
		resp.Response.Fetch = append(resp.Response.Fetch, h.fetchOne(ctx, fetch))
	}
	for _, efc := range req.EmptyFolderContents {
		resp.Response.EmptyFolder = append(resp.Response.EmptyFolder, h.emptyFolder(ctx, efc))
	}

	return wbxml.Marshal(resp)
}

// emptyFolder removes all messages from a mail collection.
func (h *ItemOperationsHandler) emptyFolder(ctx *Context, efc easEmptyFolderContentsRequest) easEmptyFolderResult {
	if !strings.HasPrefix(efc.CollectionID, "mail/") {
		return easEmptyFolderResult{Status: eas.SyncStatusObjectNotFound, CollectionID: efc.CollectionID}
	}
	guid, ok := parseMailCollectionID(efc.CollectionID)
	if !ok {
		return easEmptyFolderResult{Status: eas.SyncStatusObjectNotFound, CollectionID: efc.CollectionID}
	}
	folderPath, err := h.store.FolderPathByGUID(ctx.UserID, guid)
	if err != nil {
		return easEmptyFolderResult{Status: eas.SyncStatusObjectNotFound, CollectionID: efc.CollectionID}
	}
	client, err := imapConnect(h.cfg, ctx)
	if err != nil {
		return easEmptyFolderResult{Status: eas.SyncStatusServerError, CollectionID: efc.CollectionID}
	}
	defer client.Close()
	if err := client.EmptyMailbox(folderPath); err != nil {
		return easEmptyFolderResult{Status: eas.SyncStatusServerError, CollectionID: efc.CollectionID}
	}
	return easEmptyFolderResult{Status: eas.StatusSuccess, CollectionID: efc.CollectionID}
}

// fetchOne executes a single ItemOperations Fetch.
// Handles attachment fetch via FileReference and full message body via CollectionId+ServerId.
func (h *ItemOperationsHandler) fetchOne(ctx *Context, fetch easItemOpsFetchRequest) easItemOpsFetchResponse {
	// Attachment fetch via FileReference (collectionId:uid:partIndex).
	if fileRef := strings.TrimSpace(fetch.FileReference); fileRef != "" {
		return h.fetchAttachment(ctx, fileRef)
	}

	collectionID, serverID := resolveItemOpsIDs(fetch)
	if collectionID == "" || serverID == "" {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}
	if !strings.HasPrefix(collectionID, "mail/") {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}

	opts := defaultMailFetchOptions()
	if fetch.Options != nil && len(fetch.Options.BodyPreference) > 0 {
		opts = mailFetchOptionsFromBodyPreference(fetch.Options.BodyPreference)
	}

	props, err := fetchMailProperties(h.cfg, h.store, ctx, collectionID, serverID, opts)
	if err != nil {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}

	return easItemOpsFetchResponse{
		Status:       eas.StatusSuccess,
		CollectionID: collectionID,
		ServerID:     serverID,
		Properties:   props,
	}
}

// fetchAttachment retrieves a single MIME attachment part identified by FileReference.
//
// FileReference format: "{collectionId}:{uid}:{partIndex}" (e.g. "mail/abc123:42:2").
// The attachment data is returned base64-encoded in an inline data blob.
func (h *ItemOperationsHandler) fetchAttachment(ctx *Context, fileRef string) easItemOpsFetchResponse {
	collectionID, uid, partIdx, ok := parseFileReference(fileRef)
	if !ok {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}
	if !strings.HasPrefix(collectionID, "mail/") {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}

	guid, ok := parseMailCollectionID(collectionID)
	if !ok {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}
	folderPath, err := h.store.FolderPathByGUID(ctx.UserID, guid)
	if err != nil {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}

	client, err := imapConnect(h.cfg, ctx)
	if err != nil {
		return easItemOpsFetchResponse{Status: eas.SyncStatusServerError}
	}
	defer client.Close()

	if err := client.SelectMailbox(folderPath); err != nil {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}
	imapUID, ok := parseServerUID(uid)
	if !ok {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}
	raw, err := client.FetchRawMessage(imapUID)
	if err != nil {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}

	parsed, err := mailimap.ParseMessage(raw)
	if err != nil || partIdx <= 0 || partIdx > len(parsed.Attachments) {
		return easItemOpsFetchResponse{Status: eas.SyncStatusObjectNotFound}
	}

	att := parsed.Attachments[partIdx-1]
	b64Data := base64.StdEncoding.EncodeToString(att.Data)
	ct := att.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}

	return easItemOpsFetchResponse{
		Status:        eas.StatusSuccess,
		FileReference: fileRef,
		ContentType:   ct,
		Data:          b64Data,
	}
}

// parseFileReference splits "{collectionId}:{uid}:{partIdx}" into its components.
func parseFileReference(ref string) (collectionID, uid string, partIdx int, ok bool) {
	// collectionID may contain "/" so we split from the right by ":"
	last := strings.LastIndex(ref, ":")
	if last < 0 {
		return "", "", 0, false
	}
	partStr := ref[last+1:]
	rest := ref[:last]

	second := strings.LastIndex(rest, ":")
	if second < 0 {
		return "", "", 0, false
	}
	collectionID = rest[:second]
	uid = rest[second+1:]

	var n int
	if _, err := fmt.Sscanf(partStr, "%d", &n); err != nil || n <= 0 {
		return "", "", 0, false
	}
	return collectionID, uid, n, true
}

// resolveItemOpsIDs extracts CollectionId and ServerId from explicit fields or Search LongId (folder+uid).
func resolveItemOpsIDs(fetch easItemOpsFetchRequest) (collectionID, serverID string) {
	collectionID = strings.TrimSpace(fetch.CollectionID)
	serverID = strings.TrimSpace(fetch.ServerID)
	if collectionID != "" && serverID != "" {
		return collectionID, serverID
	}
	longID := strings.TrimSpace(fetch.LongID)
	if longID == "" {
		return collectionID, serverID
	}
	if i := strings.LastIndex(longID, "+"); i > 0 {
		return longID[:i], longID[i+1:]
	}
	return collectionID, serverID
}

type easItemOperationsRequest struct {
	XMLName             struct{}                        `wbxml:"ItemOperations.ItemOperations"`
	Fetch               []easItemOpsFetchRequest        `wbxml:"ItemOperations.Fetch"`
	EmptyFolderContents []easEmptyFolderContentsRequest `wbxml:"ItemOperations.EmptyFolderContents"`
}

type easEmptyFolderContentsRequest struct {
	CollectionID string `wbxml:"AirSyncBase.CollectionId"`
	Options      *struct {
		DeleteSubFolders int32 `wbxml:"ItemOperations.DeleteSubFolders,omitempty"`
	} `wbxml:"ItemOperations.Options,omitempty"`
}

type easItemOpsFetchRequest struct {
	Store         string                  `wbxml:"ItemOperations.Store,omitempty"`
	CollectionID  string                  `wbxml:"AirSyncBase.CollectionId,omitempty"`
	ServerID      string                  `wbxml:"AirSyncBase.ServerId,omitempty"`
	LongID        string                  `wbxml:"Search.LongId,omitempty"`
	FileReference string                  `wbxml:"AirSyncBase.FileReference,omitempty"`
	Options       *easItemOpsFetchOptions `wbxml:"ItemOperations.Options,omitempty"`
}

type easItemOpsFetchOptions struct {
	BodyPreference []eas.BodyPreference `wbxml:"AirSyncBase.BodyPreference,omitempty"`
	MIMESupport    int32                `wbxml:"AirSync.MIMESupport,omitempty"`
}

type easItemOperationsResponse struct {
	XMLName  struct{}                `wbxml:"ItemOperations.ItemOperations"`
	Status   int32                   `wbxml:"ItemOperations.Status"`
	Response *easItemOpsResponseBody `wbxml:"ItemOperations.Response,omitempty"`
}

type easItemOpsResponseBody struct {
	Fetch       []easItemOpsFetchResponse  `wbxml:"ItemOperations.Fetch"`
	EmptyFolder []easEmptyFolderResult     `wbxml:"ItemOperations.EmptyFolderContents,omitempty"`
}

type easEmptyFolderResult struct {
	Status       int32  `wbxml:"ItemOperations.Status"`
	CollectionID string `wbxml:"AirSyncBase.CollectionId,omitempty"`
}

type easItemOpsFetchResponse struct {
	Status        int32              `wbxml:"ItemOperations.Status"`
	CollectionID  string             `wbxml:"AirSyncBase.CollectionId,omitempty"`
	ServerID      string             `wbxml:"AirSyncBase.ServerId,omitempty"`
	FileReference string             `wbxml:"AirSyncBase.FileReference,omitempty"`
	ContentType   string             `wbxml:"ItemOperations.ContentType,omitempty"`
	Data          string             `wbxml:"ItemOperations.Data,omitempty"`
	Properties    *easMailFetchProps `wbxml:"ItemOperations.Properties,omitempty"`
}
