# Card Vault (PCI-DSS Tokenization Service)

**Card Vault** is a lightweight, high-performance payment card tokenization service written in Go. The service validates incoming card details, conceals sensitive cardholder data (PAN masking, CVV elimination), generates secure tokens, and provides fast, resilient access using a two-tier storage layer: **PostgreSQL** (persistent storage with transactional audit logging) and **Redis** (Cache-Aside caching layer).

---

## Key Features

* **Card Validation**:
  * Card number (PAN) verification via the Luhn algorithm (13–19 digits).
  * Expiration date validation (month/year must not be in the past).
  * CVV format checks (strictly 3 or 4 digits).
* **Data Security & Privacy**:
  * PAN masking: retains only the first 6 (BIN) and last 4 digits (`455384******3752`).
  * CVV omission: CVV is never persisted or stored in cache.
  * Sensitive data masking: custom `String()` method prevents leaking PAN and CVV in log outputs.
* **Audit Trail**:
  * Automatically records token generation events in `token_audit_logs` within an atomic database transaction.
* **Performance & Caching**:
  * Implements the **Cache-Aside** pattern using Redis with JSON serialization and configurable TTL (10 minutes).
  * Significantly reduces database load on frequent token lookups.
* **Resilience & Production Readiness**:
  * Clean **Graceful Shutdown** handling OS termination signals (`SIGINT`, `SIGTERM`) with context timeout.
  * Structured JSON logging via standard library `log/slog`.
  * HTTP request logging middleware capturing method, URL, client IP, status code, and latency.

---

## Tech Stack

* **Language**: Go 1.27
* **HTTP Router**: Standard `net/http.ServeMux` (Go 1.22+ parameterized routing)
* **Database**: PostgreSQL 18 (`pgx/v5` with connection pooling via `pgxpool`)
* **Caching**: Redis (latest Alpine, `redis/go-redis/v9`)
* **Containerization**: Docker & Docker Compose (multi-stage build on Alpine Linux)

---

## Architecture & Project Structure

The project follows Clean Architecture / DDD-lite design principles:

```text
.
├── cmd/
│   └── server/
│       └── main.go                  # Application entrypoint, DI, Graceful Shutdown
├── internal/
│   ├── domain/                      # Domain entities, validation rules, and domain errors
│   │   └── card.go
│   ├── application/                 # Business logic (tokenization, Luhn checks, PAN masking)
│   │   └── card_service.go
│   ├── repository/
│   │   ├── postgres/                # PostgreSQL adapter (transactional storage, audit logs)
│   │   │   └── card_repo.go
│   │   └── rediscache/              # Redis adapter (Cache-Aside pattern, JSON serialization)
│   │       └── card_cache.go
│   └── interface/
│       └── http/                    # HTTP handlers, routing, and middleware
│           ├── handler.go
│           └── middleware.go
├── postgres/
│   └── 001_create_cards_and_audit_tables.sql # Database schema initialization script
├── docker-compose.yaml              # Multi-container orchestration (App, Postgres, Redis)
├── Dockerfile                       # Multi-stage Docker build
├── .env.example                     # Environment variables template
└── README.md
```

---

## Configuration & Environment Variables

Copy the example environment file to `.env`:

```bash
cp .env.example .env
```

| Variable | Description | Example Value |
| :--- | :--- | :--- |
| `POSTGRES_URL` | PostgreSQL connection string | `postgres://postgres:password!@postgres:5432/card_vault_db` |
| `POSTGRES_HOST` | Database host | `postgres` |
| `POSTGRES_USER` | Database user | `postgres` |
| `POSTGRES_PASSWORD` | Database password | `password` |
| `POSTGRES_DB` | Database name | `card_vault_db` |
| `POSTGRES_PORT` | Database port | `5432` |
| `REDIS_URL` | Redis server address (`host:port`) | `redis:6379` |
| `REDIS_PASSWORD` | Redis password (if required) | `""` |

---

## Getting Started

### Running with Docker Compose (Recommended)

1. Clone the repository and navigate into the project directory:
   ```bash
   cd <project_directory>
   ```

2. Create your `.env` configuration file:
   ```bash
   cp .env.example .env
   ```

3. Build and launch all services:
   ```bash
   docker compose up -d --build
   ```

Docker Compose sets up:
* **PostgreSQL** on internal port `5432` with automated schema initialization.
* **Redis** mapped to port `6380:6379` (to avoid conflicting with any locally running Redis instance).
* **Card Vault API** exposed on port `8080`.

To view application logs in real-time:
```bash
docker compose logs -f app
```

---

## API Specification

### 1. Healthcheck
Verify service readiness and availability.

* **Method**: `GET /health`
* **Response**: `200 OK`
```json
{
  "status": "ok"
}
```

---

### 2. Tokenize Card
Validates credit card details, stores the masked card in PostgreSQL and Redis, and returns the generated token.

* **Method**: `POST /api/v1/cards/tokenize`
* **Content-Type**: `application/json`
* **Request Body:**
```json
{
  "card_number": "4553840012343752",
  "holder_name": "JOHN DOE",
  "exp_month": 10,
  "exp_year": 2030,
  "cvv": "123"
}
```
* **Response (`201 Created`):**
```json
{
  "token": "token_92c5e4c3-63d7e7aa-a282b84a-92716c82",
  "masked_pan": "455384******3752",
  "exp_month": 10,
  "exp_year": 2030,
  "created_at": "2026-10-10T20:15:00.123456Z"
}
```

#### Validation Errors (`400 Bad Request`):
* `{"error": "invalid card number"}` — PAN failed Luhn checksum or has invalid length.
* `{"error": "card is expired"}` — Card expiration date is in the past.
* `{"error": "invalid card cvv: must be 3 or 4 digits"}` — Invalid CVV format.
* `{"error": "invalid json body"}` — Malformed JSON or request payload exceeds 10 KB.

---

### 3. Retrieve Card by Token
Retrieves tokenized card information. The service first queries the Redis cache; on cache miss, it fetches data from PostgreSQL and populates the cache.

* **Method**: `GET /api/v1/cards/{token}`
* **Example URL**: `GET /api/v1/cards/token_92c5e4c3-63d7e7aa-a282b84a-92716c82`
* **Response (`200 OK`):**
```json
{
  "token": "token_92c5e4c3-63d7e7aa-a282b84a-92716c82",
  "masked_pan": "455384******3752",
  "exp_month": 10,
  "exp_year": 2030,
  "created_at": "2026-10-10T20:15:00.123456Z"
}
```

#### Error Responses:
* `404 Not Found`: `{"error": "token card not found"}` — No card associated with the provided token.
* `400 Bad Request`: `{"error": "token is required"}` — Empty token parameter.

---

## Database Schema

The database schema is initialized via `postgres/001_create_cards_and_audit_tables.sql`:

```sql
CREATE TABLE IF NOT EXISTS cards (
    id VARCHAR(64) PRIMARY KEY,
    masked_pan VARCHAR(20) NOT NULL,
    exp_month INT NOT NULL CHECK (exp_month BETWEEN 1 AND 12),
    exp_year INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS token_audit_logs (
    id SERIAL PRIMARY KEY,
    token_id VARCHAR(64) NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cards_created_at ON cards(created_at);
```
