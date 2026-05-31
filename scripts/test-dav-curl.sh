#!/usr/bin/env bash
# End-to-end CalDAV + CardDAV smoke test using curl.
# Usage: USER=you@example.com PASS=secret ./scripts/test-dav-curl.sh
#
# Auth: HTTP Basic Auth (same IMAP credentials — no CSRF cookie needed).
# Requires: curl, grep, sed, awk (standard), and optionally xmllint for pretty output.

set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
USER="${USER:-you@example.com}"
PASS="${PASS:-yourpassword}"
DAV="$BASE/dav"

# Encode user for URL path (replace @ with %40)
USER_PATH="${USER/@/%40}"

PASS_STATUS=0
FAIL_STATUS=0

ok()   { echo "  [PASS] $1"; PASS_STATUS=$((PASS_STATUS+1)); }
fail() { echo "  [FAIL] $1"; FAIL_STATUS=$((FAIL_STATUS+1)); }

check_status() {
  local label="$1" expected="$2" got="$3"
  if [ "$got" = "$expected" ]; then
    ok "$label (HTTP $got)"
  else
    fail "$label — expected HTTP $expected, got HTTP $got"
  fi
}

auth="-u ${USER}:${PASS}"
DEPTH1='-H "Depth: 1"'

dav_req() {
  # dav_req METHOD URL [-H extra] [--data body]  → prints HTTP status code
  local method="$1" url="$2"
  shift 2
  curl -s -o /dev/null -w "%{http_code}" $auth -X "$method" "$url" "$@"
}

dav_body() {
  # dav_body METHOD URL [extra curl args] → prints response body
  local method="$1" url="$2"
  shift 2
  curl -s $auth -X "$method" "$url" "$@"
}

separator() { echo; echo "════════════════════════════════════════════"; echo "  $1"; echo "════════════════════════════════════════════"; }

# ─────────────────────────────────────────────────────────────────────────────
separator "CalDAV — Calendar & Event Sync"
# ─────────────────────────────────────────────────────────────────────────────

echo
echo "--- 1. OPTIONS /.well-known/caldav ---"
STATUS=$(dav_req OPTIONS "$BASE/.well-known/caldav")
# Echo returns 204 when no explicit OPTIONS handler is registered; 200 when one is.
if [ "$STATUS" = "200" ] || [ "$STATUS" = "204" ]; then
  ok "OPTIONS well-known/caldav (HTTP $STATUS)"
else
  fail "OPTIONS well-known/caldav — expected 200 or 204, got $STATUS"
fi

DAV_HEADER=$(curl -s -X OPTIONS -D - -o /dev/null $auth "$DAV/$USER_PATH/" | grep -i "^DAV:" | tr -d '\r' || true)
if echo "$DAV_HEADER" | grep -qi "calendar"; then
  ok "DAV header advertises 'calendar' capability"
else
  fail "DAV header missing 'calendar' capability (got: $DAV_HEADER)"
fi

echo
echo "--- 2. Well-known redirect (GET) ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $auth "$BASE/.well-known/caldav")
if [ "$STATUS" = "301" ] || [ "$STATUS" = "302" ] || [ "$STATUS" = "207" ] || [ "$STATUS" = "204" ]; then
  ok "Well-known GET responds (HTTP $STATUS)"
else
  fail "Well-known GET — unexpected HTTP $STATUS"
fi

