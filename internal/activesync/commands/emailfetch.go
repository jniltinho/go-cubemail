package commands

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
	"go-cubemail/internal/activesync/state"
	"go-cubemail/internal/config"
	mailimap "go-cubemail/internal/imap"
)

const (
	bodyTypePlain = int32(1)
	bodyTypeHTML  = int32(2)
)

// easAirSyncBody is the AirSyncBase.Body element returned by ItemOperations Fetch and Search.
type easAirSyncBody struct {
	XMLName           struct{} `wbxml:"AirSyncBase.Body"`
	Type              int32    `wbxml:"AirSyncBase.Type"`
	Data              string   `wbxml:"AirSyncBase.Data,omitempty"`
	EstimatedDataSize int32    `wbxml:"AirSyncBase.EstimatedDataSize,omitempty"`
	Truncated         int32    `wbxml:"AirSyncBase.Truncated,omitempty"`
}

// easMailFetchProps holds Email + AirSyncBase fields inside ItemOperations.Properties.
type easMailFetchProps struct {
	XMLName        struct{}        `wbxml:"ItemOperations.Properties"`
	DateReceived   string          `wbxml:"Email.DateReceived,omitempty"`
	Subject        string          `wbxml:"Email.Subject,omitempty"`
	From           string          `wbxml:"Email.From,omitempty"`
	To             string          `wbxml:"Email.To,omitempty"`
	Cc             string          `wbxml:"Email.Cc,omitempty"`
	Read           bool            `wbxml:"Email.Read,omitempty"`
	Importance     int32           `wbxml:"Email.Importance,omitempty"`
	MessageClass   string          `wbxml:"Email.MessageClass,omitempty"`
	ContentClass   string          `wbxml:"Email.ContentClass,omitempty"`
	Body           *easAirSyncBody `wbxml:"AirSyncBase.Body,omitempty"`
	NativeBodyType int32           `wbxml:"AirSyncBase.NativeBodyType,omitempty"`
}

// mailFetchOptions controls body type and truncation for mail ItemOperations/Search fetch.
type mailFetchOptions struct {
	BodyType       int32 // AirSyncBase body type: 1=plain, 2=HTML.
	TruncationSize int32 // Max body bytes before Truncated=1 is set.
}

// defaultMailFetchOptions returns HTML body with a 2 MB truncation limit.
func defaultMailFetchOptions() mailFetchOptions {
	return mailFetchOptions{BodyType: bodyTypeHTML, TruncationSize: 2_000_000}
}

// mailFetchOptionsFromBodyPreference maps client BodyPreference elements to mailFetchOptions.
func mailFetchOptionsFromBodyPreference(prefs []eas.BodyPreference) mailFetchOptions {
	opts := defaultMailFetchOptions()
	for _, p := range prefs {
		if p.Type != 0 {
			opts.BodyType = p.Type
		}
		if p.TruncationSize > 0 {
			opts.TruncationSize = p.TruncationSize
		}
	}
	return opts
}

