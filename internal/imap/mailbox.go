package imap

import (
	"sort"
	"strings"

	"github.com/emersion/go-imap/v2"
)

// MailboxInfo describes a single IMAP folder including tree-structure metadata
// computed from the hierarchy delimiter and standard mailbox attributes.
type MailboxInfo struct {
	Name        string
	Delim       string
	Unseen      uint32
	Messages    uint32
	IsTrash     bool
	NoSelect    bool   // true when the folder is a hierarchy placeholder (\Noselect)
	IsSystem    bool   // true for INBOX, Sent, Drafts, Trash — cannot be renamed or deleted
	IconType    string // "inbox" | "drafts" | "sent" | "trash" | "junk" | "folder"
	DisplayName string // last path segment (used for tree display)
	ParentName  string // full name of the parent folder; empty for top-level
	HasChildren bool   // true if at least one subfolder exists under this folder
	Depth       int    // nesting level (0 = top-level)
	PaddingLeft int    // left-padding in px for tree indentation in the UI
}

// ListMailboxes returns all IMAP folders with unseen/message counts and tree metadata.
// Folders are sorted in standard email order: INBOX → Drafts → Sent → Trash → Junk → others.
func (c *Client) ListMailboxes() ([]MailboxInfo, error) {
	listCmd := c.Client.List("", "*", &imap.ListOptions{
		ReturnStatus: &imap.StatusOptions{NumUnseen: true, NumMessages: true},
	})
	data, err := listCmd.Collect()
	if err != nil {
		return nil, err
	}

	result := make([]MailboxInfo, 0, len(data))
	for _, m := range data {
		mi := MailboxInfo{Name: m.Mailbox}
		if m.Delim != 0 {
			mi.Delim = string(m.Delim)
		}
		if m.Status != nil {
			if m.Status.NumUnseen != nil {
				mi.Unseen = *m.Status.NumUnseen
			}
			if m.Status.NumMessages != nil {
				mi.Messages = *m.Status.NumMessages
			}
		}
		mi.IconType = "folder"
		for _, attr := range m.Attrs {
			switch attr {
			case imap.MailboxAttrNoSelect:
				mi.NoSelect = true
			case imap.MailboxAttrTrash:
				mi.IsTrash = true
				mi.IsSystem = true
				mi.IconType = "trash"
			case imap.MailboxAttrSent:
				mi.IsSystem = true
				mi.IconType = "sent"
			case imap.MailboxAttrDrafts:
				mi.IsSystem = true
				mi.IconType = "drafts"
			case imap.MailboxAttrJunk:
				mi.IsSystem = true
				mi.IconType = "junk"
			}
		}
		// Fall back to well-known names when server does not set attributes.
		switch mi.Name {
		case "Trash", "Lixeira", "Deleted Items", "Deleted Messages":
			mi.IsTrash = true
			mi.IsSystem = true
			mi.IconType = "trash"
		case "INBOX":
			mi.IsSystem = true
			mi.IconType = "inbox"
		case "Sent", "Sent Messages", "Sent Items":
			mi.IsSystem = true
			mi.IconType = "sent"
		case "Drafts":
			mi.IsSystem = true
			mi.IconType = "drafts"
		case "Junk", "Spam":
			mi.IsSystem = true
			mi.IconType = "junk"
		}
		// Compute depth, display name, and parent from the hierarchy path.
		if mi.Delim != "" && strings.Contains(mi.Name, mi.Delim) {
			parts := strings.Split(mi.Name, mi.Delim)
			mi.DisplayName = parts[len(parts)-1]
			mi.Depth = len(parts) - 1
			mi.ParentName = strings.Join(parts[:len(parts)-1], mi.Delim)
		} else {
			mi.DisplayName = mi.Name
		}
		mi.PaddingLeft = 20 + mi.Depth*16

		result = append(result, mi)
	}

	// Mark folders that have at least one direct child.
	for i := range result {
		if result[i].Delim == "" {
			continue
		}
		prefix := result[i].Name + result[i].Delim
		for j := range result {
			if strings.HasPrefix(result[j].Name, prefix) {
				result[i].HasChildren = true
				break
			}
		}
	}

	// Sort in standard email folder order; subfolders stay grouped with their parent.
	folderPriority := func(mi MailboxInfo) int {
		root := mi.Name
		if mi.Delim != "" && strings.Contains(mi.Name, mi.Delim) {
			root = strings.Split(mi.Name, mi.Delim)[0]
		}
		switch root {
		case "INBOX":
			return 0
		case "Drafts":
			return 10
		case "Sent", "Sent Messages", "Sent Items":
			return 20
		case "Trash", "Lixeira", "Deleted Items", "Deleted Messages":
			return 30
		case "Junk", "Spam":
			return 40
		default:
			return 50
		}
	}
	sort.Slice(result, func(i, j int) bool {
		pi, pj := folderPriority(result[i]), folderPriority(result[j])
		if pi != pj {
			return pi < pj
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// UnreadCount returns the number of unseen messages in the given mailbox.
func (c *Client) UnreadCount(mailbox string) (uint32, error) {
	data, err := c.Client.Status(mailbox, &imap.StatusOptions{NumUnseen: true}).Wait()
	if err != nil {
		return 0, err
	}
	if data.NumUnseen == nil {
		return 0, nil
	}
	return *data.NumUnseen, nil
}

// MessageCount returns the total number of messages in the given mailbox.
func (c *Client) MessageCount(mailbox string) (uint32, error) {
	data, err := c.Client.Status(mailbox, &imap.StatusOptions{NumMessages: true}).Wait()
	if err != nil {
		return 0, err
	}
	if data.NumMessages == nil {
		return 0, nil
	}
	return *data.NumMessages, nil
}

// FetchAllUIDs returns all message UIDs in the currently selected mailbox.
func (c *Client) FetchAllUIDs() ([]imap.UID, error) {
	data, err := c.Client.UIDSearch(&imap.SearchCriteria{}, nil).Wait()
	if err != nil {
		return nil, err
	}
	return data.AllUIDs(), nil
}

// SelectMailbox issues an IMAP SELECT command so subsequent operations target the named folder.
func (c *Client) SelectMailbox(mailbox string) error {
	_, err := c.Client.Select(mailbox, nil).Wait()
	return err
}

// CreateMailbox creates a new IMAP folder with the given name.
func (c *Client) CreateMailbox(name string) error {
	return c.Client.Create(name, nil).Wait()
}

// RenameMailbox renames an existing IMAP folder.
func (c *Client) RenameMailbox(oldName, newName string) error {
	return c.Client.Rename(oldName, newName, nil).Wait()
}

// DeleteMailbox deletes the named IMAP folder.
func (c *Client) DeleteMailbox(name string) error {
	return c.Client.Delete(name).Wait()
}

// EnsureSystemFolders creates Drafts, Sent, and Trash if they do not already exist.
// Missing folders are created silently; existing ones are left untouched.
func (c *Client) EnsureSystemFolders() {
	existing := make(map[string]struct{})
	if mailboxes, err := c.ListMailboxes(); err == nil {
		for _, m := range mailboxes {
			existing[strings.ToLower(m.Name)] = struct{}{}
		}
	}
	for _, name := range []string{"Drafts", "Sent", "Trash"} {
		if _, ok := existing[strings.ToLower(name)]; !ok {
			_ = c.CreateMailbox(name)
		}
	}
}

// DeleteMailboxRecursive deletes a folder and all of its subfolders.
// Subfolders are deleted deepest-first to avoid errors on servers that
// require children to be deleted before their parent.
func (c *Client) DeleteMailboxRecursive(name string) error {
	all, err := c.ListMailboxes()
	if err != nil {
		return err
	}

	// Determine the hierarchy delimiter for this folder.
	delim := "."
	for _, m := range all {
		if m.Name == name && m.Delim != "" {
			delim = m.Delim
			break
		}
	}

	// Collect subfolders and sort deepest-first.
	prefix := name + delim
	var children []string
	for _, m := range all {
		if strings.HasPrefix(m.Name, prefix) {
			children = append(children, m.Name)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return len(children[i]) > len(children[j])
	})

	for _, child := range children {
		if err := c.Client.Delete(child).Wait(); err != nil {
			return err
		}
	}
	return c.Client.Delete(name).Wait()
}

// MoveAllMessages moves every message in mailbox to dest using IMAP MOVE.
func (c *Client) MoveAllMessages(mailbox, dest string) error {
	if err := c.SelectMailbox(mailbox); err != nil {
		return err
	}
	searchData, err := c.Client.Search(&imap.SearchCriteria{}, &imap.SearchOptions{ReturnAll: true}).Wait()
	if err != nil {
		return err
	}
	if searchData.All == nil {
		return nil
	}
	_, err = c.Client.Move(searchData.All, dest).Wait()
	return err
}

// EmptyMailbox permanently deletes all messages in the given mailbox
// by flagging them \Deleted and issuing EXPUNGE.
func (c *Client) EmptyMailbox(mailbox string) error {
	if err := c.SelectMailbox(mailbox); err != nil {
		return err
	}

	searchData, err := c.Client.Search(&imap.SearchCriteria{}, &imap.SearchOptions{ReturnAll: true}).Wait()
	if err != nil {
		return err
	}
	if searchData.All == nil {
		return nil
	}

	if err := c.Client.Store(searchData.All, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil).Close(); err != nil {
		return err
	}
	return c.Client.Expunge().Close()
}

// QuotaInfo holds IMAP storage usage and limit converted to bytes.
type QuotaInfo struct {
	UsageBytes int64
	LimitBytes int64
}

// GetQuota fetches the STORAGE quota via GETQUOTAROOT on INBOX.
// Returns nil without error when the server does not support the QUOTA extension.
func (c *Client) GetQuota() (*QuotaInfo, error) {
	data, err := c.Client.GetQuotaRoot("INBOX").Wait()
	if err != nil {
		return nil, nil // quota extension not supported — ignore
	}
	for _, qd := range data {
		if res, ok := qd.Resources[imap.QuotaResourceStorage]; ok {
			// IMAP QUOTA reports storage in 1 KB blocks (RFC 2087), convert to bytes.
			return &QuotaInfo{
				UsageBytes: res.Usage * 1024,
				LimitBytes: res.Limit * 1024,
			}, nil
		}
	}
	return nil, nil
}

// AppendMessage saves a raw RFC822 message to the named mailbox with the given flags.
func (c *Client) AppendMessage(mailbox string, flags []imap.Flag, raw []byte) error {
	cmd := c.Client.Append(mailbox, int64(len(raw)), &imap.AppendOptions{
		Flags: flags,
	})
	if _, err := cmd.Write(raw); err != nil {
		return err
	}
	return cmd.Close()
}
