# Medical Scheduling Platform

> A two-service medical scheduling system built in Go using Clean Architecture and REST-based microservices.

## Why This Exists

The platform demonstrates how to decompose a domain into independently deployable services with clear ownership boundaries. Each service owns its data, enforces its own rules, and communicates with others exclusively over REST — avoiding the tight coupling that turns a distributed system into a distributed monolith.

## Architecture Overview

```
  Client
    │
    ├─── POST /doctors          ┌──────────────────┐
    ├─── GET  /doctors          │   Doctor Service  │
    └─── GET  /doctors/:id ───► │     :8080         │
                                │  [in-memory store]│
                                └──────────────────┘
                                         ▲
    Client                               │ GET /doctors/:id
    │                                    │ (verify doctor exists)
    ├─── POST /appointments      ┌───────┴──────────┐
    ├─── GET  /appointments      │ Appointment Svc  │
    ├─── GET  /appointments/:id  │     :8081         │
    └─── PATCH /appointments/…  │  [in-memory store]│
              /status       ───► └──────────────────┘
```

The Appointment Service calls the Doctor Service synchronously before persisting any appointment. No shared database exists between the two services — each service is the sole owner of its data.

## Service Responsibilities

| Service | Owns | Validates |
|---------|------|-----------|
| **Doctor Service** | Doctor profiles | `full_name` required, `email` required and unique |
| **Appointment Service** | Appointments | `title` required, `doctor_id` must resolve to a real doctor via REST call |

## How to Run

Start each service in its own terminal. The Doctor Service must be running before the Appointment Service can create appointments.

**Terminal 1 — Doctor Service**

```bash
cd doctor-service
go run .
# Listening on :8080
```

**Terminal 2 — Appointment Service**

```bash
cd appointment-service
go run .
# Listening on :8081
```

## Quick Demo

```bash
# 1. Create a doctor
DOCTOR=$(curl -s -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}')
echo $DOCTOR

# 2. Extract the ID (requires jq)
DOCTOR_ID=$(echo $DOCTOR | jq -r '.doctor.id')

# 3. Create an appointment
curl -s -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Initial consultation\",\"description\":\"Referred for palpitations\",\"doctor_id\":\"$DOCTOR_ID\"}"

# 4. Update status
APPT_ID=<id-from-above>
curl -s -X PATCH http://localhost:8081/appointments/$APPT_ID/status \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

## Folder Structure

```
doctor-appointment/
├── doctor-service/
│   ├── cmd/doctor-service/main.go
│   └── internal/
│       ├── model/          # Doctor domain model
│       ├── usecase/        # Business logic + interfaces
│       ├── repository/     # In-memory storage
│       ├── transport/http/ # HTTP handlers (Gin)
│       └── app/            # Wiring and startup
├── appointment-service/
│   ├── cmd/appointment-service/main.go
│   └── internal/
│       ├── model/          # Appointment domain model + Status type
│       ├── usecase/        # Business logic + interfaces
│       ├── repository/     # In-memory storage
│       ├── client/         # HTTP client for Doctor Service
│       ├── transport/http/ # HTTP handlers (Gin)
│       └── app/            # Wiring and startup
└── README.md
```

## Clean Architecture Inside Each Service

Each service follows the same layering pattern:

```
Handler  →  UseCase (interface)  ←  Repository implementation
                                 ←  DoctorClient implementation (appointment-service only)
```

- **Handlers** parse the HTTP request, call the use case, and write the response. Zero business logic.
- **Use cases** own all business rules. They depend only on interfaces, never on concrete types.
- **Repositories** implement persistence. Swapping in-memory storage for a real database requires no changes above this layer.
- **Domain models** contain no HTTP, JSON, or framework-specific fields.

## Why No Shared Database

Using a shared database would couple the two services at the schema level: a change to the doctors table would risk breaking the appointment service, they could not scale independently, and neither could own its data boundaries. Each service uses its own in-memory store (trivially replaceable with a real database) that no other service can read or write directly.

## Inter-Service Communication

When `POST /appointments` or `PATCH /appointments/:id/status` is called, the Appointment Service sends:

```
GET http://doctor-service:8080/doctors/:doctor_id
```

- `200 OK` → doctor exists, proceed.
- `404 Not Found` → reject with `404 Not Found`.
- Network error / timeout (5 s) → reject with `503 Service Unavailable` and log the failure.

The Doctor Service client is injected as a `DoctorServiceClient` interface, so it is fully replaceable with a test double.

## Failure Scenario

If the Doctor Service is down when an appointment is created:

1. The HTTP client times out after 5 seconds.
2. The use case receives the error and prefixes it `SERVICE_UNAVAILABLE`.
3. The handler returns `HTTP 503` with a descriptive message.
4. The failure is logged at `[ERROR]` level with the target URL and underlying error.

**Production resilience additions (not in scope for this assignment):**

| Pattern | When to apply |
|---------|---------------|
| Retry with exponential backoff | Transient failures — the Doctor Service is briefly unavailable |
| Circuit breaker | Sustained outages — stop sending requests to a failing service to prevent cascade failures |
| Timeout tuning | Match timeout to the SLA agreed between services |

## Configuration

| Service | Variable | Default |
|---------|----------|---------|
| Doctor Service | `DOCTOR_SERVICE_PORT` | `8080` |
| Appointment Service | `PORT` | `8081` |
| Appointment Service | `DOCTOR_SERVICE_URL` | `http://localhost:8080` |

## Service READMEs

- [Doctor Service →](doctor-service/README.md)
- [Appointment Service →](appointment-service/README.md)
