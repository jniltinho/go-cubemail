// Package handler — CardDAV server (RFC 6352) for address-book sync.
//
// Discovery chain used by clients (Apple Contacts, Thunderbird, DAVx⁵…):
//   1. GET|PROPFIND /.well-known/carddav               → 301 /dav/{user}/
//   2. PROPFIND /dav/{user}/                            → addressbook-home-set
//   3. PROPFIND /dav/{user}/contacts/   Depth:1         → address book list
//   4. REPORT   sync-collection on each address book    → delta since the token
//   5. GET|PUT|DELETE individual .vcf resources         → conditional via ETag
//
// The vCard a client sends is stored byte-for-byte and returned unchanged; the
// flat Contact columns are only an index. Regenerating cards from those columns
// is the classic CardDAV data-loss bug — it silently drops addresses, photos,
// birthdays, extra phone numbers and every X-* extension.
package handler

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go-cubemail/internal/config"
	"go-cubemail/internal/contacts"
	"go-cubemail/internal/dav"
	"go-cubemail/internal/model"
	"go-cubemail/internal/repository"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

// CardDAVHandler serves the CardDAV portion of the /dav namespace.
type CardDAVHandler struct {
	cfg         *config.Config
	db          *gorm.DB
	contactRepo *repository.ContactRepo
	bookRepo    *repository.AddressBookRepo
	sync        *dav.Store
}

// ── OPTIONS / well-known ──────────────────────────────────────────────────

// Options advertises the DAV capabilities and allowed methods.
func (h *CardDAVHandler) Options(c *echo.Context) error {
	c.Response().Header().Set("DAV", davCapabilities)
	c.Response().Header().Set("Allow",
		"OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, PROPPATCH, REPORT, MKCOL")
	c.Response().Header().Set("Content-Length", "0")
	return c.NoContent(http.StatusOK)
}

// WellKnown handles GET|PROPFIND /.well-known/carddav → redirect to the principal.
func (h *CardDAVHandler) WellKnown(c *echo.Context) error {
	user := caldavUsername(c)
	if user == "" {
		return c.NoContent(http.StatusUnauthorized)
	}
	c.Response().Header().Set("Location", principalHref(user))
	return c.NoContent(http.StatusMovedPermanently)
}

// ── PROPFIND ──────────────────────────────────────────────────────────────

