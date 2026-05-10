#!/usr/bin/env bash
set -euo pipefail
IMAGE_PATH="${1:-Self_image1.jpg}"
NAME="${2:-John Doe}"
DOB="${3:-1990-01-01}"
GENDER="${4:-M}"

test -f "$IMAGE_PATH" || { echo "Image not found: $IMAGE_PATH"; exit 1; }

echo "1) Health"
curl -s http://localhost:8080/health; echo

echo "2) Enroll async"
RESP=$(curl -s -X POST http://localhost:8080/kyc/enroll -F "image=@${IMAGE_PATH}" -F "name=${NAME}" -F "dob=${DOB}" -F "gender=${GENDER}")
echo "$RESP"
TXN=$(python -c 'import json,sys; print(json.load(sys.stdin)["transaction_id"])' <<< "$RESP")

echo "3) Poll status"
for i in {1..20}; do
  sleep 1
  STATUS=$(curl -s "http://localhost:8080/kyc/status/${TXN}")
  echo "$STATUS"
  python - <<'PY' <<< "$STATUS" && break || true
import json,sys
o=json.load(sys.stdin)
sys.exit(0 if o.get('status') in ('COMPLETED','FAILED') else 1)
PY
done
