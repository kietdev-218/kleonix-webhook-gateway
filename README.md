# KLeonix Webhook Gateway

A highly resilient, lightweight, and robust webhook gateway built in Go. It acts as the primary ingress for **Ory Kratos** identity webhooks, validates and standardizes the payloads, and reliably distributes them to downstream microservices via **RabbitMQ**.

## 🏗 Architecture & Design Principles

This project is built with **Clean Architecture** and adheres strictly to **SOLID** principles, ensuring long-term maintainability and high performance:

- **Dependency Inversion Principle (DIP)**: The core business logic (`internal/handler`) defines its own `Publisher` interface. The `rabbitmq` package implements this interface, ensuring the core domain is completely decoupled from the specific message broker technology (allowing easy migration to Kafka or AWS SQS in the future).
- **Zero Data Loss**: Implements RabbitMQ **Publisher Confirms** (`PublishWithDeferredConfirm`). The HTTP API will only return a `200 OK` to Kratos *after* the broker explicitly acknowledges the receipt of the message.
- **High Throughput**: Utilizes a lock-free **RabbitMQ Channel Pool** to multiplex concurrent HTTP requests efficiently without exhausting AMQP channels or causing TCP bottlenecks.
- **Self-Healing Resilience**: Features a rock-solid background reconnect loop that gracefully survives broker crashes, network partitions, and aggressive TCP resets.
- **Exponential Backoff**: In the event of temporary broker unavailability, publish attempts are retried using exponential backoff to prevent thundering herd problems, bounded by the context timeout.

## 📂 Project Structure

Following the standard Go project layout:

```text
.
├── cmd/
│   └── gateway/             # Application entrypoint (main.go)
├── internal/
│   ├── api/                 # HTTP Server initialization and Gin router wiring
│   ├── config/              # Environment variable parsing and defaults
│   ├── event/               # Domain models (e.g., standard Event Envelope)
│   ├── handler/             # Core logic: Webhook processing, HTTP handlers, and interfaces
│   ├── middleware/          # Gin middlewares (Auth, Timeout, SizeLimit, Metrics)
│   └── rabbitmq/            # AMQP infrastructure, Connection pool, and Publisher implementation
├── docker-compose.yml       # Local development environment setup
└── Dockerfile               # Production-ready multi-stage Dockerfile
```

## ⚙️ Configuration

The gateway is entirely stateless and configured via environment variables (12-Factor App methodology). Provide a `.env` file in the root directory for local development.

| Environment Variable | Default Value | Description |
|---|---|---|
| `HTTP_PORT` | `3000` | Port for the HTTP server to listen on. |
| `LOG_LEVEL` | `info` | Structured logging level (`debug`, `info`, `warn`, `error`). |
| `RABBITMQ_URL` | *(required)* | Full AMQP connection URI (e.g., `amqp://user:pass@host:5672/`). |
| `RABBITMQ_EXCHANGE` | `your_exchange_name` | The RabbitMQ Topic Exchange name to publish events to. |
| `RABBITMQ_ROUTING_KEY`| `your_routing_key` | The routing key applied to outgoing AMQP messages. |
| `WEBHOOK_SECRET` | *(required)* | The shared secret used to authenticate incoming Kratos requests. |
| `REQUEST_TIMEOUT` | `10s` | Maximum context duration allowed for a single webhook request. |
| `MAX_REQUEST_SIZE` | `1048576` | Maximum allowed payload size in bytes (prevents OOM attacks). |
| `RABMQ_MAX_RETRIES` | `5` | Maximum number of publish retries during a broker failure. |
| `RABMQ_RETRY_DELAY` | `500ms` | Base delay for the exponential backoff algorithm. |

## 🚀 API Endpoints

### Observability
- `GET /health` - Liveness probe (Returns 200 `UP`).
- `GET /ready` - Readiness probe (Returns 200 `READY`).
- `GET /metrics` - Exposes Prometheus metrics (HTTP latencies, RabbitMQ publish success/failures).

### Webhooks
- `POST /webhooks/kratos` - Ingress endpoint for Ory Kratos.
  - **Header Required**: `X-Webhook-Secret: <WEBHOOK_SECRET>`
  - **Header Required**: `Content-Type: application/json`

## 📬 Event Envelope Specification

To prevent downstream services from coupling to Kratos' specific payload structure, this gateway wraps all incoming webhooks into a standardized `Envelope`.

Consumers bound to the Exchange will receive the following JSON structure:

```json
{
  "event_id": "bf3e26a4-2060-41cf-a4b0-fb8e1fad31c2",
  "event_type": "kratos.registration",
  "source": "ory.kratos",
  "timestamp": "2026-07-03T14:56:18Z",
  "correlation_id": "ab2315a6-9ef1-4a8f-a25a-eb58354d08ec",
  "identity_id": "12345-67890",
  "payload": {
    // ... Raw, untouched Kratos payload ...
  },
  "metadata": {
    "client_ip": "172.18.0.16",
    "user_agent": "Go-http-client/1.1"
  },
  "version": "1.0"
}
```

## 🛠 Local Development & Testing

### 1. Running Locally
Ensure you have Docker and Go 1.26.4+ installed.
```bash
# Spin up RabbitMQ (and the Gateway if desired)
docker-compose up -d

# Or run the Gateway directly via Go (requires RABBITMQ_URL in env)
go run ./cmd/gateway/main.go
```

### 2. Testing & Linting
The project relies on idiomatic Go testing conventions (co-located `_test.go` files) utilizing interface mocking for isolated, fast execution.

Before opening a Pull Request, ensure you run the following commands to pass the CI pipeline:

```bash
# 1. Format your code
go fmt ./...

# 2. Run unit tests with race condition detector
go test -v -race ./...

# 3. Run the strict Code Convention linter (golangci-lint)
go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
```

### 3. Simulating a Webhook
```bash
curl -X POST http://localhost:3000/webhooks/kratos \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Secret: my-super-secret" \
  -H "X-Trace-ID: req-123" \
  -d '{
    "type": "registration",
    "identity": {
      "id": "user-abc-123",
      "traits": { "email": "dev@kleonix.com" }
    }
  }'
```

## 🔒 Security Best Practices
- **Do not commit `.env` files**. Use `.env.example` as a template for new developers.
- The `.dockerignore` file strictly prevents local binaries, IDE configurations, and secret files from being injected into the production Docker image.
