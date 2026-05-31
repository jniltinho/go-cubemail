package handler

import (
	"net/http"
	"strconv"

	"go-cubemail/internal/session"

	"github.com/emersion/go-imap/v2"
	"github.com/labstack/echo/v5"
)

// Flag godoc
// @Summary      Set or clear message flag
// @Description  Sets or clears the "seen" or "flagged" IMAP flag on a message.
// @Tags         mail
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        mailbox  path      string  true  "Mailbox folder name"
// @Param        uid      path      int     true  "Message UID"
// @Param        flag     formData  string  true  "Flag name: seen or flagged"
// @Param        value    formData  string  true  "1 to set, 0 to clear"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "invalid uid"
// @Security     CookieAuth
// @Router       /mail/{mailbox}/{uid}/flag [post]
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

// Move godoc
// @Summary      Move message to folder
// @Description  Moves a message to the destination folder specified in the form body.
// @Tags         mail
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        mailbox  path      string  true  "Source mailbox folder name"
// @Param        uid      path      int     true  "Message UID"
// @Param        dest     formData  string  true  "Destination folder name"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "invalid uid"
// @Security     CookieAuth
// @Router       /mail/{mailbox}/{uid}/move [post]
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

// Delete godoc
// @Summary      Delete message
// @Description  Moves a message to Trash. If already in Trash, permanently expunges it.
// @Tags         mail
// @Produce      json
// @Param        mailbox  path  string  true  "Mailbox folder name"
// @Param        uid      path  int     true  "Message UID"
// @Success      200  {object}  map[string]string "status ok"
// @Failure      400  {object}  map[string]string "invalid uid"
// @Security     CookieAuth
// @Router       /mail/{mailbox}/{uid} [delete]
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

// EmptyTrash godoc
// @Summary      Empty trash / clear folder
// @Description  Permanently expunges all messages in Trash. If the given mailbox is not Trash, moves all its messages to Trash instead.
// @Tags         mail
// @Produce      json
// @Param        mailbox  path  string  true  "Mailbox folder name"
// @Success      200  {object}  map[string]string "status ok"
// @Security     CookieAuth
// @Router       /mail/{mailbox} [delete]
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