// propfindHome answers PROPFIND on the addressbook-home-set.
func (h *CardDAVHandler) propfindHome(c *echo.Context, userID uint, username string, req propfindRequest, depth string) error {
	if _, err := h.bookRepo.EnsureDefault(userID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	home := newPropBag()
	home.setRaw(nsDAV, "resourcetype", "<D:collection/>")
	home.setText(nsDAV, "displayname", "Contacts")
	home.setRaw(nsDAV, "current-user-principal", hrefElement(principalHref(username)))
	home.setRaw(nsDAV, "owner", hrefElement(principalHref(username)))
	home.setRaw(nsCardDAV, "addressbook-home-set", hrefElement(addressBookHomeHref(username)))

	responses := []responseOut{{Href: addressBookHomeHref(username), Propstats: home.render(req)}}

	if depth != "0" {
		books, err := h.bookRepo.List(userID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		for i := range books {
			responses = append(responses, responseOut{
				Href:      addressBookHref(username, books[i].URI),
				Propstats: h.bookPropBag(username, &books[i]).render(req),
			})
		}
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// propfindAddressBook answers PROPFIND on one address book collection.
func (h *CardDAVHandler) propfindAddressBook(c *echo.Context, userID uint, username string, book *model.AddressBook, req propfindRequest, depth string) error {
	responses := []responseOut{{
		Href:      addressBookHref(username, book.URI),
		Propstats: h.bookPropBag(username, book).render(req),
	}}

	if depth != "0" {
		list, err := h.contactRepo.ListByBook(userID, book.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		wantData := req.wantsExplicit(nsCardDAV, "address-data")
		for i := range list {
			responses = append(responses, responseOut{
				Href:      addressObjectHref(username, book.URI, list[i].ResourceURI),
				Propstats: contactPropBag(&list[i], wantData).render(req),
			})
		}
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// propfindAddressObject answers PROPFIND on a single .vcf resource.
func (h *CardDAVHandler) propfindAddressObject(c *echo.Context, username string, book *model.AddressBook, ct *model.Contact, req propfindRequest) error {
	return writeMultistatus(c, davCapabilities, []responseOut{{
		Href:      addressObjectHref(username, book.URI, ct.ResourceURI),
		Propstats: contactPropBag(ct, req.wantsExplicit(nsCardDAV, "address-data")).render(req),
	}}, "")
}

// bookPropBag collects the properties of an address book collection.
func (h *CardDAVHandler) bookPropBag(username string, book *model.AddressBook) *propBag {
	bag := newPropBag()
	bag.setRaw(nsDAV, "resourcetype", "<D:collection/><CR:addressbook/>")
	bag.setText(nsDAV, "displayname", book.DisplayName)
	bag.setText(nsCS, "getctag", dav.CTag(book.SyncToken))
	bag.setText(nsDAV, "sync-token", dav.SyncToken(book.SyncToken))
	bag.setText(nsCardDAV, "addressbook-description", book.Description)
	// RFC 6352 §6.2.3 names the child element address-data-type, not
	// address-data — that one is the REPORT payload element. Clients that check
	// the property strictly ignore it when the name is wrong and may then assume
	// vCard 3.0 only. (CalDAV is different: RFC 4791 does reuse calendar-data.)
	bag.setRaw(nsCardDAV, "supported-address-data",
		`<CR:address-data-type content-type="text/vcard" version="3.0"/>`+
			`<CR:address-data-type content-type="text/vcard" version="4.0"/>`)
	bag.setText(nsCardDAV, "max-resource-size", strconv.Itoa(maxResourceSize))
	bag.setRaw(nsDAV, "current-user-principal", hrefElement(principalHref(username)))
	bag.setRaw(nsDAV, "owner", hrefElement(principalHref(username)))
	bag.setRaw(nsDAV, "supported-report-set", supportedReportSet(true, false))
	bag.setRaw(nsDAV, "current-user-privilege-set", privilegeSet())
	return bag
}

// contactPropBag collects the properties of an address object resource.
func contactPropBag(ct *model.Contact, withData bool) *propBag {
	card := contactVCard(ct)
	bag := newPropBag()
	bag.setText(nsDAV, "getetag", dav.Quote(contactETag(ct)))
	bag.setText(nsDAV, "getcontenttype", "text/vcard; charset=utf-8")
	bag.setText(nsDAV, "getcontentlength", strconv.Itoa(len(card)))
	bag.setText(nsDAV, "getlastmodified", ct.UpdatedAt.UTC().Format(http.TimeFormat))
	bag.setRaw(nsDAV, "resourcetype", "")
	if withData {
		bag.setText(nsCardDAV, "address-data", card)
	}
	return bag
}

// ── REPORT ────────────────────────────────────────────────────────────────

// report dispatches a CardDAV REPORT on an address book collection.
func (h *CardDAVHandler) report(c *echo.Context, userID uint, username string, book *model.AddressBook) error {
	body, tooLarge := readBody(c, davReportBodyLimit)
	if tooLarge {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}
	name, err := reportName(body)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	switch {
	case name.Space == nsDAV && name.Local == "sync-collection":
		return h.reportSyncCollection(c, userID, username, book, body)
	case name.Space == nsCardDAV && name.Local == "addressbook-multiget":
		return h.reportMultiget(c, userID, username, book, body)
	case name.Space == nsCardDAV && name.Local == "addressbook-query":
		return h.reportQuery(c, userID, username, book, body)
	default:
		return c.NoContent(http.StatusBadRequest)
	}
}

// reportSyncCollection answers the RFC 6578 delta report for an address book.
func (h *CardDAVHandler) reportSyncCollection(c *echo.Context, userID uint, username string, book *model.AddressBook, body []byte) error {
	var req syncCollectionRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	propReq := propfindRequest{Props: namesOf(req.Prop)}
	if len(propReq.Props) == 0 {
		propReq.AllProp = true
	}

	since, ok := dav.ParseSyncToken(req.SyncToken)
	if !ok {
		return invalidSyncToken(c)
	}

	var responses []responseOut
	var newToken uint64
	wantData := propReq.wantsExplicit(nsCardDAV, "address-data")

	if since == 0 {
		list, err := h.contactRepo.ListByBook(userID, book.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		newToken, err = h.sync.CurrentRevision(model.CollectionAddressBook, book.ID)
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		if over, resp := limitExceeded(req.Limit, len(list)); over {
			return resp(c)
		}
		for i := range list {
			responses = append(responses, responseOut{
				Href:      addressObjectHref(username, book.URI, list[i].ResourceURI),
				Propstats: contactPropBag(&list[i], wantData).render(propReq),
			})
		}
	} else {
		changes, current, err := h.sync.ChangesSince(model.CollectionAddressBook, book.ID, since)
		if errors.Is(err, dav.ErrInvalidSyncToken) {
			return invalidSyncToken(c)
		}
		if err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		newToken = current
		if over, resp := limitExceeded(req.Limit, len(changes)); over {
			return resp(c)
		}
		for _, ch := range changes {
			href := addressObjectHref(username, book.URI, ch.URI)
			if ch.Deleted {
				responses = append(responses, notFoundResponse(href))
				continue
			}
			ct, err := h.contactRepo.GetByResourceURI(userID, book.ID, ch.URI)
			if err != nil {
				responses = append(responses, notFoundResponse(href))
				continue
			}
			responses = append(responses, responseOut{
				Href:      href,
				Propstats: contactPropBag(ct, wantData).render(propReq),
			})
		}
	}

	trailer := "<D:sync-token>" + escapeXML(dav.SyncToken(newToken)) + "</D:sync-token>"
	return writeMultistatus(c, davCapabilities, responses, trailer)
}

// reportMultiget returns the address objects named by the client's href list.
func (h *CardDAVHandler) reportMultiget(c *echo.Context, userID uint, username string, book *model.AddressBook, body []byte) error {
	var req multigetRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	propReq := propfindRequest{Props: namesOf(req.Prop)}
	if len(propReq.Props) == 0 {
		propReq.AllProp = true
	}
	wantData := propReq.wantsExplicit(nsCardDAV, "address-data")

	responses := make([]responseOut, 0, len(req.Hrefs))
	for _, href := range req.Hrefs {
		uri := dav.ResourceURIFromHref(href)
		if uri == "" {
			responses = append(responses, notFoundResponse(href))
			continue
		}
		ct, err := h.contactRepo.GetByResourceURI(userID, book.ID, uri)
		if err != nil {
			responses = append(responses, notFoundResponse(href))
			continue
		}
		responses = append(responses, responseOut{
			Href:      addressObjectHref(username, book.URI, ct.ResourceURI),
			Propstats: contactPropBag(ct, wantData).render(propReq),
		})
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// reportQuery answers an addressbook-query, applying the prop-filter text
// matches the client asked for.
func (h *CardDAVHandler) reportQuery(c *echo.Context, userID uint, username string, book *model.AddressBook, body []byte) error {
	var req addressbookQueryRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	propReq := propfindRequest{Props: namesOf(req.Prop)}
	if len(propReq.Props) == 0 {
		propReq.AllProp = true
	}
	wantData := propReq.wantsExplicit(nsCardDAV, "address-data")

	list, err := h.contactRepo.ListByBook(userID, book.ID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	anyOf := strings.EqualFold(strings.TrimSpace(req.Filter.Test), "anyof")
	responses := make([]responseOut, 0, len(list))
	for i := range list {
		if !contactMatchesFilter(&list[i], req, anyOf) {
			continue
		}
		responses = append(responses, responseOut{
			Href:      addressObjectHref(username, book.URI, list[i].ResourceURI),
			Propstats: contactPropBag(&list[i], wantData).render(propReq),
		})
		if req.Limit != nil && req.Limit.NResults > 0 && len(responses) >= req.Limit.NResults {
			break
		}
	}
	return writeMultistatus(c, davCapabilities, responses, "")
}

// contactMatchesFilter evaluates the prop-filter text matches of a query.
// An empty filter matches everything, which is what a plain listing sends.
func contactMatchesFilter(ct *model.Contact, req addressbookQueryRequest, anyOf bool) bool {
	if len(req.Filter.Props) == 0 {
		return true
	}
	matchedAny := false
	for _, f := range req.Filter.Props {
		needle := strings.TrimSpace(f.TextMatch)
		if needle == "" {
			continue
		}
		hay := contactField(ct, strings.ToUpper(f.Name))
		hit := strings.Contains(strings.ToLower(hay), strings.ToLower(needle))
		if hit {
			matchedAny = true
		} else if !anyOf {
			return false
		}
	}
	if anyOf {
		return matchedAny
	}
	return true
}

// contactField maps a vCard property name onto the indexed column that mirrors it.
func contactField(ct *model.Contact, name string) string {
	switch name {
	case "FN", "N":
		return contacts.DisplayName(*ct)
	case "EMAIL":
		return ct.Email
	case "TEL":
		return ct.Phone
	case "ORG":
		return ct.Company
	case "TITLE":
		return ct.Title
	case "NOTE":
		return ct.Notes
	case "UID":
		return ct.UID
	default:
		// Unknown property: fall back to the raw card so the filter still works.
		return ct.VCardContent
	}
}

// ── GET / PUT / DELETE on address objects ─────────────────────────────────

// getAddressObject writes a .vcf resource.
func (h *CardDAVHandler) getAddressObject(c *echo.Context, ct *model.Contact, body bool) error {
	card := contactVCard(ct)
	c.Response().Header().Set("ETag", dav.Quote(contactETag(ct)))
	c.Response().Header().Set("Last-Modified", ct.UpdatedAt.UTC().Format(http.TimeFormat))
	if !body {
		c.Response().Header().Set("Content-Length", strconv.Itoa(len(card)))
		return c.NoContent(http.StatusOK)
	}
	return c.Blob(http.StatusOK, "text/vcard; charset=utf-8", []byte(card))
}

// putAddressObject creates or replaces a .vcf resource, storing the card exactly
// as the client sent it.
func (h *CardDAVHandler) putAddressObject(c *echo.Context, userID uint, book *model.AddressBook, resource string) error {
	if ct := c.Request().Header.Get("Content-Type"); ct != "" &&
		!strings.HasPrefix(strings.ToLower(ct), "text/vcard") &&
		!strings.HasPrefix(strings.ToLower(ct), "text/x-vcard") {
		return c.NoContent(http.StatusUnsupportedMediaType)
	}

	body, tooLarge := readBody(c, maxResourceSize)
	if tooLarge {
		return writeTooLarge(c, nsCardDAV)
	}
	if len(body) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}

	existing, err := h.contactRepo.GetByResourceURI(userID, book.ID, resource)
	exists := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return c.NoContent(http.StatusInternalServerError)
	}

	currentETag := ""
	if exists {
		currentETag = contactETag(existing)
	}
	if err := dav.CheckPreconditions(c.Request().Header, exists, currentETag); err != nil {
		return c.NoContent(http.StatusPreconditionFailed)
	}

	parsed, err := contacts.Parse(string(body))
	if err != nil {
		return writeDAVError(c, http.StatusForbidden,
			`<CR:valid-address-data xmlns:CR="urn:ietf:params:xml:ns:carddav"/>`)
	}

	if !exists {
		ct := *parsed
		ct.UserID = userID
		ct.AddressBookID = book.ID
		ct.ResourceURI = resource
		ct.VCardContent = string(body)
		if err := h.contactRepo.SaveRaw(&ct); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
		c.Response().Header().Set("ETag", dav.Quote(ct.ETag))
		return c.NoContent(http.StatusCreated)
	}

	existing.FirstName = parsed.FirstName
	existing.LastName = parsed.LastName
	existing.Email = parsed.Email
	existing.Phone = parsed.Phone
	existing.Company = parsed.Company
	existing.Title = parsed.Title
	existing.Notes = parsed.Notes
	if parsed.UID != "" {
		existing.UID = parsed.UID
	}
	existing.VCardContent = string(body)
	if err := h.contactRepo.SaveRaw(existing); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	c.Response().Header().Set("ETag", dav.Quote(existing.ETag))
	return c.NoContent(http.StatusNoContent)
}

// deleteAddressObject removes a .vcf resource, honouring If-Match.
func (h *CardDAVHandler) deleteAddressObject(c *echo.Context, userID uint, book *model.AddressBook, resource string) error {
	ct, err := h.contactRepo.GetByResourceURI(userID, book.ID, resource)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	if err := dav.CheckPreconditions(c.Request().Header, true, contactETag(ct)); err != nil {
		return c.NoContent(http.StatusPreconditionFailed)
	}
	if err := h.contactRepo.Delete(userID, ct.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Collection management ─────────────────────────────────────────────────

// mkAddressBook handles MKCOL for a new address book collection.
func (h *CardDAVHandler) mkAddressBook(c *echo.Context, userID uint, uri string) error {
	if _, err := h.bookRepo.GetByURI(userID, uri); err == nil {
		return c.NoContent(http.StatusMethodNotAllowed)
	}
	body, tooLarge := readBody(c, davRequestBodyLimit)
	if tooLarge {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}
	patch := parseProppatch(body)

	book := model.AddressBook{
		UserID:      userID,
		URI:         uri,
		DisplayName: uri,
		SyncToken:   1,
	}
	for _, p := range patch.Set {
		applyBookProperty(&book, p)
	}
	if err := h.bookRepo.Create(&book); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusCreated)
}

// propPatchAddressBook applies a PROPPATCH to an address book collection.
func (h *CardDAVHandler) propPatchAddressBook(c *echo.Context, username string, book *model.AddressBook) error {
	body, tooLarge := readBody(c, davRequestBodyLimit)
	if tooLarge {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}
	patch := parseProppatch(body)

	var okProps, failProps []string
	for _, p := range patch.Set {
		if applyBookProperty(book, p) {
			okProps = append(okProps, emptyElement(p.Name))
		} else {
			failProps = append(failProps, emptyElement(p.Name))
		}
	}
	for _, n := range patch.Remove {
		failProps = append(failProps, emptyElement(n))
	}
	if len(okProps) > 0 {
		if err := h.bookRepo.Update(book); err != nil {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	var stats []propstatOut
	if len(okProps) > 0 {
		stats = append(stats, propstatOut{Status: statusOK, Body: strings.Join(okProps, "")})
	}
	if len(failProps) > 0 {
		stats = append(stats, propstatOut{Status: statusForbidden, Body: strings.Join(failProps, "")})
	}
	return writeMultistatus(c, davCapabilities, []responseOut{
		{Href: addressBookHref(username, book.URI), Propstats: stats},
	}, "")
}

// applyBookProperty maps a settable DAV property onto the address book row.
func applyBookProperty(book *model.AddressBook, p proppatchProp) bool {
	switch {
	case p.Name.Space == nsDAV && p.Name.Local == "displayname":
		book.DisplayName = p.Value
	case p.Name.Space == nsCardDAV && p.Name.Local == "addressbook-description":
		book.Description = p.Value
	default:
		return false
	}
	return true
}

// deleteAddressBook removes a whole address book collection.
func (h *CardDAVHandler) deleteAddressBook(c *echo.Context, userID uint, book *model.AddressBook) error {
	if book.IsDefault {
		return c.NoContent(http.StatusForbidden)
	}
	if err := h.bookRepo.Delete(userID, book.ID); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// ── Shared helpers ────────────────────────────────────────────────────────

// contactVCard returns the stored card, generating one only for rows created
// before blobs were kept.
func contactVCard(ct *model.Contact) string {
	if ct.VCardContent != "" {
		return ct.VCardContent
	}
	return contacts.Build(*ct)
}

// contactETag returns the stored entity tag, deriving one when absent.
func contactETag(ct *model.Contact) string {
	if ct.ETag != "" {
		return ct.ETag
	}
	return dav.ComputeETag([]byte(contactVCard(ct)))
}
