# ActiveSync (EAS) — cURL Testing Guide

> **Project:** go-cubemail-vue  
> **Endpoint:** `http://localhost:8080/Microsoft-Server-ActiveSync`  
> **Auth:** HTTP Basic Auth (IMAP credentials) — **not** session cookies  
> **Related:** [ActiveSync Implementation Guide](ACTIVESYNC_IMPLEMENTATION.md)

---

## Prerequisites

1. Enable ActiveSync in `config.toml`:

```toml
[activesync]
enabled = true
debug = false
protocol_version = "14.1"
```

2. Run migrations (creates `eas_devices`, `eas_folder_states`, `imap_folder_mappings`):

```bash
go run . migrate
```

3. Start the server:

```bash
go run . serve
```

4. Set variables:

```bash
export BASE="http://localhost:8080"
export USER="you@example.com"
export PASS="your-imap-password"
export DEVICE_ID="testdevice000000000000000001"
export EAS="${BASE}/Microsoft-Server-ActiveSync"
```

---

## Authentication

EAS uses **HTTP Basic Auth** on every request (same IMAP user/password as web login).

```bash
curl -u "${USER}:${PASS}" ...
```

There is **no CSRF token** on ActiveSync routes.

---

## 1. OPTIONS — capability probe

Mobile clients call OPTIONS before the first POST.

```bash
curl -s -i -u "${USER}:${PASS}" -X OPTIONS "$EAS"
```

Expected headers:

```
MS-Server-ActiveSync: 14.1
MS-ASProtocolVersions: 2.5,12.0,12.1,14.0,14.1,16.0,16.1
MS-ASProtocolCommands: Sync,SendMail,FolderSync,...
Content-Type: application/vnd.ms-sync.wbxml
```

---

## 2. Provision

```bash
curl -s -u "${USER}:${PASS}" -X POST \
  "${EAS}?Cmd=Provision&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null \
  -o provision.wbxml
```

Response is **WBXML binary**. Verify non-empty file:

```bash
ls -la provision.wbxml
xxd provision.wbxml | head
```

---

## 3. FolderSync

Initial folder hierarchy (`SyncKey=0`):

```bash
curl -s -u "${USER}:${PASS}" -X POST \
  "${EAS}?Cmd=FolderSync&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null \
  -o foldersync.wbxml
```

Expected folders in response (decoded):

| ServerId | DisplayName | Type |
|----------|-------------|------|
| `mail/{guid}` | INBOX, Sent, … | 2, 5, … |
| `vevent/personal` | Calendar | 8 |
| `vtodo/personal` | Tasks | 7 |
| `vcard/personal` | Contacts | 9 |

---

## 4. Sync (stub — sync keys only)

```bash
curl -s -u "${USER}:${PASS}" -X POST \
  "${EAS}?Cmd=Sync&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null \
  -o sync.wbxml
```

Phase 2 stub returns updated collection sync keys without mail/calendar items yet.

---

## 5. Ping

```bash
curl -s -u "${USER}:${PASS}" -X POST \
  "${EAS}?Cmd=Ping&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null \
  -o ping.wbxml
```

Returns status **1** (no changes).

---

## 6. Autodiscover

```bash
curl -s -u "${USER}:${PASS}" -X POST "${BASE}/autodiscover/autodiscover.xml" \
  -H "Content-Type: text/xml" \
  -d "<?xml version=\"1.0\" encoding=\"utf-8\"?>
<Autodiscover xmlns=\"http://schemas.microsoft.com/exchange/autodiscover/outlook/requestschema/2006\">
  <Request>
    <EMailAddress>${USER}</EMailAddress>
    <AcceptableResponseSchema>http://schemas.microsoft.com/exchange/autodiscover/mobilesync/responseschema/2006</AcceptableResponseSchema>
  </Request>
</Autodiscover>"
```

Expected XML contains:

```xml
<RootUrl>http://localhost:8080/Microsoft-Server-ActiveSync</RootUrl>
```

Well-known path:

```bash
curl -s -u "${USER}:${PASS}" "${BASE}/.well-known/autodiscover/autodiscover.xml" \
  -H "Content-Type: text/xml" \
  -d "..." 
```

---

## 7. Smoke test script

```bash
chmod +x scripts/test-activesync-curl.sh
USER=you@example.com PASS=secret ./scripts/test-activesync-curl.sh
```

---

