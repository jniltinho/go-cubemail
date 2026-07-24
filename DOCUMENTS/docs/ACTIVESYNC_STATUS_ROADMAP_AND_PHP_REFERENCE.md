# ActiveSync — Status, Roadmap, Go Test Client & Horde PHP Reference

> **Project:** go-cubemail-vue  
> **Related:** [ACTIVESYNC_IMPLEMENTATION.md](ACTIVESYNC_IMPLEMENTATION.md), [ACTIVESYNC_GO_REFERENCE.md](ACTIVESYNC_GO_REFERENCE.md), [ACTIVESYNC_CURL_TESTING.md](ACTIVESYNC_CURL_TESTING.md)  
> **PHP reference:** `php/ActiveSync` (Horde ActiveSync, GPLv2)  
> **SOGo reference:** `ActiveSync/` (Objective-C, production reference in this repo)

This document consolidates **everything implemented so far**, **what remains to finish**, **how to build a Go client to test the server**, and an **analysis of Horde ActiveSync (PHP)** as a blueprint for a more complete Go server.

---

## Table of Contents

1. [Executive summary](#1-executive-summary)
2. [What was implemented (phases 0–5)](#2-what-was-implemented-phases-05)
3. [Command matrix — implemented vs pending](#3-command-matrix--implemented-vs-pending)
4. [File map (Go server)](#4-file-map-go-server)
5. [Known limitations](#5-known-limitations)
5.1. [Shared data layer with CalDAV / CardDAV](#51-shared-data-layer-with-caldav--carddav)
6. [What we still need to do](#6-what-we-still-need-to-do)
7. [Verification checklist](#7-verification-checklist)
8. [Go client for testing ActiveSync](#8-go-client-for-testing-activesync)
9. [Horde ActiveSync (PHP) — library analysis](#9-horde-activesync-php--library-analysis)
10. [PHP → Go architecture mapping](#10-php--go-architecture-mapping)
11. [Using Horde/SOGo to build a more complete Go server](#11-using-hordesogo-to-build-a-more-complete-go-server)
12. [Recommended delivery order](#12-recommended-delivery-order)

---

## 1. Executive summary

go-cubemail-vue ships a **custom Go EAS server** under `internal/activesync/`. It is **not** a port of Horde or SOGo, but reuses:

| Layer | Technology |
|-------|------------|
| WBXML codec | `github.com/remdev/go-activesync/wbxml` |
| HTTP | Echo v5 (`/Microsoft-Server-ActiveSync`, `/autodiscover`) |
| Mail backend | `internal/imap/*` |
| SMTP outbound | `internal/smtp/*` |
| Calendar/contacts | GORM + `internal/repository/*` |
| Device state | SQL (`eas_devices`, `eas_folder_states`, `imap_folder_mappings`) |

**Current maturity:** MVP suitable for development and protocol testing. **Not production-ready** without real-device validation, HTTPS, proxy tuning for Ping long-poll, and several missing commands (attachments, MoveItems, GAL, vtodo, OOF, etc.).

**Status line:** Phase 0–5 **partial** — core sync + SendMail + ItemOperations body Fetch + Ping long-poll (mail) + Search results.

---

## 2. What was implemented (phases 0–5)

### Phase 0 — Foundation ✅

| Deliverable | Location |
|-------------|----------|
| EAS HTTP endpoint (OPTIONS + POST) | `internal/activesync/handler.go` |
| Autodiscover (mobile sync) | `internal/activesync/autodiscover.go` |
| WBXML marshal/unmarshal | `github.com/remdev/go-activesync/wbxml` |
| Provision (policy status 2, SOGo-compatible) | `commands/provision.go` |
| Config `[activesync]` section | `internal/config/config.go`, `web/files/config.default.toml` |
| IMAP Basic Auth middleware | `internal/server/middleware/eas_auth.go` |
| CSRF skip for EAS routes | `internal/server/middleware/csrf.go` |

### Phase 1 — Device connect ✅

| Deliverable | Location |
|-------------|----------|
| FolderSync (IMAP mail + virtual PIM folders) | `commands/foldersync.go` |
| Ping (immediate + long-poll in Phase 5) | `commands/ping.go` |
| Device/folder SQL state | `internal/activesync/state/*` |
| Stable IMAP folder GUIDs | `state/store.go` → `ImapFolderMapping` |

### Phase 2 — Mail sync ✅

| Deliverable | Location |
|-------------|----------|
| Mail Sync (Add/Change/Delete via IMAP) | `commands/mailsync.go` |
| Sync keys + `MailSyncCache` | `state/store.go`, `state/model.go` |
| GetItemEstimate (mail counts) | `commands/getitemestimate.go` |
| IMAP `FetchAllUIDs()` | `internal/imap/mailbox.go` |

**Note:** Sync returns **envelope only** (Subject, From, Read, DateReceived). Full body is fetched via **ItemOperations** (Phase 5).

### Phase 3 — Calendar + contacts ✅

| Deliverable | Location |
|-------------|----------|
| Calendar sync (`vevent/*`) | `commands/calendarsync.go` |
| Contacts sync (`vcard/*`) | `commands/contactssync.go` |
| Shared helpers (ServerId, EAS time) | `commands/eastime.go` |
| Item change cache | `state/itemcache.go` |
| Sync router | `commands/sync.go` |

**Note:** `vtodo/*` is a **stub** (valid sync keys, no items).

### Phase 4 — Outbound + settings ✅ (MVP)

| Deliverable | Location |
|-------------|----------|
| SendMail (WBXML + raw MIME) | `commands/sendmail.go` |
| SMTP `SendRaw()` | `internal/smtp/raw.go` |
| Append to Sent (IMAP) | `commands/imapconn.go` |
| MeetingResponse (PartStat → SQL) | `commands/meetingresponse.go` |
| Search (Total only → extended in Phase 5) | `commands/search.go` |
| Settings (UserInformation, DeviceInformation) | `commands/settings.go` |

### Phase 5 — Body fetch + push + search results ⚠️ Partial

| Deliverable | Location |
|-------------|----------|
| ItemOperations Fetch (mail + body) | `commands/itemoperations.go` |
| Shared mail body builder | `commands/emailfetch.go` |
| Ping long-poll (mail vs sync cache) | `commands/ping.go`, `mailsync.go` → `PingChangedCollections` |
| Search `<Result>` list + Range | `commands/search.go` |

### Documentation & tests ✅

| Asset | Path |
|-------|------|
| Implementation guide | `DOCUMENTS/docs/ACTIVESYNC_IMPLEMENTATION.md` |
| Go function reference | `DOCUMENTS/docs/ACTIVESYNC_GO_REFERENCE.md` |
| cURL testing | `DOCUMENTS/docs/ACTIVESYNC_CURL_TESTING.md` |
| Unit tests | `internal/activesync/commands/*_test.go` |
| HTTP integration tests (`-tags integration`) | `internal/activesync/integration_test.go` |
| Integration script | `scripts/test-activesync-integration.sh` |
| cURL smoke script | `scripts/test-activesync-curl.sh` |

---

## 3. Command matrix — implemented vs pending

Commands advertised in OPTIONS (`handler.go`):

```
Sync, SendMail, FolderSync, GetItemEstimate, MeetingResponse, Search, Settings,
Ping, ItemOperations, Provision, ResolveRecipients, MoveItems
```

| Command | Server status | Notes |
|---------|---------------|-------|
| **Provision** | ✅ | Policy not enforced (Status 2) |
| **FolderSync** | ✅ | IMAP + calendar/contacts/task folders |
| **Sync** | ✅ | mail, vevent, vcard; vtodo stub |
| **GetItemEstimate** | ✅ | mail, calendar, contacts |
| **Ping** | ⚠️ | Long-poll mail only; 2s poll interval |
| **SendMail** | ✅ | SMTP + optional Sent append |
| **MeetingResponse** | ⚠️ | PartStat update; no iMIP SMTP reply |
| **Search** | ⚠️ | Mailbox + Results; no GAL |
| **Settings** | ⚠️ | User/Device info; no OOF |
| **ItemOperations** | ⚠️ | Mail Fetch + body; no FileReference/Move |
| **ResolveRecipients** | ❌ | Advertised, not dispatched |
| **MoveItems** | ❌ | Advertised, not dispatched |
| SmartForward / SmartReply | ❌ | Not advertised |
| GetAttachment | ❌ | Deprecated in favor of ItemOperations |
| GetHierarchy | ❌ | Not implemented |
| ValidateCert | ❌ | Not implemented |
| FolderCreate/Delete/Update | ❌ | Not implemented |

---

## 4. File map (Go server)

```
internal/activesync/
├── handler.go              # OPTIONS + POST dispatch, detectCommand
├── autodiscover.go         # MS-OXDISCO mobile sync
├── integration_test.go     # HTTP integration (build tag: integration)
└── commands/
    ├── dispatcher.go       # Command router (phase 0–5)
    ├── context.go          # Per-request auth + device context
    ├── provision.go
    ├── foldersync.go
    ├── ping.go             # Long-poll (Phase 5)
    ├── sync.go             # Collection router
    ├── mailsync.go         # IMAP mail + PingChangedCollections
    ├── calendarsync.go
    ├── contactssync.go
    ├── getitemestimate.go
    ├── sendmail.go
    ├── meetingresponse.go
    ├── search.go           # Results + Total (Phase 5)
    ├── settings.go
    ├── itemoperations.go   # Fetch (Phase 5)
    ├── emailfetch.go         # Body builder (Phase 5)
    ├── imapconn.go           # imapConnect, appendToSent
    ├── eastime.go
    ├── wbxmlutil.go
    └── *_test.go

internal/activesync/state/
├── model.go                # EasDevice, EasFolderState, ImapFolderMapping
├── store.go                # Sync keys, folder GUID, MailSyncCache
└── itemcache.go            # Calendar/contacts change cache

internal/smtp/raw.go        # SendRaw for SendMail
internal/server/middleware/eas_auth.go
```

---

## 5. Known limitations

### Mail

- Sync ApplicationData has **no Body** — clients must call **ItemOperations Fetch** or **Search Result** for content.
- **Attachments** not exposed (`AirSyncBase.Attachments`, ItemOperations `FileReference`).
- No **MoveItems**, **SmartForward**, **SmartReply**.
- No MIME multipart ItemOperations responses.

### Calendar / contacts / tasks

- Calendar: no RRule, timezone blob, rich Body, recurrence on EAS.
- MeetingResponse: no organizer iMIP reply via SMTP.
- `vtodo/*`: stub only, and tasks belong to no DAV collection — they are not
  exposed over CalDAV and fall back to timestamp change detection.
- A contact's phone is stored in a single column, so the type a device assigned
  (mobile / business / home) is not preserved in the index; the stored vCard
  keeps all of them.

### Search & directory

- No **GAL** search (`Search.Name = GAL`).
- No DeepTraversal / multi-folder search (Android sends CollectionId=0).

### Settings & policy

- No **OOF** (Out of Office).
- No device password / certificate settings.
- Provision returns success without enforcing policies.

### Protocol & ops

- Protocol version fixed at **14.1** in config.
- Ping long-poll requires reverse proxy timeout ≥ heartbeat.
- A client `Add` that carries nothing usable is skipped without a per-item
  status, because `eas.SyncAdd` in the WBXML library has no `Status` field.
  Everything that can be stored is accepted precisely so this path is not hit —
  a device retries an unanswered `Add` indefinitely.
- EAS production may require Microsoft licensing (see implementation guide §2).

---

## 5.1 Shared data layer with CalDAV / CardDAV

ActiveSync and DAV write to the same rows through the same repositories, so a
change made on a phone reaches Thunderbird and vice versa. Consequences worth
knowing before touching either side:

- **Collections are shared.** One calendar or address book is one EAS folder and
  one DAV collection. `internal/activesync/commands/collections.go` maps
  `vevent/<uri>` and `vcard/<uri>` to the DAV collection URI, keeping
  `personal` as an alias for the default so devices synced by earlier builds are
  not forced to resync.
- **FolderSync reports hierarchy changes**, so a calendar created in Thunderbird
  reaches an already-paired device. The last list sent is stored under the
  synthetic `__hierarchy__` collection in the folder-state table.
- **The vCard blob is authoritative.** An EAS edit updates the flat columns and
  the repository patches the stored card, so addresses, photos, birthdays and
  extra numbers survive an edit made on a phone.
- **Writes advance the DAV sync token**, which is what makes an EAS change
  visible to CalDAV/CardDAV clients on their next delta.
- **Change detection prefers `SyncRevision`** (monotonic, per collection) and
  falls back to `UpdatedAt` for rows outside any DAV collection. The timestamp
  alone cannot separate two edits made within the same second.
- **Ping does not scan rows** while a collection is quiet: it compares the
  collection sync token first — one integer read — and only then compares
  id/updated_at pairs. It never loads the iCalendar or vCard payloads.

See the [CalDAV & CardDAV Implementation Guide](DAV_IMPLEMENTATION.md) for the
DAV side of this contract.

---

## 6. What we still need to do

### Priority A — Validate end-to-end (before more code)

1. Run server with `[activesync] enabled = true`, migrations applied.
2. Configure **HTTPS** (mobile clients require TLS in production).
3. Add ActiveSync account on **iOS or Android**.
4. Run integration tests:

   ```bash
   EAS_INTEGRATION_URL=https://host/Microsoft-Server-ActiveSync \
   EAS_INTEGRATION_USER=user@example.com \
   EAS_INTEGRATION_PASS=secret \
   go test -tags integration -v ./internal/activesync/...
   ```

5. Mark items in [verification checklist](#7-verification-checklist).

### Priority B — Complete Phase 5 gaps

| Task | Reference |
|------|-----------|
| ItemOperations **FileReference** (attachments) | SOGo `SOGoActiveSyncDispatcher.m` processItemOperations; Horde `Request/ItemOperations.php` |
| **Truncated body in Mail Sync** (not just ItemOperations) | Horde `Imap/EasMessageBuilder/*`, SOGo `SOGoMailObject+ActiveSync.m` |
| Ping for **calendar/contacts** | Horde `getServerChanges(..., $ping=true)` |
| `TestIntegrationItemOperations` | New integration test |
| Search **GAL** (optional) | Horde `getSearchResults()` |

### Priority C — Missing commands (Horde parity)

| Task | Horde class |
|------|-------------|
| MoveItems | `Horde_ActiveSync_Request_MoveItems` |
| ResolveRecipients | `Horde_ActiveSync_Request_ResolveRecipients` |
| SmartForward / SmartReply | extends SendMail request |
| Settings OOF | `Horde_ActiveSync_Message/Oof.php` |
| vtodo sync | `Horde_ActiveSync_Message/Task.php` |
| FolderCreate/Delete/Update | `Horde_ActiveSync_Request_FolderCreate` |

### Priority D — Protocol polish

- EAS **16.0/16.1** headers and command surface.
- Multipart responses (`application/vnd.ms-sync.multipart`) for large attachments.
- Autodiscover HTTPS + SRV records.
- Policy enforcement / remote wipe (optional).

---

## 7. Verification checklist

From `ACTIVESYNC_IMPLEMENTATION.md` §21 — most items are **not yet checked** in a real environment:

| Phase | Check |
|-------|-------|
| 0 | OPTIONS returns protocol commands; Provision returns WBXML |
| 1 | FolderSync lists Inbox + PIM folders; `eas_devices` row created; iOS "Verifying" passes |
| 2 | New mail syncs; read/unread both ways; delete on phone removes via IMAP |
| 3 | Event/contact bidirectional sync |
| 4 | SendMail delivers; Sent append; MeetingResponse updates web calendar; Settings email |
| 5 | ItemOperations returns body; Ping status 2 on new mail; Search returns Results |

---

## 8. Go client for testing ActiveSync

There is **no production Go EAS server library**. Testing uses a **client** against our server.

### Option 1 — `github.com/remdev/go-activesync` (recommended)

Pure Go **client** for EAS 14.1. Already used for WBXML on the server side.

**Install:**

```bash
go get github.com/remdev/go-activesync@latest
```

**Client commands implemented:** `Provision`, `FolderSync`, `Sync`, `Ping`, Autodiscover.

**Example — full smoke test flow:**

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/remdev/go-activesync/autodiscover"
    "github.com/remdev/go-activesync/client"
    "github.com/remdev/go-activesync/eas"
)

func main() {
    ctx := context.Background()
    user := "you@example.com"
    pass := "secret"

    // 1. Autodiscover (optional if URL known)
    ad, err := autodiscover.New(http.DefaultClient).Discover(ctx, user,
        &autodiscover.Credentials{Username: user, Password: pass})
    if err != nil {
        log.Fatal(err)
    }

    // 2. Client
    c, err := client.New(client.Config{
        BaseURL:    ad.URL, // or "https://host/Microsoft-Server-ActiveSync"
        Auth:       &client.BasicAuth{Username: user, Password: pass},
        DeviceID:   "go-test-client-001",
        DeviceType: "GoTest",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 3. Provision
    if _, err := c.Provision(ctx, user); err != nil {
        log.Fatal(err)
    }

    // 4. FolderSync
    fs, err := c.FolderSync(ctx, user, "0")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("folder sync key: %s, folders: %d", fs.SyncKey, len(fs.Folders))

    // 5. Find inbox collection id (mail/…)
    var inboxID string
    for _, f := range fs.Folders {
        if f.Type == 2 { // default type for inbox in our FolderSync
            inboxID = f.ServerID
            break
        }
    }

    // 6. Initial Sync
    sync0, err := c.Sync(ctx, user, &eas.SyncRequest{
        Collections: eas.SyncCollections{
            Collection: []eas.SyncCollection{{SyncKey: "0", CollectionID: inboxID}},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    key := sync0.Collections.Collection[0].SyncKey

    // 7. Sync with changes
    resp, err := client.SyncTyped[eas.Email](ctx, c, user, &eas.SyncRequest{
        Collections: eas.SyncCollections{
            Collection: []eas.SyncCollection{{
                SyncKey: key, CollectionID: inboxID, GetChanges: 1, WindowSize: 25,
            }},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, col := range resp.Collections {
        for _, add := range col.Add {
            if add.ApplicationData != nil {
                log.Printf("mail %s: %s", add.ServerID, add.ApplicationData.Subject)
            }
        }
    }

    // 8. Ping (long-poll — server holds connection up to HeartbeatInterval)
    ping, err := c.Ping(ctx, user, &eas.PingRequest{
        HeartbeatInterval: 60,
        Folders: eas.PingFolders{
            Folder: []eas.PingFolder{{ID: inboxID, Class: "Email"}},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("ping status=%d (2=changes)", ping.Status)
}
```

**Runnable examples** in the upstream repo: `examples/login`, `examples/inbox-sync`, `examples/calendar-sync`, `examples/ping`.

**Gap vs our server:** remdev client does **not** yet wrap `SendMail`, `ItemOperations`, `Search`, `Settings`, `MeetingResponse`. For those, use:

- Our **integration tests** (`integration_test.go`) — raw HTTP + WBXML structs.
- **cURL** — `scripts/test-activesync-curl.sh`.
- Extend client with custom POST (see Option 3).

### Option 2 — `github.com/hstern/go-activesync`

Full EAS **client** (up to 16.1). Useful for:

- Capturing WBXML fixtures from Exchange Online.
- Testing commands remdev does not expose yet.

Add as **dev dependency** only:

```bash
go get github.com/hstern/go-activesync@latest
```

Use its client to call our server and compare responses with SOGo/Horde behavior.

### Option 3 — Custom `cmd/eas-test` in go-cubemail-vue (recommended next step)

Create a small CLI under `cmd/eas-test/` that:

1. Reads `EAS_URL`, `EAS_USER`, `EAS_PASS` from env.
2. Runs the remdev flow (Provision → FolderSync → Sync → Ping).
3. Sends **raw WBXML** for commands not in remdev client:
   - ItemOperations Fetch (using structs from `commands/itemoperations.go` patterns)
   - Search with Range
   - SendMail
   - Settings

This complements `integration_test.go` with a manual debugging tool.

**Suggested layout:**

```
cmd/eas-test/
  main.go           # subcommands: provision, foldersync, sync, ping, fetch, search, sendmail
  client.go         # HTTP + Basic Auth wrapper
  wbxml.go          # shared marshal helpers
```

Run against local server:

```bash
EAS_URL=http://localhost:8080/Microsoft-Server-ActiveSync \
EAS_USER=you@example.com EAS_PASS=secret \
go run ./cmd/eas-test foldersync
```

### Option 4 — Existing project integration tests

Already in repo — no new code required for basic HTTP coverage:

```bash
./scripts/test-activesync-integration.sh
```

| Test | Requires |
|------|----------|
| OPTIONS, Provision, Settings, Ping | URL + credentials |
| Search | IMAP mailbox |
| SendMail | `EAS_INTEGRATION_SEND_TO` + SMTP |
| MeetingResponse | `EAS_INTEGRATION_EVENT_ID` or `_EVENT_UID` |

---

## 9. Horde ActiveSync (PHP) — library analysis

**Path:** `php/ActiveSync/`  
**Package:** Horde ActiveSync (Michael J Rubinsky, Horde LLC, GPLv2)  
**Origin:** Evolved from Z-Push concepts; used in Horde Groupware and related projects.

### 9.1 Architecture overview

```
HTTP Request
    │
    ▼
Horde_ActiveSync::handleRequest($cmd, $devId)
    ├── authenticate (Horde_ActiveSync_Credentials)
    ├── Horde_ActiveSync_Device (device metadata, policy key)
    └── Horde_ActiveSync_Request_<Command> extends Request_Base
            ├── WBXML Decoder → parse request
            ├── Horde_ActiveSync_Driver_Base → backend data
            ├── Horde_ActiveSync_State_Base → sync keys, caches
            └── WBXML Encoder → response
```

**Separation of concerns (key lesson for Go):**

| Layer | PHP class | Responsibility |
|-------|-----------|----------------|
| **Server** | `Horde_ActiveSync` | HTTP, auth, version negotiation, dispatch |
| **Request handler** | `Horde_ActiveSync_Request_*` | One EAS command; WBXML in/out |
| **Driver (backend)** | `Horde_ActiveSync_Driver_Base` | **Abstract** — mail/calendar/contacts data |
| **State** | `Horde_ActiveSync_State_Base` | Device + folder sync keys, caches, maps |
| **Message models** | `Horde_ActiveSync_Message_*` | Typed ApplicationData (Mail, Appointment, Contact, Task…) |
| **WBXML** | `Horde_ActiveSync_Wbxml_Encoder/Decoder` | Code pages, streaming parse |
| **IMAP adapter** | `Horde_ActiveSync_Imap_*` | Mail-specific: MODSEQ strategies, body builders |

### 9.2 Supported protocol surface

From `ServerTest.php` and README:

- **Versions:** 2.5, 12.0, 12.1, 14.0, 14.1, 16.0
- **Commands:** Sync, SendMail, SmartForward, SmartReply, GetAttachment, GetHierarchy, CreateCollection, DeleteCollection, MoveCollection, FolderSync, FolderCreate, FolderDelete, FolderUpdate, MoveItems, GetItemEstimate, MeetingResponse, Search, Settings, Ping, ItemOperations, Provision, ResolveRecipients, ValidateCert

### 9.3 Request handler classes

| Command | PHP class | Extends |
|---------|-----------|---------|
| Sync | `Horde_ActiveSync_Request_Sync` | SyncBase |
| FolderSync | `Horde_ActiveSync_Request_FolderSync` | Base |
| Ping | `Horde_ActiveSync_Request_Ping` | Base |
| Provision | `Horde_ActiveSync_Request_Provision` | Base |
| SendMail | `Horde_ActiveSync_Request_SendMail` | Base |
| SmartForward/Reply | `Horde_ActiveSync_Request_SmartForward/Reply` | SendMail |
| ItemOperations | `Horde_ActiveSync_Request_ItemOperations` | SyncBase |
| Search | `Horde_ActiveSync_Request_Search` | SyncBase |
| Settings | `Horde_ActiveSync_Request_Settings` | Base |
| MeetingResponse | `Horde_ActiveSync_Request_MeetingResponse` | Base |
| MoveItems | `Horde_ActiveSync_Request_MoveItems` | Base |
| ResolveRecipients | `Horde_ActiveSync_Request_ResolveRecipients` | Base |
| GetItemEstimate | `Horde_ActiveSync_Request_GetItemEstimate` | Base |
| FolderCreate | `Horde_ActiveSync_Request_FolderCreate` | Base (+ Delete/Update routed here) |
| Autodiscover | `Horde_ActiveSync_Request_Autodiscover` | Base |

### 9.4 Driver contract (`Horde_ActiveSync_Driver_Base`)

The driver is the **backend plugin**. go-cubemail-vue effectively implements this inline via IMAP + GORM instead of a formal interface.

**Critical abstract methods:**

| Method | Purpose |
|--------|---------|
| `getFolderList()` / `getFolders()` | FolderSync hierarchy |
| `changeFolder()` | FolderCreate/Update/Delete |
| `getServerChanges($folder, $from_ts, $to_ts, $cutoffdate, $ping)` | Sync + **Ping change detection** |
| `statMessage()` / `getMessage()` | Sync Add/Change + ItemOperations Fetch |
| `changeMessage()` | Client Change (read flag, PIM edits) |
| `deleteMessage()` | Sync Delete |
| `sendMail()` | SendMail / SmartForward / SmartReply |
| `getAttachment()` | ItemOperations FileReference |
| `getSearchResults()` | Search mailbox + GAL |
| `getSettings()` | Settings (OOF, device info) |
| `getCurrentPolicy()` / `getProvisioning()` | Provision |
| `getSpecialFolderNameByType()` | Sent, Trash, Drafts mapping |
| `getWasteBasket()` | Trash folder for soft delete |

**go-cubemail-vue mapping today:**

| Driver method | Go implementation |
|---------------|-------------------|
| getFolders | `FolderBuilder.Build()` |
| getServerChanges | `MailSyncEngine.buildServerChanges`, calendar/contacts engines |
| getMessage | `fetchMailProperties()` (ItemOperations/Search only) |
| changeMessage | `applyClientCommands` (mail flags; PIM SQL merge) |
| deleteMessage | IMAP delete + SQL delete |
| sendMail | `SendMailHandler` + `smtp.SendRaw` |
| getSearchResults | `SearchHandler.searchMailbox` (partial) |
| getSettings | `SettingsHandler` (partial) |
| getAttachment | **Not implemented** |
| getServerChanges (ping) | `PingChangedCollections` (mail only) |

### 9.5 State management (`Horde_ActiveSync_State_Base`)

Horde state is **richer** than our SQL model:

- Per-device folder sync key
- Per-collection sync key + **sync timestamps** (`lastSyncStamp`, `thisSyncStamp`)
- **SyncCache** (`Horde_ActiveSync_SyncCache`) — UID maps, modseq, pending flags
- **Mail map** tables (IMAP UID ↔ EAS server id, draft/change/delete flags)
- Policy key per device
- Mongo or SQL backends (`State/Sql.php`, `State/Mongo.php`)

**go-cubemail-vue equivalent:**

| Horde | Go |
|-------|-----|
| Device state | `EasDevice` |
| Collection sync key | `EasFolderState.SyncKey` |
| Mail UID cache | `EasFolderState.SyncCache` → `MailSyncCache` JSON |
| PIM change cache | `ItemSyncCache` JSON |
| IMAP folder stable id | `ImapFolderMapping` |

**Gap:** Horde tracks **modseq/timestamps** for efficient incremental sync and reliable Ping. We compare full UID sets + flags (works but heavier).

### 9.6 Message layer (`Horde_ActiveSync_Message_*`)

Typed WBXML serializers for each PIM class:

- `Message/Mail.php` — MS-ASEMAIL + truncation
- `Message/Appointment.php` — MS-ASCAL
- `Message/Contact.php` — MS-ASCNTC
- `Message/Task.php` — MS-ASTASK
- `Message/AirSyncBaseBody.php` — Body, NativeBodyType, attachments
- `Message/Attachment.php`, `AirSyncBaseFileAttachment.php`

**go-cubemail-vue:** uses `github.com/remdev/go-activesync/eas` structs + custom `easMailFetchProps` / `easAirSyncBody` for ItemOperations.

### 9.7 IMAP-specific subsystem

Horde mail sync is **not trivial** — key pieces:

| Component | Role |
|-----------|------|
| `Imap/Adapter.php` | IMAP connection abstraction |
| `Imap/Strategy/Modseq.php` | CONDSTORE-based change detection |
| `Imap/Strategy/Initial.php` | First sync |
| `Imap/EasMessageBuilder/*` | Body truncation, HTML/plain/MIME for Sync vs ItemOperations |
| `Imap/MessageBodyData.php` | Body parts, attachment metadata |
| `Rfc822.php` | MIME parsing |
| `Mime.php` | TNEF, multipart |

**This is the main reference for completing go-cubemail mail body + attachments.**

### 9.8 Tests in PHP library

```
php/ActiveSync/test/Horde/ActiveSync/
├── ServerTest.php          # Protocol versions, command lists
├── MimeTest.php, Rfc822Test.php
├── AppointmentTest.php, ContactTest.php
├── ImapAdapterTest.php
├── Factory/TestServer.php  # Mock driver + server for unit tests
└── fixtures/*.eml          # Real MIME samples (signed, multipart, invitations)
```

**Use for Go:** port fixture `.eml` files into Go tests for `ParseMessage`, body selection, ItemOperations encoding.

---

## 10. PHP → Go architecture mapping

Target Go structure for a **more complete** server inspired by Horde:

```
internal/activesync/
├── handler.go                 # Horde_ActiveSync::handleRequest
├── autodiscover.go
├── driver/                    # NEW — formal backend interface
│   ├── driver.go              # interface (like Driver_Base)
│   ├── imap_mail.go           # mail collections
│   ├── sql_calendar.go
│   ├── sql_contacts.go
│   └── smtp_outbound.go
├── commands/                  # Horde_ActiveSync_Request_*
│   └── (one file per command)
├── message/                   # NEW — typed ApplicationData builders
│   ├── mail.go
│   ├── appointment.go
│   ├── contact.go
│   ├── body.go                # AirSyncBaseBody (from AirSyncBaseBody.php)
│   └── attachment.go
├── state/                     # Horde_ActiveSync_State_*
│   └── (extend with modseq/timestamps if needed)
└── wbxml/                     # already via remdev/wbxml
```

### Interface sketch (Go driver)

```go
// Driver is the backend contract (Horde_ActiveSync_Driver_Base equivalent).
type Driver interface {
    Authenticate(user, pass string) error
    Folders(ctx context.Context) ([]Folder, error)
    ServerChanges(ctx context.Context, req ChangeRequest) (Changes, error)
    GetMessage(ctx context.Context, folderID, serverID string, opts MessageOpts) ([]byte, error)
    ChangeMessage(ctx context.Context, folderID, serverID string, data []byte) error
    DeleteMessages(ctx context.Context, folderID string, serverIDs []string) error
    SendMail(ctx context.Context, rfc822 []byte, saveSent bool) error
    Search(ctx context.Context, params SearchParams) (SearchResults, error)
    Settings(ctx context.Context, req SettingsGet) (SettingsResponse, error)
}
```

Implementations wrap existing `internal/imap`, `internal/repository`, `internal/smtp`.

---

## 11. Using Horde/SOGo to build a more complete Go server

### Step-by-step methodology

1. **Pick a command** (e.g. ItemOperations attachment Fetch).
2. **Read Horde handler:** `php/ActiveSync/lib/Horde/ActiveSync/Request/ItemOperations.php` → `_handle()`.
3. **Read driver call:** trace to `getMessage()` / `getAttachment()` in `Driver/Base.php` and concrete Horde/Kolab driver if present in your deployment.
4. **Cross-check SOGo:** `ActiveSync/SOGoActiveSyncDispatcher.m` → `processItemOperations:` (behavioral reference for iOS quirks).
5. **Capture WBXML:** use Horde/Z-Push debug or `hstern/go-activesync` client against Exchange to get golden request/response bytes.
6. **Implement Go handler** in `internal/activesync/commands/`.
7. **Unit test** WBXML round-trip + **integration test** against running server.
8. **Real device test** on iOS/Android.

### High-value Horde files to port concepts from

| Goal | PHP files |
|------|-----------|
| Full mail body in Sync | `Imap/EasMessageBuilder/Html.php`, `Plain.php`, `Mime.php` |
| Attachments | `Message/AirSyncBaseAttachment.php`, `Request/ItemOperations.php` |
| Ping accuracy | `Request/Ping.php`, `Driver::getServerChanges(..., $ping=true)` |
| Search + GAL | `Request/Search.php`, `Search/Params.php`, `Search/Results.php` |
| OOF Settings | `Message/Oof.php`, `Request/Settings.php` |
| MoveItems | `Request/MoveItems.php` |
| Tasks sync | `Message/Task.php`, `Message/TaskRecurrence.php` |
| Timezone blob | `Timezone.php` |
| WBXML edge cases | `Wbxml/Decoder.php`, `Wbxml/Encoder.php` |

### SOGo files (same repo, Objective-C)

| Goal | SOGo file |
|------|-----------|
| Dispatcher | `ActiveSync/SOGoActiveSyncDispatcher.m` |
| Mail mapping | `ActiveSync/SOGoMailObject+ActiveSync.m` |
| Calendar | `ActiveSync/iCalEvent+ActiveSync.m` |
| Contacts | `ActiveSync/NGVCard+ActiveSync.m` |
| WBXML | `ActiveSync/NSData+ActiveSync.m` |

### What NOT to do

- **Do not embed PHP** in go-cubemail-vue production (different runtime, licensing, ops).
- **Do not copy WBXML tables blindly** — verify against MS-ASWBXML; use `remdev/go-activesync/wbxml` code pages.
- **Do not implement all Horde commands at once** — follow [Priority B/C](#6-what-we-still-need-to-do).

### Licensing note

- **Horde ActiveSync:** GPLv2 — study for architecture; do not copy large code blocks into MIT/proprietary Go without compliance review.
- **SOGo ActiveSync:** SOGo license + Microsoft EAS patent/licensing considerations for production.
- **Microsoft specs:** MS-ASCMD, MS-ASWBXML are the authoritative interoperability references.

---

## 12. Recommended delivery order

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Real device + integration test validation (Priority A)    │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. cmd/eas-test CLI + ItemOperations integration test        │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. ItemOperations attachments (Horde ItemOperations.php)   │
│ 4. Mail Sync body truncation (Horde EasMessageBuilder)       │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. MoveItems, ResolveRecipients                              │
│ 6. Ping calendar/contacts                                    │
│ 7. vtodo sync, Settings OOF                                  │
└──────────────────────────────┬──────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. Optional: formal Driver interface refactor              │
│ 9. EAS 16.x, multipart, policy enforcement                 │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick reference links

| Resource | URL / path |
|----------|------------|
| MS-ASCMD | https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-ascmd |
| MS-ASWBXML | https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-aswbxml |
| remdev go-activesync | https://github.com/remdev/go-activesync |
| Go server docs | [ACTIVESYNC_GO_REFERENCE.md](ACTIVESYNC_GO_REFERENCE.md) |
| Horde PHP lib | `php/ActiveSync/lib/Horde/ActiveSync/` |
| SOGo ActiveSync | `ActiveSync/` in this repository |

---

*Document version: 1.0 — ActiveSync status, roadmap, Go test client guide, and Horde PHP reference for go-cubemail-vue.*
