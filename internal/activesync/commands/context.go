// Package commands implements Exchange ActiveSync command handlers.
package commands

import (
	"net/url"
)

// Context carries authenticated request state for a single EAS command.
type Context struct {
	UserID          uint       // Internal user ID (model.User.ID).
	Username        string     // IMAP login name from Basic Auth.
	Password        string     // IMAP password from Basic Auth (used for per-command IMAP connections).
	DeviceID        string     // Client device identifier (query param DeviceId).
	DeviceType      string     // Client device type string (query param DeviceType).
	ProtocolVersion string     // MS-ASProtocolVersion request header.
	Query           url.Values // Raw query parameters from the HTTP request.
	Device          any        // *state.EasDevice set by handlers after EnsureDevice.
}
