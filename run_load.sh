#!/bin/bash
DOCTOR_ID="${1:-<doctor-id>}"
TARGET="${2:-localhost:8080}"
REQUESTS="${3:-200}"

if ! command -v grpcurl >/dev/null 2>&1; then
  echo "ERROR: grpcurl is not installed or not in PATH"
  echo "Install: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
  echo "Then: export PATH=\"\$PATH:$(go env GOPATH)/bin\""
  exit 1
fi

if [ "$DOCTOR_ID" = "<doctor-id>" ]; then
  echo "ERROR: set a real doctor id."
  echo "Usage: ./run_load.sh <doctor-id> [host:port] [requests]"
  exit 1
fi

OK=0
LIMITED=0
OTHER=0

for i in $(seq 1 "$REQUESTS"); do
  OUT=$(grpcurl -plaintext \
    -import-path doctor-service/proto \
    -proto doctor-service/proto/doctor.proto \
    -d "{\"id\":\"$DOCTOR_ID\"}" \
    "$TARGET" doctor.DoctorService/GetDoctor 2>&1)

  if echo "$OUT" | grep -q "ResourceExhausted"; then
    LIMITED=$((LIMITED+1))
  elif echo "$OUT" | grep -q "\"id\""; then
    OK=$((OK+1))
  else
    OTHER=$((OTHER+1))
  fi
done

echo "OK=$OK LIMITED=$LIMITED OTHER=$OTHER"

 # ./run_load.sh <real-doctor-id> localhost:8080 200
