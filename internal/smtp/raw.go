package smtp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	netmail "net/mail"
	"net/smtp"
	"strings"
)

// SendRaw transmits a complete RFC822 message via SMTP using the configured server.
//
// The message must include valid From, To/Cc/Bcc headers. Authentication uses PLAIN with
// the supplied user and pass. Port 465 uses implicit TLS (SMTPS); other ports use STARTTLS
// when supported (or when cfg.StartTLS requires it). Returns the same raw bytes on success
// so callers can append a copy to the Sent IMAP folder.
func SendRaw(cfg Config, user, pass string, raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	msg, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse rfc822: %w", err)
	}

	from, err := firstFromAddress(msg)
	if err != nil {
		return nil, err
	}
	rcpts := collectRecipients(msg)
	if len(rcpts) == 0 {
		return nil, fmt.Errorf("no recipients")
	}

	auth := smtp.PlainAuth("", user, pass, cfg.Host)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.Port == 465 {
		if err := sendRawSMTPS(addr, cfg.Host, auth, from, rcpts, raw); err != nil {
			return nil, err
		}
		return raw, nil
	}

	if err := sendRawSTARTTLS(addr, cfg.Host, auth, from, rcpts, raw, cfg.StartTLS); err != nil {
		return nil, err
	}
	return raw, nil
}

// firstFromAddress extracts the SMTP envelope From address from an RFC822 message.
func firstFromAddress(msg *netmail.Message) (string, error) {
	addrs, err := msg.Header.AddressList("From")
	if err != nil || len(addrs) == 0 {
		if from := strings.TrimSpace(msg.Header.Get("From")); from != "" {
			if addr, err := netmail.ParseAddress(from); err == nil {
				return addr.Address, nil
			}
			return from, nil
		}
		return "", fmt.Errorf("missing From header")
	}
	return addrs[0].Address, nil
}

// collectRecipients returns deduplicated To, Cc, and Bcc addresses from an RFC822 message.
func collectRecipients(msg *netmail.Message) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, key := range []string{"To", "Cc", "Bcc"} {
		addrs, err := msg.Header.AddressList(key)
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if a.Address == "" {
				continue
			}
			lower := strings.ToLower(a.Address)
			if _, ok := seen[lower]; ok {
				continue
			}
			seen[lower] = struct{}{}
			out = append(out, a.Address)
		}
	}
	return out
}

// sendRawSTARTTLS delivers a message over SMTP with optional STARTTLS (typical port 587).
func sendRawSTARTTLS(addr, host string, auth smtp.Auth, from string, to []string, raw []byte, requireTLS bool) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	} else if requireTLS {
		return fmt.Errorf("server does not support STARTTLS")
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
			 return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt to %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("data close: %w", err)
	}
	return client.Quit()
}

// sendRawSMTPS delivers a message over implicit TLS (typical port 465).
func sendRawSMTPS(addr, host string, auth smtp.Auth, from string, to []string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt to %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("data close: %w", err)
	}
	return client.Quit()
}