// fetchMailProperties loads a message from IMAP and builds ItemOperations/Search Properties payload.
//
// Opens a short-lived IMAP session, fetches RFC822 via BODY.PEEK[], parses MIME for text/html,
// and merges envelope headers (Subject, From, flags) into easMailFetchProps with AirSyncBase.Body.
func fetchMailProperties(cfg *config.Config, store *state.Store, ctx *Context, collectionID, serverID string, opts mailFetchOptions) (*easMailFetchProps, error) {
	guid, ok := parseMailCollectionID(collectionID)
	if !ok {
		return nil, fmt.Errorf("invalid mail collection %q", collectionID)
	}
	uid, ok := parseServerUID(serverID)
	if !ok {
		return nil, fmt.Errorf("invalid server id %q", serverID)
	}

	folderPath, err := store.FolderPathByGUID(ctx.UserID, guid)
	if err != nil {
		return nil, err
	}

	client, err := imapConnect(cfg, ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.SelectMailbox(folderPath); err != nil {
		return nil, err
	}

	raw, err := client.FetchRawMessage(uid)
	if err != nil {
		return nil, err
	}
	parsed, err := mailimap.ParseMessage(raw)
	if err != nil {
		return nil, err
	}

	envelopes, err := client.FetchEnvelopes([]imap.UID{uid})
	if err != nil || len(envelopes) == 0 {
		return nil, fmt.Errorf("fetch envelope uid=%v", uid)
	}
	env := envelopes[0]

	bodyText, nativeType := selectMailBody(parsed, opts.BodyType)
	truncated := int32(0)
	if opts.TruncationSize > 0 && int32(len(bodyText)) > opts.TruncationSize {
		bodyText = bodyText[:opts.TruncationSize]
		truncated = 1
	}

	from := env.FromEmail
	if from == "" {
		from = env.From
	}

	props := &easMailFetchProps{
		DateReceived:   easDateFromEnvelope(env.Date),
		Subject:        env.Subject,
		From:           from,
		To:             env.To,
		Read:           env.Seen,
		Importance:     eas.ImportanceNormal,
		MessageClass:   "IPM.Note",
		ContentClass:   "urn:content-classes:message",
		NativeBodyType: nativeType,
		Body: &easAirSyncBody{
			Type:              opts.BodyType,
			Data:              bodyText,
			EstimatedDataSize: int32(len(bodyText)),
			Truncated:         truncated,
		},
	}
	return props, nil
}

// selectMailBody picks plain or HTML body text for the requested EAS body type.
//
// When the preferred type is unavailable, falls back to the other representation and adjusts
// NativeBodyType accordingly (MS-ASCMD body type constants: 1=plain, 2=HTML).
func selectMailBody(parsed *mailimap.ParsedMessage, preferredType int32) (string, int32) {
	html := strings.TrimSpace(parsed.TextHTML)
	plain := strings.TrimSpace(parsed.TextPlain)

	switch preferredType {
	case bodyTypePlain:
		if plain != "" {
			return plain, bodyTypePlain
		}
		if html != "" {
			return html, bodyTypeHTML
		}
	default:
		if html != "" {
			return html, bodyTypeHTML
		}
		if plain != "" {
			return plain, bodyTypePlain
		}
	}
	return "", bodyTypePlain
}

// marshalContainerInner encodes v (with XMLName wrapper tag) and returns inner WBXML bytes.
//
// Used to strip the outer Search.Properties or ItemOperations.Properties wrapper so the payload
// can be embedded inside Search.Result or ItemOperations.Fetch responses.
func marshalContainerInner(v any) ([]byte, error) {
	data, err := wbxml.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := wbxml.NewDecoder(bytes.NewReader(data))
	if _, err := dec.ReadHeader(); err != nil {
		return nil, err
	}
	tok, err := dec.NextToken()
	if err != nil {
		return nil, err
	}
	return dec.CaptureRaw(tok.HasContent)
}

// mailSearchPropertiesRoot is the WBXML wrapper used to encode Search.Properties inner content.
type mailSearchPropertiesRoot struct {
	XMLName        struct{}        `wbxml:"Search.Properties"`
	DateReceived   string          `wbxml:"Email.DateReceived,omitempty"`
	Subject        string          `wbxml:"Email.Subject,omitempty"`
	From           string          `wbxml:"Email.From,omitempty"`
	To             string          `wbxml:"Email.To,omitempty"`
	Read           bool            `wbxml:"Email.Read,omitempty"`
	Importance     int32           `wbxml:"Email.Importance,omitempty"`
	MessageClass   string          `wbxml:"Email.MessageClass,omitempty"`
	ContentClass   string          `wbxml:"Email.ContentClass,omitempty"`
	Body           *easAirSyncBody `wbxml:"AirSyncBase.Body,omitempty"`
	NativeBodyType int32           `wbxml:"AirSyncBase.NativeBodyType,omitempty"`
}

// mailSearchPropertiesBytes encodes mail fetch properties for embedding in Search.Result.Properties.
func mailSearchPropertiesBytes(props *easMailFetchProps) ([]byte, error) {
	if props == nil {
		return nil, fmt.Errorf("nil properties")
	}
	return marshalContainerInner(&mailSearchPropertiesRoot{
		DateReceived:   props.DateReceived,
		Subject:        props.Subject,
		From:           props.From,
		To:             props.To,
		Read:           props.Read,
		Importance:     props.Importance,
		MessageClass:   props.MessageClass,
		ContentClass:   props.ContentClass,
		Body:           props.Body,
		NativeBodyType: props.NativeBodyType,
	})
}
