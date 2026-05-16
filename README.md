# Medical Scheduling Platform — Assignment 4

This project extends Assignment 3 with:
1. **Redis caching** for Doctor and Appointment read paths.
2. **Redis-backed gRPC rate limiting** (unary interceptor).
3. **Background job queue** in Notification Service for completed appointments.
4. **Mock Notification Gateway** (`POST /notify`) for external API simulation.

All existing domain models, use-case business rules, gRPC contracts, PostgreSQL repos/migrations, and broker publishing flow are preserved.

## Architecture

```text
grpc client
   |
   | DoctorService RPCs
   v
+----------------------------+         +------------------------+
| Doctor Service :8080       |-------> |                        |
| PostgreSQL: doctor_db      | events  |                        |
| Redis: cache + rate limit  |         |      NATS broker       |
+----------------------------+         |       :4222            |
                                       |                        |
grpc client                            |                        |
   |                                   +------------+-----------+
   | AppointmentService RPCs                        |
   v                                                v
+----------------------------+           +---------------------------+
| Appointment Service :8081  |---------> | Notification Service      |
| PostgreSQL: appointment_db | events    | subscriber + logger +     |
| Redis: cache + rate limit  |           | worker pool + idempotency |
+----------------------------+           | Redis + Gateway client    |
            | gRPC GetDoctor             +-------------+-------------+
            +---------------------------                |
                                                        v
                                          +---------------------------+
                                          | Mock Gateway :8090        |
                                          | POST /notify              |
                                          | 200 accepted/duplicate    |
                                          | random 20% 503            |
                                          +---------------------------+
```

## Broker choice

This implementation uses **NATS Core**.  
For durable delivery in production, use Outbox + JetStream (or RabbitMQ with durable queues + confirms).

## Environment variables

| Service | Variable | Default | Purpose |
|---|---|---|---|
| doctor-service | `DOCTOR_SERVICE_PORT` | `8080` | gRPC listen port |
| doctor-service | `DB_DSN` | local postgres | PostgreSQL DSN |
| doctor-service | `NATS_URL` | `nats://localhost:4222` | broker URL |
| doctor-service | `REDIS_URL` | `redis://localhost:6379` | Redis URL |
| doctor-service | `CACHE_TTL_SECONDS` | `60` | cache TTL |
| doctor-service | `RATE_LIMIT_RPM` | `100` | requests/min/client |
| appointment-service | `APPOINTMENT_SERVICE_PORT` | `8081` | gRPC listen port |
| appointment-service | `DB_DSN` | local postgres | PostgreSQL DSN |
| appointment-service | `DOCTOR_SERVICE_ADDR` | `localhost:8080` | doctor gRPC target |
| appointment-service | `NATS_URL` | `nats://localhost:4222` | broker URL |
| appointment-service | `REDIS_URL` | `redis://localhost:6379` | Redis URL |
| appointment-service | `CACHE_TTL_SECONDS` | `60` | cache TTL |
| appointment-service | `RATE_LIMIT_RPM` | `100` | requests/min/client |
| notification-service | `NATS_URL` | `nats://localhost:4222` | broker URL |
| notification-service | `REDIS_URL` | `redis://localhost:6379` | idempotency store |
| notification-service | `GATEWAY_URL` | `http://localhost:8090` | mock gateway URL |
| notification-service | `WORKER_POOL_SIZE` | `3` | worker count |
| mock-gateway | `GATEWAY_PORT` | `8090` | HTTP listen port |

## Infrastructure setup

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

## Migrations

Migrations run automatically on doctor/appointment service startup (before gRPC server starts).

Manual examples:

```bash
migrate -path doctor-service/migrations -database "postgres://da:pass@localhost:5432/doctor_db?sslmode=disable" up
migrate -path doctor-service/migrations -database "postgres://da:pass@localhost:5432/doctor_db?sslmode=disable" down 1

migrate -path appointment-service/migrations -database "postgres://da:pass@localhost:5432/appointment_db?sslmode=disable" up
migrate -path appointment-service/migrations -database "postgres://da:pass@localhost:5432/appointment_db?sslmode=disable" down 1
```

## Startup order

```bash
# 1) doctor service
cd doctor-service && go run ./cmd/doctor-service

# 2) appointment service
cd appointment-service && go run ./cmd/appointment-service

# 3) notification service
cd notification-service && go run .

# 4) mock gateway
cd mock-gateway && go run .
```

## Cache strategy

