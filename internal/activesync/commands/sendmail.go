package commands

import (
	"encoding/base64"
	"strings"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/config"
	smtppkg "go-cubemail/internal/smtp"
)

// SendMailHandler implements the MS-ASCMD SendMail command (ComposeMail code page).
//
// Clients submit an outgoing RFC822 message as WBXML (<MIME> field, optionally base64)
// or as a raw message/rfc822 HTTP body (EAS 12.1+). The handler sends the message via
// SMTP using the authenticated user's credentials and optionally appends a copy to the
// Sent IMAP folder when SaveInSentItems is set.
type SendMailHandler struct {
	cfg *config.Config
}

// NewSendMailHandler creates a SendMailHandler wired to the application SMTP config.
func NewSendMailHandler(cfg *config.Config) *SendMailHandler {
	return &SendMailHandler{cfg: cfg}
}

// Handle decodes the outgoing MIME message, sends it via SMTP, and optionally saves to Sent.
//
// On SMTP failure the response carries SyncStatusServerError (6) without returning a Go error,
// matching typical EAS client expectations. Parse errors (empty body) propagate as Go errors.
func (h *SendMailHandler) Handle(ctx *Context, body []byte) ([]byte, error) {
	raw, saveSent, err := parseSendMailBody(body)
	if err != nil {
		return nil, err
	}

	smtpCfg := smtppkg.Config{
		Host:       h.cfg.SMTP.Host,
		Port:       h.cfg.SMTP.Port,
		StartTLS:   h.cfg.SMTP.StartTLS,
		TimeoutSec: h.cfg.SMTP.TimeoutSec,
	}
	sent, err := smtppkg.SendRaw(smtpCfg, ctx.Username, ctx.Password, raw)
	if err != nil {
		return wbxml.Marshal(easSendMailResponse{Status: eas.SyncStatusServerError})
	}

	if saveSent {
		appendToSent(h.cfg, ctx, sent)
	}

	return wbxml.Marshal(easSendMailResponse{Status: eas.StatusSuccess})
}

type easSendMailRequest struct {
	XMLName         struct{} `wbxml:"ComposeMail.SendMail"`
	ClientID        string   `wbxml:"ComposeMail.ClientId,omitempty"`
	SaveInSentItems int32    `wbxml:"ComposeMail.SaveInSentItems,omitempty"`
	MIME            []byte   `wbxml:"ComposeMail.MIME,omitempty"`
}

type easSendMailResponse struct {
	XMLName struct{} `wbxml:"ComposeMail.SendMail"`
	Status  int32    `wbxml:"ComposeMail.Status"`
}

// parseSendMailBody extracts RFC822 bytes from a SendMail WBXML body or raw MIME payload.
//
// Detection order: empty → error; RFC822 heuristics → raw body; WBXML unmarshal → <MIME>
// field (with optional base64 decode). When WBXML unmarshal fails, the body is treated as
// opaque MIME (SaveInSentItems defaults to true).
func parseSendMailBody(body []byte) (raw []byte, saveSent bool, err error) {
	if len(body) == 0 {
		return nil, false, errEmptySendMail
	}

	// Raw MIME (some clients post message/rfc822 directly).
	if looksLikeRFC822(body) {
		return body, true, nil
	}

	var req easSendMailRequest
	if err := wbxml.Unmarshal(body, &req); err != nil {
		// Fallback: treat opaque body as MIME.
		return body, true, nil
	}

	saveSent = req.SaveInSentItems != 0
	if len(req.MIME) == 0 {
		return nil, saveSent, errEmptySendMail
	}

	raw = req.MIME
	if decoded, ok := tryBase64MIME(req.MIME); ok {
		raw = decoded
	}
	return raw, saveSent, nil
}

// looksLikeRFC822 reports whether the first bytes resemble an RFC822 message header block.
// Used when clients POST raw message/rfc822 instead of WBXML-wrapped SendMail.
func looksLikeRFC822(b []byte) bool {
	s := strings.ToUpper(string(b[:min(512, len(b))]))
	return strings.Contains(s, "MIME-VERSION:") ||
		strings.HasPrefix(s, "FROM:") ||
		strings.HasPrefix(s, "RETURN-PATH:")
}

// tryBase64MIME attempts to decode a single-line base64 string into RFC822 bytes.
// Returns false when the input looks like plain MIME text rather than base64.
func tryBase64MIME(b []byte) ([]byte, bool) {
	s := strings.TrimSpace(string(b))
	if s == "" || strings.Contains(s, "\n") && strings.Contains(s, "Content-Type:") {
		return nil, false
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(s)
	}
	if err != nil || len(decoded) == 0 {
		return nil, false
	}
	return decoded, true
}

var errEmptySendMail = errSendMail("empty SendMail MIME body")

type errSendMail string

func (e errSendMail) Error() string { return string(e) }
