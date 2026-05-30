package commands

import (
	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/model"
	"gorm.io/gorm"
)

// SettingsHandler implements the MS-ASCMD Settings command.
//
// Supports UserInformation (email address), DeviceInformation (model/agent/name),
// and OOF (Out of Office) Get/Set backed by UserSettings in the database.
type SettingsHandler struct {
	db *gorm.DB
}

// NewSettingsHandler creates a SettingsHandler with optional database access for OOF.
func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

// Handle responds to Settings Get and Set requests.
//
// When the request body is empty or Get is absent, UserInformation is returned by default.
func (h *SettingsHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	var req easSettingsRequest
	if len(body) > 0 {
		_ = wbxml.Unmarshal(body, &req)
	}

	resp := easSettingsResponse{Status: eas.StatusSuccess}

	// Handle OOF Set before building the Get response.
	if req.Set != nil && req.Set.OOF != nil {
		h.applyOOFSet(ctx, req.Set.OOF)
	}

	if req.Get != nil {
		if req.Get.UserInformation != nil {
			resp.Get = &easSettingsGet{
				UserInformation: &easSettingsUserInfo{
					Status: eas.StatusSuccess,
					Set: easSettingsUserInfoSet{
						EmailAddresses: ctx.Username,
						SmtpAddress:    ctx.Username,
					},
				},
			}
		}
		if req.Get.DeviceInformation != nil {
			if resp.Get == nil {
				resp.Get = &easSettingsGet{}
			}
			resp.Get.DeviceInformation = &easSettingsDeviceInfo{
				Status: eas.StatusSuccess,
				Set: easSettingsDeviceInfoSet{
					Model:        ctx.DeviceType,
					UserAgent:    ctx.DeviceType,
					FriendlyName: ctx.DeviceID,
				},
			}
		}
		if req.Get.OOF != nil {
			if resp.Get == nil {
				resp.Get = &easSettingsGet{}
			}
			resp.Get.OOF = h.buildOOFGet(ctx)
		}
	}

	// Default: return UserInformation when Get is absent or empty (common client probe).
	if resp.Get == nil {
		resp.Get = &easSettingsGet{
			UserInformation: &easSettingsUserInfo{
				Status: eas.StatusSuccess,
				Set: easSettingsUserInfoSet{
					EmailAddresses: ctx.Username,
					SmtpAddress:    ctx.Username,
				},
			},
		}
	}

	return wbxml.Marshal(resp)
}

// buildOOFGet loads OOF state from UserSettings and returns the EAS OOF Get response.
func (h *SettingsHandler) buildOOFGet(ctx *Context) *easSettingsOOF {
	oof := &easSettingsOOF{Status: eas.StatusSuccess}
	if h.db == nil {
		oof.Get = &easSettingsOOFGet{OofState: 0}
		return oof
	}
	var settings model.UserSettings
	if err := h.db.Where("user_id = ?", ctx.UserID).First(&settings).Error; err != nil {
		oof.Get = &easSettingsOOFGet{OofState: 0}
		return oof
	}
	oofState := int32(0)
	if settings.OOFEnabled {
		oofState = 1
	}
	var msg *easSettingsOOFMessage
	if settings.OOFMessage != "" {
		msg = &easSettingsOOFMessage{
			AppliesToInternal:        1,
			AppliesToExternalKnown:   1,
			AppliesToExternalUnknown: 1,
			ReplyMessage:             settings.OOFMessage,
		}
	}
	oof.Get = &easSettingsOOFGet{OofState: oofState, OofMessage: msg}
	return oof
}

// applyOOFSet persists OOF settings from an EAS Settings Set request.
func (h *SettingsHandler) applyOOFSet(ctx *Context, oofSet *easSettingsOOFSet) {
	if h.db == nil || ctx.UserID == 0 {
		return
	}
	var settings model.UserSettings
	h.db.Where("user_id = ?", ctx.UserID).FirstOrCreate(&settings, model.UserSettings{UserID: ctx.UserID})

	settings.OOFEnabled = oofSet.OofState != 0
	if oofSet.OofMessage != nil && oofSet.OofMessage.ReplyMessage != "" {
		settings.OOFMessage = oofSet.OofMessage.ReplyMessage
	}
	h.db.Save(&settings)
}

// ── Request / response types ───────────────────────────────────────────────

type easSettingsRequest struct {
	XMLName struct{} `wbxml:"Settings.Settings"`
	Get     *struct {
		UserInformation   *struct{} `wbxml:"Settings.UserInformation,omitempty"`
		DeviceInformation *struct{} `wbxml:"Settings.DeviceInformation,omitempty"`
		OOF               *struct{} `wbxml:"Settings.Oof,omitempty"`
	} `wbxml:"Settings.Get,omitempty"`
	Set *struct {
		OOF *easSettingsOOFSet `wbxml:"Settings.Oof,omitempty"`
	} `wbxml:"Settings.Set,omitempty"`
}

type easSettingsResponse struct {
	XMLName struct{}         `wbxml:"Settings.Settings"`
	Status  int32            `wbxml:"Settings.Status"`
	Get     *easSettingsGet  `wbxml:"Settings.Get,omitempty"`
}

type easSettingsGet struct {
	UserInformation   *easSettingsUserInfo   `wbxml:"Settings.UserInformation,omitempty"`
	DeviceInformation *easSettingsDeviceInfo `wbxml:"Settings.DeviceInformation,omitempty"`
	OOF               *easSettingsOOF        `wbxml:"Settings.Oof,omitempty"`
}

type easSettingsUserInfo struct {
	Status int32                  `wbxml:"Settings.Status"`
	Set    easSettingsUserInfoSet `wbxml:"Settings.Set"`
}

type easSettingsUserInfoSet struct {
	EmailAddresses string `wbxml:"Settings.EmailAddresses,omitempty"`
	SmtpAddress    string `wbxml:"Settings.SmtpAddress,omitempty"`
}

type easSettingsDeviceInfo struct {
	Status int32                     `wbxml:"Settings.Status"`
	Set    easSettingsDeviceInfoSet  `wbxml:"Settings.Set"`
}

type easSettingsDeviceInfoSet struct {
	Model        string `wbxml:"Settings.Model,omitempty"`
	UserAgent    string `wbxml:"Settings.UserAgent,omitempty"`
	FriendlyName string `wbxml:"Settings.FriendlyName,omitempty"`
}

type easSettingsOOF struct {
	Status int32              `wbxml:"Settings.Status"`
	Get    *easSettingsOOFGet `wbxml:"Settings.Get,omitempty"`
}

type easSettingsOOFGet struct {
	OofState      int32                  `wbxml:"Settings.OofState,omitempty"`
	OofMessage    *easSettingsOOFMessage `wbxml:"Settings.OofMessage,omitempty"`
}

type easSettingsOOFMessage struct {
	AppliesToInternal        int32  `wbxml:"Settings.AppliesToInternal,omitempty"`
	AppliesToExternalKnown   int32  `wbxml:"Settings.AppliesToExternalKnown,omitempty"`
	AppliesToExternalUnknown int32  `wbxml:"Settings.AppliesToExternalUnknown,omitempty"`
	ReplyMessage             string `wbxml:"Settings.ReplyMessage,omitempty"`
}

type easSettingsOOFSet struct {
	OofState   int32                  `wbxml:"Settings.OofState,omitempty"`
	OofMessage *easSettingsOOFMessage `wbxml:"Settings.OofMessage,omitempty"`
}
