# Calendar API Testing Guide

> **Project:** go-cubemail-vue  
> **Base URL:** `http://localhost:8080/api/v1`  
> **Auth:** Session cookie + CSRF token (same as contacts/mail API)

This guide explains how to test the calendar backend with `curl`, HTTPie, or any REST client.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Authentication Flow](#2-authentication-flow)
3. [Calendar Endpoints](#3-calendar-endpoints)
4. [Event Endpoints](#4-event-endpoints)
5. [Import / Export](#5-import--export)
6. [Full Test Script](#6-full-test-script)
7. [Error Responses](#7-error-responses)
8. [Database Verification](#8-database-verification)

---

## 1. Prerequisites

### 1.1 Run migrations

Create the calendar tables before testing:

```bash
cd go-cubemail-vue
go run . migrate
```

Expected output:

```
Running migrations...
Migrations completed.
```

### 1.2 Start the server

```bash
go run . serve
```

Default listen address: `http://localhost:8080`

### 1.3 Configure IMAP credentials

Edit `config.toml` (copy from `config.toml.example`) with a working IMAP account, or pass credentials at login time.

### 1.4 Tools

This guide uses `curl` and a cookie jar file. HTTPie works equally well.

```bash
COOKIE_JAR=/tmp/gorc-cookies.txt
BASE=http://localhost:8080/api/v1
```

---

## 2. Authentication Flow

All calendar routes require an authenticated session. Mutating requests (`POST`, `PUT`, `DELETE`) also require a CSRF token.

### Step 1 — Login

```bash
curl -s -c "$COOKIE_JAR" -X POST "$BASE/auth/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=you@example.com&password=yourpassword"
```

Success response:

```json
{"username":"you@example.com"}
```

### Step 2 — Obtain CSRF token

Any `GET` request sets the `csrf_token` cookie:

```bash
curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" "$BASE/auth/me"
```

Extract the token:

```bash
CSRF=$(grep csrf_token "$COOKIE_JAR" | awk '{print $7}')
echo "CSRF=$CSRF"
```

### Step 3 — Use cookies + CSRF on mutating requests

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"Work","color":"#e74c3c"}'
```

> **Tip:** After each mutating request, refresh the CSRF token with a GET if you receive `403 CSRF token invalid`.

---

## 3. Calendar Endpoints

### 3.1 List calendars

Auto-creates a default **Personal** calendar on first access.

```bash
curl -s -b "$COOKIE_JAR" "$BASE/calendar" | jq .
```

Example response:

```json
{
  "calendars": [
    {
      "id": 1,
      "name": "Personal",
      "color": "#3788d8",
      "is_default": true,
      "is_active": true,
      "include_in_free_busy": true,
      "sort_order": 0
    }
  ]
}
```

### 3.2 Create calendar

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "name": "Work",
    "color": "#e74c3c",
    "include_in_free_busy": true,
    "sort_order": 1
  }' | jq .
```

Example response (`201 Created`):

```json
{
  "id": 2,
  "name": "Work",
  "color": "#e74c3c",
  "is_default": false,
  "is_active": true,
  "include_in_free_busy": true,
  "sort_order": 1
}
```

### 3.3 Update calendar

```bash
curl -s -b "$COOKIE_JAR" -X PUT "$BASE/calendar/2" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "name": "Work Projects",
    "color": "#9b59b6"
  }' | jq .
```

### 3.4 Toggle calendar visibility

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar/activation" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"ids": [2], "active": false}' | jq .
```

Response:

```json
{"status":"ok"}
```

### 3.5 Delete calendar

Cannot delete the default calendar (`is_default: true`).

```bash
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/calendar/2" \
  -H "X-CSRF-Token: $CSRF" | jq .
```

---

## 4. Event Endpoints

### 4.1 List events in date range

Required query parameters: `start`, `end` (ISO 8601 / RFC 3339).

Optional: `calendar_ids` (comma-separated IDs).

```bash
curl -s -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-05-01T00:00:00Z&end=2026-06-01T00:00:00Z" | jq .
```

Filter by calendar:

```bash
curl -s -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-05-01T00:00:00Z&end=2026-06-01T00:00:00Z&calendar_ids=1" | jq .
```

Example response:

```json
{
  "events": [
    {
      "id": 1,
      "calendar_id": 1,
      "uid": "a1b2c3d4e5f6...@go-cubemail",
      "summary": "Team standup",
      "description": "Daily sync",
      "location": "Room A",
      "start_at": "2026-05-30T09:00:00Z",
      "end_at": "2026-05-30T09:30:00Z",
      "is_all_day": false,
      "is_transparent": false,
      "status": "CONFIRMED",
      "priority": 0,
      "classification": "PUBLIC",
      "categories": "",
      "organizer_name": "",
      "organizer_email": "",
      "rrule": "",
      "is_recurring": false,
      "color": "#3788d8",
      "attendees": [
        {
          "name": "Bob",
          "email": "bob@example.com",
          "partstat": "NEEDS-ACTION",
          "role": "REQ-PARTICIPANT",
          "rsvp": true
        }
      ]
    }
  ]
}
```

### 4.2 Create event

If `calendar_id` is omitted, the default **Personal** calendar is used.

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "calendar_id": 1,
    "summary": "Team standup",
    "description": "Daily sync",
    "location": "Room A",
    "start_at": "2026-05-30T09:00:00Z",
    "end_at": "2026-05-30T09:30:00Z",
    "is_all_day": false,
    "attendees": [
      {"name": "Bob", "email": "bob@example.com"}
    ]
  }' | jq .
```

Save the returned `id` for subsequent tests:

```bash
EVENT_ID=1
```

### 4.3 Get single event

```bash
curl -s -b "$COOKIE_JAR" "$BASE/events/$EVENT_ID" | jq .
```

### 4.4 Update event

```bash
curl -s -b "$COOKIE_JAR" -X PUT "$BASE/events/$EVENT_ID" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "summary": "Team standup (updated)",
    "start_at": "2026-05-30T09:00:00Z",
    "end_at": "2026-05-30T10:00:00Z",
    "location": "Room B"
  }' | jq .
```

### 4.5 Move event (drag-resize)

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/events/$EVENT_ID/move" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "start_at": "2026-05-30T14:00:00Z",
    "end_at": "2026-05-30T15:00:00Z",
    "calendar_id": 1
  }' | jq .
```

### 4.6 Create all-day event

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "calendar_id": 1,
    "summary": "Company holiday",
    "start_at": "2026-06-01T00:00:00Z",
    "end_at": "2026-06-02T00:00:00Z",
    "is_all_day": true
  }' | jq .
```

### 4.7 Delete event

```bash
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/events/$EVENT_ID" \
  -H "X-CSRF-Token: $CSRF" | jq .
```

Response:

```json
{"status":"ok"}
```

---

## 5. Import / Export

### 5.1 Export calendar as ICS

```bash
curl -s -b "$COOKIE_JAR" "$BASE/calendar/1/export" -o calendar-export.ics
head -20 calendar-export.ics
```

Expected content type: `text/calendar; charset=utf-8`

Example output:

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//go-cubemail//Calendar//EN
BEGIN:VEVENT
UID:abc123@go-cubemail
SUMMARY:Team standup
...
END:VEVENT
END:VCALENDAR
```

### 5.2 Import ICS file

Create a sample file:

```bash
cat > sample-event.ics <<'EOF'
BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:import-test@example.com
SUMMARY:Imported meeting
DTSTART:20260610T140000Z
DTEND:20260610T150000Z
END:VEVENT
END:VCALENDAR
EOF
```

Upload:

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar/1/import" \
  -H "X-CSRF-Token: $CSRF" \
  -F "file=@sample-event.ics" | jq .
```

Example response:

```json
{
  "imported": 1,
  "updated": 0,
  "skipped": 0,
  "total": 1
}
```

Re-importing the same file updates by UID:

```json
{
  "imported": 0,
  "updated": 1,
  "skipped": 0,
  "total": 1
}
```

---

## 6. Full Test Script

Save as `test-calendar-api.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://localhost:8080/api/v1}"
USER="${USER:-you@example.com}"
PASS="${PASS:-yourpassword}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/gorc-calendar-test.txt}"

rm -f "$COOKIE_JAR"

echo "== Login =="
curl -s -c "$COOKIE_JAR" -X POST "$BASE/auth/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=$USER&password=$PASS" | jq .

echo "== CSRF =="
curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" "$BASE/auth/me" > /dev/null
CSRF=$(grep csrf_token "$COOKIE_JAR" | awk '{print $7}')

echo "== List calendars =="
curl -s -b "$COOKIE_JAR" "$BASE/calendar" | jq .

echo "== Create calendar =="
CAL_ID=$(curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"API Test","color":"#2ecc71"}' | jq -r .id)
echo "Calendar ID: $CAL_ID"

echo "== Create event =="
EVENT_ID=$(curl -s -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d "{\"calendar_id\":$CAL_ID,\"summary\":\"API test event\",\"start_at\":\"2026-05-30T10:00:00Z\",\"end_at\":\"2026-05-30T11:00:00Z\"}" \
  | jq -r .id)
echo "Event ID: $EVENT_ID"

echo "== List events =="
curl -s -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-05-01T00:00:00Z&end=2026-06-01T00:00:00Z" | jq .

echo "== Move event =="
curl -s -b "$COOKIE_JAR" -X POST "$BASE/events/$EVENT_ID/move" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"start_at":"2026-05-30T15:00:00Z","end_at":"2026-05-30T16:00:00Z"}' | jq .

echo "== Export calendar =="
curl -s -b "$COOKIE_JAR" "$BASE/calendar/$CAL_ID/export" | head -5

echo "== Delete event =="
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/events/$EVENT_ID" \
  -H "X-CSRF-Token: $CSRF" | jq .

echo "== Delete calendar =="
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/calendar/$CAL_ID" \
  -H "X-CSRF-Token: $CSRF" | jq .

echo "== Done =="
```

Run:

```bash
chmod +x test-calendar-api.sh
USER=you@example.com PASS=yourpassword ./test-calendar-api.sh
```

---

## 7. Error Responses

| HTTP status | Example body | Cause |
|-------------|--------------|-------|
| `401` | `{"error":"Not authenticated"}` | Missing or expired session cookie |
| `403` | `{"error":"CSRF token invalid"}` | Missing/wrong `X-CSRF-Token` header |
| `400` | `{"error":"name is required"}` | Invalid request body |
| `400` | `{"error":"invalid start parameter"}` | Missing/invalid `start` or `end` on event list |
| `400` | `{"error":"cannot delete default calendar"}` | Attempt to delete Personal calendar |
| `404` | `{"error":"not found"}` | Calendar or event ID not found / wrong user |
| `500` | `{"error":"..."}` | Database or server error |

### Unauthenticated request

```bash
curl -s "$BASE/calendar"
```

Response:

```json
{"error":"Not authenticated"}
```

---

## 8. Database Verification

With SQLite default (`./data/app.db`):

```bash
sqlite3 ./data/app.db "SELECT id, name, is_default, is_active FROM calendars;"
sqlite3 ./data/app.db "SELECT id, calendar_id, summary, start_at, end_at FROM events;"
sqlite3 ./data/app.db "SELECT event_id, email, partstat FROM event_attendees;"
```

Tables created by migration:

| Table | Purpose |
|-------|---------|
| `calendars` | User calendar folders |
| `events` | Calendar events (VEVENT) |
| `event_attendees` | Event participants |

---

## API Reference Summary

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/v1/calendar` | List calendars (auto-creates default) |
| POST | `/api/v1/calendar` | Create calendar |
| PUT | `/api/v1/calendar/:id` | Update calendar |
| DELETE | `/api/v1/calendar/:id` | Delete calendar + events |
| POST | `/api/v1/calendar/activation` | Toggle `is_active` |
| GET | `/api/v1/calendar/:id/export` | Download `.ics` |
| POST | `/api/v1/calendar/:id/import` | Upload `.ics` (multipart `file`) |
| GET | `/api/v1/events?start=&end=` | List events in range |
| GET | `/api/v1/events/:id` | Get event |
| POST | `/api/v1/events` | Create event |
| PUT | `/api/v1/events/:id` | Update event |
| DELETE | `/api/v1/events/:id` | Delete event |
| POST | `/api/v1/events/:id/move` | Move/resize event |

---

*Document version: 1.0 — Calendar backend API testing guide for go-cubemail-vue.*
