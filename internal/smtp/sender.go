package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"regexp"
	"strings"

	"github.com/jordan-wright/email"
)

// Message representa um e-mail a ser enviado.
type Message struct {
	From        string
	DisplayName string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	TextPlain   string
	TextHTML    string
	Attachments []Attachment
}

// Attachment descreve um arquivo para envio.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Config agrupa as configurações do servidor SMTP.
type Config struct {
	Host       string
	Port       int
	StartTLS   bool
	TimeoutSec int
}

// dataURIRe captura src="data:<mime>;base64,<data>" dentro do HTML.
var dataURIRe = regexp.MustCompile(`src="data:([^;]+);base64,([^"]+)"`)

// extractInlineImages substitui data URIs no HTML por cid: references
// e retorna a lista de inline attachments gerados.
func extractInlineImages(html string) (string, []inlineImage) {
	var inlines []inlineImage
	idx := 0
	result := dataURIRe.ReplaceAllStringFunc(html, func(match string) string {
		parts := dataURIRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		mimeType := parts[1]
		b64data := parts[2]

		// Remove espaços/quebras que o TinyMCE pode ter inserido
		b64data = strings.ReplaceAll(b64data, "\n", "")
		b64data = strings.ReplaceAll(b64data, " ", "")

		data, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			return match
		}

		idx++
		cid := fmt.Sprintf("inline-img-%d", idx)
		ext := "bin"
		switch mimeType {
		case "image/png":
			ext = "png"
		case "image/jpeg", "image/jpg":
			ext = "jpg"
		case "image/gif":
			ext = "gif"
		case "image/webp":
			ext = "webp"
		}
		filename := fmt.Sprintf("inline-%d.%s", idx, ext)

		inlines = append(inlines, inlineImage{cid: cid, filename: filename, mimeType: mimeType, data: data})
		return fmt.Sprintf(`src="cid:%s"`, cid)
	})
	return result, inlines
}

type inlineImage struct {
	cid      string
	filename string
	mimeType string
	data     []byte
}

// Send envia um e-mail via SMTP e retorna os bytes brutos do e-mail.
func Send(cfg Config, user, pass string, msg *Message) ([]byte, error) {
	e := email.NewEmail()
	if msg.DisplayName != "" {
		e.From = fmt.Sprintf("%s <%s>", msg.DisplayName, msg.From)
	} else {
		e.From = msg.From
	}
	e.To = msg.To
	e.Cc = msg.Cc
	e.Bcc = msg.Bcc
	e.Subject = msg.Subject
	e.Text = []byte(msg.TextPlain)

	// Extrai imagens inline (data URIs) do HTML antes de enviar
	htmlBody := msg.TextHTML
	var inlines []inlineImage
	if htmlBody != "" {
		htmlBody, inlines = extractInlineImages(htmlBody)
		e.HTML = []byte(htmlBody)
	}

	// Adiciona imagens inline como partes CID
	for _, img := range inlines {
		part, err := e.Attach(bytes.NewReader(img.data), img.filename, img.mimeType)
		if err != nil {
			return nil, fmt.Errorf("attach inline %s: %w", img.filename, err)
		}
		part.Header.Set("Content-ID", fmt.Sprintf("<%s>", img.cid))
		part.Header.Set("Content-Disposition", "inline")
	}

	// Adiciona anexos normais
	for _, a := range msg.Attachments {
		if _, err := e.Attach(bytes.NewReader(a.Data), a.Filename, a.ContentType); err != nil {
			return nil, fmt.Errorf("attach %s: %w", a.Filename, err)
		}
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", user, pass, cfg.Host)

	// Gera os bytes brutos após montar tudo (inclusive inline images e anexos)
	raw, err := e.Bytes()
	if err != nil {
		return nil, fmt.Errorf("build email: %w", err)
	}

	if cfg.StartTLS {
		if err := e.SendWithStartTLS(addr, auth, nil); err != nil {
			return nil, err
		}
		return raw, nil
	}
	if err := e.Send(addr, auth); err != nil {
		return nil, err
	}
	return raw, nil
}

