package activesync

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"go-cubemail/internal/config"
)

// AutodiscoverHandler serves MS-OXDISCO mobile sync autodiscover responses.
type AutodiscoverHandler struct {
	cfg *config.Config
}

// NewAutodiscoverHandler creates an Autodiscover handler.
func NewAutodiscoverHandler(cfg *config.Config) *AutodiscoverHandler {
	return &AutodiscoverHandler{cfg: cfg}
}

// MobileSync handles POST /autodiscover/autodiscover.xml.
func (h *AutodiscoverHandler) MobileSync(c *echo.Context) error {
	if !h.cfg.ActiveSync.Enabled {
		return c.NoContent(http.StatusNotFound)
	}

	body, _ := io.ReadAll(c.Request().Body)
	email := extractAutodiscoverEmail(body)
	if email == "" {
		email = c.QueryParam("Email")
	}

	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	if base == "" {
		base = "http://" + c.Request().Host
	}
	rootURL := base + "/Microsoft-Server-ActiveSync"

	resp := `<?xml version="1.0" encoding="utf-8"?>` +
		`<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/responseschema/2006">` +
		`<Response xmlns="http://schemas.microsoft.com/exchange/autodiscover/mobilesync/responseschema/2006">` +
		`<Culture>en:us</Culture>` +
		`<User><DisplayName>` + xmlEscape(email) + `</DisplayName>` +
		`<EMailAddress>` + xmlEscape(email) + `</EMailAddress></User>` +
		`<Action><Settings><Setting><SettingType>MobileSync</SettingType>` +
		`<RootUrl>` + xmlEscape(rootURL) + `</RootUrl></Setting></Settings></Action>` +
		`</Response></Autodiscover>`

	c.Response().Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, err := c.Response().Write([]byte(resp))
	return err
}

// extractAutodiscoverEmail parses the EMailAddress field from an Autodiscover POST body.
func extractAutodiscoverEmail(body []byte) string {
	type autoReq struct {
		XMLName xml.Name `xml:"Autodiscover"`
		Request struct {
			EmailAddress string `xml:"EMailAddress"`
		} `xml:"Request"`
	}
	var req autoReq
	if err := xml.Unmarshal(body, &req); err != nil {
		return ""
	}
	return strings.TrimSpace(req.Request.EmailAddress)
}

// xmlEscape escapes a string for safe inclusion in Autodiscover XML responses.
func xmlEscape(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
