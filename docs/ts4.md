Advanced Programming 2
Assignment 4 – Caching Strategies & Background Jobs
Scope: Lecture 7 (Caching Strategies) & Lecture 8 (Background Jobs & External APIs)
Field Details
Scope Lecture 7 (Caching Strategies) & Lecture 8 (Background Jobs & External
APIs)
Deadline 23:59 [set by instructor]
Submission Moodle – ZIP archive named AP2_Assignment4_NameSurname.zip
Defense Week 11–12 practice classes
Cheating Policy Strictly prohibited — results in grade 0
1. Assignment Overview
In Assignments 1–3 you built a two-service Medical Scheduling Platform, migrated it to gRPC,
replaced in-memory storage with PostgreSQL managed by golang-migrate, and introduced
asynchronous event publishing through a message broker. In this final assignment you will add
two production-readiness concerns:

Caching — a Redis-backed caching layer that reduces database load and improves read
latency for the Doctor Service and the Appointment Service.
Background Jobs & External Integration — a job queue that processes domain events
asynchronously and calls a simulated external API (an email/SMS notification gateway)
for each successfully completed appointment.
The domain logic, Clean Architecture layering, bounded-context boundaries, gRPC contracts,
PostgreSQL repositories, migration files, and message-broker integration from Assignment 3
must be fully preserved. Only the infrastructure layer grows.

2. Learning Objectives
Understand why caching matters and what problems it solves (latency, database load,
scalability).
Implement an in-memory cache using Redis and the go-redis client in Go.
Apply cache invalidation strategies (Write-Through, Write-Back, Write-Around) and
articulate the trade-offs of each.
Implement rate limiting using Redis as a token-bucket or sliding-window store.
Design and implement a background job queue using a Go worker pool pattern.
AP2 — Assignment
4

Understand idempotency in job processing and implement idempotency keys.
Add retry logic with exponential backoff for transient failures in external API calls.
Integrate with a third-party (simulated) external API using webhooks or HTTP callbacks.
Explain consistency trade-offs introduced by caching (stale reads, cache stampede,
thundering herd).
3. Scope and Constraints
What Changes
Both existing services (Doctor Service, Appointment Service) integrate a Redis cache for
read operations.
Cache invalidation is applied on every write (create / update) operation.
A rate limiter backed by Redis is added to both services to protect gRPC endpoints
against overload.
The Notification Service is extended with a worker-pool-based background job queue.
When an appointment transitions to status done, a background job is enqueued and
calls a simulated External Notification API over HTTP.
Idempotency keys prevent duplicate external API calls on job retry.
What Does NOT Change
Domain models (Doctor, Appointment, Status) remain identical.
Use-case logic and business rules remain identical.
gRPC service contracts and generated stubs remain identical.
PostgreSQL schemas and migration files remain identical.
Message broker integration (NATS / RabbitMQ) and event publishing remain identical.
Clean Architecture layering and dependency direction remain identical.
Notice: REST endpoints, synchronous external HTTP calls inside gRPC handlers, and ORMs
are out of scope.