echo
echo "--- 3. PROPFIND principal /dav/${USER_PATH}/ ---"
STATUS=$(dav_req PROPFIND "$DAV/$USER_PATH/" \
  -H "Depth: 0" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop>
    <current-user-principal/>
    <calendar-home-set xmlns="urn:ietf:params:xml:ns:caldav"/>
  </prop>
</propfind>')
check_status "PROPFIND principal" "207" "$STATUS"

echo
echo "--- 4. PROPFIND calendar list /dav/${USER_PATH}/calendars/ ---"
CAL_RESPONSE=$(dav_body PROPFIND "$DAV/$USER_PATH/calendars/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<propfind xmlns="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <prop>
    <resourcetype/>
    <displayname/>
    <cs:getctag/>
    <c:supported-calendar-component-set/>
  </prop>
</propfind>')

HTTP_STATUS=$(echo "$CAL_RESPONSE" | grep -c "displayname" || true)
if [ "$HTTP_STATUS" -ge 1 ]; then
  ok "PROPFIND calendars — got displayname entries"
else
  fail "PROPFIND calendars — no displayname in response"
fi

# Extract first calendar slug from href
CAL_SLUG=$(echo "$CAL_RESPONSE" | grep -o '/calendars/[^/<"]*' | sed 's|/calendars/||' | grep -v '^$' | head -1 || true)
if [ -z "$CAL_SLUG" ]; then
  CAL_SLUG="personal"
  echo "  (no calendar slug found — using default '$CAL_SLUG')"
else
  echo "  Found calendar slug: $CAL_SLUG"
fi

echo
echo "--- 5. PROPFIND events in /dav/${USER_PATH}/calendars/${CAL_SLUG}/ ---"
STATUS=$(dav_req PROPFIND "$DAV/$USER_PATH/calendars/$CAL_SLUG/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop><getetag/><getcontenttype/></prop>
</propfind>')
check_status "PROPFIND events in calendar" "207" "$STATUS"

# ── Create a test event ───────────────────────────────────────────────────────
# UID must equal the URL filename (without .ics) — server uses TrimSuffix(".ics") to look up.
TEST_UID="curl-test-$(date +%s)"
TEST_ICS="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//go-cubemail//test//EN
BEGIN:VEVENT
UID:${TEST_UID}
SUMMARY:curl DAV test event
DTSTART:20260601T100000Z
DTEND:20260601T110000Z
DESCRIPTION:Created by test-dav-curl.sh
END:VEVENT
END:VCALENDAR"

echo
echo "--- 6. PUT event (create) ---"
STATUS=$(dav_req PUT "$DAV/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics" \
  -H "Content-Type: text/calendar; charset=utf-8" \
  --data "$TEST_ICS")
if [ "$STATUS" = "201" ] || [ "$STATUS" = "204" ]; then
  ok "PUT event — HTTP $STATUS"
else
  fail "PUT event — expected 201 or 204, got $STATUS"
fi

echo
echo "--- 7. GET event (read back) ---"
STATUS=$(dav_req GET "$DAV/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics")
check_status "GET event" "200" "$STATUS"

BODY=$(dav_body GET "$DAV/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics")
if echo "$BODY" | grep -q "$TEST_UID"; then
  ok "GET event body contains correct UID"
else
  fail "GET event body does not contain UID $TEST_UID"
fi

echo
echo "--- 8. PROPFIND event (ETag) ---"
STATUS=$(dav_req PROPFIND "$DAV/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics" \
  -H "Depth: 0" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?><propfind xmlns="DAV:"><prop><getetag/></prop></propfind>')
check_status "PROPFIND single event ETag" "207" "$STATUS"

echo
echo "--- 9. REPORT calendar-query ---"
STATUS=$(dav_req REPORT "$DAV/$USER_PATH/calendars/$CAL_SLUG/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<calendar-query xmlns="urn:ietf:params:xml:ns:caldav" xmlns:d="DAV:">
  <d:prop><d:getetag/><calendar-data/></d:prop>
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT"/>
    </comp-filter>
  </filter>
</calendar-query>')
check_status "REPORT calendar-query" "207" "$STATUS"

echo
echo "--- 10. REPORT calendar-multiget ---"
STATUS=$(dav_req REPORT "$DAV/$USER_PATH/calendars/$CAL_SLUG/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data "<?xml version=\"1.0\"?>
<calendar-multiget xmlns=\"urn:ietf:params:xml:ns:caldav\" xmlns:d=\"DAV:\">
  <d:prop><d:getetag/><calendar-data/></d:prop>
  <d:href>/dav/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics</d:href>
</calendar-multiget>")
check_status "REPORT calendar-multiget" "207" "$STATUS"

echo
echo "--- 11. DELETE event ---"
STATUS=$(dav_req DELETE "$DAV/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics")
check_status "DELETE event" "204" "$STATUS"

echo
echo "--- 12. GET deleted event (should be 404) ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $auth \
  "$DAV/$USER_PATH/calendars/$CAL_SLUG/$TEST_UID.ics")
if [ "$STATUS" = "404" ]; then
  ok "Deleted event returns 404"
else
  fail "Expected 404 after delete, got $STATUS"
fi

# ─────────────────────────────────────────────────────────────────────────────
separator "CardDAV — Contacts Sync"
# ─────────────────────────────────────────────────────────────────────────────

echo
echo "--- 1. OPTIONS /.well-known/carddav ---"
STATUS=$(dav_req OPTIONS "$BASE/.well-known/carddav")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "204" ]; then
  ok "OPTIONS well-known/carddav (HTTP $STATUS)"
else
  fail "OPTIONS well-known/carddav — expected 200 or 204, got $STATUS"
fi

DAV_HEADER=$(curl -s -X OPTIONS -D - -o /dev/null $auth "$DAV/$USER_PATH/contacts/" | grep -i "^DAV:" | tr -d '\r' || true)
if echo "$DAV_HEADER" | grep -qi "addressbook"; then
  ok "DAV header advertises 'addressbook' capability"
else
  fail "DAV header missing 'addressbook' capability (got: $DAV_HEADER)"
fi

echo
echo "--- 2. PROPFIND address book home /dav/${USER_PATH}/contacts/ ---"
STATUS=$(dav_req PROPFIND "$DAV/$USER_PATH/contacts/" \
  -H "Depth: 0" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<propfind xmlns="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
  <prop>
    <resourcetype/>
    <displayname/>
    <card:addressbook-home-set/>
  </prop>
</propfind>')
check_status "PROPFIND address book home" "207" "$STATUS"

echo
echo "--- 3. PROPFIND default address book /dav/${USER_PATH}/contacts/default/ ---"
STATUS=$(dav_req PROPFIND "$DAV/$USER_PATH/contacts/default/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<propfind xmlns="DAV:">
  <prop><getetag/><getcontenttype/><resourcetype/></prop>
</propfind>')
check_status "PROPFIND default address book (Depth:1)" "207" "$STATUS"

# ── Create a test contact ─────────────────────────────────────────────────────
CONTACT_UID="curl-contact-$(date +%s)"
# N field: LastName;FirstName → buildVCard reconstructs FN as "FirstName LastName".
# Using N:;Curl Test Contact;;; sets FirstName="Curl Test Contact", LastName="" → FN:"Curl Test Contact".
TEST_VCF="BEGIN:VCARD
VERSION:3.0
UID:$CONTACT_UID
FN:Curl Test Contact
N:;Curl Test Contact;;;
EMAIL;TYPE=INTERNET:curl-test@example.com
TEL;TYPE=CELL:+55-11-99999-0000
ORG:go-cubemail Test Suite
NOTE:Created by test-dav-curl.sh
END:VCARD"

echo
echo "--- 4. PUT contact (create) ---"
STATUS=$(dav_req PUT "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf" \
  -H "Content-Type: text/vcard; charset=utf-8" \
  --data "$TEST_VCF")
if [ "$STATUS" = "201" ] || [ "$STATUS" = "204" ]; then
  ok "PUT contact — HTTP $STATUS"
else
  fail "PUT contact — expected 201 or 204, got $STATUS"
fi

echo
echo "--- 5. GET contact (read back) ---"
STATUS=$(dav_req GET "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf")
check_status "GET contact" "200" "$STATUS"

BODY=$(dav_body GET "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf")
if echo "$BODY" | grep -q "Curl Test Contact"; then
  ok "GET contact body contains correct FN"
else
  fail "GET contact body does not contain expected FN"
fi

echo
echo "--- 6. PROPFIND contact ETag ---"
STATUS=$(dav_req PROPFIND "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf" \
  -H "Depth: 0" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?><propfind xmlns="DAV:"><prop><getetag/></prop></propfind>')
check_status "PROPFIND single contact ETag" "207" "$STATUS"

echo
echo "--- 7. REPORT addressbook-query ---"
STATUS=$(dav_req REPORT "$DAV/$USER_PATH/contacts/default/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data '<?xml version="1.0"?>
<addressbook-query xmlns="urn:ietf:params:xml:ns:carddav" xmlns:d="DAV:">
  <d:prop><d:getetag/><address-data/></d:prop>
</addressbook-query>')
check_status "REPORT addressbook-query" "207" "$STATUS"

echo
echo "--- 8. REPORT addressbook-multiget ---"
STATUS=$(dav_req REPORT "$DAV/$USER_PATH/contacts/default/" \
  -H "Depth: 1" \
  -H "Content-Type: application/xml" \
  --data "<?xml version=\"1.0\"?>
<addressbook-multiget xmlns=\"urn:ietf:params:xml:ns:carddav\" xmlns:d=\"DAV:\">
  <d:prop><d:getetag/><address-data/></d:prop>
  <d:href>/dav/$USER_PATH/contacts/default/$CONTACT_UID.vcf</d:href>
</addressbook-multiget>")
check_status "REPORT addressbook-multiget" "207" "$STATUS"

echo
echo "--- 9. PUT contact (update — change phone) ---"
UPDATED_VCF="BEGIN:VCARD
VERSION:3.0
UID:$CONTACT_UID
FN:Curl Test Contact
N:;Curl Test Contact;;;
EMAIL;TYPE=INTERNET:curl-test@example.com
TEL;TYPE=CELL:+55-11-88888-1111
ORG:go-cubemail Test Suite
NOTE:Updated by test-dav-curl.sh
END:VCARD"

STATUS=$(dav_req PUT "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf" \
  -H "Content-Type: text/vcard; charset=utf-8" \
  --data "$UPDATED_VCF")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ] || [ "$STATUS" = "204" ]; then
  ok "PUT contact update — HTTP $STATUS"
else
  fail "PUT contact update — expected 200/201/204, got $STATUS"
fi

echo
echo "--- 10. DELETE contact ---"
STATUS=$(dav_req DELETE "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf")
check_status "DELETE contact" "204" "$STATUS"

echo
echo "--- 11. GET deleted contact (should be 404) ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $auth \
  "$DAV/$USER_PATH/contacts/default/$CONTACT_UID.vcf")
if [ "$STATUS" = "404" ]; then
  ok "Deleted contact returns 404"
else
  fail "Expected 404 after delete, got $STATUS"
fi

# ─────────────────────────────────────────────────────────────────────────────
separator "Result"
# ─────────────────────────────────────────────────────────────────────────────
echo
echo "  Passed : $PASS_STATUS"
echo "  Failed : $FAIL_STATUS"
echo

if [ "$FAIL_STATUS" -eq 0 ]; then
  echo "  All DAV tests passed."
  exit 0
else
  echo "  Some tests FAILED — review output above."
  exit 1
fi
