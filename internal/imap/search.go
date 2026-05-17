package imap

import "github.com/emersion/go-imap/v2"

// SearchCriteria parametriza uma busca IMAP.
type SearchCriteria struct {
	Subject string
	From    string
	To      string
	Body    string
	Unseen  bool
	Flagged bool
}

// Search executa uma busca IMAP e retorna UIDs correspondentes.
// Campos de texto (Subject, From, To, Body) são combinados com OR.
func (c *Client) Search(criteria *SearchCriteria) ([]imap.UID, error) {
	sc := &imap.SearchCriteria{}

	// Critérios de texto: usa OR para que qualquer campo baste
	var textCriteria []imap.SearchCriteria
	if criteria.Subject != "" {
		textCriteria = append(textCriteria, imap.SearchCriteria{
			Header: []imap.SearchCriteriaHeaderField{{Key: "Subject", Value: criteria.Subject}},
		})
	}
	if criteria.From != "" {
		textCriteria = append(textCriteria, imap.SearchCriteria{
			Header: []imap.SearchCriteriaHeaderField{{Key: "From", Value: criteria.From}},
		})
		textCriteria = append(textCriteria, imap.SearchCriteria{
			Header: []imap.SearchCriteriaHeaderField{{Key: "To", Value: criteria.From}},
		})
	}
	if criteria.Body != "" {
		textCriteria = append(textCriteria, imap.SearchCriteria{Body: []string{criteria.Body}})
	}

	// Encadeia OR progressivamente: OR(a, OR(b, c)) …
	for len(textCriteria) > 1 {
		last := textCriteria[len(textCriteria)-1]
		prev := textCriteria[len(textCriteria)-2]
		merged := imap.SearchCriteria{}
		merged.Or = append(merged.Or, [2]imap.SearchCriteria{prev, last})
		textCriteria = append(textCriteria[:len(textCriteria)-2], merged)
	}
	if len(textCriteria) == 1 {
		sc = &textCriteria[0]
	}

	if criteria.Unseen {
		sc.NotFlag = append(sc.NotFlag, imap.FlagSeen)
	}
	if criteria.Flagged {
		sc.Flag = append(sc.Flag, imap.FlagFlagged)
	}

	data, err := c.Client.UIDSearch(sc, nil).Wait()
	if err != nil {
		return nil, err
	}
	return data.AllUIDs(), nil
}
