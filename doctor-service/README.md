# Doctor Service

> Manages doctor profiles for the medical scheduling platform.

## What This Service Does

The Doctor Service owns and manages all doctor data. It exposes a REST API that lets clients create, retrieve, and list doctors. The Appointment Service calls this service over HTTP to verify that a doctor exists before an appointment is created or updated.

## Quick Start

**Prerequisites**: Go 1.21+

```bash
cd doctor-service
go run .
```

The service starts on port `8080` by default. Set `DOCTOR_SERVICE_PORT` to override.

## Folder Structure

```
doctor-service/
├── cmd/doctor-service/main.go       # Entry point — calls app.Run()
└── internal/
    ├── model/doctor.go              # Domain model — no framework types
    ├── usecase/
    │   ├── interfaces.go            # DoctorUsecase and DoctorRepository interfaces
    │   └── usecase.go               # Business logic: validation, uniqueness, creation
    ├── repository/repo.go           # In-memory storage implementing DoctorRepository
    ├── transport/http/handler.go    # Thin HTTP handlers — parse, delegate, respond
    └── app/app.go                   # Wires all layers and starts the HTTP server
```

**Dependency direction**: `handler` → `usecase interface` ← `repository implementation`

The handler imports the usecase interface. The repository implements it. Neither the handler nor the repository knows about each other. This is Dependency Inversion in practice.

## API Reference

### Create a Doctor

```
POST /doctors
```

**Request**

```json
{
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

`full_name` and `email` are required. `specialization` is optional.

**Response `201 Created`**

```json
{
  "doctor": {
    "id": "3f1e2a4b-...",
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | Missing `full_name` or `email` |
| `409 Conflict` | Email already registered |

---

### Get Doctor by ID

```
GET /doctors/:id
```

**Response `200 OK`**

```json
{
  "doctor": {
    "id": "3f1e2a4b-...",
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `404 Not Found` | Doctor with the given ID does not exist |

---

### List All Doctors

```
GET /doctors
```

**Response `200 OK`**

```json
{
  "doctors": [
    { "id": "...", "full_name": "...", "specialization": "...", "email": "..." }
  ]
}
```

## Business Rules

- `full_name` is required.
- `email` is required and must be unique across all doctors. A `409 Conflict` is returned if the email is already registered.
- All business rules are enforced in the use case layer, not in the handler.

## Configuration

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `DOCTOR_SERVICE_PORT` | `8080` | Port the service listens on |

## curl Examples

```bash
# Create a doctor
curl -s -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}'

# Get a doctor by ID
curl -s http://localhost:8080/doctors/<id>

# List all doctors
curl -s http://localhost:8080/doctors
```
