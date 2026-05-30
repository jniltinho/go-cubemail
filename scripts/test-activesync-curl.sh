#!/usr/bin/env bash
# Smoke-test ActiveSync endpoints with curl.
# Usage: USER=you@example.com PASS=secret ./scripts/test-activesync-curl.sh

set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
USER="${USER:-you@example.com}"
PASS="${PASS:-yourpassword}"
DEVICE_ID="${DEVICE_ID:-testdevice000000000000000001}"
EAS="${BASE}/Microsoft-Server-ActiveSync"

auth="-u ${USER}:${PASS}"

echo "=== OPTIONS (capabilities) ==="
curl -s -i $auth -X OPTIONS "$EAS" | head -20

echo ""
echo "=== Provision ==="
curl -s $auth -X POST \
  "${EAS}?Cmd=Provision&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null | xxd | head -5

echo ""
echo "=== FolderSync (SyncKey=0) ==="
curl -s $auth -X POST \
  "${EAS}?Cmd=FolderSync&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null | xxd | head -10

echo ""
echo "=== Ping ==="
curl -s $auth -X POST \
  "${EAS}?Cmd=Ping&User=${USER}&DeviceId=${DEVICE_ID}&DeviceType=iPhone" \
  -H "MS-ASProtocolVersion: 14.1" \
  -H "Content-Type: application/vnd.ms-sync.wbxml" \
  --data-binary @/dev/null | xxd | head -5

echo ""
echo "=== Autodiscover ==="
curl -s $auth -X POST "${BASE}/autodiscover/autodiscover.xml" \
  -H "Content-Type: text/xml" \
  -d "<?xml version=\"1.0\" encoding=\"utf-8\"?><Autodiscover xmlns=\"http://schemas.microsoft.com/exchange/autodiscover/outlook/requestschema/2006\"><Request><EMailAddress>${USER}</EMailAddress><AcceptableResponseSchema>http://schemas.microsoft.com/exchange/autodiscover/mobilesync/responseschema/2006</AcceptableResponseSchema></Request></Autodiscover>" | head -20

echo ""
echo "=== Done (WBXML responses show as hex; non-empty = success) ==="
