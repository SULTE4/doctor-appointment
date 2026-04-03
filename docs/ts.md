Advanced Programming 2
Assignment 1 – Clean Architecture-Based Microservices
Scope: Lecture 1 & Lecture 2
Deadline: 23:59 03. 04.

Submission:

 ZIP archive named AP2_Assignment 1 _NameSurname.zip
 Upload to Moodle
 Project must compile and run using `go run .`
Defense: All practice classes in Week 3. You will be asked to download your

submission from the LMS, demonstrate your results, and answer questions about your

design decisions.

Cheating is strictly prohibited and will result in a grade of 0.

Scenario: Build a Two-Service Medical Scheduling Platform in Go using Clean

Architecture and REST-based microservices.

Communication Constraint: REST only (use Gin as the HTTP framework).

Assignment Overview
You are asked to build a small two-service platform composed of a Doctor
Service and an Appointment Service. Each service must follow Clean
Architecture principles, and the overall system must demonstrate service
decomposition, bounded contexts, separate data ownership, synchronous inter-
service communication, and basic failure handling.
Learning Objectives
 Apply Separation of Concerns and Dependency Inversion in a real Go project.
 Design a layered service structure with domain, use case, repository, delivery, and
application wiring layers.
 Decompose a system into two bounded contexts with clear ownership boundaries.
 Implement synchronous inter-service communication using REST.
 Explain the trade-offs of a microservices architecture compared with a monolith
in a simple system.
 Handle a basic failure scenario when one service depends on another service over
the network.
System Scope
Services
 Doctor Service - manages doctor profile data.
 Appointment Service - manages appointment data and validates doctor
existence through the Doctor Service.
Communication Model
 The system must use REST only. The Appointment Service must call the Doctor
Service over HTTP to verify that a doctor exists before an appointment is created
or updated. Message queues, event brokers, and asynchronous communication are
out of scope for this assignment.
Architecture Requirements
Clean Architecture Inside Each Service
 Thin delivery layer: handlers must parse incoming requests, delegate to use cases,
and return responses. No business logic belongs here.
 Business logic must live exclusively in the use case layer.
 Persistence logic must live in the repository layer.
 Domain models must not depend on HTTP, JSON transport concerns, or any
framework-specific types.
 Use cases must depend on interfaces, not on concrete storage or HTTP client
implementations.
Microservices Architecture Across the System

 Each service must own its own data and repository implementation.
 The Appointment Service must not access Doctor Service storage directly.
 The boundary between services must be explicit and enforced through REST
APIs only.
 Students must be able to explain why this design constitutes a microservices
decomposition rather than a distributed monolith.
Suggested Project Structure
service/
├── cmd/
│ └── service-name/
│ └── main.go
├── internal/
│ ├── model/
│ ├── usecase/
│ ├── repository/
│ ├── transport/
│ │ └── http/
│ └── app/
└── README.md
Notice: Both services should follow a similar structure. You may organize files

differently provided the layering and dependency direction remain clean and

straightforward to justify.

Domain Models
Doctor
type Doctor struct {
ID string
FullName string
Specialization string
Email string
}
Appointment
type Appointment struct {

ID string

Title string

Description string

DoctorID string

Status Status // define a custom Status type

CreatedAt time.Time

UpdatedAt time.Time

}

Functional Requirements
Doctor Service Endpoints
 POST /doctors - create a new doctor
 GET /doctors/{id} - retrieve a doctor by ID
 GET /doctors - list all doctors
Doctor Service Rules
 full_name is required.
 email is required.
 email must be unique across all doctors.
Appointment Service Endpoints
 POST /appointments - create a new appointment
 GET /appointments/{id} - retrieve an appointment by ID
 GET /appointments - list all appointments
 PATCH /appointments/{id}/status - update appointment status
Appointment Service Rules
 title is required.
 doctor_id is required.
 The referenced doctor must exist in the Doctor Service (validated over
REST).
 status must be one of: new, in_progress, done.
 Transitioning a status from done back to new is not allowed.
Example REST Payloads
Create a Doctor
{
"full_name": "Dr. Aisha Seitkali",
"specialization": "Cardiology",
"email": "a.seitkali@clinic.kz"
}
Create an Appointment
{
"title": "Initial cardiac consultation",
"description": "Patient referred for palpitations and
shortness of breath",
"doctor_id": "doctor-1"
}
Update Appointment Status
{

"status": "in_progress"

}

Required Failure Scenario
If the Doctor Service is unavailable when an appointment is being created or
updated, the Appointment Service must not proceed with the operation. It must
return a clear, descriptive error response to the client and log the failure
internally. A full circuit breaker implementation is not required. However,
students must be able to explain in their README and during the defense where
a timeout policy, retry strategy, or circuit breaker would become necessary as the
system scales.
Best Case and Worst-Case Design
Best Case
 Each service owns its own data and repository implementation
independently.
 Handlers remain thin; all business rules are enforced in the use case layer.
 The Appointment Service validates doctor existence by calling the Doctor
Service over REST.
 Interfaces are used for repositories and for the outbound Doctor Service
client.
 The codebase is structured so that it is easy to extend and easy to reason
about.
Worst Case

 Both services share one database or directly read each other's tables.
 The Appointment Service manipulates doctor data directly instead of calling
the Doctor Service API.
 Business rules are implemented inside HTTP handlers rather than use cases.
 There is no timeout or error handling when the Doctor Service fails or is
unreachable.
 The system is split into two processes but still behaves like a distributed
monolith, with tightly coupled logic and no real service boundary.
Deliverables
 Source code for both services.
 A README.md explaining the architecture and key design decisions.
 A simple architecture diagram showing both services, their owned data, and
the communication boundary between them.
 API examples or a Postman collection demonstrating all endpoints.
Documentation Requirements
 Project overview and purpose - what the system does and why it is structured
this way.
 Service responsibilities - what each service owns and manages.
 Folder structure and dependency flow - how layers are organized and in which
direction dependencies point.
 Inter-service communication - how and when the Appointment Service calls
the Doctor Service, and what HTTP contract is used.
 How to run the project - step-by-step instructions to start both services locally.
 Why a shared database was not used - a brief explanation of the data
ownership principle.
 Failure scenario - what happens when the Doctor Service is unavailable, how
the Appointment Service responds, and where more advanced resilience
patterns (timeouts, retries, circuit breakers) would fit in a production system.
Submission Format
 Submit a single compressed project folder as a .ZIP file to the LMS. The
archive must be named AP2_Assignment1_NameSurname.zip and must
contain both services in a runnable state. Submissions that do not compile will
not be graded.
Grading Rubric
Criterion Weight What Is
Evaluated
High-Performance
Indicators
Clean Architecture
inside services

30% Layering,
dependency
inversion,
separation of
concerns
Thin handlers, business
rules in use cases,
interface-based
dependencies, clean
domain boundaries
Microservice
decomposition

20% Service boundaries
and bounded
contexts
Clear data ownership per
service, no cross-service
data access, justified
decomposition
REST
communication

15% Correct
Appointment-to-
Doctor interaction
Doctor validation through
HTTP, appropriate status
codes returned, no direct
database bypass
Functionality 20% Required
endpoints and
business rules

All endpoints work
correctly and all stated
business rules are
enforced
Documentation and
explanation

15% README and
diagram quality
Architecture decisions,
trade-offs, failure
handling, and inter-service
communication are
explained clearly and
concisely
