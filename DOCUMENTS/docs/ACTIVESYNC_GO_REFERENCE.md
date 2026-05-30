# ActiveSync Backend — Go Function Reference

> **Project:** go-cubemail-vue  
> **Packages:** `internal/activesync`, `internal/activesync/commands`, `internal/activesync/state`, `internal/server/middleware`, `internal/imap` (EAS helpers)  
> **Related:** [ActiveSync Implementation Guide](ACTIVESYNC_IMPLEMENTATION.md), [ActiveSync cURL Testing Guide](ACTIVESYNC_CURL_TESTING.md)

This document lists every type and function in the ActiveSync backend with a short description. Full godoc comments live in the source files (`go doc` / IDE hover).

---

## Table of Contents

1. [HTTP layer (`internal/activesync`)](#1-http-layer-internalactivesync)
2. [Command handlers (`internal/activesync/commands`)](#2-command-handlers-internalactivesynccommands)
3. [Sync state store (`internal/activesync/state`)](#3-sync-state-store-internalactivesyncstate)
4. [Server wiring (`internal/server`)](#4-server-wiring-internalserver)
5. [Config (`internal/config`)](#5-config-internalconfig)
6. [IMAP helpers (`internal/imap`)](#6-imap-helpers-internalimap)
7. [Phase 3 — Calendar & contacts sync (detail)](#7-phase-3--calendar--contacts-sync-detail)
8. [Phase 4 — SendMail, MeetingResponse, Search, Settings](#8-phase-4--sendmail-meetingresponse-search-settings)
9. [Phase 5 — ItemOperations, Ping long-poll, Search results](#9-phase-5--itemoperations-ping-long-poll-search-results)
10. [SMTP helper (`internal/smtp`)](#10-smtp-helper-internalsmtp)
11. [View godoc locally](#11-view-godoc-locally)

---

## 1. HTTP layer (`internal/activesync`)

### `handler.go`

| Name | Kind | Description |
|------|------|-------------|
| `Handler` | struct | EAS HTTP handler: OPTIONS capability probe + POST command dispatch. |
| `NewHandler` | func | `(cfg *config.Config, db *gorm.DB) *Handler` — wires store, dispatcher, protocol headers. |
| `Options` | method | Handles `OPTIONS /Microsoft-Server-ActiveSync` (capability probe). |
| `Handle` | method | Handles `POST /Microsoft-Server-ActiveSync`; reads WBXML body, dispatches by `Cmd` query param. |
| `WriteWBXML` | func | `(v any) ([]byte, error)` — marshals a value to WBXML bytes. |
| `ReadWBXML` | func | `(data []byte, v any) error` — unmarshals WBXML into v. |
| `DebugXML` | func | `(data []byte) string` — best-effort token trace for debug logging. |
| `NowUTC` | func | `() time.Time` — current UTC time helper. |
| `setResponseHeaders` | method | Sets `MS-Server-ActiveSync`, protocol commands/versions, and Content-Type headers. |
| `userID` | method | Maps Basic Auth IMAP username → `model.User.ID` (FirstOrCreate). |
| `detectCommand` | func | Infers command name from WBXML root element when `Cmd` query param is missing. |

### `autodiscover.go`

| Name | Kind | Description |
|------|------|-------------|
| `AutodiscoverHandler` | struct | Serves MS-OXDISCO mobile sync autodiscover XML responses. |
| `NewAutodiscoverHandler` | func | `(cfg *config.Config) *AutodiscoverHandler` |
| `MobileSync` | method | Handles `POST /autodiscover/autodiscover.xml` and `GET /.well-known/autodiscover/...`. |
| `extractAutodiscoverEmail` | func | Parses `EMailAddress` from Autodiscover request XML body. |
| `xmlEscape` | func | XML-escapes a string for safe Autodiscover responses. |

---

## 2. Command handlers (`internal/activesync/commands`)

### `context.go`

| Name | Kind | Description |
|------|------|-------------|
| `Context` | struct | Per-request authenticated state: user ID, IMAP credentials, device ID/type, protocol version, query params. |

### `dispatcher.go`

| Name | Kind | Description |
|------|------|-------------|
| `Dispatcher` | struct | Routes EAS command names to handler implementations. |
| `NewDispatcher` | func | `(cfg, db, store, calRepo) *Dispatcher` — constructs phase 0–5 handlers. |
| `Dispatch` | method | `(ctx *Context, cmd string, body []byte) ([]byte, error)` — executes one EAS command. |

Supported commands: `Provision`, `FolderSync`, `Ping`, `Sync`, `GetItemEstimate`, `SendMail`, `MeetingResponse`, `Search`, `Settings`, `ItemOperations`.

### `provision.go`

| Name | Kind | Description |
|------|------|-------------|
| `ProvisionHandler` | struct | MS-ASPROV Provision command handler. |
| `Handle` | method | Returns policy-not-applied response (Status 2, SOGo-compatible). |

### `foldersync.go`

| Name | Kind | Description |
|------|------|-------------|
| `FolderSyncHandler` | struct | MS-ASCMD FolderSync command handler. |
| `Handle` | method | Returns IMAP mail folders + calendar/tasks/contacts virtual folders; manages folder sync key. |
| `FolderEntry` | struct | One folder in the EAS hierarchy (`ServerID`, `ParentID`, `DisplayName`, `Type`). |
| `FolderBuilder` | struct | Assembles folder list from IMAP + fixed PIM collections. |
| `NewFolderBuilder` | func | `(cfg, store) *FolderBuilder` |
| `Build` | method | `(ctx *Context) ([]FolderEntry, error)` — full folder list for authenticated user. |
| `mailFolders` | method | Lists IMAP mailboxes and maps each to `mail/{guid}`. |
| `displayName` | func | Best display label for an IMAP mailbox. |
| `mailFolderType` | func | Maps mailbox icon type → MS-ASCMD folder type integer. |

### `ping.go` — `PingHandler`

| Name | Kind | Description |
|------|------|-------------|
| `PingHandler` | struct | MS-ASCMD Ping command handler with long-poll change detection. |
| `NewPingHandler` | func | `(cfg, store, mail) *PingHandler` |
| `Handle` | method | Polls mail folders every 2s until HeartbeatInterval; status 2 when cache differs from IMAP. |

### `sync.go`

| Name | Kind | Description |
|------|------|-------------|
| `SyncHandler` | struct | MS-ASCMD Sync command handler (mail, calendar, contacts). |
| `NewSyncHandler` | func | `(store, mail, calendar, contacts) *SyncHandler` |
| `Handle` | method | Unmarshals Sync request, processes each collection, returns Sync response. |
| `syncCollection` | method | Routes collection ID prefix to mail, calendar, or contacts engine. |
| `syncCollectionStub` | method | Validates sync keys for unimplemented collections (e.g. `vtodo/*`). |

### `calendarsync.go` — `CalendarSyncEngine`

Bidirectional sync for **`vevent/*`** collections (default calendar via `CalendarRepo.EnsureDefault`).

#### Exported

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewCalendarSyncEngine` | `(store *state.Store, calRepo *repository.CalendarRepo, eventRepo *repository.EventRepo) *CalendarSyncEngine` | Constructs the engine. |
| `SyncCollection` | `(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error)` | Full Sync handler: client commands, server changes, sync key + cache. |
| `EstimateCount` | `(ctx *Context, collectionID string) (int32, error)` | Event count for GetItemEstimate. |

#### Internal (unexported)

| Function | Signature | Description |
|----------|-----------|-------------|
| `applyClientCommands` | `(ctx, cmds, calendarID, cache) (*eas.SyncCommands, error)` | Client Add/Change/Delete → SQL; returns ClientId→ServerId for Add. |
| `buildServerChanges` | `(userID, calendarID, cache, window) ([]SyncAdd, []SyncChange, []SyncDelete, bool, error)` | Compares DB `UpdatedAt` with ItemSyncCache. |
| `buildSyncAdd` | `(event model.Event) (eas.SyncAdd, error)` | Event → WBXML Add with `eas.Appointment` ApplicationData. |
| `buildSyncChange` | `(event model.Event) (eas.SyncChange, error)` | Updated event → WBXML Change. |
| `eventToAppointment` | `(event *model.Event) eas.Appointment` | Maps SQL event + attendees → MS-ASCAL fields. |
| `appointmentToEvent` | `(appt *eas.Appointment, userID, calendarID uint) *model.Event` | Client Add → new `model.Event` (generates UID if missing). |
| `applyAppointmentToEvent` | `(appt *eas.Appointment, event *model.Event)` | Client Change → merge into existing event. |
| `partStatToEas` | `(partStat string) int32` | iCalendar PARTSTAT → AttendeeStatus (0/2/3/4). |
| `easPartStatToICal` | `(status int32) string` | AttendeeStatus → PARTSTAT string. |
| `parseVEventCollectionID` | `(collectionID string) bool` | True when ID starts with `vevent/`. |
| `syncWindow` | `(windowSize int32) int` | Batch size; defaults to `defaultSyncWindow` (100). |

#### Calendar field mapping (`model.Event` ↔ `eas.Appointment`)

| EAS field (`eas.Appointment`) | SQL / model source |
|-------------------------------|-------------------|
| `UID` | `event.UID` |
| `Subject` | `event.Summary` |
| `Location` | `event.Location` |
| `StartTime` | `easTime(event.StartAt, event.IsAllDay)` |
| `EndTime` | `easTime(event.EndAt, event.IsAllDay)` |
| `AllDayEvent` | `1` when `event.IsAllDay`, else `0` |
| `OrganizerEmail` | `event.OrganizerEmail` |
| `OrganizerName` | `event.OrganizerName` |
| `DtStamp` | `easTime(event.UpdatedAt, false)` |
| `BusyStatus` | `eas.BusyStatusBusy` (constant) |
| `MeetingStatus` | `eas.MeetingStatusNonMeeting` (constant) |
| `Attendees` | `event.Attendees` → Email, Name, AttendeeStatus |

On client Add/Change, `ICalContent` is regenerated via `calendar.BuildICalContent`.

---

### `contactssync.go` — `ContactsSyncEngine`

Bidirectional sync for **`vcard/*`** collections (all user contacts).

#### Exported

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewContactsSyncEngine` | `(store *state.Store, contactRepo *repository.ContactRepo) *ContactsSyncEngine` | Constructs the engine. |
| `SyncCollection` | `(ctx *Context, dev *state.EasDevice, col eas.SyncCollection) (eas.SyncCollection, error)` | Full Sync handler for contacts. |
| `EstimateCount` | `(ctx *Context, collectionID string) (int32, error)` | Contact count for GetItemEstimate. |

#### Internal (unexported)

| Function | Signature | Description |
|----------|-----------|-------------|
| `applyClientCommands` | `(ctx, cmds, cache) (*eas.SyncCommands, error)` | Client Add/Change/Delete → SQL. |
| `buildServerChanges` | `(userID, cache, window) ([]SyncAdd, []SyncChange, []SyncDelete, bool, error)` | Compares DB `UpdatedAt` with ItemSyncCache. |
| `buildSyncAdd` | `(contact model.Contact) (eas.SyncAdd, error)` | Contact → WBXML Add. |
| `buildSyncChange` | `(contact model.Contact) (eas.SyncChange, error)` | Updated contact → WBXML Change. |
| `modelContactToEas` | `(c model.Contact) eas.Contact` | SQL → MS-ASCNTC ApplicationData. |
| `easContactToModel` | `(ct *eas.Contact, userID uint) *model.Contact` | Client Add → new `model.Contact`. |
| `applyEasContactToModel` | `(ct *eas.Contact, contact *model.Contact)` | Client Change → merge fields. |
| `firstNonEmpty` | `(values ...string) string` | Returns first non-empty phone/email helper. |
| `parseVCardCollectionID` | `(collectionID string) bool` | True when ID starts with `vcard/`. |

#### Contacts field mapping (`model.Contact` ↔ `eas.Contact`)

| EAS field (`eas.Contact`) | SQL / model source |
|---------------------------|-------------------|
| `FirstName` | `contact.FirstName` |
| `LastName` | `contact.LastName` |
| `Email1Address` | `contact.Email` |
| `Title` | `contact.Title` |
| `CompanyName` | `contact.Company` |
| `MobilePhoneNumber` | `contact.Phone` |

Client → server phone mapping prefers `MobilePhoneNumber`, then `BusinessPhoneNumber`, then `HomePhoneNumber`.

---

### `eastime.go` — shared EAS helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `easTime` | `(t time.Time, allDay bool) string` | Formats UTC time for MS-ASCAL/MS-ASEMAIL (`20060102T150405Z` or `20060102T000000Z`). |
| `parseEasTime` | `(s string) (time.Time, bool)` | Parses EAS datetime strings to UTC. |
| `serverIDForUint` | `(id uint) string` | DB primary key → decimal ServerId (events, contacts). |
| `parseServerID` | `(serverID string) (uint, bool)` | ServerId → DB primary key. |

---

### `mailsync.go`

| Name | Kind | Description |
|------|------|-------------|
| `MailSyncEngine` | struct | IMAP-backed mail sync for `mail/{guid}` collections. |
| `NewMailSyncEngine` | func | `(cfg, store) *MailSyncEngine` |
| `SyncCollection` | method | Processes one mail collection: client commands, server changes, sync key + cache update. |
| `EstimateCount` | method | `(ctx, collectionID) (int32, error)` — IMAP message count for GetItemEstimate. |
| `PingChangedCollections` | method | `(ctx, dev, folders) (bool, []string, error)` — detects mail folder changes vs sync cache for Ping. |
| `applyClientCommands` | method | Applies client Change (Read) and Delete commands to IMAP. |
| `buildServerChanges` | method | Compares IMAP UIDs/flags with sync cache; builds Add/Change/Delete batch. |
| `buildSyncAdd` | method | Wraps envelope as Sync Add with Email ApplicationData. |
| `buildSyncChange` | method | Wraps updated envelope as Sync Change. |
| `envelopeToEmail` | method | Maps `internal/imap.Envelope` → `eas.Email` ApplicationData. |
| `parseMailCollectionID` | func | Extracts folder GUID from `mail/{guid}`. |
| `serverIDForUID` | func | IMAP UID → decimal ServerId string. |
| `parseServerUID` | func | ServerId string → IMAP UID. |
| `easDateFromEnvelope` | func | Envelope date string → MS-ASEMAIL `DateReceived` format. |
| `defaultSyncWindow` | const | Max items per Sync batch when `WindowSize` is unset (100). |

### `getitemestimate.go`

| Name | Kind | Description |
|------|------|-------------|
| `GetItemEstimateHandler` | struct | MS-ASCMD GetItemEstimate command handler. |
| `NewGetItemEstimateHandler` | func | `(mail, calendar, contacts) *GetItemEstimateHandler` |
| `Handle` | method | Returns approximate item counts per requested collection. |
| `estimate` | method | Per-collection count + status (mail, calendar, contacts). |

### `wbxmlutil.go`

| Name | Kind | Description |
|------|------|-------------|
| `marshalApplicationDataBody` | func | Marshals `AirSync.ApplicationData` and returns inner WBXML bytes for `RawElement`. |

### `sendmail.go` — `SendMailHandler`

| Name | Kind | Description |
|------|------|-------------|
| `SendMailHandler` | struct | MS-ASCMD SendMail command handler. |
| `NewSendMailHandler` | func | `(cfg *config.Config) *SendMailHandler` |
| `Handle` | method | Parses MIME from WBXML or raw body, sends via SMTP, optionally appends to Sent. |
| `parseSendMailBody` | func | Extracts RFC822 bytes and `SaveInSentItems` flag from SendMail payload. |
| `looksLikeRFC822` | func | Heuristic detection of raw MIME bodies. |
| `tryBase64MIME` | func | Attempts base64 decode of MIME field. |

### `meetingresponse.go` — `MeetingResponseHandler`

| Name | Kind | Description |
|------|------|-------------|
| `MeetingResponseHandler` | struct | MS-ASCMD MeetingResponse command handler. |
| `NewMeetingResponseHandler` | func | `(eventRepo *repository.EventRepo) *MeetingResponseHandler` |
| `Handle` | method | Maps UserResponse (1/2/3) to attendee PartStat and updates SQL event. Returns `<Result>` WBXML (MS-ASCMD code page 8). |
| `findEvent` | method | Resolves event by CalendarId (numeric ServerId) or RequestId (UID). |
| `userResponseToPartStat` | func | Maps MS-ASCMD UserResponse → iCalendar PARTSTAT string. |

### `search.go` — `SearchHandler`

| Name | Kind | Description |
|------|------|-------------|
| `SearchHandler` | struct | MS-ASCMD Search command handler (mailbox search). |
| `NewSearchHandler` | func | `(cfg, store) *SearchHandler` |
| `Handle` | method | IMAP SEARCH → `<Result>` list with LongId, Properties, Range, and Total. |
| `searchMailbox` | method | Builds Search.Result entries for a UID slice with body preview. |
| `parseSearchRange` | func | Parses `"start-end"` Range strings (default `0-99`). |

### `itemoperations.go` — `ItemOperationsHandler`

| Name | Kind | Description |
|------|------|-------------|
| `ItemOperationsHandler` | struct | MS-ASCMD ItemOperations command handler (Fetch). |
| `NewItemOperationsHandler` | func | `(cfg, store) *ItemOperationsHandler` |
| `Handle` | method | Processes Fetch requests; returns Properties with Email + AirSyncBase.Body. |
| `fetchOne` | method | Single Fetch for one mail collection + ServerId. |
| `resolveItemOpsIDs` | func | Resolves CollectionId/ServerId from explicit fields or Search LongId. |

### `emailfetch.go` — mail body helpers

| Name | Kind | Description |
|------|------|-------------|
| `easAirSyncBody` | struct | WBXML `AirSyncBase.Body` (Type, Data, Truncated, EstimatedDataSize). |
| `easMailFetchProps` | struct | WBXML `ItemOperations.Properties` (Email.* + Body + NativeBodyType). |
| `mailFetchOptions` | struct | Body type and truncation size for fetch operations. |
| `defaultMailFetchOptions` | func | Default HTML body, 2 MB truncation. |
| `mailFetchOptionsFromBodyPreference` | func | Maps client `BodyPreference` → `mailFetchOptions`. |
| `fetchMailProperties` | func | IMAP RFC822 fetch + MIME parse → `easMailFetchProps`. |
| `selectMailBody` | func | Chooses plain/HTML text; sets NativeBodyType. |
| `marshalContainerInner` | func | Strips outer WBXML wrapper tag; returns inner bytes. |
| `mailSearchPropertiesBytes` | func | Encodes properties for `Search.Result.Properties` RawElement. |

### `settings.go` — `SettingsHandler`

| Name | Kind | Description |
|------|------|-------------|
| `SettingsHandler` | struct | MS-ASCMD Settings command handler. |
| `NewSettingsHandler` | func | `() *SettingsHandler` |
| `Handle` | method | Returns UserInformation (email) and/or DeviceInformation (device type/id). |

### `imapconn.go`

| Name | Kind | Description |
|------|------|-------------|
| `imapConnect` | func | Opens authenticated IMAP session using EAS Basic Auth credentials from ctx. |
| `appendToSent` | func | Best-effort APPEND of sent RFC822 to the user's Sent mailbox. |

---

## 3. Sync state store (`internal/activesync/state`)

### `model.go`

| Name | Kind | Description |
|------|------|-------------|
| `EasDevice` | struct | Per-user device row: folder sync key, policy key, device type. |
| `EasFolderState` | struct | Per-device collection sync key + JSON sync cache (`MailSyncCache` for mail, `ItemSyncCache` for calendar/contacts). |
| `ImapFolderMapping` | struct | Stable mapping from IMAP mailbox path → EAS folder GUID. |

### `store.go`

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewStore` | `(db *gorm.DB) *Store` | Creates a GORM-backed sync state store. |
| `EnsureDevice` | `(userID, deviceID, deviceType) (*EasDevice, error)` | Returns existing device or creates a new row. |
| `SaveDevice` | `(dev *EasDevice) error` | Persists device changes. |
| `NextFolderSyncKey` | `(dev *EasDevice) string` | Increments and saves folder sync key. |
| `GetCollectionSyncKey` | `(deviceID, collectionID) (string, error)` | Returns collection sync key; defaults to `"0"`. |
| `SetCollectionSyncKey` | `(deviceID, collectionID, syncKey) error` | Upserts collection sync key. |
| `NextCollectionSyncKey` | `(deviceID, collectionID) (string, error)` | Increments and returns new collection sync key. |
| `FolderGUID` | `(userID, folderPath) (string, error)` | Returns stable GUID for an IMAP mailbox path (creates mapping if needed). |
| `GetFolderState` | `(deviceID, collectionID) (*EasFolderState, error)` | Returns folder state row or a zero-valued default. |
| `SaveFolderState` | `(st *EasFolderState) error` | Inserts or updates folder state (sync key + cache). |
| `FolderPathByGUID` | `(userID, guid) (string, error)` | Reverse lookup: GUID → IMAP mailbox path. |
| `LoadMailSyncCache` | `(raw string) MailSyncCache` | Decodes JSON sync cache; returns empty map on error. |
| `EncodeMailSyncCache` | `(c MailSyncCache) string` | Serializes sync cache to JSON. |
| `MailSyncCache` | struct | Map of ServerId → last-known UID and flags. |
| `MailSyncItem` | struct | One cached message: UID, Seen, Flagged. |
| `hashFolder` | func | FNV-1a hash for `(userID, folderPath)` when creating new GUIDs. |

### `itemcache.go` — calendar/contacts sync cache

Used by `CalendarSyncEngine` and `ContactsSyncEngine`. Stored in `EasFolderState.SyncCache` as JSON.

| Name | Kind | Description |
|------|------|-------------|
| `ItemSyncCache` | struct | Map of ServerId → `ItemSyncItem` (`json:"items"`). |
| `ItemSyncItem` | struct | `UpdatedAt int64` — last synced `model.Event.UpdatedAt` or `model.Contact.UpdatedAt` unix time. |
| `LoadItemSyncCache` | func | `(raw string) ItemSyncCache` — decode; empty map on blank/invalid JSON. |
| `EncodeItemSyncCache` | func | `(c ItemSyncCache) string` — serialize to JSON for DB storage. |

**Change detection logic:** for each DB row, if ServerId absent from cache → **Add**; if `UpdatedAt` differs → **Change**; if ServerId in cache but row deleted → **Delete**.

---

## 4. Server wiring (`internal/server`)

### `routes.go`

| Function | Signature | Description |
|----------|-----------|-------------|
| `registerActiveSyncRoutes` | `(e *echo.Echo, cfg, db)` | Mounts EAS + Autodiscover routes when `activesync.enabled` is true. |

Routes registered:

| Method | Path | Handler |
|--------|------|---------|
| OPTIONS | `/Microsoft-Server-ActiveSync` | `Handler.Options` |
| POST | `/Microsoft-Server-ActiveSync` | `Handler.Handle` |
| POST | `/autodiscover/autodiscover.xml` | `AutodiscoverHandler.MobileSync` |
| GET | `/.well-known/autodiscover/autodiscover.xml` | `AutodiscoverHandler.MobileSync` |

All routes use `middleware.EASAuth` (HTTP Basic Auth validated against IMAP).

### `middleware/eas_auth.go`

| Function | Signature | Description |
|----------|-----------|-------------|
| `EASAuth` | `(cfg *config.Config) echo.MiddlewareFunc` | Validates Basic Auth credentials via IMAP LOGIN; sets `eas_user` and `eas_password` on context. |

### `middleware/csrf.go`

CSRF middleware skips paths starting with `/Microsoft-Server-ActiveSync` and `/autodiscover` (EAS clients do not send CSRF tokens).

---

## 5. Config (`internal/config`)

### `ActiveSyncConfig`

| Field | Description |
|-------|-------------|
| `Enabled` | Master switch for EAS endpoints (`activesync.enabled`). |
| `Debug` | Log EAS dispatch errors to Echo logger. |
| `MaxPingIntervalSec` | Reserved for long-poll Ping (not yet enforced). |
| `MaxSyncWindowSize` | Reserved cap on Sync window size (not yet enforced). |
| `ProtocolVersion` | Value for `MS-Server-ActiveSync` header (default `"14.1"`). |

---

## 6. IMAP helpers (`internal/imap`)

Functions used by ActiveSync (existing package; one function added for EAS):

| Function | Signature | Description |
|----------|-----------|-------------|
| `Connect` | `(host, port, tls, timeout, user, pass, debug) (*Client, error)` | IMAP LOGIN wrapper used by EAS auth and mail sync. |
| `ListMailboxes` | `() ([]MailboxInfo, error)` | FolderSync: list all IMAP folders. |
| `SelectMailbox` | `(mailbox string) error` | SELECT before UID operations. |
| `FetchAllUIDs` | `() ([]imap.UID, error)` | **Added for EAS** — UID SEARCH ALL on selected mailbox. |
| `FetchEnvelopes` | `(uids []imap.UID) ([]Envelope, error)` | Mail Sync: headers + flags for message list. |
| `FetchRawMessage` | `(uid imap.UID) ([]byte, error)` | **Used by EAS** — BODY.PEEK[] RFC822 download for ItemOperations/Search. |
| `ParseMessage` | `(raw []byte) (*ParsedMessage, error)` | **Used by EAS** — decodes text/plain and text/html from RFC822. |
| `MarkSeen` | `(uid imap.UID) error` | Client Sync Change: set `\Seen`. |
| `MarkUnseen` | `(uid imap.UID) error` | Client Sync Change: clear `\Seen`. |
| `DeleteMessage` | `(uid imap.UID) error` | Client Sync Delete: `\Deleted` + EXPUNGE. |
| `MessageCount` | `(mailbox string) (uint32, error)` | GetItemEstimate: total messages in folder. |

---

## 7. Phase 3 — Calendar & contacts sync (detail)

### Collection ID routing (`sync.go` → `syncCollection`)

| Collection ID prefix | Engine | Backend | ServerId format |
|---------------------|--------|---------|-----------------|
| `mail/` | `MailSyncEngine` | IMAP | Decimal IMAP UID |
| `vevent/` | `CalendarSyncEngine` | SQL `events` (default calendar) | Decimal `event.ID` |
| `vcard/` | `ContactsSyncEngine` | SQL `contacts` | Decimal `contact.ID` |
| `vtodo/` | `syncCollectionStub` | Not implemented | — |

### Sync flow (calendar & contacts)

1. Validate client `SyncKey` against `EasFolderState.SyncKey` (status 3 if stale).
2. Apply client `Commands` (Add/Change/Delete) to SQL.
3. When `GetChanges != 0` or initial sync (`SyncKey == "0"`), build server Add/Change/Delete from cache diff.
4. Increment sync key; persist `ItemSyncCache` JSON in `EasFolderState.SyncCache`.

### AttendeeStatus mapping (calendar)

| iCalendar PARTSTAT | MS-ASCAL AttendeeStatus |
|--------------------|-------------------------|
| `NEEDS-ACTION` (default) | `0` |
| `TENTATIVE` | `2` |
| `ACCEPTED` | `3` |
| `DECLINED` | `4` |

### Tests (`calendarsync_test.go`)

| Test | Covers |
|------|--------|
| `TestEasTimeRoundTrip` | `easTime` / `parseEasTime` |
| `TestEventToAppointment` | `eventToAppointment` + attendees |
| `TestModelContactToEas` | `modelContactToEas` |
| `TestParseCollectionIDs` | `parseVEventCollectionID`, `parseVCardCollectionID` |
| `TestServerIDForUint` | `serverIDForUint` / `parseServerID` |

---

## 8. Phase 4 — SendMail, MeetingResponse, Search, Settings

### SendMail flow

1. `parseSendMailBody` — WBXML `<MIME>` (optional base64) or raw `message/rfc822` body.
2. `smtp.SendRaw` — SMTP AUTH with EAS credentials; STARTTLS (587) or SMTPS (465).
3. When `SaveInSentItems != 0`, `appendToSent` — IMAP APPEND to Sent folder.

### MeetingResponse flow

| UserResponse | PARTSTAT | Meaning |
|--------------|----------|---------|
| `1` | `ACCEPTED` | Accept |
| `2` | `DECLINED` | Decline |
| `3` | `TENTATIVE` | Tentative |

Event lookup: `CalendarId` as decimal `event.ID`, fallback `RequestId` as iCalendar UID.

### Search (Phase 5 update)

- Resolves `CollectionId` `mail/{guid}` → IMAP mailbox path via `state.Store.FolderPathByGUID`.
- Returns `<Result>` entries with `LongId`, `CollectionId`, and `Properties` (including body preview).
- Supports `Range` (default `0-99`) and `<Total>`.
- GAL / contact search not implemented.

### Settings limitations

- Supports `Get.UserInformation` and `Get.DeviceInformation`.
- OOF (Out of Office), DevicePassword, and certificate settings not implemented.

### Tests (`phase4_handlers_test.go`, `phase4_test.go`, `internal/smtp/raw_test.go`, `integration_test.go`)

| Test | Covers |
|------|--------|
| `TestSettingsHandlerDefaultUserInformation` | Settings default Get response |
| `TestSettingsHandlerDeviceInformation` | Settings DeviceInformation |
| `TestMeetingResponseUpdatesAttendeePartStat` | Accept → PartStat ACCEPTED in SQL |
| `TestParseSendMailBodyRawMIME` / `WBXML` / `Empty` | SendMail MIME extraction |
| `TestUserResponseToPartStat` | MeetingResponse mapping |
| `TestLooksLikeRFC822` | SendMail raw MIME detection |
| `TestCollectRecipients` / `TestFirstFromAddress` | SMTP recipient parsing |
| `TestIntegration*` (`-tags integration`) | Live HTTP against EAS server |

---

## 9. Phase 5 — ItemOperations, Ping long-poll, Search results

### ItemOperations Fetch flow

1. Client sends `<ItemOperations><Fetch>` with `CollectionId`, `ServerId`, optional `BodyPreference`.
2. `resolveItemOpsIDs` — also accepts `Search.LongId` format `mail/{guid}+{uid}`.
3. `fetchMailProperties` — IMAP `FetchRawMessage` + `ParseMessage` + envelope flags.
4. Response: `<Fetch><Status>1</Status><Properties>` with `Email.*` and `AirSyncBase.Body`.

**Not implemented:** `FileReference` attachment fetch, Move, EmptyFolderContents.

### Ping long-poll flow

1. Client sends folder IDs (`Ping.Id` = collection id, e.g. `mail/{guid}`) and `HeartbeatInterval`.
2. Server polls every **2 seconds** calling `MailSyncEngine.PingChangedCollections`.
3. Compares IMAP UID set + `\Seen`/`\Flagged` against `MailSyncCache` in `EasFolderState`.
4. **Status 2** + changed folder IDs, or **status 1** when heartbeat expires.

Calendar/contacts folders are not monitored yet.

### Body type constants (`emailfetch.go`)

| Value | Meaning |
|-------|---------|
| `1` | Plain text (`bodyTypePlain`) |
| `2` | HTML (`bodyTypeHTML`) |

### Tests (`phase5_test.go`)

| Test | Covers |
|------|--------|
| `TestSelectMailBodyPrefersHTML` | HTML vs plain selection |
| `TestSelectMailBodyPlainFallback` | HTML request with plain-only message |
| `TestResolveItemOpsIDsFromLongId` | LongId parsing |
| `TestParseSearchRange` | Search Range `"5-15"` |

---

## 10. SMTP helper (`internal/smtp`)

### `raw.go`

| Name | Kind | Description |
|------|------|-------------|
| `SendRaw` | func | `(cfg Config, user, pass string, raw []byte) ([]byte, error)` — sends RFC822 via SMTP. |
| `firstFromAddress` | func | Parses From header from `net/mail.Message`. |
| `collectRecipients` | func | Collects unique To/Cc/Bcc addresses. |
| `sendRawSTARTTLS` | func | SMTP on port 587 with optional STARTTLS. |
| `sendRawSMTPS` | func | Implicit TLS on port 465. |

---

## 11. View godoc locally

From the project root:

```bash
# Package overviews
go doc go-cubemail/internal/activesync
go doc go-cubemail/internal/activesync/commands
go doc go-cubemail/internal/activesync/state

# Key types (Phase 3–5)
go doc go-cubemail/internal/activesync/commands.CalendarSyncEngine
go doc go-cubemail/internal/activesync/commands.ContactsSyncEngine
go doc go-cubemail/internal/activesync/commands.SendMailHandler
go doc go-cubemail/internal/activesync/commands.MeetingResponseHandler
go doc go-cubemail/internal/activesync/commands.ItemOperationsHandler
go doc go-cubemail/internal/activesync/commands.fetchMailProperties
go doc go-cubemail/internal/activesync/commands.PingChangedCollections
go doc go-cubemail/internal/smtp.SendRaw
go doc go-cubemail/internal/activesync/state.ItemSyncCache

# Single functions
go doc go-cubemail/internal/activesync/commands.CalendarSyncEngine.SyncCollection
go doc go-cubemail/internal/activesync/commands.eventToAppointment
go doc go-cubemail/internal/activesync/commands.modelContactToEas
go doc go-cubemail/internal/activesync/state.LoadItemSyncCache
go doc go-cubemail/internal/activesync/commands.MailSyncEngine.SyncCollection
go doc go-cubemail/internal/activesync/state.Store.FolderGUID
go doc go-cubemail/internal/server/middleware.EASAuth

# Run unit tests (always)
go test ./internal/activesync/commands/... ./internal/smtp/...

# Run HTTP integration tests (live server required)
go test -tags integration -v ./internal/activesync/...
```

---

*Document version: 1.4 — Phase 5 ItemOperations, Ping long-poll, Search results.*
