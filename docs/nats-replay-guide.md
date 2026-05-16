# NATS Replay Guide (Idempotency Check)

Use this when the grader asks you to replay the same `appointments.status_updated` event.

## Goal

Replay the **same** event (same `id` + same `occurred_at`) so Notification Service computes the same idempotency key and drops it as duplicate.

Expected result:
- event log appears in Notification Service
- job is detected as duplicate
- no second successful delivery effect to gateway

## Option A — Use NATS CLI

## 1) Install CLI

```bash
go install github.com/nats-io/natscli/nats@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

## 2) Publish replay event

```bash
nats --server nats://localhost:4222 pub appointments.status_updated '{"event_type":"appointments.status_updated","occurred_at":"2026-05-15T07:55:30Z","id":"appt-123","doctor_id":"doc-456","old_status":"new","new_status":"done"}'
```

Replace values with the original event values from your logs.

## Option B — Use Docker nats-box (if CLI not installed)

```bash
docker run --rm --network host natsio/nats-box:latest sh -c \
'nats --server nats://127.0.0.1:4222 pub appointments.status_updated "{\"event_type\":\"appointments.status_updated\",\"occurred_at\":\"2026-05-15T07:55:30Z\",\"id\":\"appt-123\",\"doctor_id\":\"doc-456\",\"old_status\":\"in_progress\",\"new_status\":\"done\"}"'
```

## How to get correct values

1. Trigger `UpdateAppointmentStatus` to `done` once (normal flow).
2. In Notification Service logs, copy:
   - `id`
   - `occurred_at`
   - `doctor_id`
3. Replay with exactly the same values.

## Verify in defense

After replay, show:
1. Notification Service logs the event.
2. Duplicate detection behavior (no new successful job delivery effect).
3. Mock gateway does not get a new accepted delivery for same idempotency key.
