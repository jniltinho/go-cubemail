package handler

import (
	"fmt"
	"strings"
	"time"

	"go-cubemail/internal/config"
	imappkg "go-cubemail/internal/imap"
	"go-cubemail/internal/session"
)

// MessageHandler handles reading, flagging, moving, and deleting individual email messages.
type MessageHandler struct {
	cfg *config.Config
}

// messageDownloadName returns a filesystem-safe filename for downloading a message as .eml.
// Special characters that are invalid in filenames are replaced or removed.
func messageDownloadName(subject string, uid uint64) string {
	name := strings.TrimSpace(subject)
	if name == "" {
		return fmt.Sprintf("message-%d.eml", uid)
	}

	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	name = replacer.Replace(name)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return fmt.Sprintf("message-%d.eml", uid)
	}
	return name + ".eml"
}

// imapConn opens an authenticated IMAP connection using the current session credentials.
func (h *MessageHandler) imapConn(s *session.IMAPSession) (*imappkg.Client, error) {
	pass, err := s.Password(h.cfg.Server.SecretKey)
	if err != nil {
		return nil, err
	}
	return imappkg.Connect(s.IMAPHost, s.IMAPPort, h.cfg.IMAP.TLS,
		time.Duration(h.cfg.IMAP.TimeoutSec)*time.Second, s.Username, pass, h.cfg.Server.Debug)
}

// findTrashFolder resolves the real Trash folder name from IMAP mailbox attributes.
// Returns "Trash" as a fallback when no folder is flagged with the \Trash attribute.
func (h *MessageHandler) findTrashFolder(conn *imappkg.Client) string {
	trashFolder := "Trash"
	if boxes, err := conn.ListMailboxes(); err == nil {
		for _, mb := range boxes {
			if mb.IsTrash {
				return mb.Name
			}
		}
	}
	return trashFolder
}
