#!/usr/bin/env bash
# Run ActiveSync HTTP integration tests against a live go-cubemail server.
#
# Required:
#   EAS_INTEGRATION_URL   e.g. http://localhost:8080/Microsoft-Server-ActiveSync
#   EAS_INTEGRATION_USER  IMAP login (Basic Auth)
#   EAS_INTEGRATION_PASS  IMAP password
#
# Optional:
#   EAS_INTEGRATION_DEVICE_ID      default: integration-test-device-001
#   EAS_INTEGRATION_SEND_TO        enable SendMail tests (recipient address)
#   EAS_INTEGRATION_EVENT_ID       enable MeetingResponse by event ServerId
#   EAS_INTEGRATION_EVENT_UID      enable MeetingResponse by iCalendar UID
#   EAS_INTEGRATION_SEARCH_QUERY   free-text query for Search test
#   EAS_INTEGRATION_SKIP_SEARCH=1  skip Search test

set -euo pipefail

cd "$(dirname "$0")/.."

: "${EAS_INTEGRATION_URL:?set EAS_INTEGRATION_URL}"
: "${EAS_INTEGRATION_USER:?set EAS_INTEGRATION_USER}"
: "${EAS_INTEGRATION_PASS:?set EAS_INTEGRATION_PASS}"

export EAS_INTEGRATION_URL EAS_INTEGRATION_USER EAS_INTEGRATION_PASS
export EAS_INTEGRATION_DEVICE_ID="${EAS_INTEGRATION_DEVICE_ID:-integration-test-device-001}"

echo "Running ActiveSync integration tests against ${EAS_INTEGRATION_URL}"
go test -tags integration -v -count=1 ./internal/activesync/...
