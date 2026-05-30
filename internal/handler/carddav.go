// Package handler — CardDAV server (RFC 6352) for address-book sync.
//
// Discovery chain used by clients (Apple Contacts, Thunderbird, etc.):
//   1. GET /.well-known/carddav                    → 301 /dav/{user}/
//   2. PROPFIND /dav/{user}/                        → addressbook-home-set
//   3. PROPFIND /dav/{user}/contacts/   Depth:1     → address-book collection
//   4. PROPFIND /dav/{user}/contacts/default/ Depth:1 → vCard list with getetag
//   5. REPORT   /dav/{user}/contacts/default/       → addressbook-query / multiget
//   6. GET|PUT|DELETE individual .vcf resources
//
// Auth: same CalDAVAuth Basic Auth middleware (sets caldav_user_id/caldav_username).
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-cubemail/internal/config"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CardDAVHandler handles all CardDAV HTTP methods.
type CardDAVHandler struct {
	cfg         *config.Config
	db          *gorm.DB
	contactRepo *repository.ContactRepo
}

// ── OPTIONS ───────────────────────────────────────────────────────────────

// Options returns CardDAV capability headers.
func (h *CardDAVHandler) Options(c *echo.Context) error {
	c.Response().Header().Set("DAV", "1, 2, addressbook")
	c.Response().Header().Set("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, REPORT")
	return c.NoContent(http.StatusOK)
}

// ── Well-known ────────────────────────────────────────────────────────────

// WellKnown handles GET|PROPFIND /.well-known/carddav → redirect to principal.
func (h *CardDAVHandler) WellKnown(c *echo.Context) error {
	user := caldavUsername(c)
	if user == "" {
		return c.NoContent(http.StatusUnauthorized)
	}
	target := fmt.Sprintf("%s/dav/%s/", strings.TrimRight(h.cfg.Server.BaseURL, "/"), user)
	c.Response().Header().Set("Location", target)
	return c.NoContent(http.StatusMovedPermanently)
}

// ── PROPFIND ──────────────────────────────────────────────────────────────

// PropFind dispatches PROPFIND based on URL depth.
func (h *CardDAVHandler) PropFind(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	username := caldavUsername(c)
	depth := c.Request().Header.Get("Depth")
	if depth == "" {
		depth = "1"
	}
	abSlug := c.Param("ab")
	uidParam := strings.TrimSuffix(c.Param("uid"), ".vcf")

	switch {
	case uidParam != "":
		return h.propfindContact(c, userID, username, uidParam)
	case abSlug != "":
		return h.propfindAddressBook(c, userID, username, abSlug, depth)
	default:
		return h.propfindHome(c, userID, username, depth)
	}
}

// propfindHome returns the address-book home collection.
func (h *CardDAVHandler) propfindHome(c *echo.Context, userID uint, username, depth string) error {
	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	homeURL := fmt.Sprintf("%s/dav/%s/contacts/", base, username)

	responses := []propResponse{
		{
			Href: homeURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					rawXML(`<D:resourcetype><D:collection/></D:resourcetype>`),
					rawXML(`<D:displayname>Contacts</D:displayname>`),
					rawXML(`<D:current-user-principal><D:href>` + fmt.Sprintf("%s/dav/%s/", base, username) + `</D:href></D:current-user-principal>`),
					rawXML(`<CR:addressbook-home-set xmlns:CR="urn:ietf:params:xml:ns:carddav"><D:href>` + homeURL + `</D:href></CR:addressbook-home-set>`),
				},
			}},
		},
	}

	if depth == "1" {
		abURL := fmt.Sprintf("%s/dav/%s/contacts/default/", base, username)
		ctag := h.abCTag(userID)
		responses = append(responses, propResponse{
			Href: abURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					rawXML(`<D:resourcetype><D:collection/><CR:addressbook xmlns:CR="urn:ietf:params:xml:ns:carddav"/></D:resourcetype>`),
					rawXML(`<D:displayname>Default</D:displayname>`),
					rawXML(`<CS:getctag xmlns:CS="http://calendarserver.org/ns/">` + ctag + `</CS:getctag>`),
					rawXML(`<D:sync-token>` + ctag + `</D:sync-token>`),
				},
			}},
		})
	}

	return writeCardDAVPropfind(c, cdMultistatus(responses))
}

