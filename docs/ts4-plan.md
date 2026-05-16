# Assignment 4 Implementation Plan

## Problem statement
We need to extend the TS3 architecture with:
1. Redis caching for Doctor and Appointment read endpoints.
2. Redis-backed per-client rate limiting via gRPC interceptors.
3. Notification Service background jobs for completed appointments.
4. Mock external notification gateway with idempotent behavior and transient failures.

All existing domain logic, gRPC contracts, PostgreSQL repositories/migrations, and broker flow must be preserved.

## Implementation approach

### 1) Shared configuration and dependency wiring
- Add `REDIS_URL`, `CACHE_TTL_SECONDS`, `RATE_LIMIT_RPM` to Doctor/Appointment services.
- Add `WORKER_POOL_SIZE`, `GATEWAY_URL` to Notification Service.
- Keep safe defaults and read from env in app wiring.
- On Redis startup failure: log warning and continue in degraded mode.

### 2) Caching layer (Doctor + Appointment)
- Introduce `CacheRepository` abstraction and Redis implementation.
- Add no-op cache fallback for resilience.
- Implement required key naming:
  - `doctor:<id>`, `doctors:list`
  - `appointment:<id>`, `appointments:list`
- Enforce required behavior:
  - Cache-aside for reads.
  - Invalidation/update after successful DB writes and before RPC response.
  - Cache failures logged only (best-effort).

### 3) Rate limiting (Doctor + Appointment)
- Add `internal/middleware` interceptor in each service.
- Use Redis sliding-window counter per client IP.
- Apply limit globally through `grpc.NewServer(grpc.UnaryInterceptor(...))`.
- Return `codes.ResourceExhausted` with retry-after details.

### 4) Notification Service refactor and job queue
- Separate packages:
  - `internal/subscriber`
  - `internal/logger`
  - `internal/jobqueue`
- Keep event log output structured.
- When receiving `appointments.status_updated` with `new_status=done`, enqueue job.
- Job includes deterministic idempotency key and required payload fields.
- Worker pool:
  - configurable size,
  - buffered channel,
  - retries (1s, 2s, 4s, max 3),
  - dead-letter JSON to stderr.

### 5) Mock gateway service
- New service `mock-gateway` with `POST /notify`.
- Behavior:
  - new idempotency key => `{"status":"accepted"}`
  - duplicate key => `{"status":"duplicate"}`
  - random 20% => `HTTP 503`
- Structured stdout logging for every request.

### 6) Documentation and defense readiness
- Update root/service READMEs with:
  - cache strategy per endpoint,
  - rate-limiter algorithm and Redis structure,
  - job lifecycle and dead-letter behavior,
  - startup order for all services + infra.
- Extend grpcurl/demo artifact to cover all defense checkpoints.

## Execution todos
1. Add env/config and dependencies for Redis + HTTP gateway integration.
2. Build doctor-service cache layer and wire use case invalidation rules.
3. Build appointment-service cache layer and wire use case invalidation/update rules.
4. Add rate-limiter interceptors in both gRPC services.
5. Refactor notification service into subscriber/logger/jobqueue packages.
6. Implement job queue worker pool with idempotency and retries.
7. Implement mock gateway with idempotent behavior + 20% transient failures.
8. Update docs and demo commands for defense checkpoints.
9. Run end-to-end verification across cache, limiter, queue, idempotency, dead-letter flows.

## Key risks / decisions
- Event payload for `appointments.status_updated` should include `doctor_id` to satisfy TS4 job contract while preserving required fields.
- Redis downtime must never break primary request flow; degraded mode is expected and should be visible in logs.
- Keep all Redis and middleware logic in infrastructure packages (no Redis imports in domain/usecase packages).
