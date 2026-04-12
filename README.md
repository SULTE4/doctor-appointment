# Medical Scheduling Platform (gRPC)

Two Go microservices using Clean Architecture:
- **Doctor Service** owns doctor profiles.
- **Appointment Service** owns appointments and validates doctors through the Doctor Service over **gRPC**.

All communication is gRPC (client-to-service and inter-service). No REST endpoints are used.

## Architecture and data ownership

```text
grpc client
   |
   | DoctorService RPCs
   v
+-------------------------+        gRPC GetDoctor
| Doctor Service :8080    |<-----------------------------+
| owns doctor data only   |                              |
+-------------------------+                              |
                                                         |
grpc client                                               |
   |                                                      |
   | AppointmentService RPCs                              |
   v                                                      |
+-------------------------+                               |
| Appointment Service :8081|------------------------------+
| owns appointment data only|
+-------------------------+
```

Each service keeps its own repository and business rules. Appointment service never reads Doctor repository directly.

## Clean Architecture layering

Inside each service:

`transport/grpc` -> `usecase` (interface) <- `repository`  
`usecase` (appointment only) -> `DoctorServiceClient` interface <- `internal/client` gRPC implementation

Business logic stays in use cases. gRPC handlers only map proto <-> domain/usecase.

## Proto contracts

- `doctor-service/proto/doctor.proto`
  - `CreateDoctor`, `GetDoctor`, `ListDoctors`
- `appointment-service/proto/appointment.proto`
  - `CreateAppointment`, `GetAppointment`, `ListAppointments`, `UpdateAppointmentStatus`

Generated files are committed:
- `doctor-service/proto/doctor.pb.go`, `doctor_grpc.pb.go`
- `appointment-service/proto/appointment.pb.go`, `appointment_grpc.pb.go`

## Required gRPC status behavior

| Situation | Status code |
|---|---|
| Missing required field | `InvalidArgument` |
| Duplicate doctor email | `AlreadyExists` |
| Local entity not found | `NotFound` |
| Doctor service unreachable | `Unavailable` |
| Remote doctor does not exist | `FailedPrecondition` |
| Invalid status transition (`done -> new`) | `InvalidArgument` |

## Running locally

Start Doctor Service first, then Appointment Service.

```bash
# Terminal 1
cd doctor-service
go run ./cmd/doctor-service
```

```bash
# Terminal 2
cd appointment-service
go run ./cmd/appointment-service
```

Default ports:
- Doctor: `8080` (`DOCTOR_SERVICE_PORT`)
- Appointment: `8081` (`APPOINTMENT_SERVICE_PORT`)
- Appointment->Doctor target: `localhost:8080` (`DOCTOR_SERVICE_ADDR`)

## Regenerate protobuf stubs

Prerequisites:
1. `protoc` installed
2. Go plugins installed:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$PATH:$(go env GOPATH)/bin"
```

Generate doctor stubs:

```bash
cd doctor-service
protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. proto/doctor.proto
```

Generate appointment stubs:

```bash
cd appointment-service
protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. proto/appointment.proto
```

## RPC contract summary

### Doctor Service

1. `CreateDoctor(CreateDoctorRequest) -> DoctorResponse`
   - `full_name` and `email` required, `email` unique.
2. `GetDoctor(GetDoctorRequest) -> DoctorResponse`
   - returns `NotFound` if ID does not exist.
3. `ListDoctors(ListDoctorsRequest) -> ListDoctorsResponse`
   - returns all doctors.

### Appointment Service

1. `CreateAppointment(CreateAppointmentRequest) -> AppointmentResponse`
   - `title` and `doctor_id` required.
   - validates doctor through Doctor Service `GetDoctor`.
2. `GetAppointment(GetAppointmentRequest) -> AppointmentResponse`
   - returns `NotFound` if ID does not exist.
3. `ListAppointments(ListAppointmentsRequest) -> ListAppointmentsResponse`
   - returns all appointments.
4. `UpdateAppointmentStatus(UpdateStatusRequest) -> AppointmentResponse`
   - status must be `new`, `in_progress`, or `done`.
   - `done -> new` is forbidden.
   - validates doctor through Doctor Service before update.

## Failure scenario

If Doctor Service is down/unreachable:
- Appointment service gRPC client call fails.
- Use case returns service-unavailable failure.
- gRPC handler maps it to `codes.Unavailable`.
- Appointment creation/update is rejected (no write is performed).

Where production resilience would fit:
- **timeouts** (already used),
- **retries with backoff** for transient failures,
- **circuit breaker** to prevent cascading failure.

## REST vs gRPC trade-offs

1. **Serialization**: gRPC uses protobuf (compact, strongly typed), REST commonly uses JSON (human-readable but larger payloads).
2. **Contract strictness**: gRPC enforces schema-first contracts (`.proto`), REST is often looser unless strict OpenAPI governance is used.
3. **Performance**: gRPC over HTTP/2 is generally better for service-to-service latency/throughput; REST is simpler for browser/public API consumption.

Choose gRPC for internal high-throughput service calls and strict contracts; choose REST when broad client compatibility and human-readable APIs are priorities.

## Testing artifact

See `docs/grpcurl-commands.txt` for runnable grpcurl commands that demonstrate all required RPCs and error paths.