| Service | Operation | Strategy | Key | Behavior |
|---|---|---|---|---|
| doctor | `GetDoctor` | Cache-Aside | `doctor:<id>` | cache miss -> DB -> cache set |
| doctor | `ListDoctors` | Cache-Aside | `doctors:list` | cache miss -> DB -> cache set |
| doctor | `CreateDoctor` | Write-Through invalidation | `doctors:list` | DB success -> list delete |
| appointment | `GetAppointment` | Cache-Aside | `appointment:<id>` | cache miss -> DB -> cache set |
| appointment | `ListAppointments` | Cache-Aside | `appointments:list` | cache miss -> DB -> cache set |
| appointment | `CreateAppointment` | Write-Around invalidation | `appointments:list` | DB success -> list delete |
| appointment | `UpdateAppointmentStatus` | Write-Through update + invalidation | `appointment:<id>`, `appointments:list` | DB success -> item set + list delete |

### Cache invalidation and consistency

- Invalidation/update happens after successful DB write and before RPC response.
- Cache miss never returns an error to caller; request falls through to DB.
- Cache write/read failure is logged and treated as best-effort.
- If Redis is unavailable at startup, services continue in degraded mode (DB-only reads/writes).
- Stale-read window exists only until next write invalidation or TTL expiration.

## Rate limiter design

- Algorithm: **Redis sliding-window counter** using `ZSET`.
- Key: `ratelimit:<client_ip>`.
- Window: 60 seconds.
- Default limit: `RATE_LIMIT_RPM=100`.
- Enforcement point: `grpc.UnaryServerInterceptor` (no handler changes).
- Exceeded limit response: `codes.ResourceExhausted` with retry-after seconds.

### Rate-limiting trade-offs

1. Per-instance in-memory limiters break fairness in horizontal scale.
2. Redis central counter keeps limits consistent across instances.
3. IP-based identity can be weak behind shared proxies/NAT unless forwarded identity is standardized.

## Event contract

| Subject | Published by | Trigger | Payload fields |
|---|---|---|---|
| `doctors.created` | doctor-service | `CreateDoctor` success | `event_type`, `occurred_at`, `id`, `full_name`, `specialization`, `email` |
| `appointments.created` | appointment-service | `CreateAppointment` success | `event_type`, `occurred_at`, `id`, `title`, `doctor_id`, `status` |
| `appointments.status_updated` | appointment-service | `UpdateAppointmentStatus` success | `event_type`, `occurred_at`, `id`, `doctor_id`, `old_status`, `new_status` |

## Job queue design

- Trigger: `appointments.status_updated` where `new_status="done"`.
- Queue model: buffered channel + worker pool.
- Worker count: `WORKER_POOL_SIZE` (default `3`).
- Queue backpressure: enqueue blocks when channel is full.

Job payload:
- `idempotency_key` (SHA-256 of `event_type + id + occurred_at`)
- `appointment_id`
- `doctor_id`
- `occurred_at`
- `channel="email"`
- `recipient="patient@clinic.kz"`
- `message="Your appointment <id> with doctor <doctor_id> is complete."`

### Idempotency

- Keys are stored in Redis as `idempotency:<hash> = done` with `24h` TTL.
- If key is already marked done, job is dropped as duplicate and not sent to gateway.

### Retry and dead-letter strategy

- Retry on gateway `503` or network failure with backoff `1s -> 2s -> 4s` (max 3 attempts).
- After max retries: emit dead-letter JSON log to `stderr`; worker continues processing next jobs.
- Production evolution: move dead-letter to durable DLQ topic/queue + alerting.

## Mock Notification Gateway

`POST /notify` request:

```json
{
  "idempotency_key": "...",
  "channel": "email",
  "recipient": "patient@clinic.kz",
  "message": "Your appointment appt-1 with doctor doc-1 is complete."
}
```

Behavior:
- `200 {"status":"accepted"}` for new key.
- `200 {"status":"duplicate"}` for repeated key.
- `503 {"status":"temporary_unavailable"}` randomly in ~20% calls.
- every request is logged to stdout as JSON.

## Defense checkpoints mapping

1. **Cache hit**: call same `GetDoctor`/`GetAppointment` twice, observe Redis hit on second call.
2. **Rate limiter**: exceed RPM, receive `ResourceExhausted`, then recover after window reset.
3. **Job queue/gateway**: `UpdateAppointmentStatus -> done` logs event, processing, and success/retry.
4. **Idempotency**: replay same event, duplicate is dropped (no second gateway delivery).
5. **Dead letter**: stop gateway, trigger done event, observe retries then dead-letter stderr log.

## Testing artifact

Use `docs/grpcurl-commands.txt` for command sequence and expected log behavior.
