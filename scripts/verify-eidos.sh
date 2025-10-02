#!/usr/bin/env bash
# usage: ./scripts/verify-eidos.sh http://localhost:5173/#/assets
set -euo pipefail
URL="${1:-http://localhost:3000/ledger}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$URL" || true)
if [[ "$STATUS" != "200" ]]; then
  echo "PAGE FAIL"
  exit 1
fi
CONTENT=$(curl -s "$URL")
if ! echo "$CONTENT" | egrep -q "eidos-ledger-root|eidos-ledger-container|eidos-ledger-list"; then
  echo "CLASSES MISSING"
  exit 2
fi
echo "OK"
