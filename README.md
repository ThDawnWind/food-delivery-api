# Food Delivery API

A production-style REST API for a food delivery service built with Go and PostgreSQL.

The project is designed as a practical backend application with authentication, role-based authorization, product catalog management, saved delivery addresses, order processing, database migrations, automated tests, and Docker-based local development.

## Features

* User registration and authentication
* JWT-based authorization
* Role-based access control
* Product catalog
* Product images
* Category management
* Saved delivery addresses
* Order creation
* Delivery address snapshots
* Order history
* Order cancellation
* Admin order management
* Controlled order status transitions
* PostgreSQL transactions
* Database migrations with Goose
* Integration and unit tests
* Docker Compose development environment
* Graceful HTTP server shutdown
* HTTP server timeouts

## Tech Stack

* Go 1.27
* PostgreSQL 18
* pgx
* chi
* JWT
* bcrypt
* Goose
* Docker
* Docker Compose

## Architecture

The application follows a layered architecture:

```text
HTTP Request
     ↓
Handler
     ↓
Service
     ↓
Repository
     ↓
PostgreSQL
```

Responsibilities are separated between layers:

* **Handler** handles HTTP requests and responses.
* **Service** contains business logic and validation.
* **Repository** handles database access.
* **PostgreSQL** stores persistent application data.

## Project Structure

```text
food-delivery-api/
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── migrate/
│       └── main.go
│
├── internal/
│   ├── address/
│   ├── auth/
│   ├── category/
│   ├── config/
│   ├── database/
│   ├── httpx/
│   ├── order/
│   ├── product/
│   └── user/
│
├── migrations/
├── openapi/
│   └── openapi.yaml
│
├── Dockerfile
├── compose.yml
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

## Requirements

For the Docker-based setup:

* Docker
* Docker Compose

For local development without Docker:

* Go 1.27+
* PostgreSQL 18+

## Environment Variables

Copy the example configuration:

```bash
cp .env.example .env
```

On Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

Available environment variables:

```env
APP_ENV=local

HTTP_PORT=8080
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s

DATABASE_URL=postgres://food_delivery:food_delivery@localhost:5432/food_delivery?sslmode=disable
TEST_DATABASE_URL=postgres://food_delivery:food_delivery@localhost:55432/food_delivery_test?sslmode=disable

JWT_SECRET=replace-with-a-long-random-secret
JWT_TTL=24h

APP_TIMEZONE=UTC
```

`JWT_SECRET` should contain at least 32 characters.

Do not commit the real `.env` file.

## Running with Docker

Build and start the application:

```bash
docker compose up --build
```

Docker Compose starts the services in the following order:

```text
PostgreSQL
     ↓
Database migrations
     ↓
Food Delivery API
```

The migration container waits until PostgreSQL is healthy, applies all pending database migrations, and exits successfully.

The API starts only after the migration step has completed.

The API is available at:

```text
http://localhost:8080
```

Health check:

```text
GET /health
```

Example:

```bash
curl http://localhost:8080/health
```

Expected response:

```text
OK
```

## Running Locally

Start PostgreSQL:

```bash
docker compose up -d postgres
```

Apply migrations:

```bash
go run ./cmd/migrate up
```

Run the API:

```bash
go run ./cmd/api
```

## Database Migrations

Database migrations are stored in:

```text
migrations/
```

Current migrations:

```text
001_create_users.sql
002_create_categories.sql
003_create_products.sql
004_create_addresses.sql
005_create_orders.sql
006_create_order_items.sql
007_add_indexes.sql
008_create_product_images.sql
```

Apply pending migrations:

```bash
go run ./cmd/migrate up
```

Rollback the latest migration:

```bash
go run ./cmd/migrate down
```

Check migration status:

```bash
go run ./cmd/migrate status
```

## Testing

Start the test PostgreSQL instance:

```bash
docker compose --profile test up -d postgres-test
```

Run all tests:

```bash
go test ./...
```

With `gotestsum`:

```bash
gotestsum --format testname -- -count=1 ./...
```

Run static analysis:

```bash
go vet ./...
```

Format the project:

```bash
go fmt ./...
```

## API Overview

### Authentication

Public endpoints:

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
```

### Categories

Public endpoints:

```text
GET /api/v1/categories
GET /api/v1/categories/{id}
```

Admin endpoints:

```text
POST   /api/v1/categories
PUT    /api/v1/categories/{id}
DELETE /api/v1/categories/{id}
```

### Products

Public endpoints:

```text
GET /api/v1/products
GET /api/v1/products/{id}
```

Product listing supports:

```text
search
category_id
limit
offset
```

Admin endpoints:

```text
POST   /api/v1/products
PUT    /api/v1/products/{id}
DELETE /api/v1/products/{id}
```

`DELETE /api/v1/products/{id}` soft-deactivates the product instead of physically deleting it from the database.

### Addresses

Authenticated users can manage their saved delivery addresses.

```text
POST   /api/v1/addresses
GET    /api/v1/addresses
GET    /api/v1/addresses/{id}
PATCH  /api/v1/addresses/{id}
DELETE /api/v1/addresses/{id}
```

Users can access only their own addresses.

### Orders

Authenticated user endpoints:

```text
POST  /api/v1/orders
GET   /api/v1/orders
GET   /api/v1/orders/{id}
PATCH /api/v1/orders/{id}/cancel
```

User order listing supports pagination:

```text
limit
offset
```

Admin endpoints:

```text
GET   /api/v1/admin/orders
GET   /api/v1/admin/orders/{id}
PATCH /api/v1/admin/orders/{id}/status
```

Admin order listing supports:

```text
status
user_id
from
to
limit
offset
```

## Order Lifecycle

Orders follow a controlled status lifecycle:

```text
new
 ↓
confirmed
 ↓
cooking
 ↓
ready
 ↓
delivering
 ↓
completed
```

Orders can be cancelled only from supported early states:

```text
new ────────→ cancelled

confirmed ──→ cancelled
```

Invalid status transitions are rejected by the service layer.

Status updates also use the current database state to prevent conflicting concurrent transitions.

## Delivery Address Snapshot

An order references a saved delivery address when it is created.

The delivery address is also copied into the order as a snapshot.

This means changing or deleting the saved address later does not modify the historical delivery address stored with an existing order.

## Authentication and Authorization

The API uses JWT access tokens.

Protected requests use:

```text
Authorization: Bearer <token>
```

Two roles are supported:

```text
user
admin
```

Users can access only their own protected resources.

Administrative catalog and order endpoints require the `admin` role.

## Error Responses

API errors use a consistent JSON structure:

```json
{
  "error": "error message"
}
```

Depending on the endpoint, the API can return:

```text
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
500 Internal Server Error
```

## API Documentation

The API specification is being documented using:

* OpenAPI 3.1
* Scalar

The OpenAPI specification is stored in:

```text
openapi/openapi.yaml
```

Planned HTTP endpoints for interactive documentation:

```text
GET /openapi.yaml
GET /docs
```

Scalar will provide an interactive API reference based on the OpenAPI specification.

## Development Status

Core API functionality is implemented.

Current focus:

* OpenAPI specification
* Scalar API reference
* CI pipeline
* final production-readiness review

## License

This project is intended primarily for learning, portfolio, and backend development practice.
