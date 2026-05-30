package commands

import (
	"bytes"
	"fmt"

	"github.com/remdev/go-activesync/wbxml"
)

// marshalApplicationDataBody encodes a value whose XMLName is AirSync.ApplicationData
// and returns the inner WBXML bytes for use in SyncAdd/SyncChange RawElement.
func marshalApplicationDataBody(v any) ([]byte, error) {
	data, err := wbxml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal application data: %w", err)
	}
	dec := wbxml.NewDecoder(bytes.NewReader(data))
	if _, err := dec.ReadHeader(); err != nil {
		return nil, fmt.Errorf("read wbxml header: %w", err)
	}
	tok, err := dec.NextToken()
	if err != nil {
		return nil, fmt.Errorf("read application data tag: %w", err)
	}
	body, err := dec.CaptureRaw(tok.HasContent)
	if err != nil {
		return nil, fmt.Errorf("capture application data body: %w", err)
	}
	return body, nil
}
