package handler

import (
	"net/http"
	"strconv"

	"go-cubemail/internal/session"

	"github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
)

// Flag sets or clears the "seen" or "flagged" IMAP flag on a message.
// The request must include form fields "flag" ("seen"|"flagged") and "value" ("1"|"0").
func (h *MessageHandler) Flag(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	flag := c.FormValue("flag")
	value := c.FormValue("value") == "1"

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	imapUID := imap.UID(uid)
	switch flag {
	case "seen":
		if value {
			err = conn.MarkSeen(imapUID)
		} else {
			err = conn.MarkUnseen(imapUID)
		}
	case "flagged":
		err = conn.MarkFlagged(imapUID, value)
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Move moves a message to the folder specified in the "dest" form field.
func (h *MessageHandler) Move(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}
	dest := c.FormValue("dest")

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}
	if err := conn.MoveMessage(imap.UID(uid), dest); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Delete moves a message to the Trash folder.
// If the message is already in Trash, it is permanently deleted instead.
func (h *MessageHandler) Delete(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	uid, err := strconv.ParseUint(c.Param("uid"), 10, 32)
	if err != nil {
		return echo.ErrBadRequest
	}

	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SelectMailbox(mailbox); err != nil {
		return err
	}

	trashFolder := h.findTrashFolder(conn)

	if mailbox == trashFolder {
		err = conn.DeleteMessage(imap.UID(uid))
	} else {
		err = conn.MoveMessage(imap.UID(uid), trashFolder)
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// EmptyTrash permanently deletes all messages in the Trash folder,
// or moves all messages from the given mailbox to Trash if it is not the Trash folder.
func (h *MessageHandler) EmptyTrash(c *echo.Context) error {
	mailbox := c.Param("mailbox")
	s := c.Get("imap_session").(*session.IMAPSession)
	conn, err := h.imapConn(s)
	if err != nil {
		return err
	}
	defer conn.Close()

	trashFolder := h.findTrashFolder(conn)

	if mailbox == trashFolder {
		if err := conn.EmptyMailbox(mailbox); err != nil {
			return err
		}
	} else {
		if err := conn.MoveAllMessages(mailbox, trashFolder); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
