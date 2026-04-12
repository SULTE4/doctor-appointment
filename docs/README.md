# Medical Scheduling Platform Documentation (gRPC)

This project is migrated to **gRPC-only** communication:
- client -> Doctor Service: gRPC
- client -> Appointment Service: gRPC
- Appointment Service -> Doctor Service: gRPC

No REST endpoints are used.

## Service overview

| Service | Owns | Port (default) |
|---|---|---|
| Doctor Service | Doctors | `8080` |
| Appointment Service | Appointments | `8081` |

## Architecture

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

Each service follows Clean Architecture:

`transport/grpc` -> `usecase` (interface) <- `repository`  
`usecase` (appointment) -> `DoctorServiceClient` interface <- gRPC client implementation

## Proto contracts

- `doctor-service/proto/doctor.proto`
  - `CreateDoctor`
  - `GetDoctor`
  - `ListDoctors`
- `appointment-service/proto/appointment.proto`
  - `CreateAppointment`
  - `GetAppointment`
  - `ListAppointments`
  - `UpdateAppointmentStatus`

Generated files are committed:
- `doctor-service/proto/doctor.pb.go`, `doctor_grpc.pb.go`
- `appointment-service/proto/appointment.pb.go`, `appointment_grpc.pb.go`

## gRPC error behavior

| Situation | gRPC code |
|---|---|
| Missing required field | `InvalidArgument` |
| Duplicate doctor email | `AlreadyExists` |
| Local entity not found | `NotFound` |
| Doctor service unreachable | `Unavailable` |
| Remote doctor does not exist | `FailedPrecondition` |
| Invalid status transition (`done -> new`) | `InvalidArgument` |

## Run locally

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

Environment variables:
- `DOCTOR_SERVICE_PORT` (default `8080`)
- `APPOINTMENT_SERVICE_PORT` (default `8081`)
- `DOCTOR_SERVICE_ADDR` (default `localhost:8080`)

## Regenerate protobuf stubs

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$PATH:$(go env GOPATH)/bin"
```

```bash
cd doctor-service
protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. proto/doctor.proto
```

```bash
cd appointment-service
protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. proto/appointment.proto
```

## Testing artifact

Use:

`docs/grpcurl-commands.txt`

It contains ready grpcurl commands for:
- all required RPCs,
- doctor-not-found scenario,
- invalid status transition scenario,
- doctor-service-unavailable scenario.

## Related files

- Root docs: `README.md`
- Doctor service docs: `doctor-service/README.md`
- Appointment service docs: `appointment-service/README.md`