## 8. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| HTTP 404 | `[activesync] enabled = false` | Set `enabled = true` |
| HTTP 401 | Wrong IMAP credentials | Check user/pass and `config.toml` IMAP |
| HTTP 500 on POST | Missing `DeviceId` query param | Add `DeviceId=` to URL |
| Empty FolderSync | IMAP unreachable | Verify IMAP host/port in config |
| `no such table: eas_devices` | Migration not run | `go run . migrate` |

### Verify database

```bash
sqlite3 ./data/app.db "SELECT id, device_id, folder_sync_key FROM eas_devices;"
sqlite3 ./data/app.db "SELECT collection_id, sync_key FROM eas_folder_states;"
```

---

## Implemented commands (current)

| Command | Status |
|---------|--------|
| OPTIONS | ✅ |
| Provision | ✅ |
| FolderSync | ✅ (IMAP mail + calendar/contacts/task folders) |
| Ping | ✅ (immediate no-change response) |
| Sync | ✅ (mail, calendar, contacts; vtodo stub) |
| GetItemEstimate | ✅ (mail, calendar, contacts counts) |
| SendMail | ✅ (SMTP + optional Sent append) |
| MeetingResponse | ✅ (PartStat update; no iMIP reply) |
| Search | ⚠️ (Result list + Total; no GAL) |
| Settings | ⚠️ (UserInformation + DeviceInformation) |
| ItemOperations | ⚠️ (mail Fetch + body) |
| Ping | ⚠️ (long-poll mail folders) |
| ItemOperations (advanced) | ❌ Attachments, Move, EmptyFolder |

---

## Integration tests (Go)

Automated HTTP integration tests live in `internal/activesync/integration_test.go` (build tag `integration`). They require a **running** go-cubemail server with valid IMAP credentials.

### Required environment

| Variable | Example | Purpose |
|----------|---------|---------|
| `EAS_INTEGRATION_URL` | `http://localhost:8080/Microsoft-Server-ActiveSync` | EAS endpoint base URL |
| `EAS_INTEGRATION_USER` | `you@example.com` | Basic Auth / IMAP user |
| `EAS_INTEGRATION_PASS` | `secret` | Basic Auth / IMAP password |

### Optional environment

| Variable | Purpose |
|----------|---------|
| `EAS_INTEGRATION_DEVICE_ID` | DeviceId query param (default: `integration-test-device-001`) |
| `EAS_INTEGRATION_SEND_TO` | Enable SendMail tests (recipient address) |
| `EAS_INTEGRATION_EVENT_ID` | MeetingResponse test — decimal event ServerId |
| `EAS_INTEGRATION_EVENT_UID` | MeetingResponse test — iCalendar UID |
| `EAS_INTEGRATION_SEARCH_QUERY` | Free-text for Search test |
| `EAS_INTEGRATION_SKIP_SEARCH=1` | Skip Search test |

### Run

```bash
# 1. Start server with activesync.enabled = true and run migrations
go run . migrate
go run . serve

# 2. In another terminal — always-run unit tests (no live server):
go test ./internal/activesync/commands/... ./internal/smtp/...

# 3. HTTP integration tests against live server:
EAS_INTEGRATION_URL=http://localhost:8080/Microsoft-Server-ActiveSync \
EAS_INTEGRATION_USER=you@example.com \
EAS_INTEGRATION_PASS=secret \
go test -tags integration -v ./internal/activesync/...

# Or use the helper script:
EAS_INTEGRATION_URL=http://localhost:8080/Microsoft-Server-ActiveSync \
EAS_INTEGRATION_USER=you@example.com \
EAS_INTEGRATION_PASS=secret \
EAS_INTEGRATION_SEND_TO=recipient@example.com \
./scripts/test-activesync-integration.sh
```

### Tests included

| Test | Requires |
|------|----------|
| `TestIntegrationOPTIONS` | URL + credentials |
| `TestIntegrationProvision` | URL + credentials |
| `TestIntegrationSettings` | URL + credentials |
| `TestIntegrationPing` | URL + credentials |
| `TestIntegrationSearch` | URL + credentials + IMAP mailbox |
| `TestIntegrationSendMail` | `EAS_INTEGRATION_SEND_TO` + working SMTP |
| `TestIntegrationSendMailRawMIME` | `EAS_INTEGRATION_SEND_TO` + working SMTP |
| `TestIntegrationMeetingResponse` | `EAS_INTEGRATION_EVENT_ID` or `_EVENT_UID` |

Unit tests for Phase 4 handlers (no live server): `internal/activesync/commands/phase4_handlers_test.go`.

---

*Document version: 1.2 — ActiveSync server phase 0–4 testing guide (unit + integration).*
