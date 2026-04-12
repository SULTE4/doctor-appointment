# Doctor Service (gRPC)

Owns doctor profile data and exposes `DoctorService` gRPC API.

## Run

```bash
cd doctor-service
go run ./cmd/doctor-service
```

Default port: `8080` (`DOCTOR_SERVICE_PORT`).

## RPCs

Defined in `proto/doctor.proto`:

1. `CreateDoctor(CreateDoctorRequest) returns (DoctorResponse)`
2. `GetDoctor(GetDoctorRequest) returns (DoctorResponse)`
3. `ListDoctors(ListDoctorsRequest) returns (ListDoctorsResponse)`

## Business rules

- `full_name` required -> `InvalidArgument`
- `email` required -> `InvalidArgument`
- unique `email` -> `AlreadyExists`
- doctor ID missing in get -> `InvalidArgument`
- doctor ID not found -> `NotFound`

## Structure

```text
cmd/doctor-service/main.go
internal/model
internal/usecase
internal/repository
internal/transport/grpc
internal/app
proto/doctor.proto
proto/doctor.pb.go
proto/doctor_grpc.pb.go
```

Dependency flow remains:

`transport/grpc` -> `usecase` (interface) <- `repository`

## Regenerate stubs

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$PATH:$(go env GOPATH)/bin"

protoc --go_out=paths=source_relative:. \
  --go-grpc_out=paths=source_relative:. \
  proto/doctor.proto
```
