package imap

import (
	"fmt"
	"github.com/emersion/go-imap/v2"
)

// Envelope resume cabeçalhos de uma mensagem para listagem.
type Envelope struct {
	UID       imap.UID  `json:"uid"`
	Subject   string    `json:"subject"`
	From      string    `json:"from"`
	FromEmail string    `json:"from_email"`
	To        string    `json:"to"`
	Date      string    `json:"date"`
	Seen      bool      `json:"seen"`
	Flagged   bool      `json:"flagged"`
}

// FetchEnvelopes busca envelopes de uma lista de UIDs.
func (c *Client) FetchEnvelopes(uids []imap.UID) ([]Envelope, error) {
	if len(uids) == 0 {
		return nil, nil
	}

	seqSet := imap.UIDSetNum(uids...)
	msgs, err := c.Client.Fetch(seqSet, &imap.FetchOptions{
		UID:      true,
		Flags:    true,
		Envelope: true,
	}).Collect()
	if err != nil {
		return nil, err
	}

	result := make([]Envelope, 0, len(msgs))
	for _, m := range msgs {
		env := Envelope{UID: m.UID}
		if m.Envelope != nil {
			env.Subject = m.Envelope.Subject
			if len(m.Envelope.From) > 0 {
				addr := m.Envelope.From[0]
				env.FromEmail = addr.Mailbox + "@" + addr.Host
				if addr.Name != "" {
					env.From = addr.Name
				} else {
					env.From = env.FromEmail
				}
			}
			if len(m.Envelope.To) > 0 {
				addr := m.Envelope.To[0]
				if addr.Name != "" {
					env.To = addr.Name + " <" + addr.Mailbox + "@" + addr.Host + ">"
				} else {
					env.To = addr.Mailbox + "@" + addr.Host
				}
			}
			env.Date = m.Envelope.Date.Format("02/01/2006 15:04")
		}
		for _, f := range m.Flags {
			switch f {
			case imap.FlagSeen:
				env.Seen = true
			case imap.FlagFlagged:
				env.Flagged = true
			}
		}
		result = append(result, env)
	}
	return result, nil
}

// FetchRawMessage baixa o conteúdo RFC822 completo de uma mensagem
func (c *Client) FetchRawMessage(uid imap.UID) ([]byte, error) {
	seqSet := imap.UIDSetNum(uid)
	section := &imap.FetchItemBodySection{Peek: true}

	msgs, err := c.Client.Fetch(seqSet, &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{section},
	}).Collect()
	
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message not found")
	}

	return msgs[0].FindBodySection(section), nil
}

// MarkSeen marca uma mensagem como lida.
func (c *Client) MarkSeen(uid imap.UID) error {
	return c.Client.Store(imap.UIDSetNum(uid), &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}, nil).Close()
}

// MarkUnseen desmarca a flag \Seen.
func (c *Client) MarkUnseen(uid imap.UID) error {
	return c.Client.Store(imap.UIDSetNum(uid), &imap.StoreFlags{
		Op:    imap.StoreFlagsDel,
		Flags: []imap.Flag{imap.FlagSeen},
	}, nil).Close()
}

// MarkFlagged marca/desmarca a flag \Flagged.
func (c *Client) MarkFlagged(uid imap.UID, flagged bool) error {
	op := imap.StoreFlagsAdd
	if !flagged {
		op = imap.StoreFlagsDel
	}
	return c.Client.Store(imap.UIDSetNum(uid), &imap.StoreFlags{
		Op:    op,
		Flags: []imap.Flag{imap.FlagFlagged},
	}, nil).Close()
}

// MoveMessage move uma mensagem para outra pasta.
func (c *Client) MoveMessage(uid imap.UID, dest string) error {
	_, err := c.Client.Move(imap.UIDSetNum(uid), dest).Wait()
	return err
}

// DeleteMessage marca como \Deleted e executa EXPUNGE.
func (c *Client) DeleteMessage(uid imap.UID) error {
	if err := c.Client.Store(imap.UIDSetNum(uid), &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil).Close(); err != nil {
		return err
	}
	return c.Client.Expunge().Close()
}
