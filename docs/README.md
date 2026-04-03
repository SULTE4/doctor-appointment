
## Architecture Overview
[ASCII or Mermaid diagram here]

+------------------+          REST GET /doctors/{id}       +------------------+
| Appointment Svc  | ------------------------------------> |   Doctor Svc     |
|  :8081           |                                       |   :8080          |
|  [in-memory DB]  |                                       |  [in-memory DB]  |
+------------------+                                       +------------------+

## Dependency Direction (per service)
Handler → UseCase (interface) ← Repository implementation
                              ← DoctorClient implementation

## Why No Shared Database
Each service owns its bounded context. A shared DB would couple schema,
deployment, and scaling — defeating the purpose of service decomposition.

## Failure Scenario
When Doctor Service is down, the HTTP client times out after 5s.
The use case receives the error, prefixes it SERVICE_UNAVAILABLE,
the handler maps it to HTTP 503 with a descriptive message.
The failure is logged internally.

Production resilience additions:
- Retries with exponential backoff (transient failures)
- Circuit breaker (prevent cascade failures under sustained outage)
- Timeout tuning per SLA requirement

## How to Run
# Terminal 1
cd doctor-service && go run .

# Terminal 2
cd appointment-service && go run .

## API Examples (curl)
# Create doctor
curl -X POST localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Dr. Aisha","specialization":"Cardiology","email":"a@clinic.kz"}'

# Create appointment
curl -X POST localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{"title":"Consultation","doctor_id":"<id-from-above>"}'

# Update status
curl -X PATCH localhost:8081/appointments/<id>/status \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```
