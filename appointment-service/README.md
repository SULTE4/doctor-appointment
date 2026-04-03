# Appointment Service

> Manages appointment scheduling and enforces doctor-existence validation via the Doctor Service.

## What This Service Does

The Appointment Service owns all appointment data. Before creating or updating an appointment, it calls the Doctor Service over HTTP to verify the referenced doctor exists. If the Doctor Service is unreachable, the operation is rejected with a `503 Service Unavailable` response and the failure is logged.

## Quick Start

**Prerequisites**: Go 1.21+, Doctor Service running on port `8080`

```bash
cd appointment-service
go run .
```

The service starts on port `8081` by default.

## Folder Structure

```
appointment-service/
├── cmd/appointment-service/main.go       # Entry point — calls app.Run()
└── internal/
    ├── model/appointment.go              # Domain model with Status type
    ├── usecase/
    │   ├── interfaces.go                 # AppointmentUseCase, AppointmentRepository, DoctorServiceClient interfaces
    │   └── usecase.go                    # Business logic: validation, status transitions, doctor check
    ├── repository/repo.go                # In-memory storage implementing AppointmentRepository
    ├── client/doctorClient.go            # HTTP client implementing DoctorServiceClient
    ├── transport/http/handler.go         # Thin HTTP handlers — parse, delegate, respond
    └── app/app.go                        # Wires all layers and starts the HTTP server
```

**Dependency direction**: `handler` → `usecase interface` ← `repository implementation`
                                                          ← `doctor client implementation`

The use case depends only on interfaces. The HTTP client for the Doctor Service implements `DoctorServiceClient`, so it can be swapped for a mock in tests without touching business logic.

## API Reference

### Create an Appointment

```
POST /appointments
```

**Request**

```json
{
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctor_id": "3f1e2a4b-..."
}
```

`title` and `doctor_id` are required. `description` is optional.

**Response `201 Created`**

```json
{
  "ID": "9c2d...",
  "Title": "Initial cardiac consultation",
  "Description": "Patient referred for palpitations and shortness of breath",
  "DoctorID": "3f1e2a4b-...",
  "Status": "new",
  "CreatedAt": "2026-04-03T10:00:00Z",
  "UpdatedAt": "2026-04-03T10:00:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | Missing `title` or `doctor_id` |
| `404 Not Found` | Doctor does not exist in the Doctor Service |
| `503 Service Unavailable` | Doctor Service is unreachable or timed out |

---

### Get Appointment by ID

```
GET /appointments/:id
```

**Response `200 OK`** — appointment object as above.

**Error responses**

| Status | Condition |
|--------|-----------|
| `404 Not Found` | Appointment with the given ID does not exist |

---

### List All Appointments

```
GET /appointments
```

**Response `200 OK`** — array of appointment objects.

---

### Update Appointment Status

```
PATCH /appointments/:id/status
```

**Request**

```json
{ "status": "in_progress" }
```

Valid values: `new`, `in_progress`, `done`.

**Response `200 OK`** — updated appointment object.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | Invalid status value |
| `404 Not Found` | Appointment not found |
| `422 Unprocessable Entity` | Transitioning from `done` back to `new` |

## Business Rules

- `title` is required.
- `doctor_id` is required, and the referenced doctor must exist in the Doctor Service (validated over REST).
- `status` must be one of: `new`, `in_progress`, `done`.
- Transitioning from `done` back to `new` is forbidden.
- All business rules are enforced in the use case layer, not in the handler.

## Failure Scenario

When the Doctor Service is unavailable:

1. The HTTP client times out after **5 seconds**.
2. The use case receives the error and wraps it with the `SERVICE_UNAVAILABLE` prefix.
3. The handler maps this to `HTTP 503` with a descriptive message.
4. The failure is logged at `[ERROR]` level with the target URL and underlying error.

The operation is never completed when the Doctor Service cannot be reached.

**Where resilience patterns would be added in production:**

- **Timeout**: already implemented (5s). Tune per SLA.
- **Retries with exponential backoff**: for transient network failures where the Doctor Service is briefly unavailable.
- **Circuit breaker**: to stop sending requests after repeated failures, preventing cascade overload and giving the Doctor Service time to recover.

## Configuration

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `PORT` | `8081` | Port the service listens on |
| `DOCTOR_SERVICE_URL` | `http://localhost:8080` | Base URL of the Doctor Service |

## curl Examples

```bash
# Create an appointment (replace <doctor-id> with a real ID from the Doctor Service)
curl -s -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{"title":"Initial consultation","description":"Referred for palpitations","doctor_id":"<doctor-id>"}'

# Get an appointment by ID
curl -s http://localhost:8081/appointments/<id>

# List all appointments
curl -s http://localhost:8081/appointments

# Update status
curl -s -X PATCH http://localhost:8081/appointments/<id>/status \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```
