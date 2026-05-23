package imap

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"strings"

	"github.com/emersion/go-message/mail"
)

// ParsedMessage contém o corpo e anexos extraídos de uma mensagem.
type ParsedMessage struct {
	TextPlain   string
	TextHTML    string
	Attachments []Attachment
}

// Attachment descreve um arquivo anexado.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Part        int
	Data        []byte
}

// ParseMessage faz o parse de uma mensagem MIME em bytes raw.
// Imagens inline com cid: são convertidas para data: URI no HTML.
func ParseMessage(raw []byte) (*ParsedMessage, error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	pm := &ParsedMessage{}
	part := 0
	// mapa de Content-ID → data URI para imagens inline
	cidMap := make(map[string]string)

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		part++

		ctRaw := p.Header.Get("Content-Type")
		dispRaw := p.Header.Get("Content-Disposition")
		contentID := strings.Trim(p.Header.Get("Content-ID"), "<>")

		mediatype, _, _ := mime.ParseMediaType(ctRaw)
		dispType, dispParams, _ := mime.ParseMediaType(dispRaw)

		data, _ := io.ReadAll(p.Body)
		filename := dispParams["filename"]
		if filename == "" {
			_, ctParams, _ := mime.ParseMediaType(ctRaw)
			filename = ctParams["name"]
		}

		switch {
		case dispType == "attachment" || (filename != "" && dispType != "inline"):
			pm.Attachments = append(pm.Attachments, Attachment{
				Filename:    filename,
				ContentType: mediatype,
				Size:        int64(len(data)),
				Part:        part,
				Data:        data,
			})

		case contentID != "" && strings.HasPrefix(mediatype, "image/"):
			// Imagem inline referenciada por cid: → converter para data: URI
			b64 := base64.StdEncoding.EncodeToString(data)
			cidMap[contentID] = fmt.Sprintf("data:%s;base64,%s", mediatype, b64)

		case strings.HasPrefix(mediatype, "text/html"):
			pm.TextHTML = string(data)

		case strings.HasPrefix(mediatype, "text/plain"):
			pm.TextPlain = string(data)
		}
	}

	// Substituir referências cid: pelo data: URI correspondente
	if pm.TextHTML != "" && len(cidMap) > 0 {
		for cid, dataURI := range cidMap {
			pm.TextHTML = strings.ReplaceAll(pm.TextHTML, "cid:"+cid, dataURI)
		}
	}

	return pm, nil
}