// propfindAddressBook returns the address-book collection with optional vCard listing.
func (h *CardDAVHandler) propfindAddressBook(c *echo.Context, userID uint, username, abSlug, depth string) error {
	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	abURL := fmt.Sprintf("%s/dav/%s/contacts/%s/", base, username, abSlug)
	ctag := h.abCTag(userID)

	responses := []propResponse{
		{
			Href: abURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					rawXML(`<D:resourcetype><D:collection/><CR:addressbook xmlns:CR="urn:ietf:params:xml:ns:carddav"/></D:resourcetype>`),
					rawXML(`<D:displayname>` + xmlEsc(abSlug) + `</D:displayname>`),
					rawXML(`<CS:getctag xmlns:CS="http://calendarserver.org/ns/">` + ctag + `</CS:getctag>`),
					rawXML(`<D:sync-token>` + ctag + `</D:sync-token>`),
				},
			}},
		},
	}

	if depth == "1" {
		contacts, err := h.contactRepo.List(userID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		for _, ct := range contacts {
			uid := contactUID(ct)
			vcfURL := fmt.Sprintf("%s/dav/%s/contacts/%s/%s.vcf", base, username, abSlug, uid)
			etag := contactETag(ct)
			responses = append(responses, propResponse{
				Href: vcfURL,
				Props: []propstat{{
					Status: "HTTP/1.1 200 OK",
					Props: []xml.TokenReader{
						rawXML(`<D:getetag>"` + etag + `"</D:getetag>`),
						rawXML(`<D:getcontenttype>text/vcard; charset=utf-8</D:getcontenttype>`),
						rawXML(`<D:resourcetype/>`),
					},
				}},
			})
		}
	}

	return writeCardDAVPropfind(c, cdMultistatus(responses))
}

// propfindContact returns properties for a single vCard resource.
func (h *CardDAVHandler) propfindContact(c *echo.Context, userID uint, username, uid string) error {
	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	ct, err := h.contactRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	abSlug := "default"
	vcfURL := fmt.Sprintf("%s/dav/%s/contacts/%s/%s.vcf", base, username, abSlug, uid)
	etag := contactETag(*ct)
	return writeCardDAVPropfind(c, cdMultistatus([]propResponse{{
		Href: vcfURL,
		Props: []propstat{{
			Status: "HTTP/1.1 200 OK",
			Props: []xml.TokenReader{
				rawXML(`<D:getetag>"` + etag + `"</D:getetag>`),
				rawXML(`<D:getcontenttype>text/vcard; charset=utf-8</D:getcontenttype>`),
				rawXML(`<D:resourcetype/>`),
			},
		}},
	}}))
}

// ── REPORT ────────────────────────────────────────────────────────────────

// Report handles addressbook-query and addressbook-multiget REPORT.
func (h *CardDAVHandler) Report(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	username := caldavUsername(c)
	abSlug := c.Param("ab")

	body, _ := io.ReadAll(io.LimitReader(c.Request().Body, 256*1024))
	bodyStr := string(body)

	if strings.Contains(bodyStr, "addressbook-multiget") {
		return h.reportMultiget(c, userID, username, abSlug, bodyStr)
	}
	return h.reportQuery(c, userID, username, abSlug, bodyStr)
}

func (h *CardDAVHandler) reportQuery(c *echo.Context, userID uint, username, abSlug, body string) error {
	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	contacts, err := h.contactRepo.List(userID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	wantData := strings.Contains(body, "address-data") || strings.Contains(body, "addressbook-data")
	responses := buildContactResponses(base, username, abSlug, contacts, wantData)
	return writeCardDAVPropfind(c, cdMultistatus(responses))
}

func (h *CardDAVHandler) reportMultiget(c *echo.Context, userID uint, username, abSlug, body string) error {
	base := strings.TrimRight(h.cfg.Server.BaseURL, "/")
	hrefs := parseHrefs(body)
	var responses []propResponse
	for _, href := range hrefs {
		uid := hrefToUID(href)
		uid = strings.TrimSuffix(uid, ".vcf")
		if uid == "" {
			continue
		}
		ct, err := h.contactRepo.GetByUID(userID, uid)
		if err != nil {
			responses = append(responses, propResponse{
				Href:  href,
				Props: []propstat{{Status: "HTTP/1.1 404 Not Found"}},
			})
			continue
		}
		vcf := buildVCard(*ct)
		vcfURL := fmt.Sprintf("%s/dav/%s/contacts/%s/%s.vcf", base, username, abSlug, uid)
		etag := contactETag(*ct)
		responses = append(responses, propResponse{
			Href: vcfURL,
			Props: []propstat{{
				Status: "HTTP/1.1 200 OK",
				Props: []xml.TokenReader{
					rawXML(`<D:getetag>"` + etag + `"</D:getetag>`),
					rawXML(`<CR:address-data xmlns:CR="urn:ietf:params:xml:ns:carddav">` + xmlEsc(vcf) + `</CR:address-data>`),
				},
			}},
		})
	}
	return writeCardDAVPropfind(c, cdMultistatus(responses))
}

// ── GET / PUT / DELETE ────────────────────────────────────────────────────

// GetContact handles GET /dav/{user}/contacts/{ab}/{uid}.vcf.
func (h *CardDAVHandler) GetContact(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".vcf")
	ct, err := h.contactRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	vcf := buildVCard(*ct)
	etag := contactETag(*ct)
	c.Response().Header().Set("ETag", `"`+etag+`"`)
	c.Response().Header().Set("Content-Type", "text/vcard; charset=utf-8")
	return c.Blob(http.StatusOK, "text/vcard; charset=utf-8", []byte(vcf))
}

