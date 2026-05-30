# Calendar API — cURL Testing Guide

> **Project:** go-cubemail-vue  
> **Base URL:** `http://localhost:8080/api/v1`  
> **Auth:** Session cookie (`gorc_session`) + CSRF header on mutating requests  
> **Related:** [Go Function Reference](CALENDAR_GO_REFERENCE.md)

Step-by-step guide to test every calendar backend endpoint using **curl**.

---

## Table of Contents

1. [Setup](#1-setup)
2. [Authentication with curl](#2-authentication-with-curl)
3. [Calendar endpoints](#3-calendar-endpoints)
4. [Event endpoints](#4-event-endpoints)
5. [Import and export (ICS)](#5-import-and-export-ics)
6. [Complete test script](#6-complete-test-script)
7. [Troubleshooting](#7-troubleshooting)
8. [Quick reference](#8-quick-reference)

---

## 1. Setup

### 1.1 Install dependencies

- **curl** — pre-installed on most Linux/macOS systems
- **jq** (optional) — pretty-print JSON: `sudo apt install jq`

### 1.2 Prepare the server

```bash
cd go-cubemail-vue

# Copy and edit config if needed
cp config.toml.example config.toml

# Create database tables (calendars, events, event_attendees)
go run . migrate

# Start API server
go run . serve
```

Server listens on **port 8080** by default (`config.toml` → `[server] port`).

### 1.2 Environment variables for curl

```bash
export BASE="http://localhost:8080/api/v1"
export COOKIE_JAR="/tmp/gorc-cookies.txt"
export USER="you@example.com"
export PASS="your-imap-password"
```

---

## 2. Authentication with curl

Calendar routes require login. Mutating requests (`POST`, `PUT`, `DELETE`) also need a **CSRF token**.

### Step 1 — Login (save session cookie)

```bash
curl -s -c "$COOKIE_JAR" -X POST "$BASE/auth/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=${USER}&password=${PASS}"
```

**Expected (200):**

```json
{"username":"you@example.com"}
```

**On failure (401):**

```json
{"error":"Invalid credentials or server unreachable."}
```

### Step 2 — Obtain CSRF token

Any GET request sets the `csrf_token` cookie:

```bash
curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" "$BASE/auth/me"
```

Extract token into a shell variable:

```bash
export CSRF=$(grep csrf_token "$COOKIE_JAR" | awk '{print $7}')
echo "CSRF=$CSRF"
```

### Step 3 — Use both on every mutating request

```bash
curl -s -b "$COOKIE_JAR" \
  -H "X-CSRF-Token: $CSRF" \
  -H "Content-Type: application/json" \
  ...
```

> **Note:** If you get `403 CSRF token invalid`, run Step 2 again to refresh the token.

### Verify session

```bash
curl -s -b "$COOKIE_JAR" "$BASE/auth/me" | jq .
```

---

## 3. Calendar endpoints

### 3.1 List calendars

Auto-creates default **Personal** calendar on first call.

```bash
curl -s -b "$COOKIE_JAR" "$BASE/calendar" | jq .
```

**Expected (200):**

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

Save default calendar ID:

```bash
CAL_ID=$(curl -s -b "$COOKIE_JAR" "$BASE/calendar" | jq -r '.calendars[0].id')
echo "CAL_ID=$CAL_ID"
```

---

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

**Expected (201):** JSON object with `"name": "Work"` and new `"id"`.

**Validation error (400):**

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"name": ""}' | jq .
# {"error":"name is required"}
```

---

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

**Expected (200):** Updated calendar object.

---

### 3.4 Toggle visibility (activation)

Hide calendar from views:

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar/activation" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"ids": [2], "active": false}' | jq .
```

Show again:

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar/activation" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"ids": [2], "active": true}' | jq .
```

**Expected (200):** `{"status":"ok"}`

---

### 3.5 Delete calendar

Cannot delete the default calendar (`is_default: true`).

```bash
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/calendar/2" \
  -H "X-CSRF-Token: $CSRF" | jq .
```

**Expected (200):** `{"status":"ok"}`

**Blocked (400):**

```bash
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/calendar/1" \
  -H "X-CSRF-Token: $CSRF" | jq .
# {"error":"cannot delete default calendar"}
```

---

## 4. Event endpoints

### 4.1 List events (date range)

**Required query params:** `start`, `end` (ISO 8601 UTC).

```bash
curl -s -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-05-01T00:00:00Z&end=2026-06-01T00:00:00Z" | jq .
```

Filter by calendar:

```bash
curl -s -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-05-01T00:00:00Z&end=2026-06-01T00:00:00Z&calendar_ids=1" | jq .
```

**Expected (200):**

```json
{"events": []}
```

**Missing params (400):**

```bash
curl -s -b "$COOKIE_JAR" "$BASE/events" | jq .
# {"error":"invalid start parameter"}
```

---

### 4.2 Create event

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
    "attendees": [
      {"name": "Bob", "email": "bob@example.com"}
    ]
  }' | jq .
```

Save event ID:

```bash
EVENT_ID=$(curl -s -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "summary": "API test event",
    "start_at": "2026-05-30T10:00:00Z",
    "end_at": "2026-05-30T11:00:00Z"
  }' | jq -r .id)
echo "EVENT_ID=$EVENT_ID"
```

**Expected (201):** Event object with `"uid"`, `"color"`, `"attendees"`.

---

### 4.3 Get single event

```bash
curl -s -b "$COOKIE_JAR" "$BASE/events/$EVENT_ID" | jq .
```

**Not found (404):**

```bash
curl -s -b "$COOKIE_JAR" "$BASE/events/99999" | jq .
# {"error":"not found"}
```

---

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

---

### 4.5 Move event (drag-resize)

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/events/$EVENT_ID/move" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "start_at": "2026-05-30T14:00:00Z",
    "end_at": "2026-05-30T15:00:00Z"
  }' | jq .
```

Move to another calendar:

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

---

### 4.6 Create all-day event

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{
    "calendar_id": 1,
    "summary": "Holiday",
    "start_at": "2026-06-01T00:00:00Z",
    "end_at": "2026-06-02T00:00:00Z",
    "is_all_day": true
  }' | jq .
```

---

### 4.7 Delete event

```bash
curl -s -b "$COOKIE_JAR" -X DELETE "$BASE/events/$EVENT_ID" \
  -H "X-CSRF-Token: $CSRF" | jq .
```

**Expected (200):** `{"status":"ok"}`

---

## 5. Import and export (ICS)

### 5.1 Export calendar to file

```bash
curl -s -b "$COOKIE_JAR" "$BASE/calendar/1/export" -o calendar-export.ics
file calendar-export.ics
head -15 calendar-export.ics
```

**Expected:** `text/calendar` content starting with `BEGIN:VCALENDAR`.

---

### 5.2 Import ICS file

Create sample file:

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

Upload with curl multipart:

```bash
curl -s -b "$COOKIE_JAR" -X POST "$BASE/calendar/1/import" \
  -H "X-CSRF-Token: $CSRF" \
  -F "file=@sample-event.ics" | jq .
```

**Expected (200):**

```json
{"imported":1,"updated":0,"skipped":0,"total":1}
```

Re-import same file (upsert by UID):

```json
{"imported":0,"updated":1,"skipped":0,"total":1}
```

Verify imported event appears in range query:

```bash
curl -s -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-06-01T00:00:00Z&end=2026-07-01T00:00:00Z" | jq .
```

---

## 6. Complete test script

Save as `scripts/test-calendar-curl.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://localhost:8080/api/v1}"
USER="${USER:-you@example.com}"
PASS="${PASS:-yourpassword}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/gorc-calendar-curl.txt}"

rm -f "$COOKIE_JAR"

echo "=== 1. Login ==="
curl -sf -c "$COOKIE_JAR" -X POST "$BASE/auth/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=$USER&password=$PASS" | jq .

echo "=== 2. CSRF ==="
curl -sf -b "$COOKIE_JAR" -c "$COOKIE_JAR" "$BASE/auth/me" > /dev/null
CSRF=$(grep csrf_token "$COOKIE_JAR" | awk '{print $7}')

echo "=== 3. List calendars ==="
curl -sf -b "$COOKIE_JAR" "$BASE/calendar" | jq .

echo "=== 4. Create calendar ==="
CAL_ID=$(curl -sf -b "$COOKIE_JAR" -X POST "$BASE/calendar" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -d '{"name":"curl-test","color":"#2ecc71"}' | jq -r .id)
echo "Calendar ID: $CAL_ID"

echo "=== 5. Create event ==="
EVENT_ID=$(curl -sf -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -d "{\"calendar_id\":$CAL_ID,\"summary\":\"curl event\",\"start_at\":\"2026-05-30T10:00:00Z\",\"end_at\":\"2026-05-30T11:00:00Z\"}" \
  | jq -r .id)
echo "Event ID: $EVENT_ID"

echo "=== 6. List events ==="
curl -sf -b "$COOKIE_JAR" \
  "$BASE/events?start=2026-05-01T00:00:00Z&end=2026-06-01T00:00:00Z" | jq .

echo "=== 7. Move event ==="
curl -sf -b "$COOKIE_JAR" -X POST "$BASE/events/$EVENT_ID/move" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -d '{"start_at":"2026-05-30T15:00:00Z","end_at":"2026-05-30T16:00:00Z"}' | jq .

echo "=== 8. Export ICS ==="
curl -sf -b "$COOKIE_JAR" "$BASE/calendar/$CAL_ID/export" | head -5

echo "=== 9. Delete event ==="
curl -sf -b "$COOKIE_JAR" -X DELETE "$BASE/events/$EVENT_ID" \
  -H "X-CSRF-Token: $CSRF" | jq .

echo "=== 10. Delete calendar ==="
curl -sf -b "$COOKIE_JAR" -X DELETE "$BASE/calendar/$CAL_ID" \
  -H "X-CSRF-Token: $CSRF" | jq .

echo "=== All tests passed ==="
```

Run:

```bash
chmod +x scripts/test-calendar-curl.sh
USER=you@example.com PASS=secret ./scripts/test-calendar-curl.sh
```

---

## 7. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `{"error":"Not authenticated"}` | No session cookie | Run login (Step 1) |
| `{"error":"CSRF token invalid"}` | Missing/wrong header | Refresh CSRF (Step 2) |
| `{"error":"Invalid credentials..."}` | Wrong IMAP password/host | Check `config.toml` + credentials |
| `Connection refused` | Server not running | `go run . serve` |
| `no such table: calendars` | Migration not run | `go run . migrate` |
| Empty `"events": []` | Range does not overlap event times | Adjust `start`/`end` query params |

### Test without authentication (expect 401)

```bash
curl -s "$BASE/calendar" | jq .
```

### Inspect cookies

```bash
cat "$COOKIE_JAR"
```

### Check database (SQLite default)

```bash
sqlite3 ./data/app.db "SELECT id, name, is_default FROM calendars;"
sqlite3 ./data/app.db "SELECT id, summary, start_at FROM events;"
```

---

## 8. Quick reference

| Method | curl path | CSRF required |
|--------|-----------|---------------|
| GET | `$BASE/calendar` | No |
| POST | `$BASE/calendar` | Yes |
| PUT | `$BASE/calendar/:id` | Yes |
| DELETE | `$BASE/calendar/:id` | Yes |
| POST | `$BASE/calendar/activation` | Yes |
| GET | `$BASE/calendar/:id/export` | No |
| POST | `$BASE/calendar/:id/import` | Yes |
| GET | `$BASE/events?start=&end=` | No |
| GET | `$BASE/events/:id` | No |
| POST | `$BASE/events` | Yes |
| PUT | `$BASE/events/:id` | Yes |
| DELETE | `$BASE/events/:id` | Yes |
| POST | `$BASE/events/:id/move` | Yes |

**Minimal authenticated GET:**

```bash
curl -b "$COOKIE_JAR" "$BASE/calendar"
```

**Minimal authenticated POST:**

```bash
curl -b "$COOKIE_JAR" -X POST "$BASE/events" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"summary":"Test","start_at":"2026-05-30T10:00:00Z","end_at":"2026-05-30T11:00:00Z"}'
```

---

*Document version: 1.0 — Calendar API cURL testing guide for go-cubemail-vue.*
