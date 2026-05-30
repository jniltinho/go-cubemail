# Implementing an ActiveSync Server in Go — A Phased Guide Based on Horde ActiveSync (PHP)

**Version:** 1.0  
**Date:** 2026-05-30  
**Audience:** Backend developers implementing or extending the go-cubemail-vue ActiveSync server  
**Reference implementation:** [Horde ActiveSync](../../php/ActiveSync/) (GPL-2.0, derived from Z-Push)  
**Target codebase:** `go-cubemail-vue/internal/activesync/`

---

## Table of Contents

1. [Purpose and Scope](#1-purpose-and-scope)
2. [Specifications and External References](#2-specifications-and-external-references)
3. [Horde ActiveSync — Architectural Analysis](#3-horde-activesync--architectural-analysis)
4. [Request Lifecycle (End-to-End)](#4-request-lifecycle-end-to-end)
5. [Layer Mapping: PHP Horde → Go](#5-layer-mapping-php-horde--go)
6. [Recommended Go Package Layout](#6-recommended-go-package-layout)
7. [Core Abstractions to Port First](#7-core-abstractions-to-port-first)
8. [Phased Implementation Plan](#8-phased-implementation-plan)
9. [Command-by-Command Reference Table](#9-command-by-command-reference-table)
10. [State and Persistence Design](#10-state-and-persistence-design)
11. [WBXML, Messages, and IMAP Integration](#11-wbxml-messages-and-imap-integration)
12. [Testing Strategy per Phase](#12-testing-strategy-per-phase)
13. [Current Status in go-cubemail-vue](#13-current-status-in-go-cubemail-vue)
14. [Anti-Patterns and Common Pitfalls](#14-anti-patterns-and-common-pitfalls)
15. [Related Documentation](#15-related-documentation)

---

## 1. Purpose and Scope

This document explains **how to implement a Microsoft Exchange ActiveSync (EAS) server in Go** by studying and translating the design of **Horde ActiveSync** (`php/ActiveSync`). It is not a line-by-line port; it is an **architectural blueprint** with concrete phases, deliverables, and acceptance criteria.

### Goals

- Provide a **faithful mental model** of Horde’s layering (HTTP → Request handler → Collections/State → Driver → backend).
- Define **phases** that can be implemented incrementally, each producing a testable milestone.
- Map every Horde class/file to a **Go package or interface** so developers know where to look when stuck.
- Align with what already exists in `internal/activesync/` and highlight gaps.

### Non-goals

- Supporting every EAS version (16.x, multipart, SmartForward) in the first release.
- Replacing Horde’s full SQL schema on day one (SQLite/GORM MVP is acceptable).
- Documenting MS-AS* protocol field-by-field (use Microsoft specs for that).

### Success criteria (project-level)

A mobile client (iOS Mail, Outlook, Samsung Email) can:

1. Autodiscover the server and authenticate.
2. Provision policies and complete FolderSync.
3. Sync mail (and optionally calendar/contacts) with Ping-driven updates.
4. Send mail and perform basic search/item fetch.

---

## 2. Specifications and External References

| Spec | Role |
|------|------|
| [MS-ASHTTP](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-ashttp/) | HTTP headers, query params (`Cmd`, `User`, `DeviceId`, `DeviceType`) |
| [MS-ASWBXML](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-aswbxml/) | WBXML code pages and tag names |
| [MS-ASCMD](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-ascmd/) | Command semantics (Sync, Ping, Provision, …) |
| [MS-ASPROV](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-asprov/) | Device provisioning |
| [MS-ASTA](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-asta/) | Tasks (vtodo) |
| [MS-ASCAL](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-ascal/) | Calendar |
| [MS-ASCON](https://learn.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-ascon/) | Contacts |

**WBXML library in Go:** `github.com/remdev/go-activesync/wbxml` (already used in this project).

**Test clients:**

- `github.com/remdev/go-activesync` — EAS 14.1 client (good for integration tests).
- `github.com/hstern/go-activesync` — EAS 16.1 client with fixtures.

---

## 3. Horde ActiveSync — Architectural Analysis

Horde ActiveSync is a **protocol engine** separated from mail/calendar storage. Storage is accessed only through a **Driver** interface. The engine handles WBXML, device state, sync keys, collections, and command dispatch.

### 3.1 Layer diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  HTTP entry (ActiveSync.php / front controller)                 │
│  - Validates MS-ASHTTP headers                                  │
│  - Basic auth, device id/type, protocol version                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  Horde_ActiveSync (facade)                                      │
│  - handleRequest(): auth → device → instantiate Request_*       │
│  - getCollectionsObject(), checkGlobalError()                   │
│  - Encoder/Decoder (WBXML)                                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  Horde_ActiveSync_Request_<Command>  (22 handlers)              │
│  - Extends Request_Base or Request_SyncBase                     │
│  - Parses WBXML request, builds WBXML response                  │
│  - Uses Collections + State + Driver                            │
└──────────────┬─────────────────────────────┬────────────────────┘
               │                             │
┌──────────────▼──────────────┐   ┌──────────▼──────────────────────┐
│  Collections + SyncCache     │   │  State (SQL)                   │
│  - Per-device folder list    │   │  - sync keys, device registry  │
│  - Heartbeat for Ping        │   │  - collection cache blob       │
│  - Hierarchy vs pingable     │   │  - mailmap (UID ↔ sync state)  │
└──────────────┬──────────────┘   └──────────┬──────────────────────┘
               │                             │
               └──────────────┬──────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│  Horde_ActiveSync_Driver_Base (abstract backend)                │
│  - getFolderHierarchy(), getServerChanges(), getMessage()       │
│  - sendMail(), meetingResponse(), getSearchResults(), …         │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│  Concrete driver (e.g. IMAP + CalDAV + LDAP in SOGo/Horde)      │
│  Horde_ActiveSync_Imap_* — message building, MODSEQ, attachments│
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Key classes and responsibilities

| PHP class / file | Responsibility |
|------------------|----------------|
| `lib/Horde/ActiveSync.php` | Main facade: constants (versions, folder types), `handleRequest()`, global errors, factory for Collections |
| `lib/Horde/ActiveSync/Driver/Base.php` | **Backend contract** — ~25 abstract methods every command ultimately calls |
| `lib/Horde/ActiveSync/State/Base.php` | Persist device id, sync keys, collection cache, user ↔ device mapping |
| `lib/Horde/ActiveSync/Collections.php` | In-memory view of monitored folders; heartbeat; pingable vs hierarchy flags |
| `lib/Horde/ActiveSync/SyncCache.php` | Serialize/deserialize collection state between requests |
| `lib/Horde/ActiveSync/Request/Base.php` | Shared request logic: logging, provision-required WBXML, pairing cleanup (`validate` device id) |
| `lib/Horde/ActiveSync/Request/SyncBase.php` | Sync-specific: BodyPreference, truncation, GetChanges loop, status codes |
| `lib/Horde/ActiveSync/Request/*.php` | One class per EAS command (Provision, FolderSync, Sync, Ping, …) |
| `lib/Horde/ActiveSync/Message/*.php` | Typed property bags with `_mapping` arrays → WBXML encode/decode |
| `lib/Horde/ActiveSync/Imap/EasMessageBuilder/*.php` | Build `AirSyncBaseBody`, attachments from IMAP MIME |
| `lib/Horde/ActiveSync/Imap/Message.php` | Parse IMAP message structure for ActiveSync |
| `lib/Horde/ActiveSync/Imap/Adapter.php` | IMAP connection abstraction |
| `lib/Horde/ActiveSync/Imap/Strategy/Modseq.php` | CONDSTORE/MODSEQ-based change detection |
| `migration/*` | SQL tables: `horde_activesync_state`, `_map`, `_device`, `_device_users`, `_mailmap` |

### 3.3 Design principles worth copying

1. **Thin HTTP, fat command handlers** — Each EAS command is an isolated unit with its own status constants.
2. **Driver boundary** — Never let IMAP/SMTP/CalDAV leak into WBXML code; handlers call Driver methods.
3. **Collections as first-class state** — Ping and Sync share the same folder/collection model; do not treat Ping as a separate hack.
4. **Sync keys are opaque to the client but structured server-side** — Horde increments keys per collection change window.
5. **Message objects mirror WBXML** — `Horde_ActiveSync_Message_Mail` knows how to encode itself; avoids scattered tag strings.
6. **Incremental sync via change lists** — `getServerChanges($folder, $syncKey, …)` returns adds/updates/deletes + new key.
7. **Global error short-circuit** — `checkGlobalError()` can force re-provision or FolderSync before handling body.

### 3.4 Horde request handlers (complete list)

Located under `lib/Horde/ActiveSync/Request/`:

| Handler | EAS command |
|---------|-------------|
| `Provision.php` | Provision |
| `FolderSync.php` | FolderSync |
| `FolderCreate.php` | FolderCreate |
| `FolderDelete.php` | FolderDelete |
| `FolderUpdate.php` | FolderUpdate |
| `Sync.php` | Sync |
| `Ping.php` | Ping |
| `GetItemEstimate.php` | GetItemEstimate |
| `ItemOperations.php` | ItemOperations |
| `SendMail.php` | SendMail |
| `SmartForward.php` | SmartForward |
| `SmartReply.php` | SmartReply |
| `MoveItems.php` | MoveItems |
| `MeetingResponse.php` | MeetingResponse |
| `Search.php` | Search |
| `Settings.php` | Settings |
| `ResolveRecipients.php` | ResolveRecipients |
| `ValidateCert.php` | ValidateCert |

(`SyncBase.php` and `Base.php` are base classes, not commands.)

---

## 4. Request Lifecycle (End-to-End)

Understanding one Sync request in Horde clarifies the whole architecture.

```
Client POST /Microsoft-Server-ActiveSync?Cmd=Sync&User=...&DeviceId=...&DeviceType=...
    │
    ├─► HTTP layer: MS-ASProtocolVersion, Authorization, Content-Type application/vnd.ms-sync.wbxml
    │
    ├─► Horde_ActiveSync::handleRequest()
    │       ├─ Authenticate user (HTTP Basic → Driver)
    │       ├─ Load/create device in State
    │       └─ new Horde_ActiveSync_Request_Sync(...)->handle()
    │
    ├─► Request_Sync::_handle()
    │       ├─ checkGlobalError() → may return status 142/144 (provision/folder sync required)
    │       ├─ Decode WBXML: Collections[] with SyncKey, CollectionId, Options (BodyPreference, FilterType)
    │       ├─ Collections: validate keys against State
    │       ├─ For each collection:
    │       │     Driver::getServerChanges() → change list
    │       │     For each change: Driver::getMessage() → Horde_ActiveSync_Message_*
    │       │     Message::encode() → WBXML fragment
    │       ├─ Update sync keys in State + SyncCache
    │       └─ Return application/vnd.ms-sync.wbxml
    │
    └─► Client applies changes; if Ping previously returned STATUS_NEEDSYNC(2), this Sync clears the pending flag
```

**Ping lifecycle (simplified):**

1. Client sends folder list + HeartbeatInterval.
2. Server stores pingable collections in cache.
3. Server loops until heartbeat expires or Driver detects change (`getServerChanges` with quick check / MODSEQ / cache diff).
4. Returns `Status=1` (no changes) or `Status=2` (need Sync).

See `Request/Ping.php` lines 98–175 for empty-PING handling, heartbeat bounds, and `STATUS_FOLDERSYNCREQD`.

---

## 5. Layer Mapping: PHP Horde → Go

| Horde (PHP) | Go (recommended) | Current in go-cubemail-vue |
|-------------|------------------|----------------------------|
| `ActiveSync.php` | `internal/activesync/handler.go` | ✅ `Handler`, routing |
| Autodiscover XML | `internal/activesync/autodiscover.go` | ✅ |
| `Request_*` | `internal/activesync/commands/*.go` | ✅ partial (11 commands) |
| `Driver/Base.php` | `internal/activesync/driver/driver.go` (interface) | ⚠️ implicit in handlers |
| IMAP backend | `commands/imapconn.go`, `mailsync.go` | ✅ MVP |
| `State/Base.php` | `internal/activesync/state/store.go`, `model.go` | ✅ SQLite/GORM |
| `Collections.php` | `state/itemcache.go` + future `collections.go` | ⚠️ partial |
| `SyncCache.php` | embedded in `state/store.go` | ⚠️ partial |
| `Message/*` | inline WBXML in handlers + future `message/` pkg | ⚠️ minimal |
| `Imap/EasMessageBuilder/*` | `commands/emailfetch.go` | ✅ Phase 5 start |
| Encoder/Decoder | `github.com/remdev/go-activesync/wbxml` | ✅ |
| SMTP send | `internal/smtp/raw.go` | ✅ |

### Recommended next refactor

Extract a formal **`Driver` interface** mirroring `Horde_ActiveSync_Driver_Base` so command handlers stop calling IMAP directly. This matches Horde’s separation and simplifies calendar/contacts backends.

---

## 6. Recommended Go Package Layout

```
internal/activesync/
├── handler.go              # HTTP entry (Horde_ActiveSync::handleRequest)
├── autodiscover.go
├── context.go              # Per-request: user, device, version, logger
├── commands/
│   ├── dispatcher.go       # Cmd → handler registry
│   ├── provision.go
│   ├── foldersync.go
│   ├── sync.go             # dispatches by collection class
│   ├── mailsync.go
│   ├── calendarsync.go
│   ├── contactssync.go
│   ├── ping.go
│   ├── getitemestimate.go
│   ├── itemoperations.go
│   ├── sendmail.go
│   ├── meetingresponse.go
│   ├── search.go
│   ├── settings.go
│   └── ...                 # future commands
├── driver/
│   └── driver.go           # interface + shared types (Change, Folder, MessageRef)
├── state/
│   ├── model.go            # Device, Collection, SyncKey
│   ├── store.go            # GORM persistence
│   ├── collections.go      # (new) pingable collections, heartbeat
│   └── mailmap.go          # (new) UID mapping like horde_activesync_mailmap
├── message/
│   ├── mail.go             # (future) typed Mail message + Encode
│   ├── appointment.go
│   └── contact.go
└── imap/
    ├── adapter.go          # (future) move from imapconn.go
    └── builder.go          # AirSyncBaseBody from MIME
```

Keep **English-only** identifiers and comments per project policy.

---

## 7. Core Abstractions to Port First

Before adding more commands, implement these four abstractions (order matters).

### 7.1 Device and protocol context

**Horde:** `$this->_device` (id, type, version), `$this->_state->getDevice()`.

**Go:** Extend `commands.Context` with:

- `DeviceID`, `DeviceType`, `ProtocolVersion`
- `Provisioned bool`, `PolicyKey string`

**Acceptance:** Provision updates policy key in state; subsequent Sync without key returns status 142.

### 7.2 State store

**Horde tables:**

- `horde_activesync_device` — device metadata
- `horde_activesync_device_users` — user ↔ device
- `horde_activesync_state` — sync keys per folder/collection
- `horde_activesync_map` — server id ↔ client id mapping
- `horde_activesync_mailmap` — IMAP UID + flags per synced message

**Go:** GORM models in `state/model.go`; add `MailMap` when mail flag/delete sync becomes reliable.

### 7.3 Collections manager

**Horde:** `Horde_ActiveSync_Collections` — tracks which folders are pingable, hierarchy stale flags, heartbeat.

**Go:** New `state/collections.go`:

- `LoadFromStore(deviceID)`
- `SetPingableFolders([]CollectionRef)`
- `NeedsFolderResync() bool`
- `GetHeartbeat() int`

**Acceptance:** Empty Ping with no collections returns status 3 (MISSING); stale hierarchy returns 7 (FOLDERSYNCREQD) — see Horde `Ping.php`.

### 7.4 Driver interface

Port method groups from `Driver/Base.php`:

```go
type Driver interface {
    Authenticate(user, pass string) error

    GetFolderHierarchy(syncKey string) (newKey string, folders []Folder, err error)

    GetServerChanges(collection CollectionRef, syncKey string, opts SyncOptions) (newKey string, changes []Change, err error)
    GetMessage(collection CollectionRef, serverID string, opts SyncOptions) (MessageEncoder, error)

    SendMail(raw []byte, saveToSent bool) error
    MeetingResponse(request MeetingResponseRequest) error

    FetchItem(collection CollectionRef, serverID string, opts FetchOptions) (FetchResult, error)
    Search(query SearchQuery) (SearchResult, error)

    GetHeartbeatConfig() HeartbeatConfig
    GetGlobalSettings() SettingsSnapshot
}
```

Handlers depend on `Driver`, not on IMAP types.

---

## 8. Phased Implementation Plan

Each phase has: **Objective**, **Horde reference**, **Go deliverables**, **Verification**, **Exit criteria**.

Phases 0–5 partially exist in go-cubemail-vue; phases 6+ are forward work.

---

### Phase 0 — HTTP shell and Autodiscover

**Objective:** Accept EAS traffic and respond to Autodiscover.

| Item | Detail |
|------|--------|
| Horde ref | HTTP front controller, Autodiscover XML templates |
| Go deliverables | Echo route `/Microsoft-Server-ActiveSync`, OPTIONS, Autodiscover POST |
| Headers | `MS-ASProtocolVersion`, `X-MS-PolicyKey`, `Content-Type: application/vnd.ms-sync.wbxml` |

**Verification:** `curl` OPTIONS; Autodiscover returns ASUrl.

**Exit criteria:** Client can discover server URL (see [ACTIVESYNC_CURL_TESTING.md](ACTIVESYNC_CURL_TESTING.md)).

**Status:** ✅ Done (`handler.go`, `autodiscover.go`).

---

### Phase 1 — WBXML infrastructure + Provision

**Objective:** Decode/encode WBXML; implement device policy handshake.

| Item | Detail |
|------|--------|
| Horde ref | `Request/Provision.php`, policy key in State |
| Go deliverables | `commands/provision.go`, `wbxmlutil.go`, policy key in `state/store.go` |
| Status codes | 1=OK, 2=invalid policy key, 3=unknown policy |

**Verification:** Unit test round-trip Provision request/response.

**Exit criteria:** Device stores policy key; handler sends `X-MS-PolicyKey` header.

**Status:** ✅ Done.

---

### Phase 2 — Authentication + FolderSync

**Objective:** IMAP Basic auth; return folder hierarchy with sync key.

| Item | Detail |
|------|--------|
| Horde ref | `Request/FolderSync.php`, `Driver::getFolderHierarchy()` |
| Go deliverables | `imapconn.go`, `foldersync.go`, folder types (Inbox=2, Drafts=3, …) |
| Edge cases | Wipe `validate` device id after pairing (`Request/Base.php::_cleanUpAfterPairing`) |

**Verification:** FolderSync returns Inbox/Sent/Drafts/Trash with new SyncKey.

**Exit criteria:** iOS/Android completes folder list step.

**Status:** ✅ Done.

---

### Phase 3 — Sync (mail) + GetItemEstimate

**Objective:** Incremental mail sync with adds/deletes/flag changes.

| Item | Detail |
|------|--------|
| Horde ref | `Request/Sync.php`, `SyncBase.php`, `Driver::getServerChanges()`, `Imap/Strategy/Modseq.php` |
| Go deliverables | `sync.go`, `mailsync.go`, `state/itemcache.go` |
| Options | FilterType, MIME support, truncation (start with headers + snippet) |

**Verification:** Two consecutive Syncs: first returns messages, second returns empty with new key unless IMAP changed.

**Exit criteria:** Mobile inbox lists messages.

**Status:** ✅ MVP done; truncation/BodyPreference incomplete vs Horde.

---

### Phase 4 — Write path and auxiliary commands

**Objective:** Send mail, meeting response, search, settings.

| Item | Detail |
|------|--------|
| Horde ref | `SendMail.php`, `MeetingResponse.php`, `Search.php`, `Settings.php` |
| Go deliverables | `sendmail.go`, `meetingresponse.go`, `search.go`, `settings.go`, `internal/smtp/raw.go` |
| Notes | MeetingResponse WBXML uses `<Result>` not `<Response>` (MS-ASWBXML code page 8) |

**Verification:** Unit tests in `phase4_handlers_test.go`; optional integration send.

**Exit criteria:** Client can send mail; Settings returns device-expected OOF stub or status.

**Status:** ✅ Done (Search/GAL/OOF advanced parts stubbed).

---

### Phase 5 — ItemOperations + real Ping + Search results

**Objective:** On-demand body fetch; long-poll Ping; Search with `<Result>` nodes.

| Item | Detail |
|------|--------|
| Horde ref | `ItemOperations.php`, `Ping.php`, `Search.php`, `Imap/EasMessageBuilder/*` |
| Go deliverables | `itemoperations.go`, `emailfetch.go`, enhanced `ping.go`, `search.go`, `mailsync.PingChangedCollections()` |
| Ping | Heartbeat loop, compare cache vs IMAP MODSEQ/UIDVALIDITY |

**Verification:** `phase5_test.go`, integration Ping timeout test.

**Exit criteria:** Client opens full message body via ItemOperations; Ping returns 2 when new mail arrives.

**Status:** ⚠️ Partial — Fetch body ✅; FileReference attachments ❌; calendar/contacts Ping ❌.

---

### Phase 6 — Collections hardening + mailmap + body in Sync

**Objective:** Match Horde reliability for flags, deletes, and truncated bodies in Sync.

| Item | Detail |
|------|--------|
| Horde ref | `Collections.php`, `SyncCache.php`, `migration/6_horde_activesync_addmailmap.php`, `SyncBase.php` BodyPreference |
| Go deliverables | `state/collections.go`, `state/mailmap.go`, body truncation in `mailsync.go`, `message/mail.go` |
| Driver | `GetServerChanges` returns proper `ChangeType` for flag updates |

**Verification:** Mark read on phone propagates to IMAP; delete on phone `\Deleted`; Sync returns `Truncation` when body too large.

**Exit criteria:** No duplicate messages after reconnect; read/unread stable across Sync cycles.

**Status:** ❌ Not started.

---

### Phase 7 — Calendar, contacts, tasks

**Objective:** Multi-collection Sync for non-mail folder types.

| Item | Detail |
|------|--------|
| Horde ref | `Message/Appointment.php`, `Contact.php`, `Task.php`, calendar/contact Driver methods |
| Go deliverables | Complete `calendarsync.go`, `contactssync.go`, new `tasksync.go`, CalDAV/CardDAV backend |
| Folder types | Appointment=4, Contacts=9, Tasks=7 |

**Verification:** Sync calendar folder; create/update event; Sync contacts.

**Exit criteria:** Mobile calendar and address book usable.

**Status:** ⚠️ Calendar/contacts sync skeleton exists; tasks ❌.

---

### Phase 8 — Remaining commands and enterprise features

**Objective:** Feature parity with common Horde deployments.

| Command | Horde ref | Priority |
|---------|-----------|----------|
| MoveItems | `MoveItems.php` | High |
| ResolveRecipients | `ResolveRecipients.php` | Medium |
| FolderCreate/Delete/Update | `FolderCreate.php`, … | Medium |
| SmartForward/SmartReply | `SmartForward.php`, `SmartReply.php` | Low |
| ValidateCert | `ValidateCert.php` | Low (EAS 12+) |
| GAL Search | `Search.php` + Driver | Medium |
| OOF Settings | `Settings.php` | Medium |
| Attachments (FileReference) | `ItemOperations.php`, `Message/Attachment.php` | High |

**Exit criteria:** Production checklist in [ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md](ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md) mostly green.

**Status:** ❌ Not started.

---

### Phase 9 — Protocol version upgrades (optional)

**Objective:** EAS 16.x, multipart responses, improved conflict handling.

| Item | Detail |
|------|--------|
| Reference | `hstern/go-activesync`, MS-ASHTTP multipart |
| Go deliverables | Version negotiation, optional `cmd/eas-test` CLI |

**Exit criteria:** Outlook mobile 16.x connects with feature downgrade gracefully.

**Status:** ❌ Not started.

---

## 9. Command-by-Command Reference Table

| Cmd | Horde Request class | Driver methods (typical) | go-cubemail status |
|-----|---------------------|--------------------------|-------------------|
| Provision | `Provision.php` | (policy only) | ✅ |
| FolderSync | `FolderSync.php` | `getFolderHierarchy` | ✅ |
| Sync | `Sync.php` | `getServerChanges`, `getMessage` | ✅ mail; ⚠️ cal/contacts |
| Ping | `Ping.php` | `getServerChanges` (quick) | ⚠️ mail only |
| GetItemEstimate | `GetItemEstimate.php` | `getServerChanges` (count) | ✅ |
| ItemOperations | `ItemOperations.php` | `getMessage`, `getAttachment` | ⚠️ Fetch only |
| SendMail | `SendMail.php` | `sendMail` | ✅ |
| MeetingResponse | `MeetingResponse.php` | `meetingResponse` | ✅ basic |
| Search | `Search.php` | `getSearchResults` | ⚠️ mailbox |
| Settings | `Settings.php` | `getGlobalSettings`, OOF | ⚠️ stub |
| MoveItems | `MoveItems.php` | `moveItems` | ❌ |
| ResolveRecipients | `ResolveRecipients.php` | `resolveRecipients` | ❌ |
| FolderCreate/Delete/Update | `Folder*.php` | folder CRUD | ❌ |

---

## 10. State and Persistence Design

### 10.1 Minimum viable schema (Go/GORM)

```
devices(id, device_id, device_type, protocol_version, policy_key, user_id)
collections(device_id, server_id, sync_key, folder_type, class, last_sync_at)
collection_cache(device_id, blob)          -- JSON or gob: pingable set + heartbeat
mailmap(device_id, folder_id, imap_uid, modseq, read, deleted)  -- Phase 6
```

### 10.2 Sync key strategy

Horde treats sync keys as **monotonic strings** per collection. On successful Sync:

1. Validate client key matches stored key (or `0` for initial).
2. Apply changes since that key.
3. Issue new key (e.g. increment integer or UUID generation).

**Do not** expose IMAP UIDVALIDITY directly as sync key — clients expect opaque keys.

### 10.3 Device pairing edge cases

| Case | Horde behavior | Implement in Go |
|------|----------------|-----------------|
| Android `DeviceId=validate` | Remove after FolderSync/Provision | `_cleanUpAfterPairing()` |
| Missing policy key | Status 142 on Sync | `checkGlobalError` equivalent |
| Stale folder hierarchy | Ping status 7 | `collections.NeedsFolderResync()` |

---

## 11. WBXML, Messages, and IMAP Integration

### 11.1 Message mapping pattern (Horde)

`Horde_ActiveSync_Message_Base` defines:

- `$_mapping` — property name → [code page, tag, type]
- `encode()` / `decode()` — walk mapping

**Go approach (Phase 6+):** Introduce typed structs with `Encode(*wbxml.Encoder)` methods to replace ad-hoc tag strings in `mailsync.go`.

### 11.2 Body building (Horde IMAP)

`Imap/EasMessageBuilder/` hierarchy:

- `Html.php` — prefers HTML part; falls back to plain
- Sets `AirSyncBaseBody`: Type, EstimatedDataSize, Truncated, Data
- Attachments via `getAttachments($version)`

**Go:** `emailfetch.go` already builds AirSyncBaseBody for ItemOperations; reuse same builder in Sync when `BodyPreference` requests body inline.

### 11.3 Change detection

Horde prefers **IMAP CONDSTORE/MODSEQ** when available (`Imap/Strategy/Modseq.php`). Fallback: UID tracking + periodic scan.

**Go:** Extend `imapconn.go` to store `{uidvalidity, highestmodseq}` per folder in state; Ping compares cached vs live.

---

## 12. Testing Strategy per Phase

| Phase | Unit tests | Integration tests | Manual |
|-------|------------|-------------------|--------|
| 0–1 | WBXML Provision round-trip | curl Autodiscover | — |
| 2 | FolderSync parse | IMAP auth + FolderSync HTTP | Android account setup |
| 3 | Sync key validation | `integration_test.go` Sync | Inbox listing |
| 4 | `phase4_handlers_test.go` | SendMail with `SEND_TO` env | Send from phone |
| 5 | `phase5_test.go` | Ping + ItemOperations HTTP | Open message body |
| 6+ | mailmap, truncation | flag/delete round-trip | Read/unread sync |
| 7 | cal/contact WBXML | CalDAV test container | Calendar sync |
| 8 | per-command | full script | GAL, MoveItems |

**Integration env vars:** `EAS_INTEGRATION_URL`, `EAS_INTEGRATION_USER`, `EAS_INTEGRATION_PASS` (see `integration_test.go`).

**External client library test:**

```go
// Example pattern with remdev/go-activesync (EAS 14.1)
client := activesync.NewClient(url, user, pass, deviceID, "iPhone", "14.1")
client.Provision()
client.FolderSync("")
client.Sync(inboxCollectionID, syncKey)
```

---

## 13. Current Status in go-cubemail-vue

| Phase | Status | Notes |
|-------|--------|-------|
| 0 | ✅ | Handler + Autodiscover |
| 1 | ✅ | Provision |
| 2 | ✅ | FolderSync + IMAP auth |
| 3 | ✅ | Mail Sync MVP |
| 4 | ✅ | SendMail, MeetingResponse, Search, Settings |
| 5 | ⚠️ | ItemOperations Fetch, Ping mail, Search results — attachments pending |
| 6 | ❌ | mailmap, Collections manager, Sync body truncation |
| 7 | ⚠️ | Calendar/contacts partial |
| 8 | ❌ | MoveItems, ResolveRecipients, folder CRUD, GAL |
| 9 | ❌ | EAS 16.x |

**Immediate recommended work (Phase 5 completion → Phase 6 start):**

1. FileReference in ItemOperations (attachments).
2. Formal `driver.Driver` interface; migrate handlers off direct IMAP.
3. `state/collections.go` for Ping status 3/7 correctness.
4. `mailmap` table for read/delete sync.

---

## 14. Anti-Patterns and Common Pitfalls

| Pitfall | Why it breaks clients | Horde lesson |
|---------|----------------------|--------------|
| Treating Ping as optional | Battery drain or no push | Collections cache must persist pingable folders |
| Using IMAP UID as ServerId without mapping | UIDVALIDITY change wipes client | Use stable ServerEntryId + mailmap |
| Wrong WBXML tag names | Silent parse failure | MeetingResponse: `Result` not `Response` |
| Ignoring empty Ping | Status 3 vs 7 confusion | Read `Ping.php` empty request branch |
| Huge body in Sync | Timeouts, OOM | Truncate in Sync; full body via ItemOperations |
| Skipping OPTIONS | Some clients refuse to connect | MS-ASHTTP requires proper OPTIONS response |
| No HTTPS in production | Autodiscover may fail | TLS required for real devices |

---

## 15. Related Documentation

| Document | Use when |
|----------|----------|
| [ACTIVESYNC_IMPLEMENTATION.md](ACTIVESYNC_IMPLEMENTATION.md) | Original step-by-step plan (phases 0–4 focus) |
| [ACTIVESYNC_GO_REFERENCE.md](ACTIVESYNC_GO_REFERENCE.md) | Go API / godoc for implemented functions |
| [ACTIVESYNC_CURL_TESTING.md](ACTIVESYNC_CURL_TESTING.md) | Manual HTTP testing |
| [ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md](ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md) | Live status, roadmap, PHP file index |
| [php/ActiveSync/lib/Horde/ActiveSync/Driver/Base.php](../../php/ActiveSync/lib/Horde/ActiveSync/Driver/Base.php) | Authoritative backend contract |

---

## Appendix A — Suggested reading order for new developers

1. MS-ASHTTP (1 hour) — headers and URL shape.
2. This document — architecture and phases.
3. `php/ActiveSync/lib/Horde/ActiveSync.php` — `handleRequest()`.
4. `php/ActiveSync/lib/Horde/ActiveSync/Request/Sync.php` + `SyncBase.php`.
5. `php/ActiveSync/lib/Horde/ActiveSync/Request/Ping.php`.
6. `php/ActiveSync/lib/Horde/ActiveSync/Driver/Base.php`.
7. `go-cubemail-vue/internal/activesync/handler.go` + `commands/dispatcher.go`.
8. Run integration tests with [ACTIVESYNC_CURL_TESTING.md](ACTIVESYNC_CURL_TESTING.md).

---

## Appendix B — Glossary

| Term | Meaning |
|------|---------|
| Collection | A synced folder + class (Email, Calendar, Contacts) + sync key |
| SyncKey | Opaque token; client must send last acknowledged key |
| ServerEntryId | Stable id for item in a collection (not always IMAP UID) |
| Heartbeat | Ping interval in seconds |
| BodyPreference | Client asks for plain/HTML/truncation in Sync options |
| Driver | Backend adapter abstracting IMAP/CalDAV/SMTP |
| WBXML | Binary XML used on the wire for all EAS commands |

---

*This guide is maintained alongside the Go implementation. When a phase is completed, update Section 13 and cross-link from [ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md](ACTIVESYNC_STATUS_ROADMAP_AND_PHP_REFERENCE.md).*
