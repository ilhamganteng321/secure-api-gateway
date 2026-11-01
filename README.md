# Secure API Gateway

A security-focused API Gateway built with **Go, Fiber v3, PostgreSQL, and Redis**. Designed to provide centralized authentication, authorization, rate limiting, request auditing, and reverse proxy capabilities for backend services.

The project explores backend engineering concepts such as API key management, JWT authentication, RBAC, idempotency, distributed rate limiting, circuit breaking, observability, and production readiness.

---

## Overview

Secure API Gateway acts as an entry point between API consumers and backend services. It centralizes cross-cutting concerns so individual services do not need to implement every gateway-level security and traffic management feature independently.

### Key Capabilities

- **Authentication:** JWT access tokens and refresh token rotation.
- **Authorization:** Role-Based Access Control (RBAC) and API key scopes.
- **API Key Management:** Secure API key generation, hashed storage, validation, and revocation.
- **Rate Limiting:** Redis-backed request limiting.
- **Reverse Proxy:** Forward requests to configured upstream services.
- **Idempotency:** Protection against duplicate request processing.
- **Audit Logging:** Record security-sensitive and administrative activities.
- **Resilience:** Timeout handling, retry policies, and circuit breaker patterns.
- **Observability:** Structured logging, Prometheus metrics, and OpenTelemetry tracing.
- **Security:** Request validation, CORS policies, security headers, and upstream protection.

---

## Architecture

```mermaid
flowchart TD
    A[API Consumer] --> B[Secure API Gateway]

    B --> C[Request ID & Security Middleware]
    C --> D[Authentication]
    D --> E[Authorization & API Scopes]
    E --> F[Rate Limiter]
    F --> G[Request Validation]
    G --> H[Idempotency]
    H --> I[Gateway Router]

    I --> J[Load Balancer]
    J --> K[Circuit Breaker]
    K --> L[Upstream Services]

    B --> M[(PostgreSQL)]
    B --> N[(Redis)]

    B --> O[Audit Logging]
    B --> P[Prometheus Metrics]
    B --> Q[OpenTelemetry Tracing]
```

### Technology Stack

| Category | Technology |
|---|---|
| Language | Go |
| HTTP Framework | Fiber v3 |
| Database | PostgreSQL |
| Database Driver | pgx |
| Cache & Rate Limiting | Redis |
| Authentication | JWT |
| Password Hashing | Argon2id |
| Logging | Go `slog` |
| Metrics | Prometheus |
| Distributed Tracing | OpenTelemetry |
| Resilience | Circuit Breaker, Retry, Timeout |
| Load Testing | k6 |
| Testing | Go Testing, `httptest` |
| CI/CD | GitHub Actions |

---

## Core Features

### 1. Authentication & Authorization

The gateway supports two distinct authorization flows:

**JWT Authentication**
- Access token validation.
- Refresh token rotation.
- Refresh token storage using hashed values.
- Token revocation and logout.
- Refresh token reuse detection.

**API Key Authentication**
- Cryptographically secure API key generation.
- Store API key hashes instead of plaintext keys.
- API key validation and revocation.
- Client-based authentication.
- Scope-based access control.

RBAC is used to control access to management operations, while API key scopes control access to gateway resources.

### 2. Redis Rate Limiting

Redis is used to coordinate rate limiting across requests and application instances.

Capabilities:
- Request counting.
- Per-client rate limiting.
- Atomic operations using Redis Lua scripts.
- Configurable rate limit policies.
- Controlled handling of Redis failures.

### 3. Idempotency

Idempotency protects operations from being processed multiple times when clients retry requests.

Example:

```http
POST /api/orders
Idempotency-Key: order-request-123
Content-Type: application/json
```

The gateway/application can use the idempotency key to identify duplicate operations and prevent unintended repeated processing.

### 4. Reverse Proxy & Upstream Routing

The gateway forwards requests to configured backend services.

Capabilities include:
- Upstream registration.
- Request forwarding.
- Load balancing.
- Upstream timeout handling.
- Error handling.
- Health checking.
- Circuit breaker integration.
- Retry policies for eligible requests.

Retries must be restricted to operations that are safe to retry or protected by idempotency guarantees.

### 5. Resilience Engineering

The project explores techniques for handling dependency and upstream failures.

| Pattern | Purpose |
|---|---|
| Timeout | Prevent requests from waiting indefinitely |
| Retry | Recover from eligible transient failures |
| Circuit Breaker | Stop repeatedly calling failing upstreams |
| Health Check | Monitor upstream availability |
| Load Balancing | Distribute requests across upstream instances |
| Graceful Shutdown | Allow controlled application termination |

### 6. Observability

The gateway is designed to expose operational information through:

**Structured Logging**
- Request ID.
- HTTP method and route.
- Response status.
- Request duration.
- Error information.

