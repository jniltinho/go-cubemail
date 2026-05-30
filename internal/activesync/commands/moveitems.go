package commands

import (
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	mailimap "go-cubemail/internal/imap"
)

// MoveItemsHandler implements the MS-ASCMD MoveItems command.
//
// Moves one or more mail messages from a source mail collection to a destination
// collection by issuing IMAP MoveMessage per item.
type MoveItemsHandler struct {
	cfg   *config.Config
	store *state.Store
}

// NewMoveItemsHandler creates a MoveItemsHandler.
func NewMoveItemsHandler(cfg *config.Config, store *state.Store) *MoveItemsHandler {
	return &MoveItemsHandler{cfg: cfg, store: store}
}

// Handle processes a MoveItems request and returns per-item results.
func (h *MoveItemsHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req easMoveItemsRequest
	if len(body) > 0 {
		if err := wbxml.Unmarshal(body, &req); err != nil {
			return wbxml.Marshal(easMoveItemsResponse{
				Moves: []easMoveResult{{Status: eas.SyncStatusProtocolError}},
			})
		}
	}
	if len(req.Move) == 0 {
		return wbxml.Marshal(easMoveItemsResponse{
			Moves: []easMoveResult{{Status: eas.SyncStatusProtocolError}},
		})
	}

	client, err := imapConnect(h.cfg, ctx)
	if err != nil {
		return wbxml.Marshal(easMoveItemsResponse{
			Moves: []easMoveResult{{Status: eas.SyncStatusServerError}},
		})
	}
	defer client.Close()

	results := make([]easMoveResult, 0, len(req.Move))
	for _, mv := range req.Move {
		results = append(results, h.moveOne(ctx, client, mv))
	}
	return wbxml.Marshal(easMoveItemsResponse{Moves: results})
}

// moveOne moves a single message identified by SrcMsgId from SrcFldId to DstFldId.
func (h *MoveItemsHandler) moveOne(ctx *Context, client *mailimap.Client, mv easMoveItem) easMoveResult {
	if !strings.HasPrefix(mv.SrcFldId, "mail/") || !strings.HasPrefix(mv.DstFldId, "mail/") {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	srcGUID, ok := parseMailCollectionID(mv.SrcFldId)
	if !ok {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	dstGUID, ok := parseMailCollectionID(mv.DstFldId)
	if !ok {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	srcPath, err := h.store.FolderPathByGUID(ctx.UserID, srcGUID)
	if err != nil {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	dstPath, err := h.store.FolderPathByGUID(ctx.UserID, dstGUID)
	if err != nil {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	uid, ok := parseServerUID(mv.SrcMsgId)
	if !ok {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	if err := client.SelectMailbox(srcPath); err != nil {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusObjectNotFound}
	}
	if err := client.MoveMessage(uid, dstPath); err != nil {
		return easMoveResult{SrcMsgId: mv.SrcMsgId, Status: eas.SyncStatusServerError}
	}
	// Return same ID; client reconciles via next Sync.
	return easMoveResult{SrcMsgId: mv.SrcMsgId, DstMsgId: mv.SrcMsgId, Status: eas.StatusSuccess}
}

type easMoveItemsRequest struct {
	XMLName struct{}      `wbxml:"MoveItems.MoveItems"`
	Move    []easMoveItem `wbxml:"MoveItems.Move"`
}

type easMoveItem struct {
	SrcMsgId string `wbxml:"MoveItems.SrcMsgId"`
	SrcFldId string `wbxml:"MoveItems.SrcFldId"`
	DstFldId string `wbxml:"MoveItems.DstFldId"`
}

type easMoveItemsResponse struct {
	XMLName struct{}        `wbxml:"MoveItems.MoveItems"`
	Moves   []easMoveResult `wbxml:"MoveItems.Response,omitempty"`
}

type easMoveResult struct {
	SrcMsgId string `wbxml:"MoveItems.SrcMsgId,omitempty"`
	DstMsgId string `wbxml:"MoveItems.DstMsgId,omitempty"`
	Status   int32  `wbxml:"MoveItems.Status"`
}
