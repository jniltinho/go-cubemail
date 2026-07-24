# CalDAV & CardDAV Implementation Guide

> **Project:** go-cubemail-vue
> **Stack:** Go 1.26 + Echo v5 + GORM (SQLite / MariaDB / PostgreSQL) + Vue 3, single binary
> **Status:** Core protocol complete and covered by tests — contact groups, GAL, collected contacts and listing projection are pending (see [§6](#6-remaining-work))
> **Related:** [DAV & Sync Setup](DAV_AND_SYNC_SETUP.md) (client configuration) · [SDD §11.4](SDD.md) · [ActiveSync Implementation](ACTIVESYNC_IMPLEMENTATION.md)

This document unifies and supersedes the two earlier planning drafts
(`PLANO-AGENDA-CONTATOS.md` and `PLANO-AGENDA-CONTATOS_2.md`). It records the
architecture that was built, the reasoning behind the decisions that shaped it,
the mistakes that were made and corrected, and the work that remains.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Design Principles](#2-design-principles)
3. [Data Model](#3-data-model)
4. [Synchronisation Layer](#4-synchronisation-layer)
5. [Protocol Surface](#5-protocol-surface)
6. [Remaining Work](#6-remaining-work)
7. [Testing Strategy](#7-testing-strategy)
8. [Known Pitfalls](#8-known-pitfalls)
9. [File & Directory Map](#9-file--directory-map)
10. [References](#10-references)

---

## 1. Executive Summary

`/dav` serves **CalDAV** (RFC 4791) and **CardDAV** (RFC 6352) to external
clients: Thunderbird, Apple Calendar/Contacts, DAVx⁵, Evolution, GNOME Calendar,
and Outlook through the CalDAV Synchronizer add-in.

The DAV server and the web UI's REST API share the same tables. A change made in
the browser reaches a phone on its next sync, and vice versa, without any
reconciliation code — there is exactly one source of truth.

### Why not `go-webdav`

The earlier drafts proposed building on `emersion/go-webdav` with its
`caldav.Backend` / `carddav.Backend` interfaces. That path was **not** taken.

Those interfaces impose their own data model (`Collection`, `CalendarObject`,
`AddressObject`). The existing `model.Event` and `model.Contact` are already
consumed by the REST API, the Vue frontend and the ActiveSync server, so
adopting them would have meant rewriting three subsystems to gain a protocol
layer that was already largely present. The decision was to keep the shared
model and fix the protocol layer where it was actually wrong.

That trade is worth restating for anyone revisiting it: `go-webdav` would give
better spec coverage per line of code, at the cost of a second data model and a
translation layer between the two. With ActiveSync in the picture the second
model is a liability, not an asset.

### Current state at a glance

| Area | State |
|---|---|
| Collections with stable URIs, provisioned on first access | ✅ |
| Blob fidelity (iCalendar / vCard stored verbatim) | ✅ |
| ETag, CTag, sync-token (RFC 6578) with transactional changelog | ✅ |
| Conditional requests (`If-Match` / `If-None-Match` → 412) | ✅ |
| `sync-collection`, `*-multiget`, `calendar-query`, `addressbook-query` | ✅ |
| `MKCALENDAR`, `MKCOL`, `PROPPATCH`, `PROPFIND`, `REPORT`, `HEAD` | ✅ |
| Basic auth against IMAP with credential cache | ✅ |
| Discovery (`.well-known`, `current-user-principal`, home sets) | ✅ |
| Contact groups (`KIND:group`) | ❌ [§6.2](#62-contact-groups) |
| Global Address List | ❌ [§6.4](#64-gal--global-address-list) |
| Collected contacts | ❌ [§6.3](#63-collected-contacts) |
| Listing projection (blobs are loaded even when not requested) | ❌ [§6.5](#65-listing-projection) |
| `EXDATE` / `RECURRENCE-ID` in the web UI's expansion | ❌ [§6.6](#66-recurrence-completeness) |
| Validation against real clients | ❌ [§7.2](#72-real-client-matrix) |

---

## 2. Design Principles

These four rules explain most of the code. Violating any of them reintroduces a
class of bug that is painful to diagnose because it only appears after days in
production.

### 2.1 The blob is the truth

Store the iCalendar / vCard payload **exactly as the client sent it**, byte for
byte. The exploded columns (`summary`, `start_at`, `email`…) exist only as a
query index. Never rebuild the payload from them when answering a client.

Rebuilding drops `VALARM`, `VTIMEZONE`, `X-*` extensions, postal addresses,
photos, birthdays, secondary phone numbers, and the `RECURRENCE-ID` overrides
that share a master's UID inside a single resource.

There is a second, subtler consequence. If the server rewrites what it stores,
the client reads back something different from what it sent, concludes the
server holds a newer version, and writes again. The server rewrites again. This
is an **infinite sync loop**, and it is the most common failure mode of
home-grown CardDAV servers.

The same rule governs vCard versions: iOS and Evolution speak 3.0, DAVx⁵ and
recent Thunderbird speak 4.0, and they can share one collection. Do not convert.
`vcard.ToV4()`-style normalisation is acceptable **in memory** to populate index
columns — never to store or to respond.

Edits from the web UI are the one case where the payload must change. Those
**patch** the stored card in place (`contacts.ApplyToVCard`) instead of
regenerating it, so properties outside the UI's model survive.

### 2.2 Resource identity is the URL, not the UID

The client picks the resource name in its `PUT`. It is not obliged to use
`<uid>.ics`, and Apple Calendar does not. Store the resource name and the UID
separately; DAV identity is `(collection, resource name)`.

A corollary that bit this codebase: **UID cannot be unique**. A recurrence
override is a separate row sharing the master's UID, and two users may
legitimately import the same public calendar. The original schema had
`UNIQUE(events.uid)` and the migration drops it.

### 2.3 Deletions must be observable

A change token derived from surviving rows — `MAX(updated_at)`, say — can never
express "this object is gone". Clients that cannot learn about a deletion keep
ghost entries forever and fall back to full sync on every poll.

This is the entire justification for the `dav_changes` table.

### 2.4 Never leak across users

Every request path carries a user segment. It must be compared against the
authenticated user, and a mismatch answered with **404**, not 403 — a 403
confirms that the other user's collection exists.

Path parsing lives in exactly one place ([`dav_paths.go`](#9-file--directory-map))
because scattered string manipulation is where traversal bugs are born.

---

## 3. Data Model

### 3.1 Collections

Calendars and address books are collections. Both carry the same DAV fields:

```go
// model.Calendar / model.AddressBook (abridged)
URI            string // stable path segment: /dav/{user}/calendars/{URI}/
DisplayName    string // Name on Calendar
Description    string
SyncToken      uint64 // current revision, starts at 1
PrunedRevision uint64 // oldest revision still in the changelog
```

**`URI` is assigned once at creation and never derived from the display name.**
Deriving it means renaming a calendar invalidates every client's bookmark and
sync state, and that two calendars named `Work` and `work!` collide on the same
slug. `repository.Slugify` seeds the value; `freeCollectionURI` resolves
collisions with a numeric suffix.

A `default` calendar and a `default` address book are provisioned lazily on
first access (`EnsureDefault`).

### 3.2 Objects

```go
// model.Event / model.Contact (DAV fields)
ResourceURI  string // "a1b2c3.ics" — the name the client chose
ETag         string // sha256 prefix of the blob, unquoted
SyncRevision uint64 // collection revision of the last write
ICalContent  string // model.Event   — raw iCalendar
VCardContent string // model.Contact — raw vCard
```

Uniqueness is `(collection_id, resource_uri)`. On MariaDB the index is
8 + 1020 = 1028 bytes with `varchar(255)` in `utf8mb4`, comfortably inside the
3072-byte limit of `ROW_FORMAT=DYNAMIC` (the default since MariaDB 10.11).

### 3.3 Changelog

```go
type DAVChange struct {
    ID             uint64
    CollectionKind string // "calendar" | "addressbook"
    CollectionID   uint
    SyncRevision   uint64
    URI            string // resource name, not the UID
    Deleted        bool
    CreatedAt      time.Time
}
// INDEX (collection_kind, collection_id, sync_revision)
```

The covering index matters: some clients poll every minute, and this table is
read on every one of those requests.

### 3.4 Migration notes

`cmd/migrate.go` runs three steps in a fixed order, and the order is load-bearing:

1. `AutoMigrate(&AddressBook{}, &DAVChange{})` — new tables.
2. `database.PrepareDAVSchema(db)` — add the DAV columns and **backfill resource
   names**.
3. Full `AutoMigrate`, then `database.FinishDAVMigration(db)`.

The unique indexes over `(collection_id, resource_uri)` cannot be created while
existing rows all hold the empty default, so the values must exist before
AutoMigrate builds the index — hence step 2.

**GORM's SQLite driver rebuilds a table to alter any column**, copying only the
columns its DDL parser recognised. A hand-edited schema can therefore arrive at
step 3 with the freshly backfilled names blanked out. `FinishDAVMigration`
repeats the backfill and calls `ensureDAVIndexes` for that reason; both passes
are idempotent.

One naming trap: GORM maps the Go field `ETag` to the column `e_tag`, and
`VCardContent` to `v_card_content`. Both carry explicit `gorm:"column:..."`
tags so raw SQL stays readable.

---

## 4. Synchronisation Layer

Three mechanisms, all required, all distinct.

| Mechanism | Scope | Question it answers |
|---|---|---|
| **ETag** | one object | "Has *this resource* changed since I last read it?" |
| **CTag** | one collection | "Is it worth listing this collection at all?" |
| **sync-token** | one collection | "What changed since revision N?" |

### 4.1 ETag

```go
func ComputeETag(data []byte) string {
    sum := sha256.Sum256(data)
    return hex.EncodeToString(sum[:16])
}
```

Stored unquoted; `dav.Quote` adds the quotes that are part of the HTTP header
value. Content-derived, so it changes if and only if the payload changes.

Used for conditional requests, which is what stops two clients from silently
overwriting each other:

- `If-None-Match: *` on `PUT` — create only; **412** if the resource exists.
- `If-Match: "<etag>"` on `PUT`/`DELETE` — **412** if it is gone or changed.

### 4.2 CTag and sync-token

Both derive from the same counter, so any change that advances one advances the
other. The sync-token is an opaque URI
(`http://go-cubemail.local/ns/sync/<rev>`); the CTag is the bare decimal.

### 4.3 Recording a change

Every write bumps the collection revision and appends a changelog row **inside
the same transaction as the object write**:

```go
// internal/dav/sync.go
func Record(tx *gorm.DB, kind string, collID uint, uri string, deleted bool) (uint64, error)
```

The increment uses a SQL expression (`sync_token + 1`) rather than a
read-modify-write, so concurrent writers cannot be handed the same revision.

This lives in the **repositories**, not the handlers. `EventRepo` and
`ContactRepo` are the single write path for the REST API, the DAV server and
ActiveSync alike, so instrumenting them means every writer participates for
free — and a rollback takes the changelog entry with it.

`RecordIfExists` tolerates a missing collection instead of failing the write.
ActiveSync stores VTODO tasks that belong to no calendar; those simply are not
exposed over DAV, and failing their write because there is nothing to
synchronise would be wrong.

### 4.4 Answering `sync-collection`

```
Client sends token → ParseSyncToken → revision N
  N == 0  → initial sync: enumerate the collection
  N  > 0  → SELECT uri, deleted FROM dav_changes
            WHERE collection = ? AND sync_revision > N ORDER BY sync_revision
```

Three details that are easy to get wrong:

- **Initial sync enumerates the collection**, it does not replay the changelog.
  The changelog may have been pruned, or may predate the rows entirely after a
  migration.
- **Collapse per URI.** A resource created, modified twice and then deleted must
  appear once, as deleted.
- **Deleted resources get a bare `<D:status>404</D:status>`** on the response,
  not a propstat.

The new token goes in a `<D:sync-token>` element inside the multistatus.

### 4.5 Retention

A background goroutine prunes entries older than 30 days every 6 hours,
**always keeping the head revision** so a client sitting exactly at it is never
forced to resync. The highest discarded revision is written to the collection's
`PrunedRevision`.

A client presenting a token below that gets **403** with `<D:valid-sync-token/>`,
which is the standard instruction to discard local state and start over.

---

## 5. Protocol Surface

### 5.1 URL layout

```
/dav/                                        service root
/dav/{user}/                                 principal
/dav/{user}/calendars/                       calendar-home-set
/dav/{user}/calendars/{uri}/                 a calendar
/dav/{user}/calendars/{uri}/{resource}       a calendar object
/dav/{user}/contacts/                        addressbook-home-set
/dav/{user}/contacts/{uri}/                  an address book
/dav/{user}/contacts/{uri}/{resource}        an address object
```

**`href` values are emitted as absolute paths, not absolute URLs.** A path is
correct regardless of how the reverse proxy presents the host; a URL built from
a misconfigured `base_url` silently sends clients to the wrong origin, and that
failure is invisible until someone changes the proxy.

### 5.2 Routing in Echo v5

WebDAV uses methods the router does not know — `PROPFIND`, `PROPPATCH`,
`REPORT`, `MKCOL`, `MKCALENDAR` — and clients disagree about trailing slashes.

The earlier drafts proposed bypassing the router with an `e.Pre` middleware
delegating to an `http.Handler`. What is implemented instead: each method is
registered once against a **single catch-all handler**, which parses the path
itself.

```go
for _, m := range davMethods {
    dav.Add(m, "", h.DAV.Handle)
    dav.Add(m, "/", h.DAV.Handle)
    dav.Add(m, "/*", h.DAV.Handle)
}
```

This keeps the Echo middleware chain (auth, logging, recovery) intact, which the
`Pre` bypass would have skipped, and removes the whole class of trailing-slash
and 405 bugs that came from enumerating routes per depth.

### 5.3 Request handling

Request bodies are **parsed with `encoding/xml`**, not matched with
`strings.Contains`. A client asking for three properties gets those three, and
anything unknown comes back in a `404` propstat.

`allprop` deliberately excludes `calendar-data` and `address-data`: those are
payloads, not live properties, and including them would inline every object in
the collection into one PROPFIND response.

Request bodies are size-bounded and **oversized bodies are rejected, not
truncated** — a truncated `PUT` would store a corrupted resource and still
answer `201` with a valid-looking ETag. Limits are 64 KB for property documents,
1 MB for REPORT, 10 MB per resource; exceeding the last returns **413** with the
`max-resource-size` precondition element.

### 5.4 Authentication and discovery

HTTP Basic validated against IMAP, with a **5-minute in-memory cache** keyed by
`HMAC-SHA256(user + "\0" + password)` under a per-process random key. The
password itself is never stored, and failures are never cached.

Without the cache, every request opens a TCP + TLS connection and a LOGIN.
Clients poll every 5–15 minutes and issue several requests per cycle, which
turns the mail server into a bottleneck and can trip its own brute-force
protection. Measured effect: **10 authenticated requests → 1 IMAP connection**.

Discovery chain — memorise it, it is what you will debug:

```
user's e-mail address
  → DNS SRV _caldavs._tcp.<domain>
  → GET /.well-known/caldav  (301)
  → PROPFIND / with DAV:current-user-principal
  → PROPFIND <principal> with CALDAV:calendar-home-set
  → PROPFIND <home-set> Depth:1  → collection list
  → REPORT sync-collection on each collection
```

DNS records for the domain:

```dns
_caldavs._tcp.example.com.   86400 IN SRV 0 1 443 mail.example.com.
_carddavs._tcp.example.com.  86400 IN SRV 0 1 443 mail.example.com.
_caldavs._tcp.example.com.   86400 IN TXT "path=/dav/"
_carddavs._tcp.example.com.  86400 IN TXT "path=/dav/"
```

Reverse proxy (Caddy passes unknown methods through unchanged): check that
`max_request_body` is not below the resource limit, that `Expect: 100-continue`
works (some Apple clients use it), and that timeouts are generous — an initial
full sync of a large calendar takes a while.

---

## 6. Remaining Work

Ordered by value per unit of effort.

### 6.1 vCard fidelity gaps

**Quoted-printable decoding.** Older clients still send
`N;ENCODING=QUOTED-PRINTABLE;CHARSET=UTF-8:Concei=C3=A7=C3=A3o` in vCard 3.0.
The blob is stored correctly, but `contacts.Parse` does not decode it, so the
index columns and the web UI show the raw escape sequence. Decode in
`Parse` only — never touch the blob. Test with *"José Antônio da Conceição"*.

**`Version` column.** Add `Version string` to `model.Contact`, populated from
the stored card. Needed to decide which convention to emit for contacts created
in the web UI, and to report the collection's predominant version.

Other version differences worth knowing:

| | vCard 3.0 | vCard 4.0 |
|---|---|---|
| Type parameters | `TEL;TYPE=HOME,VOICE:` | `TEL;TYPE="home,voice":` |
| UID | free text | usually `urn:uuid:...` (already normalised in the index) |

### 6.2 Contact groups

Two competing conventions, and supporting only one loses groups in half the
clients:

```
# RFC 6350 (vCard 4.0)              # Apple extension (over vCard 3.0)
KIND:group                          X-ADDRESSBOOKSERVER-KIND:group
MEMBER:urn:uuid:1a2b3c4d-...        X-ADDRESSBOOKSERVER-MEMBER:urn:uuid:1a2b...
```

Round-tripping already works by accident — the blob is preserved, so a group
created in iOS comes back to iOS intact. What is missing is that the server does
not *understand* groups: `contacts.Parse` treats one as an ordinary contact, so
it shows up in the UI as a person.

Plan:

1. On `PUT`, detect a group by `KIND:group` **or** `X-ADDRESSBOOKSERVER-KIND:group`;
   extract members from `MEMBER` **or** `X-ADDRESSBOOKSERVER-MEMBER`.
2. Add `Kind string` to `model.Contact` (`individual` | `group` | `org`).
3. Add a derived membership table, repopulated on every group `PUT` (delete +
   insert inside the same transaction as the revision bump):

   ```go
   type GroupMember struct {
       ID           uint64
       CollectionID uint64 `gorm:"index:idx_grp,priority:1"`
       GroupUID     string `gorm:"index:idx_grp,priority:2"`
       MemberUID    string `gorm:"index"`
   }
   ```

   It is a **cache derived from the blob** — if the two disagree, the blob wins.
4. On `GET`, keep returning the original blob. Each client gets back its own
   convention.
5. **Do not enforce referential integrity.** A client may send the group before
   its members exist; store the `MEMBER` reference anyway and resolve on read.

### 6.3 Collected contacts

Zimbra's behaviour: replying to someone creates a contact automatically. Cheap
to build, high perceived value.

- Dedicated collection `collected/`, provisioned alongside `default/`.
- Hook the SMTP send path (`internal/smtp`); for each `To`/`Cc` recipient,
  dispatch asynchronously. **Never block the send** — if the contact write
  fails, the mail has already gone out.
- Rules that keep the collection from becoming landfill: lowercase before
  comparing; skip addresses already present in **any** of the user's
  collections; skip the user's own aliases; skip `noreply@`, `no-reply@`,
  `bounce@`, `mailer-daemon@`, `postmaster@` and VERP-style local parts; cap the
  collection (≈2000) evicting least-used; keep a use counter and last-contact
  date to rank composer autocomplete.

**LGPD.** Collected contacts are third-party personal data captured without an
explicit action by either the data subject or the user. Minimum: a per-user
preference to disable collection (**off by default** is the defensible
posture), a "clear collected contacts" button, and a mention in the privacy
notice. This will show up in a corporate compliance questionnaire eventually;
better to be born compliant.

### 6.4 GAL — Global Address List

A **read-only** collection at `/dav/{user}/contacts/gal/`, materialised on
demand from the mail domain's account tables (`go-postfixadmin`). Do not
duplicate the rows into `contacts`.

- **Deterministic UID** — UUID v5 over the lowercased address with a fixed
  namespace. If the UID changes between restarts, every client reprocesses the
  whole catalogue.
- ETag from the source row's `updated_at`; collection sync-token from
  `MAX(updated_at)` mapped to a monotonic counter.
- Reject `PUT`/`DELETE`/`PROPPATCH` with **403** and `<D:need-privileges/>`, and
  advertise `current-user-privilege-set` as `DAV:read` only so well-behaved
  clients hide the edit button.

> **Cross-domain isolation is the critical requirement.** The deployment is
> multi-domain: a user of `company-a.com` must never see mailboxes of
> `company-b.com`. Filter the GAL by the authenticated account's domain, always,
> and write a dedicated automated test for it. This is the kind of leak that
> passes code review and becomes a privacy incident between paying customers.

Also consider a per-mailbox `gal_visible` flag — system mailboxes, technical
aliases and service accounts should not appear.

### 6.5 Listing projection

`PROPFIND Depth:1` currently loads full rows, including `vcard_content` and
`i_cal_content`, even when the client asked only for `getetag`. With 500
contacts carrying `PHOTO` data that is hundreds of megabytes per listing
request. A 2 MP portrait becomes ~2.7 MB of base64.

```go
// listing: never select the blob column
db.Model(&model.Contact{}).
    Select("id, address_book_id, resource_uri, uid, etag, sync_revision, " +
           "first_name, last_name, updated_at").
    Where("address_book_id = ?", bookID).
    Find(&rows)
```

Load the blob only when the client actually requested `address-data` /
`calendar-data`. Note `getcontentlength` is computed from the blob today — add
a `Size` column so listings can report it without loading anything.

If the volume still hurts on MariaDB, move the blob to a 1:1 side table
(`address_object_data`). The payload stays intact; the index queries get light.

Applies equally to `EventRepo.ListByCalendar`, which also eager-loads attendees.

### 6.6 Recurrence completeness

The DAV side is correct — the blob round-trips, so a client's own expansion is
authoritative. The gaps are on the **web UI's** side, which reads the index
columns:

- **`EXDATE` and `RDATE` are neither stored nor parsed.** Deleting one
  occurrence of a series in a phone shows correctly on the phone but the web UI
  still renders it.
- **`<C:expand>`** is not implemented in `calendar-query`. Clients expand
  locally, so this is low priority.
- **Range pre-filter is a superset.** `ListByCalendarRange` returns every
  recurring master starting before the window. The refinement is to store, at
  write time, the end of the **last** occurrence in `DTEnd` (`NULL` for infinite
  recurrence, computed with `rrule-go` from `UNTIL`/`COUNT`), turning the filter
  into:

  ```sql
  WHERE collection_id = ?
    AND (dtend IS NULL OR dtend >= ?)   -- window start
    AND dtstart <= ?                    -- window end
  ```

  Still a pre-filter: expand the candidates in memory and discard what falls
  outside, honouring `EXDATE` and `RECURRENCE-ID` overrides.

### 6.7 Frontend

The REST API under `/api/v1` is the frontend's only interface — the browser
never speaks DAV. Both hit the same repositories.

`CalendarPane.vue` is a hand-built month/week/day grid with drag-and-drop that
already consumes the REST API. The earlier drafts proposed replacing it with
`@fullcalendar/vue3`; that is a product decision, not an interoperability
requirement, and it costs a dependency install. Open questions if revisited:
overlap/lane layout, an all-day row, and multi-day events (which currently
render only on their start day).

Two integrations that make this feel like one product rather than three modules:
a `text/calendar` attachment offering **Accept / Tentative / Decline** (partly
present), and composer autocomplete backed by contacts (present).

### 6.8 iTIP / iMIP scheduling

Leave for last — scheduling is the most complex part of CalDAV.

Present: RSVP endpoints and the iMIP `REPLY` mail. Missing: emitting
`method=REQUEST` when an event with `ATTENDEE` is created, and `method=CANCEL`
on deletion. That minimal scope interoperates with Google, Outlook and Zimbra
without implementing RFC 6638 `calendar-auto-schedule`, whose
`schedule-inbox`/`schedule-outbox` collections are a project of their own.

---

## 7. Testing Strategy

### 7.1 Automated tests

`make test` runs the whole suite in well under a second. **No network, no mail
server, no external database**: in-memory SQLite plus go-imap's own in-memory
IMAP server (`imapserver/imapmemserver`) for the auth path.

```bash
make test        # go test ./internal/...
make test-race   # same, with the race detector
```

Scoped to `./internal/...` because the root package embeds `web/dist`, which
does not exist until `make frontend` has run.

| Package | Tests | Covers |
|---|---|---|
| `internal/dav` | 12 | ETag, token round-trip, changelog collapse, retention, rollback |
| `internal/contacts` | 10 | folding, `FN` without `N`, `PREF`, grouped properties, patch preserves unknown props |
| `internal/database` | 5 | legacy-schema upgrade, index creation, idempotency |
| `internal/handler` | 31 | full HTTP surface through Echo |
| `internal/server/middleware` | 7 | auth against a real IMAP server, cache behaviour |

The handler tests exercise the real routing table with the IMAP-backed auth
middleware replaced by a stub. Worth knowing what they pin down, because each
corresponds to a bug that existed:

- a PUT payload comes back byte-identical, `VALARM` and `X-*` included;
- a deletion appears in the next delta as a tombstone;
- a pruned token is answered with `403 valid-sync-token`;
- a stale `If-Match` is refused with 412;
- a recurring series appears **once** in a `calendar-query`;
- another user's path is 404, and `%2e%2e%2f` does not escape the collection;
- `allprop` does not inline object bodies;
- an oversized `PUT` is refused and leaves the existing resource untouched.

### 7.2 Real-client matrix

Not yet done — it needs a running server. Test in this order, most forgiving
first:

| # | Client | Why |
|---|---|---|
| 1 | `curl` | isolates the protocol from any client's quirks (`scripts/test-dav-curl.sh`) |
| 2 | Thunderbird | tolerant, readable errors, useful error console |
| 3 | DAVx⁵ (Android) | verbose logs, open source, aggressively bidirectional |
| 4 | Apple Calendar / iOS | strictest; surfaces what the others forgive |
| 5 | Outlook + CalDAV Synchronizer | see [§8](#8-known-pitfalls) |
| 6 | Evolution (GNOME) | good cross-check |

Conformance suites: **litmus** for the WebDAV base, then **CalDAVTester** (from
Apple's CalendarServer) once the basics are stable.

### 7.3 Exit criteria

Both directions, for each client:

```
[ ] Create in webmail → appears in client, and the reverse
[ ] Edit in one → reflected in the other
[ ] Delete in one → disappears in the other
[ ] Weekly recurring event → occurrence count matches
[ ] Delete ONE occurrence → only that one disappears (EXDATE)
[ ] Edit ONE occurrence → only that one changes (RECURRENCE-ID)
[ ] All-day event → does not shift by a day in any timezone
[ ] Simultaneous edit in two clients → conflict detected via If-Match
[ ] iOS (vCard 3.0) and DAVx⁵ (vCard 4.0) on the SAME collection
[ ] Accented contact survives a full round trip
[ ] Group created in iOS appears as a group in Thunderbird
[ ] Two consecutive syncs with no changes → NO ETag changes
[ ] PROPFIND Depth:1 over 500 contacts does not transfer photo blobs
[ ] User of domain A cannot see domain B in the GAL
```

The all-day item fails frequently. The simultaneous-edit item is what separates
a toy from something you put in production.

**The ETag-stability item is the most revealing of all**: run two syncs back to
back without touching anything and confirm no ETag moved. If one did, something
is rewriting objects — find it now, not in production, because that is the
infinite-loop failure of [§2.1](#21-the-blob-is-the-truth).

---

## 8. Known Pitfalls

Collected from both drafts, plus what this implementation actually got wrong.

**Do not normalise the blob.** Re-serialising loses `X-*` properties and breaks
clients. See [§2.1](#21-the-blob-is-the-truth).

**UID ≠ file name.** The client names the resource. Store both, separately.

**One resource may hold several components.** A recurring series with overrides
is *one* `.ics` containing multiple `VEVENT`s sharing a UID. Do not split them.

**Line breaks are CRLF, and lines fold at 75 octets.** Fold on UTF-8 boundaries;
splitting a multi-byte character corrupts the value.

**`Depth` matters.** `0` = the resource itself, `1` = resource plus direct
children. Ignoring it produces enormous responses and slow clients.

**Status lives inside the multistatus.** A `207` carries each resource's real
status in the body. Do not return a blanket `200` when an item failed.

**`supported-address-data` uses `address-data-type`.** RFC 6352 §6.2.3 names the
child element `CARDDAV:address-data-type`; `address-data` is the REPORT payload
element. CalDAV is genuinely different — RFC 4791 does reuse `calendar-data`.
This codebase had it wrong; strict clients ignore the property and may then
assume vCard 3.0 only.

**`io.LimitReader` truncates silently.** Reading a body with a cap and acting on
the result stores a corrupted resource and returns success. Read one byte past
the limit and reject on overflow.

**Escape CRLF inside XML, do not emit it raw.** `xml.EscapeText` encodes it as
`&#xD;&#xA;`, which is required: an XML parser normalises a literal `\r` to
`\n`, silently corrupting the payload the client reads back.

**Aggressive polling.** Some clients sync every minute. Measure early; the
covering index on `dav_changes` is there for this.

**Outlook does not speak CalDAV/CardDAV natively** — Microsoft discontinued the
connector in 2014. Options: **CalDAV Synchronizer** (open source, mature, but
per-machine install; GPO helps in a corporate fleet), or **ActiveSync**, which
is native but is a Microsoft-patented protocol requiring a licence for
commercial use. `remdev/go-activesync` is already in `go.mod` and the EAS server
reuses the data layer described here — see
[ActiveSync Implementation](ACTIVESYNC_IMPLEMENTATION.md).

---

## 9. File & Directory Map

```
internal/
  dav/                          synchronisation primitives (no HTTP, no protocol)
    etag.go                     ComputeETag, Quote, MatchETag
    token.go                    SyncToken, ParseSyncToken, CTag
    sync.go                     Store, Record, ChangesSince, Cleanup
    conditional.go              CheckPreconditions, resource-name helpers
  contacts/
    vcard.go                    Parse, Build, ApplyToVCard, folding, escaping
  handler/
    dav_entry.go                single entry point; path scoping per user
    dav_paths.go                DAV URL construction and parsing
    dav_xml.go                  request decoding, multistatus building, limits
    caldav.go                   CalDAV collections, objects, REPORTs
    carddav.go                  CardDAV collections, objects, REPORTs
  repository/
    address_book.go             CardDAV collection CRUD
    collection_uri.go           Slugify, freeCollectionURI
    event.go / contact.go       object CRUD + changelog instrumentation
  database/
    dav_migrate.go              column preparation, backfill, index creation
  model/
    address_book.go             CardDAV collection
    dav_change.go               RFC 6578 changelog
    calendar.go / event.go      CalDAV collection and object (DAV fields)
    contact.go                  CardDAV object (DAV fields)
  server/
    routes.go                   /dav registration (catch-all per method)
    middleware/caldav_auth.go   Basic auth + credential cache
```

Tests sit beside the code they cover; `handler/dav_testutil_test.go` holds the
shared harness.

---

## 10. References

| RFC | Subject |
|-----|---------|
| 4918 | WebDAV |
| 4791 | CalDAV |
| 6352 | CardDAV |
| 6578 | Collection synchronisation (sync-token) |
| 5545 | iCalendar |
| 6350 | vCard 4.0 |
| 6764 | Discovery via DNS SRV and well-known URIs |
| 6638 | Automatic scheduling |
| 5546 | iTIP — scheduling transport |
| 6047 | iMIP — iTIP over e-mail |