**Prometheus Metrics**
- Total gateway requests.
- Request duration.
- Upstream request duration.
- Upstream errors.
- Rate limit rejections.
- Active requests.
- Circuit breaker state.

**Distributed Tracing**
- OpenTelemetry instrumentation.
- Request trace propagation.
- Outgoing upstream HTTP tracing.
- OTLP exporter integration.

Sensitive information such as JWTs, passwords, API keys, and private request data must never be included in logs or trace attributes.

---

## Project Structure

```text
secure-api-gateway/
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── app/
│   ├── config/
│   ├── database/
│   ├── redis/
│   ├── auth/
│   ├── middleware/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   ├── gateway/
│   ├── observability/
│   └── resilience/
│
├── migrations/
├── tests/
│   ├── integration/
│   └── load/
│
├── docs/
├── scripts/
├── .github/
│   └── workflows/
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

The directory structure may evolve as the project develops.

---

## Getting Started

### Prerequisites

Install the following tools:

- Go
- PostgreSQL
- Redis
- Git

Docker is not required for local development.

### 1. Clone the Repository

```bash
git clone https://github.com/ilhamganteng321/secure-api-gateway.git

cd secure-api-gateway
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Configure Environment Variables

Copy the example environment file:

**Windows PowerShell**

```powershell
Copy-Item .env.example .env
```

**Linux / macOS**

```bash
cp .env.example .env
```

Configure the required database, Redis, and authentication settings.

Example configuration:

```env
APP_ENV=development
APP_PORT=3000

DATABASE_URL=postgres://postgres:password@localhost:5432/secure_gateway?sslmode=disable

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=replace-with-a-secure-random-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

LOG_LEVEL=debug
```

The variable names must match the configuration loader in the application.

Never commit real credentials or production secrets.

### 4. Prepare the Database

Create a PostgreSQL database:

```sql
CREATE DATABASE secure_gateway;
```

Run the project's database migrations using the migration procedure provided in the repository.

### 5. Start the Application

```bash
go run ./cmd/api
```

If the application entry point is located elsewhere, adjust the command accordingly.

---

## Testing

Run all tests:

```bash
go test ./... -count=1
```

Run tests with verbose output:

```bash
go test ./... -v
```

Run the race detector:

```bash
go test -race ./...
```

Run static analysis:

```bash
go vet ./...
```

Run benchmarks:

```bash
go test ./internal/... -run '^$' -bench . -benchmem
```

Generate coverage:

```bash
go test ./... -coverprofile=coverage.out

go tool cover -func=coverage.out
```

### Load Testing

Load testing uses k6.

```bash
k6 run tests/load/gateway.js
```

The load testing scenarios cover request throughput, latency, concurrency, and gateway behavior under increasing traffic.

---

## Security Considerations

Security is a core design concern of this project.

- API keys are stored as hashes.
- Passwords use Argon2id hashing.
- JWT secrets are managed through environment configuration.
- Management endpoints use authentication and authorization.
- Rate limiting protects API resources.
- Upstream destinations must follow explicit allowlist policies.
- DNS resolution and actual outbound connections must be protected against SSRF and DNS rebinding.
- Request size limits and security headers are configured.
- Sensitive data must not be exposed through logs, metrics, or traces.

This project is intended for engineering and educational purposes. Production deployment requires environment-specific security review, configuration validation, and operational testing.

---

## CI/CD

GitHub Actions is used to automate development quality checks.

The CI pipeline is designed to run:

- Go formatting validation.
- Dependency verification.
- Static analysis with `go vet`.
- Automated tests.
- Race detection.
- Linux binary builds.
- Security vulnerability checks.

Build artifacts can be used in a subsequent release and deployment workflow.

---

## Development Roadmap

| Stage | Focus |
|---|---|
| 01–08 | Core application, database, authentication, middleware, and security |
| 09 | Performance, observability, and resilience foundations |
| 10 | API key management, reverse proxy, and gateway security |
| 11 | Distributed tracing, metrics, SLO, and operational readiness |
| 12A–12B | Integration, security, and failure injection testing |
| 12C | Performance and scalability testing |
| 12D | Deployment and production release |
| 12E | GitHub Actions CI/CD |
| 12F | Automated security scanning |
| 12G | Blue-green deployment |
| 12H | Final architecture review |

---

## Learning Objectives

This project is also an exploration of advanced backend engineering concepts:

- API Gateway architecture.
- Authentication and authorization design.
- Distributed rate limiting.
- Database transactions and concurrency.
- Idempotent request processing.
- Reverse proxy and load balancing.
- Resilience engineering.
- Distributed tracing and observability.
- Security testing.
- CI/CD and production deployment.

---

## License

Choose and add an appropriate open-source license before publishing this repository.

---

**Built with Go. Focused on Backend Engineering, API Security, and System Reliability.**
