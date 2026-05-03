# Appointment Service (gRPC)

Owns appointment data and exposes `AppointmentService` gRPC API.
Before create/update, it validates doctor existence by calling Doctor Service gRPC `GetDoctor`.
Uses PostgreSQL for persistence and publishes appointment events to NATS.

## Run

```bash
cd appointment-service
go run ./cmd/appointment-service
```

Default port: `8081` (`APPOINTMENT_SERVICE_PORT`)  
Doctor target: `localhost:8080` (`DOCTOR_SERVICE_ADDR`)  
DB env: `DB_DSN`  
Broker env: `NATS_URL` (default `nats://localhost:4222`)

## RPCs

Defined in `proto/appointment.proto`:

1. `CreateAppointment(CreateAppointmentRequest) returns (AppointmentResponse)`
2. `GetAppointment(GetAppointmentRequest) returns (AppointmentResponse)`
3. `ListAppointments(ListAppointmentsRequest) returns (ListAppointmentsResponse)`
4. `UpdateAppointmentStatus(UpdateStatusRequest) returns (AppointmentResponse)`

## Business rules and gRPC errors

- `title` required -> `InvalidArgument`
- `doctor_id` required -> `InvalidArgument`
- doctor must exist remotely -> `FailedPrecondition`
- doctor service unreachable -> `Unavailable`
- appointment id not found -> `NotFound`
- status must be one of `new/in_progress/done` -> `InvalidArgument`
- transition `done -> new` forbidden -> `InvalidArgument`
- broker unavailable at startup -> service still starts, warning is logged
- broker publish failure during RPC -> error is logged, RPC response is not affected

## Structure

```text
cmd/appointment-service/main.go
internal/model
internal/usecase
internal/repository
internal/client              # Doctor service gRPC client implementation
internal/transport/grpc
internal/app
proto/appointment.proto
proto/appointment.pb.go
proto/appointment_grpc.pb.go
```

Dependency flow:

`transport/grpc` -> `usecase` (interface) <- `repository`  
`usecase` -> `DoctorServiceClient` interface <- `internal/client`

## Regenerate stubs

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$PATH:$(go env GOPATH)/bin"

protoc --go_out=paths=source_relative:. \
  --go-grpc_out=paths=source_relative:. \
  proto/appointment.proto
```

## Doctor service dependency wiring

`appointment-service/go.mod` includes:
- `require doctor-service v0.0.0`
- `replace doctor-service => ../doctor-service`

This allows importing Doctor proto stubs (`doctor-service/proto`) for the injected gRPC client without breaking use case dependency inversion.