4. Infrastructure Requirements
4.1 Redis
Run a single Redis instance locally or via Docker (redis:7-alpine recommended).
The connection string must be read from the environment variable REDIS_URL (e.g.
redis://localhost:6379).
AP2 — Assignment
4

Use the github.com/redis/go-redis/v9 client. Direct tcp dial or other clients are not
permitted.
All cache keys must follow a namespaced convention: :: (e.g.
doctor:doc-1, appointment:appt-1, doctors:list).
Cache TTL values must be configurable via environment variables
(CACHE_TTL_SECONDS). A hardcoded default of 60 seconds is acceptable only as a
fallback.
4.2 Rate Limiter
Implement a per-client rate limiter using Redis as the backing store (sliding-window
counter or token-bucket — your choice; document your choice in the README).
The rate limit must be applied as a gRPC interceptor (UnaryServerInterceptor) so that no
handler code is modified.
Default limit: 100 requests per minute per client IP. This must be configurable via the
RATE_LIMIT_RPM environment variable.
When the limit is exceeded, return codes.ResourceExhausted with a descriptive
message.
4.3 Background Job Queue
Implement a simple in-process worker pool in the Notification Service. You may not use
an external job queue library (e.g. Asynq, Machinery) — implement the queue using Go
channels and goroutines.
The pool size must be configurable via the WORKER_POOL_SIZE environment variable
(default: 3).
Each job must have an idempotency key (a deterministic string derived from the event
payload, e.g. SHA-256 of event_type + id + occurred_at) stored in Redis to prevent
duplicate processing on retry.
Failed jobs must be retried up to 3 times with exponential backoff (1s, 2s, 4s). After 3
failures, the job is moved to a dead-letter log (a structured JSON line written to stderr).
4.4 External Notification API (Simulated)
You must build a minimal fourth Go binary — the Mock Notification Gateway — that exposes
one HTTP endpoint:

POST /notify
Request body (JSON):

{ "idempotency_key": "...", "channel": "email|sms", "recipient": "...", "message":
"..." }
The gateway must:

Return HTTP 200 with {"status": "accepted"} for new idempotency keys.
Return HTTP 200 with {"status": "duplicate"} for repeated idempotency keys (simulate
idempotent external API behavior).
AP2 — Assignment
4

Return HTTP 503 randomly 20% of the time to simulate transient failures (so your retry
logic is exercised during defense).
Log every received request to stdout in JSON format.
The gateway URL must be read from the GATEWAY_URL environment variable in the
Notification Service.

5. Caching Requirements
5.1 Doctor Service
Operation Cache Strategy Key Pattern TTL
GetDoctor (by ID) Cache-Aside (Read-
Through)
doctor:<id> CACHE_TTL_SECONDS
ListDoctors Cache-Aside doctors:list CACHE_TTL_SECONDS
CreateDoctor
(write)
Write-Through —
invalidate list key
doctors:list immediate eviction
5.2 Appointment Service
Operation Cache Strategy Key Pattern TTL
GetAppointment (by ID) Cache-Aside appointment:<id> CACHE_TTL_SECONDS
ListAppointments Cache-Aside appointments:list CACHE_TTL_SECONDS
CreateAppointment (write) Write-Around —
invalidate list key
appointments:list immediate eviction
UpdateAppointmentStatus
(write)
Write-Through —
update &
invalidate
appointment:<id>,
appointments:list
immediate eviction
5.3 Cache Invalidation Rules
Invalidation must happen after the database write succeeds and before the gRPC
response is returned.
A cache miss must never return an error to the caller — fall through to the database
transparently.
A cache write failure must be logged but must not block the gRPC response (best-effort
caching).
The cache layer must be implemented behind a CacheRepository interface and injected
— no Redis calls inside use cases or gRPC handlers.
AP2 — Assignment
4

6. Background Job Requirements
6.1 Job Trigger
The Notification Service already subscribes to appointments.status_updated events. When the
event payload contains new_status = "done", a background job must be enqueued immediately
after the event is logged.

6.2 Job Contract
Each job must carry the following fields:

Field Type Description
idempotency_key string SHA-256 hex of event_type + id + occurred_at
appointment_id string ID of the completed appointment
doctor_id string Doctor ID from the event payload
occurred_at string
(RFC3339)
Timestamp from the original event
channel string Always "email" for this assignment
recipient string Hardcoded placeholder: "patient@clinic.kz"
message string "Your appointment <id> with doctor <doctor_id> is complete."
6.3 Job Lifecycle
Job is created from the appointments.status_updated event consumer.
Idempotency key is checked in Redis. If already processed (value = "done"), job is
dropped silently.
Job is dispatched to the worker pool via a buffered channel.
A worker picks up the job and calls POST /notify on the Mock Notification Gateway.
On HTTP 200: mark the idempotency key in Redis as "done" (TTL: 24 h). Log success.
On HTTP 503 or network error: wait exponential backoff, retry (up to 3 times).
After 3 failures: write a dead-letter log entry to stderr and drop the job.
6.4 Required Log Output
Every job state transition must produce a structured JSON log line on stdout (success) or stderr
(dead-letter). Minimum fields:

Field Type Description
time string
(RFC3339)
When the log line was written
AP2 — Assignment
4

level string "info" | "warn" | "error"
job_id string The idempotency key
attempt integer Current attempt number (1-based)
status string "enqueued" | "processing" | "success" | "retry" | "dead_letter"
error string Error message (omit on success)
7. Notification Service — Updated Responsibilities
The Notification Service now has three concerns that must be cleanly separated:

Subscriber — unchanged from Assignment 3: connects to the broker and routes events.
Logger — unchanged: prints one structured JSON log line per event to stdout.
Job Queue — new: manages the worker pool, idempotency store (Redis), and retry
logic.
These three concerns must live in separate packages (e.g. internal/subscriber, internal/logger,
internal/jobqueue). Cross-package dependencies must follow the same Clean Architecture rules
as the other services.

8. Event Definitions (Unchanged from Assignment 3)
All three events remain identical. The table below is reproduced for reference.

Service Trigger Subject / Routing Key Payload Fields
Doctor Service CreateDoctor succeeds doctors.created event_type,
occurred_at, id,
full_name,
specialization, email
Appointment
Service
CreateAppointment
succeeds
appointments.created event_type,
occurred_at, id, title,
doctor_id, status
Appointment
Service
UpdateAppointmentStatus
succeeds
appointments.status_updated event_type,
occurred_at, id,
old_status, new_status
The job queue is triggered only by appointments.status_updated events where new_status =
"done".

9. Error Handling Requirements
AP2 — Assignment
4

All gRPC status codes from Assignments 2 and 3 remain in force. The following cases are
added:

Situation Expected Behaviour
Redis unavailable on startup Log a warning and continue. Caching is best-effort; the service must
not crash.
Cache miss at runtime Fall through to the database transparently. Never return an error for
a cache miss.
Cache write failure Log the error with full context. Do not block the gRPC response.
Rate limit exceeded Return codes.ResourceExhausted with a descriptive message
including the retry-after interval.
External gateway returns 503 Retry with exponential backoff (1s, 2s, 4s). Log each retry attempt.
External gateway unreachable Treat as transient failure; apply same retry logic.
Job reaches max retries Write a dead-letter JSON entry to stderr. Do not crash the worker.
Duplicate idempotency key Log at info level and drop the job silently.
10. Suggested Project Structure
ap2-assignment4/
├── doctor-service/
│ ├── internal/
│ │ ├── cache/ ← Redis CacheRepository implementation
│ │ ├── middleware/ ← Rate-limiter gRPC interceptor
│ │ └── ... (unchanged from Assignment 3)
├── appointment-service/
│ ├── internal/
│ │ ├── cache/ ← Redis CacheRepository implementation
│ │ ├── middleware/ ← Rate-limiter gRPC interceptor
│ │ └── ... (unchanged from Assignment 3)
├── notification-service/
│ ├── internal/
│ │ ├── subscriber/ (unchanged)
│ │ ├── logger/ (unchanged)
│ │ └── jobqueue/ ← NEW: worker pool, idempotency, retry
├── mock-gateway/ ← NEW: simulated external notification API
│ └── main.go
└── README.md
AP2 — Assignment
4

11. Best-Case and Worst-Case Design
Best Case
Redis is optional at startup: if unavailable, services log a warning and serve all requests
from the database. Caching is a pure performance optimisation and never a single point
of failure.
The cache layer is hidden behind a CacheRepository interface and injected into the use
case — no Redis imports in domain or use-case code.
The rate limiter is implemented as a gRPC interceptor, keeping all handler code clean.
The job queue uses a buffered channel with a configurable worker pool. The pool size,
retry count, and backoff durations are all configurable via environment variables.
Idempotency keys are stored in Redis with a 24-hour TTL so that a service restart does
not re-process already-delivered jobs.
The Mock Notification Gateway correctly simulates idempotency (duplicate key → 200
duplicate) and transient failure (20% 503 rate), so retry logic is exercised end-to-end.
Dead-letter entries are written to stderr as structured JSON — no silent job drops.
The README documents the cache strategy choice for each endpoint, the rate-limiting
algorithm, and the job lifecycle with a state diagram.
Worst Case
Redis unavailability crashes the service or blocks gRPC responses.
Redis calls appear directly inside use cases or gRPC handlers.
Cache invalidation is missing or applied inconsistently, causing stale reads that are
never corrected.
The rate limiter is implemented inside individual RPC handlers rather than as an
interceptor.
The job queue is implemented with a single goroutine and no retry logic.
Idempotency keys are not stored — re-processing events on restart causes duplicate
external API calls.
The Mock Notification Gateway is absent or always returns 200, making retry logic
impossible to verify.
Connection URLs are hardcoded in source code.
12. Environment Variables — Complete List
Service Variable Example Value Purpose
All REDIS_URL redis://localhost:6379 Redis connection string
All CACHE_TTL_SECONDS 60 Default cache entry TTL
AP2 — Assignment
4

Doctor / Appt RATE_LIMIT_RPM 100 Max requests per minute per
client IP
Doctor Service DATABASE_URL postgres://... PostgreSQL connection
(unchanged)
Doctor Service NATS_URL /
AMQP_URL
nats://localhost:4222 Broker URL (unchanged)
Doctor Service GRPC_PORT 50051 gRPC listen port
(unchanged)
Appt Service DATABASE_URL postgres://... PostgreSQL connection
(unchanged)
Appt Service NATS_URL /
AMQP_URL
nats://localhost:4222 Broker URL (unchanged)
Appt Service GRPC_PORT 50052 gRPC listen port
(unchanged)
Notification NATS_URL /
AMQP_URL
nats://localhost:4222 Broker URL (unchanged)
Notification GATEWAY_URL http://localhost:8080 Mock Notification Gateway
URL
Notification WORKER_POOL_SIZE 3 Number of background job
workers
Mock Gateway GATEWAY_PORT 8080 HTTP listen port
13. Deliverables
Source code for both updated services (Doctor Service, Appointment Service) with
Redis cache layer and rate-limiter interceptor.
Updated Notification Service with job queue, idempotency store, and retry logic.
New Mock Notification Gateway binary.
All migration files, proto files, and generated Go stubs from Assignment 3 (unchanged).
README.md — see Section 14 for required content.
Updated architecture diagram showing all four services, two PostgreSQL databases,
Redis, the message broker, and all communication arrows with labels.
Testing artifact: extended grpcurl command list with expected Notification Service log
output and expected job-queue log output per command.
14. Documentation Requirements
Project overview — what changed compared to Assignment 3 and why.
AP2 — Assignment
4

Cache strategy — which strategy (Cache-Aside, Write-Through, Write-Around, Write-
Back) was chosen for each endpoint, and the reason for each choice.
Rate-limiting algorithm — which algorithm (sliding window, token bucket, etc.) was
chosen and why. Include the Redis data structure used.
Cache invalidation — when and how cache keys are invalidated, and what stale-read
window exists between a write and the next TTL expiry.
Job queue design — a description of the worker pool architecture, buffered channel size,
and how backpressure is handled when the channel is full.
Idempotency — how idempotency keys are derived, where they are stored, and how
they prevent duplicate gateway calls on retry.
Dead-letter strategy — what happens after max retries, how to inspect dead-letter
entries, and what a production system would do with them (e.g., DLQ in the broker,
alerting).
Infrastructure setup — how to start Redis, PostgreSQL, the broker, and the Mock
Gateway (Docker commands or equivalent).
Service startup order — correct startup sequence for all four services with exact go run.
commands.
Cache consistency trade-offs — what happens when Redis is unavailable, which reads
become eventually consistent, and how a distributed cache (Redis Cluster) would affect
consistency guarantees.
Rate-limiting trade-offs — discuss at least two limitations of per-instance rate limiting in a
horizontally scaled system and how a centralised Redis-backed counter solves them.
1 5. Grading Rubric
Criterion Weight What Is Evaluated High-Performance Indicators
Caching Layer 25% Redis integration;
cache-aside reads;
invalidation on writes;
TTL configuration;
CacheRepository
interface
Cache misses fall through to DB
transparently; invalidation is correct and
consistent; no Redis calls in domain or
use-case layer; Redis failure never
crashes the service
Rate Limiting 15% Redis-backed rate
limiter; gRPC interceptor
implementation;
configurable limit;
correct error codes
Implemented as a UnaryServerInterceptor;
returns codes.ResourceExhausted with
retry-after; limit configurable via env var;
algorithm documented in README
Background Job
Queue
25% Worker pool;
idempotency keys; retry
with backoff; dead-letter
logging; Mock Gateway
integration
Pool size configurable; idempotency keys
stored in Redis with TTL; all 3 retry
attempts logged; dead-letter entries written
to stderr as JSON; duplicate keys silently
dropped
AP2 — Assignment
4

Clean Architecture
Preservation
15% Layering and
dependency direction
after infrastructure
changes
Cache and rate-limiter are infrastructure
concerns injected via interfaces; no Redis
or net/http imports in use-case or domain
packages; job queue cleanly separated
from subscriber and logger
Functionality 10% All RPCs work end-to-
end; cache hits
observable; job lifecycle
visible in logs
GetDoctor/GetAppointment return cached
values on second call (verifiable with
Redis CLI MONITOR);
UpdateAppointmentStatus to done triggers
job; gateway call logged; retry triggered by
503
Documentation &
Defense
10% README quality and
answers during defense
Cache strategy choice justified per
endpoint; rate-limiting algorithm explained;
job lifecycle documented; consistency
trade-offs and dead-letter strategy
explained clearly
16. Defense Checkpoints
Checkpoint 1 — Cache Hit
Call GetDoctor twice with the same ID using grpcurl. The grader will run redis-cli MONITOR and
verify that only the first call produces a GET miss followed by a SET, while the second call
produces only a GET hit with no database query.

Checkpoint 2 — Rate Limiter
The grader will send more than RATE_LIMIT_RPM requests per minute to any gRPC endpoint.
The service must return codes.ResourceExhausted for requests exceeding the limit. It must
resume accepting requests after the window resets.

Checkpoint 3 — Job Queue & Gateway
Call UpdateAppointmentStatus with status = "done" via grpcurl. The grader will observe:

The Notification Service terminal prints the appointments.status_updated event log line.
Within a few seconds, the job-queue log shows status = "processing" and then status =
"success" (or status = "retry" if the gateway returned 503, followed eventually by status =
"success").
The Mock Gateway terminal prints the received POST /notify request.
Checkpoint 4 — Idempotency
AP2 — Assignment
4

The grader will replay the same appointments.status_updated event message manually into the
broker. The Notification Service must log the event, check the idempotency key in Redis, find it
already processed, and drop the job silently (no second gateway call, no new job log entry
beyond the drop notice).

Checkpoint 5 — Dead Letter
The grader will stop the Mock Gateway and trigger an UpdateAppointmentStatus to done. The
Notification Service must attempt the job 3 times (each logged as status = "retry"), then write a
status = "dead_letter" entry to stderr. The service must continue running and processing
subsequent events normally.

If the caching layer is absent or Redis is not used, the Caching Layer criterion is capped at 0%.
If the job queue produces no output or the Mock Gateway is absent, the Background Job Queue
criterion is capped at 50%.

17. Submission Format
Submit a single compressed project folder as a .ZIP file to the LMS. The archive must be named
AP2_Assignment4_NameSurname.zip and must contain all four services in a runnable state.

All four services must compile and start with go run. from their respective directories, with no
additional setup steps beyond setting environment variables and starting infrastructure (Redis,
PostgreSQL, broker).

Submissions that do not compile will not be graded.

Recommended resources:

github.com/redis/go-redis/v9 — official Go Redis client documentation.
redis.io/docs/manual/patterns — Redis patterns including rate limiting and idempotency.
grpc.io/docs/guides/interceptors — gRPC interceptor documentation for Go.
pkg.go.dev/sync — Go sync primitives for worker pool implementation.
You are expected to read and apply the documentation independently.
