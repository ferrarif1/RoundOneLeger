#!/usr/bin/env bash
# usage: ./scripts/verify-roledger.sh http://localhost:5173/#/assets
set -euo pipefail
URL="${1:-http://localhost:3000/ledger}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$URL" || true)
if [[ "$STATUS" != "200" ]]; then
  echo "PAGE FAIL"
  exit 1
fi
CONTENT=$(curl -s "$URL")
if ! echo "$CONTENT" | egrep -q "roledger-ledger-root|roledger-ledger-container|roledger-ledger-list"; then
  echo "CLASSES MISSING"
  exit 2
fi
echo "OK"
