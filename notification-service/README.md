# Notification Service

Standalone subscriber for Assignment 3.

## Responsibilities

- Connect to NATS (`NATS_URL`)
- Subscribe to:
  - `doctors.created`
  - `appointments.created`
  - `appointments.status_updated`
- Print one structured JSON log line per message:
  - `time` (RFC3339)
  - `subject`
  - `event` (full deserialized payload)

## What it does not do

- No gRPC server
- No HTTP server
- No database writes

## Run

```bash
cd notification-service
go run .
```

## Startup behavior

- If broker is unavailable, retries with exponential backoff.
- After max retries, exits with non-zero status and descriptive error.

## Shutdown behavior

- Handles `SIGINT` / `SIGTERM`
- Drains broker connection and exits cleanly
