# Medical Scheduling Platform — Assignment 3

This version extends Assignment 2 with:
1. **PostgreSQL + migrations** (`golang-migrate`) in Doctor and Appointment services.
2. **Asynchronous events** via **NATS Core** (`doctors.created`, `appointments.created`, `appointments.status_updated`).
3. A third binary: **Notification Service**, which subscribes to events and prints structured JSON logs.

## Broker choice

This project uses **NATS Core** because setup is simple and fast for stateless notifications.  
For durable delivery in production, move to **NATS JetStream** (or RabbitMQ durable queues + confirms).

## Architecture

```text
grpc client
   |
   | DoctorService RPCs
   v
+---------------------------+                 +-----------------------+
| Doctor Service :8080      | --publish-----> |                       |
| DB: doctor_db (PostgreSQL)|                 |                       |
+---------------------------+                 |                       |
                                              |      NATS broker      |
grpc client                                   |     :4222 (Core)      |
   |                                          |                       |
   | AppointmentService RPCs                  |                       |
   v                                          |                       |
+-------------------------------+             |                       |
| Appointment Service :8081     | --publish-> |                       |
| DB: appointment_db (PostgreSQL)|            +-----------+-----------+
+-------------------------------+                         |
                | gRPC GetDoctor                          |
                +-----------------------------------------+
                                                          |
                                                          v
                                             +---------------------------+
                                             | Notification Service      |
                                             | subscribes + logs JSON    |
                                             | no DB, no gRPC, no HTTP   |
                                             +---------------------------+
```

## Environment variables

| Service | Variable | Example |
|---|---|---|
| doctor-service | `DOCTOR_SERVICE_PORT` | `8080` |
| doctor-service | `DB_DSN` | `postgres://da:pass@localhost:5432/doctor_db?sslmode=disable` |
| doctor-service | `NATS_URL` | `nats://localhost:4222` |
| appointment-service | `APPOINTMENT_SERVICE_PORT` | `8081` |
| appointment-service | `DB_DSN` | `postgres://da:pass@localhost:5432/appointment_db?sslmode=disable` |
| appointment-service | `DOCTOR_SERVICE_ADDR` | `localhost:8080` |
| appointment-service | `NATS_URL` | `nats://localhost:4222` |
| notification-service | `NATS_URL` | `nats://localhost:4222` |

## Infrastructure setup

Start PostgreSQL (two databases in one instance):

```bash
docker run --name ap2-postgres -e POSTGRES_USER=da -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=postgres -p 5432:5432 -d postgres:16
docker exec -it ap2-postgres psql -U da -d postgres -c "CREATE DATABASE doctor_db;"
docker exec -it ap2-postgres psql -U da -d postgres -c "CREATE DATABASE appointment_db;"
```

Start NATS:

```bash
docker run --name ap2-nats -p 4222:4222 -d nats:2.11-alpine
```

## Migrations

Migrations run automatically on service startup (before gRPC server starts).

Manual CLI example:

```bash
# doctor-service
migrate -path doctor-service/migrations -database "postgres://da:pass@localhost:5432/doctor_db?sslmode=disable" up
migrate -path doctor-service/migrations -database "postgres://da:pass@localhost:5432/doctor_db?sslmode=disable" down 1

# appointment-service
migrate -path appointment-service/migrations -database "postgres://da:pass@localhost:5432/appointment_db?sslmode=disable" up
migrate -path appointment-service/migrations -database "postgres://da:pass@localhost:5432/appointment_db?sslmode=disable" down 1
```

## Startup order

1. Start PostgreSQL and NATS.
2. Start `doctor-service` first (needed by appointment-service doctor validation).
3. Start `appointment-service`.
4. Start `notification-service`.

```bash
cd doctor-service && go run ./cmd/doctor-service
cd appointment-service && go run ./cmd/appointment-service
cd notification-service && go run .
```

## Event contract

| Subject | Published by | Trigger | Payload |
|---|---|---|---|
| `doctors.created` | doctor-service | `CreateDoctor` success | `event_type`, `occurred_at`, `id`, `full_name`, `specialization`, `email` |
| `appointments.created` | appointment-service | `CreateAppointment` success | `event_type`, `occurred_at`, `id`, `title`, `doctor_id`, `status` |
| `appointments.status_updated` | appointment-service | `UpdateAppointmentStatus` success | `event_type`, `occurred_at`, `id`, `old_status`, `new_status` |

## Notification Service behavior

- Subscribes to all three subjects.
- For each message, prints one JSON line to stdout:

```json
{"time":"2026-05-01T10:24:01Z","subject":"appointments.created","event":{"event_type":"appointments.created","occurred_at":"2026-05-01T10:24:01Z","id":"appt-1","title":"Initial cardiac consultation","doctor_id":"doc-1","status":"new"}}
```

- On startup broker failure: retries with exponential backoff, then exits non-zero.
- On SIGINT/SIGTERM: drains and closes broker connection.

## Consistency trade-offs

- DB write is the source of truth.
- Event publishing is **best-effort** for doctor/appointment services.
- If broker is unavailable, RPC still succeeds and publish failure is logged.
- If process crashes after DB commit but before publish, event can be lost.
- Durability improvements: **Outbox pattern**, **NATS JetStream**, or **RabbitMQ publisher confirms + durable queues**.

## NATS vs RabbitMQ (quick comparison)

1. **Durability**: NATS Core is fire-and-forget; RabbitMQ supports durable queues and acknowledgements by default patterns.
2. **Operational model**: NATS Core is lighter/simpler; RabbitMQ gives richer queue routing and persistence controls.

Choose NATS Core for simple transient events; choose RabbitMQ (or JetStream) when at-least-once durability is required.

## Testing artifact

`docs/grpcurl-commands.txt` contains grpcurl calls and expected Notification Service output mapping.
