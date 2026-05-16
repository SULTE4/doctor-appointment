# Notification Service (Assignment 4)

## Responsibilities

1. Subscribe to broker events (`doctors.created`, `appointments.created`, `appointments.status_updated`).
2. Log each received event as JSON (`time`, `subject`, `event`).
3. For `appointments.status_updated` with `new_status=done`, enqueue background job.
4. Process jobs with worker pool and call Mock Gateway `POST /notify`.
5. Enforce idempotency using Redis keys with `24h` TTL.

## Packages

- `internal/subscriber` — broker connection + subscriptions
- `internal/logger` — structured stdout/stderr JSON logs
- `internal/jobqueue` — queue, workers, retries, idempotency, gateway calls

## Environment

- `NATS_URL` (default `nats://localhost:4222`)
- `REDIS_URL` (default `redis://localhost:6379`)
- `GATEWAY_URL` (default `http://localhost:8090`)
- `WORKER_POOL_SIZE` (default `3`)

## Run

```bash
cd notification-service
go run .
```

## Retry / dead-letter policy

- Retry count: 3 attempts.
- Backoff: 1s, 2s, 4s.
- On max retry exhaustion: write dead-letter JSON entry to `stderr`.