// PutContact handles PUT /dav/{user}/contacts/{ab}/{uid}.vcf — create or update.
func (h *CardDAVHandler) PutContact(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".vcf")

	body, _ := io.ReadAll(io.LimitReader(c.Request().Body, 256*1024))
	parsed := parseVCardSimple(string(body))
	if parsed == nil {
		return c.NoContent(http.StatusBadRequest)
	}
	parsed.UID = uid

	existing, err := h.contactRepo.GetByUID(userID, uid)
	if err == gorm.ErrRecordNotFound {
		parsed.UserID = userID
		if err := h.contactRepo.Create(parsed); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		c.Response().Header().Set("ETag", `"`+contactETag(*parsed)+`"`)
		return c.NoContent(http.StatusCreated)
	}
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	existing.FirstName = parsed.FirstName
	existing.LastName = parsed.LastName
	existing.Email = parsed.Email
	existing.Phone = parsed.Phone
	existing.Company = parsed.Company
	existing.Title = parsed.Title
	existing.Notes = parsed.Notes
	if err := h.contactRepo.Update(existing); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	c.Response().Header().Set("ETag", `"`+contactETag(*existing)+`"`)
	return c.NoContent(http.StatusNoContent)
}

// DeleteContact handles DELETE /dav/{user}/contacts/{ab}/{uid}.vcf.
func (h *CardDAVHandler) DeleteContact(c *echo.Context) error {
	userID, ok := caldavUserID(c)
	if !ok {
		return c.NoContent(http.StatusUnauthorized)
	}
	uid := strings.TrimSuffix(c.Param("uid"), ".vcf")
	ct, err := h.contactRepo.GetByUID(userID, uid)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	if err := h.contactRepo.Delete(userID, ct.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── vCard helpers ─────────────────────────────────────────────────────────

// buildVCard serialises a Contact to a vCard 3.0 string.
func buildVCard(c model.Contact) string {
	uid := contactUID(c)
	var sb strings.Builder
	sb.WriteString("BEGIN:VCARD\r\nVERSION:3.0\r\n")
	sb.WriteString("UID:" + uid + "\r\n")
	sb.WriteString("FN:" + vcfEsc(strings.TrimSpace(c.FirstName+" "+c.LastName)) + "\r\n")
	sb.WriteString("N:" + vcfEsc(c.LastName) + ";" + vcfEsc(c.FirstName) + ";;;\r\n")
	if c.Email != "" {
		sb.WriteString("EMAIL;TYPE=INTERNET:" + vcfEsc(c.Email) + "\r\n")
	}
	if c.Phone != "" {
		sb.WriteString("TEL;TYPE=CELL:" + vcfEsc(c.Phone) + "\r\n")
	}
	if c.Company != "" {
		sb.WriteString("ORG:" + vcfEsc(c.Company) + "\r\n")
	}
	if c.Title != "" {
		sb.WriteString("TITLE:" + vcfEsc(c.Title) + "\r\n")
	}
	if c.Notes != "" {
		sb.WriteString("NOTE:" + vcfEsc(c.Notes) + "\r\n")
	}
	sb.WriteString("REV:" + c.UpdatedAt.UTC().Format("20060102T150405Z") + "\r\n")
	sb.WriteString("END:VCARD\r\n")
	return sb.String()
}

// parseVCardSimple parses a minimal vCard 3.0/4.0 into a Contact struct.
func parseVCardSimple(raw string) *model.Contact {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	c := &model.Contact{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "BEGIN:") || strings.HasPrefix(strings.ToUpper(line), "END:") {
			continue
		}
		prop, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		propUpper := strings.ToUpper(strings.SplitN(prop, ";", 2)[0])
		val = vcfUnesc(strings.TrimSpace(val))
		switch propUpper {
		case "FN":
			// FN is handled below via N
		case "N":
			parts := strings.SplitN(val, ";", 5)
			if len(parts) >= 1 {
				c.LastName = parts[0]
			}
			if len(parts) >= 2 {
				c.FirstName = parts[1]
			}
			if c.FirstName == "" && c.LastName == "" {
				// Use FN as full name
				c.FirstName = val
			}
		case "EMAIL":
			if c.Email == "" {
				c.Email = val
			}
		case "TEL":
			c.Phone = val
		case "ORG":
			c.Company = strings.SplitN(val, ";", 2)[0]
		case "TITLE":
			c.Title = val
		case "NOTE":
			c.Notes = val
		case "UID":
			c.UID = val
		}
	}
	if c.Email == "" && c.FirstName == "" && c.LastName == "" {
		return nil
	}
	return c
}

func vcfEsc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func vcfUnesc(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\N`, "\n")
	s = strings.ReplaceAll(s, `\;`, ";")
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// contactUID returns a stable UID for a contact (stored UID or generated from ID).
func contactUID(c model.Contact) string {
	if c.UID != "" {
		return c.UID
	}
	return fmt.Sprintf("contact-%d@go-cubemail", c.ID)
}

// contactETag produces a stable ETag for a contact.
func contactETag(c model.Contact) string {
	return fmt.Sprintf("%d", c.UpdatedAt.Unix())
}

// newContactUID generates a random UID for a new contact.
func newContactUID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) + "@go-cubemail"
}

// abCTag returns a change token for the address book based on latest contact update.
func (h *CardDAVHandler) abCTag(userID uint) string {
	contacts, err := h.contactRepo.List(userID)
	if err != nil || len(contacts) == 0 {
		return "0"
	}
	var latest int64
	for _, c := range contacts {
		if t := c.UpdatedAt.Unix(); t > latest {
			latest = t
		}
	}
	return fmt.Sprintf("%d", latest)
}

func buildContactResponses(base, username, abSlug string, contacts []model.Contact, wantData bool) []propResponse {
	out := make([]propResponse, 0, len(contacts))
	for _, ct := range contacts {
		uid := contactUID(ct)
		vcfURL := fmt.Sprintf("%s/dav/%s/contacts/%s/%s.vcf", base, username, abSlug, uid)
		etag := contactETag(ct)
		props := []xml.TokenReader{
			rawXML(`<D:getetag>"` + etag + `"</D:getetag>`),
			rawXML(`<D:getcontenttype>text/vcard; charset=utf-8</D:getcontenttype>`),
		}
		if wantData {
			vcf := buildVCard(ct)
			props = append(props, rawXML(`<CR:address-data xmlns:CR="urn:ietf:params:xml:ns:carddav">`+xmlEsc(vcf)+`</CR:address-data>`))
		}
		out = append(out, propResponse{
			Href:  vcfURL,
			Props: []propstat{{Status: "HTTP/1.1 200 OK", Props: props}},
		})
	}
	return out
}

// ── XML helpers (CardDAV-specific) ────────────────────────────────────────

// cdMultistatus produces a CardDAV multistatus XML string.
func cdMultistatus(responses []propResponse) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<D:multistatus xmlns:D="DAV:" xmlns:CR="urn:ietf:params:xml:ns:carddav" xmlns:CS="http://calendarserver.org/ns/">`)
	for _, r := range responses {
		sb.WriteString(`<D:response>`)
		sb.WriteString(`<D:href>` + xmlEsc(r.Href) + `</D:href>`)
		for _, ps := range r.Props {
			sb.WriteString(`<D:propstat><D:prop>`)
			for _, p := range ps.Props {
				if rp, ok := p.(rawXML); ok {
					sb.WriteString(string(rp))
				}
			}
			sb.WriteString(`</D:prop>`)
			sb.WriteString(`<D:status>` + ps.Status + `</D:status>`)
			sb.WriteString(`</D:propstat>`)
		}
		sb.WriteString(`</D:response>`)
	}
	sb.WriteString(`</D:multistatus>`)
	return sb.String()
}

func writeCardDAVPropfind(c *echo.Context, body string) error {
	c.Response().Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.Response().Header().Set("DAV", "1, 2, addressbook")
	return c.Blob(http.StatusMultiStatus, "application/xml; charset=utf-8", []byte(body))
}

// ensure newContactUID and time are used
var _ = newContactUID
var _ = time.Now
