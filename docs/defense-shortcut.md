# TS4 Defense (Run + Test + Postman)

This guide is a practical script you can follow during defense so everything works in order.

## 1) What you will run

- **doctor-service** (gRPC): `localhost:8080`
- **appointment-service** (gRPC): `localhost:8081`
- **notification-service** (subscriber + job queue)
- **mock-gateway** (HTTP): `localhost:8090`  ✅ (moved from 8080 to avoid conflict)
- Infra: PostgreSQL, Redis, NATS

## 2) Start infrastructure

```bash
# PostgreSQL
docker run --name ap2-postgres -e POSTGRES_USER=da -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=postgres -p 5432:5432 -d postgres:16
docker exec -it ap2-postgres psql -U da -d postgres -c "CREATE DATABASE doctor_db;"
docker exec -it ap2-postgres psql -U da -d postgres -c "CREATE DATABASE appointment_db;"

# NATS
docker run --name ap2-nats -p 4222:4222 -d nats:2.11-alpine

# Redis
docker run --name ap2-redis -p 6379:6379 -d redis:7-alpine
```

## 3) Start services (4 terminals)

```bash
# Terminal 1
cd doctor-service && go run ./cmd/doctor-service
```

```bash
# Terminal 2
cd appointment-service && go run ./cmd/appointment-service
```

```bash
# Terminal 3
cd notification-service && go run .
```

```bash
# Terminal 4
cd mock-gateway && go run .
```

## 4) Postman setup (important)

Your APIs are **gRPC** (not REST).

### Doctor Service in Postman
1. New -> **gRPC Request**
2. Server: `localhost:8080`
3. Import proto: `doctor-service/proto/doctor.proto`
4. Use methods:
   - `doctor.DoctorService/CreateDoctor`
   - `doctor.DoctorService/GetDoctor`
   - `doctor.DoctorService/ListDoctors`

### Appointment Service in Postman
1. New -> **gRPC Request**
2. Server: `localhost:8081`
3. Import proto: `appointment-service/proto/appointment.proto`
4. Use methods:
   - `appointment.AppointmentService/CreateAppointment`
   - `appointment.AppointmentService/GetAppointment`
   - `appointment.AppointmentService/ListAppointments`
   - `appointment.AppointmentService/UpdateAppointmentStatus`

## 5) Defense flow (checkpoint by checkpoint)

## Checkpoint A — Base flow + event logs
1. Postman -> `CreateDoctor`:
```json
{
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```
2. Copy returned `doctor_id`.
3. Postman -> `CreateAppointment` with that doctor id.
4. Postman -> `UpdateAppointmentStatus` to `"done"`.
5. Show:
   - Notification terminal: event log + job logs (`enqueued`, `processing`, `success` or `retry`)
   - Mock gateway terminal: received `/notify` JSON request

## Checkpoint B — Cache hit
1. In another terminal:
```bash
redis-cli MONITOR
```
2. In Postman call `GetDoctor` twice with same id.
3. Explain:
   - first request -> cache miss then set
   - second request -> cache hit path

## Checkpoint C — Rate limiter
Option 1 (recommended): Postman Runner sends 120+ requests quickly to `GetDoctor`.

Option 2 (CLI burst):
```bash
for i in $(seq 1 120); do
  grpcurl -plaintext \
    -import-path doctor-service/proto \
    -proto doctor-service/proto/doctor.proto \
    -d '{"id":"<doctor-id>"}' \
    localhost:8080 doctor.DoctorService/GetDoctor >/dev/null 2>&1
done
```
Expected: some requests return `ResourceExhausted`, then recover after 1-minute window.

## Checkpoint D — Idempotency
Replay same `appointments.status_updated` event (same `id`, `occurred_at`) into NATS:
```bash
nats pub appointments.status_updated '{"event_type":"appointments.status_updated","occurred_at":"<same-occurred-at>","id":"<same-appointment-id>","doctor_id":"<same-doctor-id>","old_status":"in_progress","new_status":"done"}'
```
Expected:
- event is logged
- job is detected as duplicate (already processed)
- no second successful external delivery effect

## Checkpoint E — Dead letter
1. Stop mock gateway terminal (`Ctrl+C`).
2. Trigger another `UpdateAppointmentStatus` to `"done"` for a different appointment.
3. Show notification logs:
   - retry attempt 1
   - retry attempt 2
   - retry attempt 3
   - `dead_letter` JSON in **stderr**
4. Service must keep running.

## 6) Quick troubleshooting

- If doctor service says it is listening on `:8080`, it is OK.
- If mock gateway port is busy, confirm it is `8090` in:
  - `mock-gateway/.env`
  - `notification-service/.env`
- If Redis warning appears but services run, still continue (cache/rate limiting fallback behavior is expected by spec).

## 7) What to show verbally (short answers)

1. Cache strategy used per endpoint (cache-aside + required invalidations).
2. Rate limiter algorithm (Redis sliding window with ZSET in unary interceptor).
3. Job lifecycle (enqueue -> processing -> success/retry/dead_letter).
4. Idempotency key formula (`SHA-256(event_type + id + occurred_at)`).
5. Reliability trade-offs (best-effort eventing, retries, dead-letter, production DLQ/outbox options).
