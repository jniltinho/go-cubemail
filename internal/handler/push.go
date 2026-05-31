package handler

import (
	"encoding/json"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// PushHandler manages Web Push subscriptions and notification delivery.
type PushHandler struct {
	cfg      *config.Config
	db       *gorm.DB
	pushRepo *repository.PushSubscriptionRepo
}

type subscribePayload struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// VAPIDPublicKey godoc
// @Summary      Get VAPID public key
// @Description  Returns the server VAPID public key required for the browser to create a Web Push subscription.
// @Tags         push
// @Produce      json
// @Success      200  {object}  map[string]string "public_key"
// @Failure      503  {object}  map[string]string "push not configured"
// @Security     CookieAuth
// @Router       /push/vapid-public-key [get]
func (h *PushHandler) VAPIDPublicKey(c *echo.Context) error {
	key := h.cfg.Push.VAPIDPublicKey
	if key == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "push not configured"})
	}
	return c.JSON(http.StatusOK, map[string]string{"public_key": key})
}

// Subscribe godoc
// @Summary      Register push subscription
// @Description  Stores the browser PushSubscription (endpoint + keys) after the user grants notification permission.
// @Tags         push
// @Accept       json
// @Produce      json
// @Param        body  body      subscribePayload  true  "Push subscription payload"
// @Success      200  {object}  map[string]string "status subscribed"
// @Failure      400  {object}  map[string]string "invalid body or missing fields"
// @Failure      500  {object}  map[string]string "database error"
// @Security     CookieAuth
// @Router       /push/subscribe [post]
func (h *PushHandler) Subscribe(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var payload subscribePayload
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if payload.Endpoint == "" || payload.Keys.P256DH == "" || payload.Keys.Auth == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "endpoint, p256dh, auth required"})
	}
	sub := model.PushSubscription{
		UserID:   userID,
		Endpoint: payload.Endpoint,
		P256DH:   payload.Keys.P256DH,
		Auth:     payload.Keys.Auth,
	}
	if err := h.pushRepo.Upsert(&sub); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "subscribed"})
}

// Unsubscribe godoc
// @Summary      Remove push subscription
// @Description  Deletes the push subscription for the given endpoint when the user revokes notification permission.
// @Tags         push
// @Accept       json
// @Produce      json
// @Param        body  body      object            true  "endpoint to remove"
// @Success      200  {object}  map[string]string "status unsubscribed"
// @Failure      400  {object}  map[string]string "invalid body"
// @Security     CookieAuth
// @Router       /push/unsubscribe [delete]
func (h *PushHandler) Unsubscribe(c *echo.Context) error {
	userID, err := getUserID(c, h.db)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user lookup failed"})
	}
	var payload struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	_ = h.pushRepo.Delete(userID, payload.Endpoint)
	return c.JSON(http.StatusOK, map[string]string{"status": "unsubscribed"})
}

// SendNewMailNotification sends a Web Push notification to all subscriptions
// of the given user. Called from the SSE handler when new mail is detected.
func SendNewMailNotification(cfg *config.Config, pushRepo *repository.PushSubscriptionRepo, userID uint, unread uint32) {
	if cfg.Push.VAPIDPublicKey == "" || cfg.Push.VAPIDPrivateKey == "" {
		return
	}
	subs, err := pushRepo.ListForUser(userID)
	if err != nil || len(subs) == 0 {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"title": "New mail",
		"body":  "You have new messages in your inbox.",
		"unread": unread,
	})
	opts := &webpush.Options{
		VAPIDPublicKey:  cfg.Push.VAPIDPublicKey,
		VAPIDPrivateKey: cfg.Push.VAPIDPrivateKey,
		Subscriber:      cfg.Push.VAPIDContact,
		TTL:             60,
	}
	for _, sub := range subs {
		wSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256DH,
				Auth:   sub.Auth,
			},
		}
		resp, err := webpush.SendNotification(payload, wSub, opts)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
		// 404/410 means the subscription expired — remove it.
		if err == nil && resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone) {
			_ = pushRepo.Delete(userID, sub.Endpoint)
		}
	}
}
