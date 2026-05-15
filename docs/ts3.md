Advanced Programming 2
Assignment 3 – Message Queue & Database Migrations
Scope: Lecture 5 & Lecture 6
Field Details
Scope Lecture 5 (Message Queues) & Lecture 6 (Migrations & Transactions)
Deadline 23:59 03.05.
Submission Moodle – ZIP archive named AP2_Assignment3_NameSurname.zip
Defense Week 7- 8 practice classes
Cheating Policy Strictly prohibited — results in grade 0
1. Assignment Overview
In Assignments 1 and 2 you built a two-service Medical Scheduling Platform using REST, then migrated all inter-
service communication to gRPC. In this assignment you will extend the system in two directions:

Introduce asynchronous, event-driven communication using a message broker. After every successful
write operation, the responsible service publishes a domain event. A new third service — the
Notification Service — subscribes to those events and reacts to them.
Replace in-memory storage with a real PostgreSQL database and manage the schema exclusively
through versioned migration files using golang-migrate.
The domain logic, Clean Architecture layering, bounded-context boundaries, and gRPC contracts from
Assignment 2 must be fully preserved. Only the infrastructure layer changes.

2. Message Broker Choice
You must choose one of the following two brokers and use it consistently across all three services. You may not
mix brokers.

NATS (Core)^ RabbitMQ^
Go client github.com/nats-io/nats.go github.com/rabbitmq/amqp091-go
Local start nats-server binary or Docker Docker (rabbitmq:3-management)
Delivery model Pub/Sub, fire-and-forget Pub/Sub via fanout exchange, persistent
queues
Persistence None (core NATS) Queue-level durability
Config variable NATS_URL (e.g. nats://localhost:4222) AMQP_URL (e.g.
amqp://guest:guest@localhost:5672/)
When to choose Simpler setup, stateless notifications Need durable queues or guaranteed
delivery
Important — document your choice
State in your README which broker you chose and why.
Be prepared during defense to explain what would need to change if you switched to the other broker,
and where durable delivery (RabbitMQ queues or NATS JetStream) would become necessary in production.
3. Learning Objectives
Publish domain events from a Go service to a message broker after a successful state change.
Build a standalone subscriber service that consumes events from a broker and reacts to them.
Understand Pub/Sub vs. Point-to-Point messaging and articulate when each model is appropriate.
Provision a PostgreSQL database and write versioned SQL migration files.
Use golang-migrate to apply, verify, and roll back migrations.
Replace in-memory repository implementations with PostgreSQL-backed ones while keeping repository
interfaces unchanged.
Wrap multi-step write operations in database transactions and explain the ACID properties they provide.
Explain the consistency trade-offs introduced by asynchronous event publishing.
4. Scope and Constraints
What changes
Both existing services connect to PostgreSQL. In-memory maps are replaced by database-backed
repositories.
Schema is managed exclusively through migration files — no auto-migration or raw DDL in application
code.
After each successful write operation, the responsible service publishes a domain event to the chosen
broker.
A new Notification Service subscribes to those events and handles them (see Section 6 for full details).
What does NOT change
Domain models (Doctor, Appointment, Status) remain identical.
Use-case logic and business rules remain identical.
gRPC service contracts and generated stubs remain identical.
Clean Architecture layering and dependency direction remain identical.
The gRPC failure scenario: if the Doctor Service is unreachable, the Appointment Service must still return
a descriptive gRPC error.
Notice: REST endpoints, mixed-transport layers, and synchronous event handling via HTTP callbacks or gRPC
streaming are out of scope.

5. Infrastructure Requirements
5.1 PostgreSQL
Each service must have its own database or its own schema — shared tables across services are not
permitted.
The connection string must be read from an environment variable (DATABASE_URL or DB_DSN).
The repository layer must use database/sql or pgx/v5 directly. ORMs are not permitted.
All write operations that involve multiple steps must be wrapped in a database transaction.
5.2 Migrations (golang-migrate)
Use github.com/golang-migrate/migrate/v4 to manage schema versions.
Migration files must live at migrations/ inside each service directory and follow this naming convention:
000001_create_doctors.up.sql
000001_create_doctors.down.sql
Migrations must run automatically on service startup, before the gRPC server begins accepting requests.
Down migrations must correctly undo the corresponding up migration — they will be tested during
defense.
No application code may contain raw DDL (CREATE TABLE, ALTER TABLE, DROP TABLE, etc.) outside
migration files.
5.3 Message Broker
Use a single broker instance started locally or via Docker.
The broker connection URL must be read from an environment variable (NATS_URL or AMQP_URL).
Published events must be serialized as JSON and include at minimum: event_type, occurred_at
(RFC3339), and the relevant entity payload.
NATS: use core NATS Pub/Sub (not JetStream). Publishing is fire-and-forget.
RabbitMQ: declare a fanout exchange named ap2.events. Each subscriber binds its own exclusive queue
to that exchange.
6. Event Definitions
Each service must publish the following events after a successful operation. The subject/routing-key column
shows the NATS subject or RabbitMQ routing key to use.

Service Trigger Subject / Routing Key Required JSON Fields
Doctor Service CreateDoctor succeeds doctors.created event_type, occurred_at,
id, full_name,
specialization, email
Appointment
Service
CreateAppointment
succeeds
appointments.created event_type, occurred_at,
id, title, doctor_id, status
Appointment
Service
UpdateAppointmentStatus
succeeds
appointments.status_updated event_type, occurred_at,
id, old_status, new_status
Example event payload (appointments.created):

{
"event_type": "appointments.created",
"occurred_at": "2026- 05 - 01T10:23:44Z",
"id": "appt-1",
"title": "Initial cardiac consultation",
"doctor_id": "doc-1",
"status": "new"
}
7. Notification Service
The Notification Service is the third Go binary in this system. Its sole responsibility is to listen for events
published by the Doctor Service and the Appointment Service, and to react to each event by logging a structured
record to standard output.

It has no gRPC server, no HTTP server, and no database. It does nothing except receive and log events.

7.1 What it must do
On startup the Notification Service must:

◦ Connect to the message broker (same broker you chose in Section 2).
◦ Subscribe to all three subjects / routing keys: doctors.created, appointments.created, and
appointments.status_updated.
◦ If the broker is unavailable at startup, retry with an exponential backoff (e.g. 1s, 2s, 4s) for a
reasonable number of attempts, then exit with a non-zero status code and a descriptive error
message.
Each time a message arrives the Notification Service must:

◦ Deserialize the JSON payload.
◦ Print one structured log line to stdout in JSON format. The log line must include at minimum: time,
subject, and the full deserialized event object.
◦ Acknowledge the message (relevant for RabbitMQ; NATS core requires no explicit ack).
7.2 Required log output format
The log line printed to stdout must be a single JSON object. It must include at minimum:

JSON Field Type Description Example Value
time string (RFC3339) When the Notification Service received
and processed the event
"2026- 05 -
01T10:23:44Z"
subject string The NATS subject or RabbitMQ routing
key on which the message arrived
"doctors.created"
event object The full deserialized payload as
published by the source service
{"id":"doc-
1","full_name":"..."}
Example stdout lines after a doctor is created and an appointment is created:

{"time":"2026- 05 -
01T10:23:44Z","subject":"doctors.created","event":{"event_type":"doctors.created","occurr
ed_at":"2026- 05 - 01T10:23:44Z","id":"doc-1","full_name":"Dr. Aisha
Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
{"time":"20 26 - 05 -
01T10:24:01Z","subject":"appointments.created","event":{"event_type":"appointments.create
d","occurred_at":"2026- 05 - 01T10:24:01Z","id":"appt-1","title":"Initial cardiac
consultation","doctor_id":"doc-1","status":"new"}}
{"time":"2026- 05 -
01T10:25:10Z","subject":"appointments.status_updated","event":{"event_type":"appointments
.status_updated","occurred_at":"2026- 05 - 01T10:25:10Z","id":"appt-
1","old_status":"new","new_status":"in_progress"}}
7.3 What it must NOT do
It must not call any gRPC endpoint.
It must not write to any database.
It must not expose any network port.
It must not silently drop messages without logging an error.
7.4 Startup and shutdown behaviour
Environment variable: use the same NATS_URL or AMQP_URL as the other services.
The service must start with go run. from the notification-service/ directory.
It must stay running and keep consuming messages until the process is stopped.
On clean shutdown (SIGTERM / SIGINT) it must drain in-flight messages, close the broker connection,
and exit with code 0.
Defense checkpoint — Notification Service
During defense you will start all three services and all infrastructure, then call CreateDoctor and
CreateAppointment via grpcurl.
The grader will check that the Notification Service terminal prints a correctly structured log line for each event
within a few seconds.
If the Notification Service is absent or produces no output, the NATS/RabbitMQ criterion is capped at 50%.
8. Schema Requirements
The following schemas are the minimum required. Additional columns and indexes are allowed but must also be
managed through migration files.

doctors table (Doctor Service database)
CREATE TABLE doctors (
id TEXT PRIMARY KEY,
full_name TEXT NOT NULL,
specialization TEXT NOT NULL DEFAULT '',
email TEXT NOT NULL UNIQUE,
created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
appointments table (Appointment Service database)
CREATE TABLE appointments (
id TEXT PRIMARY KEY,
title TEXT NOT NULL,
description TEXT NOT NULL DEFAULT '',
doctor_id TEXT NOT NULL,
status TEXT NOT NULL DEFAULT 'new',
created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
9. Suggested Project Structure
ap2-assignment3/
├── doctor-service/
│ ├── cmd/doctor-service/main.go
│ ├── internal/
│ │ ├── model/
│ │ ├── usecase/
│ │ ├── repository/ ← PostgreSQL implementation
│ │ ├── transport/grpc/ ← unchanged from Assignment 2
│ │ ├── event/ ← broker publisher
│ │ └── app/
│ ├── migrations/
│ │ ├── 000001_create_doctors.up.sql
│ │ └── 000001_create_doctors.down.sql
│ └── proto/ ← unchanged from Assignment 2
├── appointment-service/
│ ├── cmd/appointment-service/main.go
│ ├── internal/
│ │ ├── model/
│ │ ├── usecase/
│ │ ├── repository/ ← PostgreSQL implementation
│ │ ├── transport/grpc/ ← unchanged from Assignment 2
│ │ ├── client/ ← Doctor Service gRPC client
│ │ ├── event/ ← broker publisher
│ │ └── app/
│ ├── migrations/
│ │ ├── 000001_create_appointments.up.sql
│ │ └── 000001_create_appointments.down.sql
│ └── proto/ ← unchanged from Assignment 2
├── notification-service/
│ ├── cmd/notification-service/
│ │ └── main.go ← subscriber + logger
│ └── internal/
│ └── subscriber/ ← broker connection and message handling
└── README.md
Notice: You may organize files differently provided layering and dependency direction remain clean and easy to
justify.

10. Error Handling Requirements
All gRPC status codes from Assignment 2 remain in force. The following cases are added:

Situation Expected Behaviour
Database unavailable on startup Service exits with non-zero code and a descriptive log message.
Database query fails at runtime Return codes.Internal with a descriptive message.
Broker unavailable at startup (Doctor /
Appointment Service)
Service starts normally. Log a warning. The RPC still succeeds —
broker publishing is best-effort.
Broker publish fails during an RPC Log the error with enough context to debug. The RPC response is
not affected.
Broker unavailable at startup (Notification
Service)
Retry with backoff. Exit with non-zero code after max retries.
Duplicate email in DB Return codes.AlreadyExists.
Row not found in DB Return codes.NotFound.
Consistency note
Because broker publishing is best-effort, a process crash between the DB commit and the publish will cause
the event to be lost.
Be prepared to explain this trade-off during defense, and to describe how the Outbox pattern or durable
broker features
(RabbitMQ publisher confirms, NATS JetStream) would address it.
11. Best-Case and Worst-Case Design
Best Case
Each service has its own isolated database. No cross-service table access.
Migration files are correctly numbered, clean SQL, and reversible. Down migrations are verified.
Repository interfaces are unchanged from Assignment 2; only the implementation swaps to PostgreSQL.
The broker publisher is abstracted behind an EventPublisher interface and injected into the use case.
A broker publish failure never blocks the gRPC response; it is logged with full context.
The Notification Service connects to the broker, subscribes to all three subjects, prints one structured
JSON log line per event, and handles broker reconnection.
All three services are started with a single go run. each, and no URLs are hardcoded.
Worst Case
Both services share one database or access each other's tables directly.
Schema is created by raw SQL in application code; migration files are missing or incorrect.
Broker publishing is placed inside the gRPC handler rather than in a dedicated publisher.
A broker failure causes the RPC to return an error.
The Notification Service is missing, produces no output, or prints unstructured plain text.
Connection URLs are hardcoded in source code.
12. Deliverables
Source code for both existing services (updated) and the new Notification Service.
Migration files (.up.sql and .down.sql) for both services.
All proto files and generated Go stubs from Assignment 2 (unchanged).
README.md — see Section 13 for required content.
Updated architecture diagram showing: Doctor Service, Appointment Service, Notification Service, two
PostgreSQL databases, the message broker, gRPC arrows, and event-publish arrows with subject labels.
Testing artifact: grpcurl command list (from Assignment 2) extended with the expected Notification
Service log output that results from each command.
13. Documentation Requirements
Project overview — what changed compared to Assignment 2 and why.
Broker choice — which broker you chose (NATS or RabbitMQ) and the reason for that choice.
Environment variables — complete list for all three services (DATABASE_URL / DB_DSN, NATS_URL /
AMQP_URL, gRPC ports).
Infrastructure setup — how to start PostgreSQL and the broker (Docker commands or equivalent).
Migration instructions — how to apply and roll back migrations using golang-migrate CLI or the
automatic startup behaviour.
Service startup order — which service to start first and why, with the exact go run. command for each.
Event contract — each subject/routing key, which service publishes it, the trigger, and the full JSON
structure.
Notification Service explanation — what it does, how to read its log output, and how to verify that
events arrive during a live demo.
Consistency trade-offs — what happens when the broker is unavailable, which events can be lost, and
how durable delivery (Outbox pattern, JetStream, RabbitMQ confirms) would improve reliability.
Broker comparison — at least two concrete differences between NATS (core) and RabbitMQ, and when
you would choose one over the other.
14. Submission Format
Submit a single compressed project folder as a .ZIP file to the LMS. The archive must be named
AP2_Assignment3_NameSurname.zip and must contain all three services in a runnable state.

All three services must compile and start with go run. from their respective directories, with no additional setup
steps beyond setting environment variables. Submissions that do not compile will not be graded.

15. Grading Rubric
Criterion Weight What Is Evaluated High-Performance Indicators
Database & Migrations 30% PostgreSQL integration;
golang-migrate usage;
schema correctness;
transaction handling
Separate DB per service; migration files
correctly numbered, clean SQL, and
reversible; no raw DDL in app code;
repository interfaces unchanged from
Assignment 2
Message Broker &
Notification Service
25% Correct events published
after every write;
Notification Service
subscribes to all three
subjects and prints
structured JSON log lines
All three events published with correct
payloads; publisher behind an
interface; broker failures logged and do
not block RPC; Notification Service
prints one JSON line per event to stdout
Clean Architecture
Preservation
15% Layering and dependency
direction after
infrastructure changes
Handlers remain thin; use cases
unchanged; DB and broker
dependencies injected via interfaces;
no infrastructure types leak into
domain or use-case layer
Functionality 15% All RPCs work end-to-end
with real DB; events
observed in Notification
Service logs
All RPCs satisfy business rules with
PostgreSQL; down migrations work
during defense; Notification Service
logs all three event types correctly
Documentation &
Defense
15% README quality and
answers during defense
Setup instructions complete and
accurate; event contract documented;
broker choice justified; consistency
trade-offs and broker comparison
explained clearly
Recommended resources: nats.io and github.com/nats-io/nats.go — official NATS documentation and Go client.
rabbitmq.com/tutorials and github.com/rabbitmq/amqp091-go — RabbitMQ Go tutorial and client. github.com/golang-
migrate/migrate — golang-migrate documentation and CLI reference. You are expected to read and apply the
documentation independently.
