package commands

import (
	"time"

	goimap "github.com/emersion/go-imap/v2"
	"go-cubemail/internal/config"
	mailimap "go-cubemail/internal/imap"
)

// imapConnect opens an authenticated IMAP session for the EAS user credentials in ctx.
//
// Each command handler that needs IMAP opens its own short-lived connection using the
// Basic Auth username and password stored on the request Context.
func imapConnect(cfg *config.Config, ctx *Context) (*mailimap.Client, error) {
	timeout := time.Duration(cfg.IMAP.TimeoutSec) * time.Second
	return mailimap.Connect(cfg.IMAP.Host, cfg.IMAP.Port, cfg.IMAP.TLS, timeout, ctx.Username, ctx.Password, cfg.Server.Debug)
}

// appendToSent saves a copy of raw RFC822 bytes to the user's Sent folder when possible.
//
// Failures are silently ignored (best-effort). The Sent mailbox is detected by IMAP
// icon type "sent", falling back to the literal name "Sent".
func appendToSent(cfg *config.Config, ctx *Context, raw []byte) {
	if len(raw) == 0 {
		return
	}
	client, err := imapConnect(cfg, ctx)
	if err != nil {
		return
	}
	defer client.Close()

	sentFolder := "Sent"
	if boxes, err := client.ListMailboxes(); err == nil {
		for _, mb := range boxes {
			if mb.IconType == "sent" {
				sentFolder = mb.Name
				break
			}
		}
	}
	_ = client.AppendMessage(sentFolder, []goimap.Flag{goimap.FlagSeen}, raw)
}
