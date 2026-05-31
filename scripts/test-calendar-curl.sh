#!/usr/bin/env bash
# End-to-end calendar API test using curl.
# Usage: USER=you@example.com PASS=secret ./scripts/test-calendar-curl.sh

set -euo pipefail

BASE="${BASE:-http://localhost:8080/api/v1}"
USER="${USER:-you@example.com}"
PASS="${PASS:-yourpassword}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/gorc-calendar-curl.txt}"

rm -f "$COOKIE_JAR"

echo "=== 0. Fetch CSRF cookie ==="
curl -sf -c "$COOKIE_JAR" "$BASE/auth/me" > /dev/null || true
CSRF=$(grep csrf_token "$COOKIE_JAR" | awk '{print $7}')
echo "CSRF token: $CSRF"

echo "=== 1. Login ==="
curl -sf -b "$COOKIE_JAR" -c "$COOKIE_JAR" -X POST "$BASE/auth/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-CSRF-Token: $CSRF" \
  -d "username=$USER&password=$PASS" | jq .

echo "=== 2. Refresh CSRF after login ==="
curl -sf -b "$COOKIE_JAR" -c "$COOKIE_JAR" "$BASE/auth/me" > /dev/null || true
CSRF=$(grep csrf_token "$COOKIE_JAR" | awk '{print $7}')
echo "CSRF token: $CSRF"

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
