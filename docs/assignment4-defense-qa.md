# Assignment 4 — What Was Added + Defense Q&A

This file is a quick speaking guide for defense.

## 1) What was added in Assignment 4

## Doctor Service
- Added **Redis cache layer** for:
  - `GetDoctor` (`doctor:<id>`)
  - `ListDoctors` (`doctors:list`)
- Added cache invalidation on write:
  - `CreateDoctor` invalidates `doctors:list`
- Added **Redis-backed rate limiter** as gRPC unary interceptor (`RATE_LIMIT_RPM`).

## Appointment Service
- Added **Redis cache layer** for:
  - `GetAppointment` (`appointment:<id>`)
  - `ListAppointments` (`appointments:list`)
- Added write-side cache handling:
  - `CreateAppointment` invalidates `appointments:list`
  - `UpdateAppointmentStatus` updates `appointment:<id>` and invalidates `appointments:list`
- Added **Redis-backed rate limiter** as gRPC unary interceptor.

## Notification Service
- Kept broker subscription and event logging.
- Added **background job queue**:
  - worker pool (`WORKER_POOL_SIZE`)
  - idempotency keys in Redis (TTL 24h)
  - retries with exponential backoff (1s, 2s, 4s)
  - dead-letter JSON logs to stderr after max retries
- On `appointments.status_updated` with `new_status=done`, enqueue external notification job.

## Mock Gateway (new 4th binary)
- Added HTTP service with `POST /notify`.
- Behavior:
  - new key -> `{"status":"accepted"}`
  - repeated key -> `{"status":"duplicate"}`
  - random 20% -> HTTP 503 (to exercise retry logic)

## 2) What each addition does (in one sentence)

- **Cache**: reduces DB load and lowers read latency on repeated requests.
- **Rate limiter**: protects gRPC endpoints from bursts and abuse.
- **Job queue**: moves external API calls out of request path for reliability and isolation.
- **Idempotency key**: prevents duplicate external notifications on replay/retry.
- **Retry + dead letter**: handles transient failures safely and makes permanent failures visible.

## 3) Likely defense questions and answer templates

## Q1. Why cache-aside here?
**Answer:**  
I used cache-aside for reads because it is simple, keeps DB as source of truth, and gracefully falls back on cache miss. Cache failure does not break RPC behavior.

## Q2. Why invalidate list keys on create/update?
**Answer:**  
List caches become stale after writes. I invalidate list keys immediately after successful DB write so next read repopulates from fresh DB data.

## Q3. Why update `appointment:<id>` on status update?
**Answer:**  
Status update changes one known entity, so updating that key keeps point-read cache fresh, while list key is invalidated to avoid stale aggregates.

## Q4. Why implement rate limiting in interceptor, not handlers?
**Answer:**  
Interceptor keeps handlers clean and enforces policy consistently across all RPC methods in one place.

## Q5. What algorithm did you choose for rate limiting?
**Answer:**  
Redis sliding-window with ZSET. I remove old timestamps, count requests in current 60s window, and reject with `ResourceExhausted` when the limit is exceeded.

## Q6. What happens if Redis is down?
**Answer:**  
Services still run. Cache and limiter are treated as infrastructure best-effort/degraded concerns; core business RPC flow continues using DB and existing logic.

## Q7. How is idempotency key generated and why?
**Answer:**  
SHA-256 of `event_type + id + occurred_at`. It is deterministic for the same logical event, so replay/retry maps to the same key and avoids duplicate delivery.

## Q8. Why retries 1s, 2s, 4s?
**Answer:**  
Exponential backoff reduces pressure on a failing dependency and gives transient failures time to recover.

## Q9. What is dead-letter in your implementation?
**Answer:**  
After 3 failed attempts, job is not retried further and a structured dead-letter entry is written to stderr. Service stays alive and continues processing other jobs.

## Q10. Why separate subscriber/logger/jobqueue packages?
**Answer:**  
To preserve clean architecture and separation of concerns: message intake, structured logging, and background processing are isolated and easier to test/maintain.

## Q11. Why include `doctor_id` in `appointments.status_updated` payload?
**Answer:**  
Job contract needs doctor context to build the outgoing message. Adding `doctor_id` keeps event self-contained for asynchronous processing.

## Q12. What reliability gap still exists?
**Answer:**  
Event publishing is still best-effort; crash between DB commit and publish can lose an event. Production fix is Outbox pattern and/or durable broker features.

## 4) One-minute summary you can say

“In Assignment 4 I added Redis cache-aside reads with proper invalidation on writes, Redis-based gRPC rate limiting via interceptors, and a worker-pool job queue in Notification Service for done appointments. Jobs use deterministic idempotency keys in Redis, retry with exponential backoff, and go to dead-letter logs after max retries. I also added a mock external gateway with duplicate detection and random 503 to prove retry behavior. Core domain logic and gRPC contracts remain unchanged.”
