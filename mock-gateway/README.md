# Mock Notification Gateway

Simulated external API for Assignment 4.

## Endpoint

- `POST /notify`

Request JSON:

```json
{
  "idempotency_key": "...",
  "channel": "email|sms",
  "recipient": "patient@clinic.kz",
  "message": "..."
}
```

## Behavior

- New idempotency key -> `200 {"status":"accepted"}`
- Repeated idempotency key -> `200 {"status":"duplicate"}`
- Random 20% failures -> `503 {"status":"temporary_unavailable"}`

Every request is logged to stdout as JSON.

## Run

```bash
cd mock-gateway
go run .
```
